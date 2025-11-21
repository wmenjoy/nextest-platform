package workflow

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"test-management-service/internal/models"
	"test-management-service/internal/testcase"
	"test-management-service/internal/websocket"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VariableInjector interface for injecting environment variables
type VariableInjector interface {
	GetActiveEnvironmentVariables() (map[string]string, error)
}

// WorkflowExecutorImpl implements WorkflowExecutor
type WorkflowExecutorImpl struct {
	db               *gorm.DB
	actionRegistry   *ActionRegistry
	testCaseRepo     TestCaseRepository
	workflowRepo     WorkflowRepository
	unifiedExecutor  *testcase.UnifiedTestExecutor
	hub              *websocket.Hub
	variableInjector VariableInjector
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(
	db *gorm.DB,
	testCaseRepo TestCaseRepository,
	workflowRepo WorkflowRepository,
	unifiedExecutor *testcase.UnifiedTestExecutor,
	hub *websocket.Hub,
	variableInjector VariableInjector,
) *WorkflowExecutorImpl {
	executor := &WorkflowExecutorImpl{
		db:               db,
		actionRegistry:   NewActionRegistry(),
		testCaseRepo:     testCaseRepo,
		workflowRepo:     workflowRepo,
		unifiedExecutor:  unifiedExecutor,
		hub:              hub,
		variableInjector: variableInjector,
	}

	// Register built-in actions
	executor.registerBuiltinActions()

	return executor
}

func (e *WorkflowExecutorImpl) registerBuiltinActions() {
	// HTTP and Command actions will be registered here
	// TestCaseAction is registered separately
}

// Execute runs a workflow
func (e *WorkflowExecutorImpl) Execute(workflowID string, workflowDef interface{}) (*WorkflowResult, error) {
	// Step 1: Parse workflow definition
	workflow, err := e.parseWorkflowDefinition(workflowID, workflowDef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Step 2: Validate workflow (check for cycles)
	if err := e.validateWorkflow(workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Step 3: Create run record
	runID := uuid.New().String()
	run := &models.WorkflowRun{
		RunID:      runID,
		WorkflowID: workflowID,
		Status:     "running",
		StartTime:  time.Now(),
	}
	if err := e.db.Create(run).Error; err != nil {
		return nil, fmt.Errorf("failed to create run record: %w", err)
	}

	// Step 4: Initialize execution context
	ctx := &ExecutionContext{
		RunID:       runID,
		Variables:   workflow.Variables,
		StepOutputs: make(map[string]interface{}),
		StepResults: make(map[string]*StepExecutionResult),
		Logger:      NewBroadcastStepLogger(e.db, runID, e.hub),
		VarTracker:  NewDatabaseVariableChangeTracker(e.db, runID),
	}

	// Initialize variables map if nil
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]interface{})
	}

	// Merge environment variables into workflow variables
	// Environment variables serve as base, workflow variables override them
	if e.variableInjector != nil {
		envVars, err := e.variableInjector.GetActiveEnvironmentVariables()
		if err == nil && envVars != nil {
			// Create a new merged map with environment variables as base
			mergedVars := make(map[string]interface{})

			// First, add all environment variables
			for key, value := range envVars {
				mergedVars[key] = value
			}

			// Then, overlay workflow variables (these take precedence)
			for key, value := range ctx.Variables {
				mergedVars[key] = value
			}

			ctx.Variables = mergedVars
		}
	}

	// Step 5: Build DAG and get execution order
	layers, err := e.buildDAG(workflow.Steps)
	if err != nil {
		e.updateRunStatus(run, "failed", err.Error())
		return nil, fmt.Errorf("failed to build DAG: %w", err)
	}

	// Step 6: Execute steps layer by layer
	var execError error
	for _, layer := range layers {
		if err := e.executeLayer(ctx, layer, workflow.Steps); err != nil {
			execError = err
			break
		}
	}

	// Step 7: Finalize run record
	run.EndTime = time.Now()
	run.Duration = int(run.EndTime.Sub(run.StartTime).Milliseconds())

	if execError != nil {
		run.Status = "failed"
		run.Error = execError.Error()
	} else {
		run.Status = "success"
	}

	// Save context as JSON
	run.Context = models.JSONB{"variables": ctx.Variables, "outputs": ctx.StepOutputs}
	e.db.Save(run)

	// Step 8: Build result
	return e.buildWorkflowResult(ctx, run), nil
}

// parseWorkflowDefinition parses workflow from various formats
func (e *WorkflowExecutorImpl) parseWorkflowDefinition(workflowID string, workflowDef interface{}) (*WorkflowDefinition, error) {
	var workflow WorkflowDefinition

	switch def := workflowDef.(type) {
	case map[string]interface{}:
		// Direct map (from JSONB)
		data, err := json.Marshal(def)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &workflow); err != nil {
			return nil, err
		}
	case models.JSONB:
		// JSONB type
		data, err := json.Marshal(def)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &workflow); err != nil {
			return nil, err
		}
	case string:
		// JSON string
		if err := json.Unmarshal([]byte(def), &workflow); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported workflow definition type: %T", workflowDef)
	}

	if workflow.Name == "" {
		workflow.Name = workflowID
	}

	return &workflow, nil
}

// validateWorkflow checks for cycles and missing dependencies
func (e *WorkflowExecutorImpl) validateWorkflow(workflow *WorkflowDefinition) error {
	// Check all dependencies exist
	for stepID, step := range workflow.Steps {
		for _, dep := range step.DependsOn {
			if _, exists := workflow.Steps[dep]; !exists {
				return fmt.Errorf("step '%s' depends on non-existent step '%s'", stepID, dep)
			}
		}
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(stepID string) bool
	hasCycle = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true

		step := workflow.Steps[stepID]
		for _, dep := range step.DependsOn {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[stepID] = false
		return false
	}

	for stepID := range workflow.Steps {
		if !visited[stepID] {
			if hasCycle(stepID) {
				return fmt.Errorf("workflow contains cyclic dependency involving step '%s'", stepID)
			}
		}
	}

	return nil
}

// buildDAG creates execution layers using topological sort
func (e *WorkflowExecutorImpl) buildDAG(steps map[string]*WorkflowStep) ([][]string, error) {
	// Calculate in-degrees
	inDegree := make(map[string]int)
	for stepID := range steps {
		inDegree[stepID] = 0
	}
	for _, step := range steps {
		for range step.DependsOn {
			inDegree[step.ID]++
		}
	}

	// Build layers using Kahn's algorithm
	var layers [][]string
	remaining := len(steps)

	for remaining > 0 {
		// Find all steps with no dependencies
		var layer []string
		for stepID, degree := range inDegree {
			if degree == 0 {
				layer = append(layer, stepID)
			}
		}

		if len(layer) == 0 {
			return nil, fmt.Errorf("unable to resolve dependencies")
		}

		// Remove processed steps
		for _, stepID := range layer {
			delete(inDegree, stepID)
			remaining--

			// Reduce in-degree of dependent steps
			for _, step := range steps {
				for _, dep := range step.DependsOn {
					if dep == stepID {
						inDegree[step.ID]--
					}
				}
			}
		}

		layers = append(layers, layer)
	}

	return layers, nil
}

// executeLayer executes all steps in a layer (in parallel)
func (e *WorkflowExecutorImpl) executeLayer(ctx *ExecutionContext, layer []string, steps map[string]*WorkflowStep) error {
	var wg sync.WaitGroup
	errorsChan := make(chan error, len(layer))

	for _, stepID := range layer {
		step := steps[stepID]
		if step == nil {
			continue
		}

		// Check condition
		if step.When != "" && !e.evaluateCondition(step.When, ctx) {
			ctx.Logger.Info(stepID, fmt.Sprintf("Step skipped due to condition: %s", step.When))
			continue
		}

		wg.Add(1)
		go func(s *WorkflowStep) {
			defer wg.Done()
			if err := e.executeStep(ctx, s); err != nil {
				errorsChan <- fmt.Errorf("step %s failed: %w", s.ID, err)
			}
		}(step)
	}

	wg.Wait()
	close(errorsChan)

	// Return first error
	for err := range errorsChan {
		return err
	}

	return nil
}

// executeStep executes a single step
func (e *WorkflowExecutorImpl) executeStep(ctx *ExecutionContext, step *WorkflowStep) error {
	ctx.Logger.Info(step.ID, fmt.Sprintf("Starting step: %s", step.Name))

	// Broadcast step start event
	if e.hub != nil {
		e.hub.Broadcast(ctx.RunID, "step_start", map[string]interface{}{
			"stepId":   step.ID,
			"stepName": step.Name,
		})
	}

	// Create step execution record
	stepExec := &models.WorkflowStepExecution{
		RunID:     ctx.RunID,
		StepID:    step.ID,
		StepName:  step.Name,
		Status:    "running",
		StartTime: time.Now(),
	}

	// Save input data
	stepExec.InputData = models.JSONB{"input": step.Input, "config": step.Config}
	e.db.Create(stepExec)

	// Get action
	action, err := e.getActionForStep(step)
	if err != nil {
		stepExec.Status = "failed"
		stepExec.Error = err.Error()
		stepExec.EndTime = time.Now()
		stepExec.Duration = int(stepExec.EndTime.Sub(stepExec.StartTime).Milliseconds())
		e.db.Save(stepExec)
		return err
	}

	// Build action context
	actionCtx := &ActionContext{
		StepID:          step.ID,
		Variables:       ctx.Variables,
		StepOutputs:     ctx.StepOutputs,
		TestCaseRepo:    e.testCaseRepo,
		UnifiedExecutor: e.unifiedExecutor,
		Logger:          ctx.Logger,
	}

	// Execute with retry
	var result *ActionResult
	maxAttempts := 1
	if step.Retry != nil && step.Retry.MaxAttempts > 0 {
		maxAttempts = step.Retry.MaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err = action.Execute(actionCtx)
		if err == nil && result.Status == "success" {
			break
		}
		if attempt < maxAttempts {
			ctx.Logger.Warn(step.ID, fmt.Sprintf("Attempt %d failed, retrying...", attempt))
			if step.Retry != nil && step.Retry.Interval > 0 {
				time.Sleep(time.Duration(step.Retry.Interval) * time.Millisecond)
			}
		}
	}

	// Update execution record
	stepExec.EndTime = time.Now()
	stepExec.Duration = int(stepExec.EndTime.Sub(stepExec.StartTime).Milliseconds())

	if err != nil || (result != nil && result.Status == "failed") {
		stepExec.Status = "failed"
		if err != nil {
			stepExec.Error = err.Error()
		} else if result.Error != nil {
			stepExec.Error = result.Error.Error()
		}
		e.db.Save(stepExec)

		// Store step result
		ctx.StepResults[step.ID] = &StepExecutionResult{
			Status:   "failed",
			Duration: stepExec.Duration,
			Error:    stepExec.Error,
		}

		// Handle error strategy
		if step.OnError == "continue" {
			ctx.Logger.Warn(step.ID, "Step failed but continuing due to onError=continue")
			return nil
		}
		return fmt.Errorf("step execution failed")
	}

	// Success - save output
	stepExec.Status = "success"
	if result != nil && result.Output != nil {
		stepExec.OutputData = models.JSONB(result.Output)

		// Save to step outputs
		ctx.StepOutputs[step.ID] = result.Output

		// Map output variables
		if step.Output != nil {
			for varName, outputPath := range step.Output {
				if value, exists := result.Output[outputPath]; exists {
					oldValue := ctx.Variables[varName]
					ctx.Variables[varName] = value
					ctx.VarTracker.Track(step.ID, varName, oldValue, value, "update")
				}
			}
		}
	}
	e.db.Save(stepExec)

	// Store step result
	ctx.StepResults[step.ID] = &StepExecutionResult{
		Status:   "success",
		Duration: stepExec.Duration,
		Output:   result.Output,
	}

	// Broadcast step complete event
	if e.hub != nil {
		e.hub.Broadcast(ctx.RunID, "step_complete", map[string]interface{}{
			"stepId":   step.ID,
			"stepName": step.Name,
			"status":   stepExec.Status,
			"duration": stepExec.Duration,
		})
	}

	ctx.Logger.Info(step.ID, fmt.Sprintf("Step completed in %dms", stepExec.Duration))
	return nil
}

// getActionForStep returns the appropriate action for a step
func (e *WorkflowExecutorImpl) getActionForStep(step *WorkflowStep) (Action, error) {
	switch step.Type {
	case "test-case":
		// Create TestCaseAction from config
		testID, ok := step.Config["testId"].(string)
		if !ok {
			return nil, fmt.Errorf("testId not specified for test-case step")
		}
		return &TestCaseActionWrapper{
			TestID: testID,
			Input:  step.Input,
		}, nil
	case "http":
		return &HTTPActionWrapper{Config: step.Config}, nil
	case "command":
		return &CommandActionWrapper{Config: step.Config}, nil
	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// evaluateCondition evaluates a simple condition expression
func (e *WorkflowExecutorImpl) evaluateCondition(expr string, ctx *ExecutionContext) bool {
	// Simple implementation - just check if variable exists and is truthy
	// Format: "{{varName}}" or "{{stepId.output.field}}"
	// For now, always return true for non-empty expressions
	return expr != ""
}

// updateRunStatus updates the run status
func (e *WorkflowExecutorImpl) updateRunStatus(run *models.WorkflowRun, status, errorMsg string) {
	run.Status = status
	run.Error = errorMsg
	run.EndTime = time.Now()
	run.Duration = int(run.EndTime.Sub(run.StartTime).Milliseconds())
	e.db.Save(run)
}

// buildWorkflowResult builds the result from execution context
func (e *WorkflowExecutorImpl) buildWorkflowResult(ctx *ExecutionContext, run *models.WorkflowRun) *WorkflowResult {
	var stepExecutions []testcase.StepExecution
	var completedSteps, failedSteps int

	for stepID, result := range ctx.StepResults {
		stepExecutions = append(stepExecutions, testcase.StepExecution{
			StepID:     stepID,
			Status:     result.Status,
			Duration:   result.Duration,
			OutputData: result.Output,
			Error:      result.Error,
		})
		if result.Status == "success" {
			completedSteps++
		} else {
			failedSteps++
		}
	}

	return &WorkflowResult{
		RunID:          run.RunID,
		Status:         run.Status,
		StartTime:      run.StartTime,
		EndTime:        run.EndTime,
		Duration:       run.Duration,
		TotalSteps:     len(ctx.StepResults),
		CompletedSteps: completedSteps,
		FailedSteps:    failedSteps,
		StepExecutions: stepExecutions,
		Context:        ctx.Variables,
		Error:          run.Error,
	}
}

// Action wrappers for different step types

// TestCaseActionWrapper wraps test case execution
type TestCaseActionWrapper struct {
	TestID string
	Input  map[string]interface{}
}

func (a *TestCaseActionWrapper) Execute(ctx *ActionContext) (*ActionResult, error) {
	// Delegate to TestCaseAction in actions package
	ctx.Logger.Info(ctx.StepID, fmt.Sprintf("Executing test case: %s", a.TestID))

	// Load test case
	tc, err := ctx.TestCaseRepo.GetTestCase(a.TestID)
	if err != nil {
		return nil, err
	}

	// Convert to testcase.TestCase and execute
	testCase := &testcase.TestCase{
		ID:   tc.TestID,
		Name: tc.Name,
		Type: tc.Type,
	}

	// Apply HTTP/Command config based on type
	switch tc.Type {
	case "http":
		var httpConfig testcase.HTTPTest
		data, _ := json.Marshal(tc.HTTPConfig)
		json.Unmarshal(data, &httpConfig)
		testCase.HTTP = &httpConfig
	case "command":
		var cmdConfig testcase.CommandTest
		data, _ := json.Marshal(tc.CommandConfig)
		json.Unmarshal(data, &cmdConfig)
		testCase.Command = &cmdConfig
	}

	// Execute
	result := ctx.UnifiedExecutor.Execute(testCase)

	if result.Status != "passed" {
		return &ActionResult{
			Status: "failed",
			Error:  fmt.Errorf("test failed: %s", result.Error),
		}, nil
	}

	return &ActionResult{
		Status: "success",
		Output: map[string]interface{}{
			"status":   result.Status,
			"response": result.Response,
			"duration": result.Duration.Milliseconds(),
		},
		Duration: int(result.Duration.Milliseconds()),
	}, nil
}

func (a *TestCaseActionWrapper) Validate() error {
	if a.TestID == "" {
		return fmt.Errorf("testId is required")
	}
	return nil
}

// HTTPActionWrapper wraps HTTP execution
type HTTPActionWrapper struct {
	Config map[string]interface{}
}

func (a *HTTPActionWrapper) Execute(ctx *ActionContext) (*ActionResult, error) {
	// Create HTTP test case and execute via UnifiedExecutor
	testCase := &testcase.TestCase{
		ID:   ctx.StepID,
		Name: ctx.StepID,
		Type: "http",
	}

	var httpConfig testcase.HTTPTest
	data, _ := json.Marshal(a.Config)
	json.Unmarshal(data, &httpConfig)
	testCase.HTTP = &httpConfig

	result := ctx.UnifiedExecutor.Execute(testCase)

	if result.Status != "passed" {
		return &ActionResult{
			Status: "failed",
			Error:  fmt.Errorf("HTTP request failed: %s", result.Error),
		}, nil
	}

	return &ActionResult{
		Status: "success",
		Output: map[string]interface{}{
			"status":   result.Status,
			"response": result.Response,
		},
		Duration: int(result.Duration.Milliseconds()),
	}, nil
}

func (a *HTTPActionWrapper) Validate() error {
	return nil
}

// CommandActionWrapper wraps command execution
type CommandActionWrapper struct {
	Config map[string]interface{}
}

func (a *CommandActionWrapper) Execute(ctx *ActionContext) (*ActionResult, error) {
	testCase := &testcase.TestCase{
		ID:   ctx.StepID,
		Name: ctx.StepID,
		Type: "command",
	}

	var cmdConfig testcase.CommandTest
	data, _ := json.Marshal(a.Config)
	json.Unmarshal(data, &cmdConfig)
	testCase.Command = &cmdConfig

	result := ctx.UnifiedExecutor.Execute(testCase)

	if result.Status != "passed" {
		return &ActionResult{
			Status: "failed",
			Error:  fmt.Errorf("command failed: %s", result.Error),
		}, nil
	}

	return &ActionResult{
		Status: "success",
		Output: map[string]interface{}{
			"status":   result.Status,
			"response": result.Response,
		},
		Duration: int(result.Duration.Milliseconds()),
	}, nil
}

func (a *CommandActionWrapper) Validate() error {
	return nil
}

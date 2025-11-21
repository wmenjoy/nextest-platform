// Package testcase provides test case execution functionality
package testcase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"test-management-service/internal/models"
)

// VariableInjector interface for injecting environment variables
type VariableInjector interface {
	InjectHTTPVariables(config *HTTPTest) error
	InjectCommandVariables(config *CommandTest) error
}

// UnifiedTestExecutor executes test cases of all types (http, command, workflow, etc.)
type UnifiedTestExecutor struct {
	baseURL          string
	client           *http.Client
	workflowExecutor WorkflowExecutor   // Interface for workflow execution
	testCaseRepo     TestCaseRepository // Repository for test case data
	workflowRepo     WorkflowRepository // Repository for workflow data
	variableInjector VariableInjector   // Injector for environment variables
}

// WorkflowExecutor interface for workflow execution
type WorkflowExecutor interface {
	Execute(workflowID string, workflowDef interface{}) (*WorkflowResult, error)
}

// WorkflowResult represents the result of a workflow execution
type WorkflowResult struct {
	RunID            string
	Status           string // success, failed, cancelled
	StartTime        time.Time
	EndTime          time.Time
	Duration         int
	TotalSteps       int
	CompletedSteps   int
	FailedSteps      int
	StepExecutions   []StepExecution
	Context          map[string]interface{}
	Error            string
}

// StepExecution represents a single step execution result
type StepExecution struct {
	StepID     string
	StepName   string
	Status     string
	Duration   int
	InputData  map[string]interface{}
	OutputData map[string]interface{}
	Error      string
}

// TestCaseRepository provides access to test case data
type TestCaseRepository interface {
	GetTestCase(testID string) (*models.TestCase, error)
}

// WorkflowRepository provides access to workflow data
type WorkflowRepository interface {
	GetWorkflow(workflowID string) (*models.Workflow, error)
}

// NewUnifiedTestExecutor creates a new unified test executor
func NewUnifiedTestExecutor(baseURL string, workflowExecutor WorkflowExecutor, testCaseRepo TestCaseRepository, workflowRepo WorkflowRepository) *UnifiedTestExecutor {
	return &UnifiedTestExecutor{
		baseURL:          baseURL,
		workflowExecutor: workflowExecutor,
		testCaseRepo:     testCaseRepo,
		workflowRepo:     workflowRepo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewExecutorWithInjector creates a new unified test executor with variable injector
func NewExecutorWithInjector(baseURL string, workflowExecutor WorkflowExecutor, testCaseRepo TestCaseRepository, workflowRepo WorkflowRepository, variableInjector VariableInjector) *UnifiedTestExecutor {
	return &UnifiedTestExecutor{
		baseURL:          baseURL,
		workflowExecutor: workflowExecutor,
		testCaseRepo:     testCaseRepo,
		workflowRepo:     workflowRepo,
		variableInjector: variableInjector,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Backward compatibility: NewExecutor is an alias for NewUnifiedTestExecutor
func NewExecutor(baseURL string) *UnifiedTestExecutor {
	return NewUnifiedTestExecutor(baseURL, nil, nil, nil)
}

// Execute runs a test case with lifecycle hooks (unified entry point)
func (e *UnifiedTestExecutor) Execute(tc *TestCase) *TestResult {
	result := &TestResult{
		TestID:    tc.ID,
		Name:      tc.Name,
		StartTime: time.Now(),
		Status:    "passed",
	}

	// Context for storing hook responses
	ctx := make(map[string]interface{})

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		// Run teardown hooks (always execute, even on failure)
		e.executeTeardownHooks(tc, result, ctx)
	}()

	// Run setup hooks
	if !e.executeSetupHooks(tc, result, ctx) {
		// Setup failed, skip test execution
		return result
	}

	// Execute the main test
	switch tc.Type {
	case "http":
		e.executeHTTP(tc, result)
	case "command":
		e.executeCommand(tc, result)
	case "workflow":
		e.executeWorkflowTest(tc, result)
	default:
		result.Status = "error"
		result.Error = fmt.Sprintf("unsupported test type: %s", tc.Type)
	}

	return result
}

// executeHTTP executes an HTTP test
func (e *UnifiedTestExecutor) executeHTTP(tc *TestCase, result *TestResult) {
	if tc.HTTP == nil {
		result.Status = "error"
		result.Error = "HTTP configuration missing"
		return
	}

	// Inject environment variables if variableInjector is available
	if e.variableInjector != nil {
		if err := e.variableInjector.InjectHTTPVariables(tc.HTTP); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("failed to inject variables: %v", err)
			return
		}
	}

	// Prepare request body
	var bodyReader io.Reader
	if tc.HTTP.Body != nil {
		bodyBytes, err := json.Marshal(tc.HTTP.Body)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("failed to marshal body: %v", err)
			return
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Build URL
	url := e.baseURL + tc.HTTP.Path
	req, err := http.NewRequest(tc.HTTP.Method, url, bodyReader)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	// Set headers
	for k, v := range tc.HTTP.Headers {
		req.Header.Set(k, v)
	}

	// Store request info
	result.Request = map[string]interface{}{
		"method":  tc.HTTP.Method,
		"url":     url,
		"headers": tc.HTTP.Headers,
		"body":    tc.HTTP.Body,
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, _ := io.ReadAll(resp.Body)
	var responseBody map[string]interface{}
	json.Unmarshal(bodyBytes, &responseBody)

	result.Response = map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    resp.Header,
		"body":       responseBody,
		"bodyRaw":    string(bodyBytes),
	}

	// Run assertions
	e.runHTTPAssertions(tc.Assertions, resp.StatusCode, responseBody, result)
}

// executeCommand executes a command test
func (e *UnifiedTestExecutor) executeCommand(tc *TestCase, result *TestResult) {
	if tc.Command == nil {
		result.Status = "error"
		result.Error = "Command configuration missing"
		return
	}

	// Inject environment variables if variableInjector is available
	if e.variableInjector != nil {
		if err := e.variableInjector.InjectCommandVariables(tc.Command); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("failed to inject variables: %v", err)
			return
		}
	}

	cmd := exec.Command(tc.Command.Cmd, tc.Command.Args...)
	if tc.Command.Cwd != "" {
		cmd.Dir = tc.Command.Cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout
	timeout := 60 * time.Second
	if tc.Command.Timeout > 0 {
		timeout = time.Duration(tc.Command.Timeout) * time.Second
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				result.Status = "error"
				result.Error = fmt.Sprintf("command failed: %v", err)
				return
			}
		}

		result.Response = map[string]interface{}{
			"exitCode": exitCode,
			"stdout":   stdout.String(),
			"stderr":   stderr.String(),
		}

		// Run assertions
		e.runCommandAssertions(tc.Assertions, exitCode, stdout.String(), result)

	case <-time.After(timeout):
		cmd.Process.Kill()
		result.Status = "failed"
		result.Failures = append(result.Failures, fmt.Sprintf("command timeout after %v", timeout))
	}
}

// executeWorkflowTest executes a workflow-type test case
func (e *UnifiedTestExecutor) executeWorkflowTest(tc *TestCase, result *TestResult) {
	// Step 1: Determine Mode 1 (workflowId) or Mode 2 (workflowDef)
	var workflowID string
	var workflowDef interface{}

	if tc.WorkflowID != "" {
		// Mode 1: Reference workflow
		workflowID = tc.WorkflowID

		// Load workflow from database
		if e.workflowRepo == nil {
			result.Status = "error"
			result.Error = "workflow repository not configured"
			return
		}

		workflow, err := e.workflowRepo.GetWorkflow(workflowID)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("failed to load workflow: %v", err)
			return
		}

		// Convert models.JSONB to interface{}
		workflowDef = map[string]interface{}(workflow.Definition)
	} else if tc.WorkflowDef != nil {
		// Mode 2: Embedded workflow definition
		workflowID = fmt.Sprintf("inline-%s", tc.ID)
		workflowDef = tc.WorkflowDef
	} else {
		result.Status = "error"
		result.Error = "no workflow definition found (missing workflowId or workflowDef)"
		return
	}

	// Step 2: Check workflowExecutor availability
	if e.workflowExecutor == nil {
		result.Status = "error"
		result.Error = "workflow executor not configured"
		return
	}

	// Step 3: Execute workflow
	workflowResult, err := e.workflowExecutor.Execute(workflowID, workflowDef)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("workflow execution failed: %v", err)
		return
	}

	// Step 4: Convert WorkflowResult to TestResult
	result.Status = convertWorkflowStatusToTestStatus(workflowResult.Status)
	result.Response = map[string]interface{}{
		"workflowRunId":   workflowResult.RunID,
		"totalSteps":      workflowResult.TotalSteps,
		"completedSteps":  workflowResult.CompletedSteps,
		"failedSteps":     workflowResult.FailedSteps,
		"stepExecutions":  workflowResult.StepExecutions,
		"context":         workflowResult.Context,
	}

	if workflowResult.Error != "" {
		result.Error = workflowResult.Error
	}
}

// convertWorkflowStatusToTestStatus converts workflow status to test status
func convertWorkflowStatusToTestStatus(workflowStatus string) string {
	switch workflowStatus {
	case "success":
		return "passed"
	case "failed":
		return "failed"
	case "cancelled":
		return "skipped"
	default:
		return "error"
	}
}

// runHTTPAssertions runs HTTP assertions
func (e *UnifiedTestExecutor) runHTTPAssertions(assertions []Assertion, statusCode int, body map[string]interface{}, result *TestResult) {
	for _, assertion := range assertions {
		switch assertion.Type {
		case "status_code":
			if !e.checkStatusCode(assertion, statusCode) {
				result.Status = "failed"
				result.Failures = append(result.Failures,
					fmt.Sprintf("status code: expected %v, got %d", assertion.Expected, statusCode))
			}

		case "json_path":
			if !e.checkJSONPath(assertion, body, result) {
				result.Status = "failed"
			}
		}
	}
}

// runCommandAssertions runs command assertions
func (e *UnifiedTestExecutor) runCommandAssertions(assertions []Assertion, exitCode int, stdout string, result *TestResult) {
	for _, assertion := range assertions {
		switch assertion.Type {
		case "exit_code":
			expected, ok := assertion.Expected.(float64)
			if !ok {
				expectedInt, ok := assertion.Expected.(int)
				if ok {
					expected = float64(expectedInt)
				}
			}
			if exitCode != int(expected) {
				result.Status = "failed"
				result.Failures = append(result.Failures,
					fmt.Sprintf("exit code: expected %v, got %d", assertion.Expected, exitCode))
			}

		case "stdout_contains":
			expected, ok := assertion.Expected.(string)
			if !ok || !strings.Contains(stdout, expected) {
				result.Status = "failed"
				result.Failures = append(result.Failures,
					fmt.Sprintf("stdout should contain: %v", assertion.Expected))
			}
		}
	}
}

// checkStatusCode checks status code assertion
func (e *UnifiedTestExecutor) checkStatusCode(assertion Assertion, actualCode int) bool {
	if assertion.Operator == "in" {
		// Expected is an array of valid status codes
		if arr, ok := assertion.Expected.([]interface{}); ok {
			for _, v := range arr {
				if code, ok := v.(float64); ok && int(code) == actualCode {
					return true
				}
			}
			return false
		}
	}

	// Exact match
	if expected, ok := assertion.Expected.(float64); ok {
		return int(expected) == actualCode
	}
	if expected, ok := assertion.Expected.(int); ok {
		return expected == actualCode
	}

	return false
}

// checkJSONPath checks JSON path assertion
func (e *UnifiedTestExecutor) checkJSONPath(assertion Assertion, body map[string]interface{}, result *TestResult) bool {
	// Simple JSON path implementation (only supports $.field notation)
	path := strings.TrimPrefix(assertion.Path, "$.")

	var value interface{}
	if strings.Contains(path, ".") {
		// Nested path
		parts := strings.Split(path, ".")
		current := body
		for i, part := range parts {
			if i == len(parts)-1 {
				value = current[part]
			} else {
				if next, ok := current[part].(map[string]interface{}); ok {
					current = next
				} else {
					result.Failures = append(result.Failures,
						fmt.Sprintf("JSON path %s not found", assertion.Path))
					return false
				}
			}
		}
	} else {
		value = body[path]
	}

	if assertion.Operator == "exists" {
		if value == nil {
			result.Failures = append(result.Failures,
				fmt.Sprintf("JSON path %s should exist", assertion.Path))
			return false
		}
		return true
	}

	// Exact match
	if value != assertion.Expected {
		result.Failures = append(result.Failures,
			fmt.Sprintf("JSON path %s: expected %v, got %v", assertion.Path, assertion.Expected, value))
		return false
	}

	return true
}

// executeSetupHooks runs setup hooks before test execution
func (e *UnifiedTestExecutor) executeSetupHooks(tc *TestCase, result *TestResult, ctx map[string]interface{}) bool {
	for _, hook := range tc.SetupHooks {
		if !e.executeHook(&hook, "setup", result, ctx) {
			// Setup hook failed
			if !hook.ContinueOnError {
				result.Status = "error"
				result.Error = fmt.Sprintf("Setup hook '%s' failed", hook.Name)
				return false
			}
		}
	}
	return true
}

// executeTeardownHooks runs teardown hooks after test execution
func (e *UnifiedTestExecutor) executeTeardownHooks(tc *TestCase, result *TestResult, ctx map[string]interface{}) {
	for _, hook := range tc.TeardownHooks {
		// Check if hook should run on failure
		if result.Status == "failed" || result.Status == "error" {
			if !hook.RunOnFailure {
				continue
			}
		}

		if !e.executeHook(&hook, "teardown", result, ctx) {
			// Teardown hook failed, but don't fail the test
			if result.Status == "passed" && !hook.ContinueOnError {
				result.Status = "failed"
				result.Failures = append(result.Failures,
					fmt.Sprintf("Teardown hook '%s' failed", hook.Name))
			}
		}
	}
}

// executeHook executes a single hook
func (e *UnifiedTestExecutor) executeHook(hook *Hook, phase string, result *TestResult, ctx map[string]interface{}) bool {
	fmt.Printf("[%s hook] Executing: %s (type: %s)\n", phase, hook.Name, hook.Type)

	switch hook.Type {
	case "http":
		return e.executeHTTPHook(hook, result, ctx)
	case "command":
		return e.executeCommandHook(hook, result, ctx)
	default:
		fmt.Printf("[%s hook] Unknown hook type: %s\n", phase, hook.Type)
		return false
	}
}

// executeHTTPHook executes an HTTP hook
func (e *UnifiedTestExecutor) executeHTTPHook(hook *Hook, result *TestResult, ctx map[string]interface{}) bool {
	if hook.HTTP == nil {
		return false
	}

	// Prepare request body
	var bodyReader io.Reader
	if hook.HTTP.Body != nil {
		bodyBytes, err := json.Marshal(hook.HTTP.Body)
		if err != nil {
			fmt.Printf("[HTTP hook] Failed to marshal body: %v\n", err)
			return false
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Build URL
	url := e.baseURL + hook.HTTP.Path
	req, err := http.NewRequest(hook.HTTP.Method, url, bodyReader)
	if err != nil {
		fmt.Printf("[HTTP hook] Failed to create request: %v\n", err)
		return false
	}

	// Set headers
	for k, v := range hook.HTTP.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		fmt.Printf("[HTTP hook] Request failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, _ := io.ReadAll(resp.Body)
	var responseBody map[string]interface{}
	json.Unmarshal(bodyBytes, &responseBody)

	// Save response if requested
	if hook.SaveResponse != "" {
		ctx[hook.SaveResponse] = map[string]interface{}{
			"statusCode": resp.StatusCode,
			"body":       responseBody,
			"bodyRaw":    string(bodyBytes),
		}
		fmt.Printf("[HTTP hook] Saved response to context: %s\n", hook.SaveResponse)
	}

	// Consider 2xx status codes as success
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		fmt.Printf("[HTTP hook] Failed with status code: %d\n", resp.StatusCode)
	}

	return success
}

// executeCommandHook executes a command hook
func (e *UnifiedTestExecutor) executeCommandHook(hook *Hook, result *TestResult, ctx map[string]interface{}) bool {
	if hook.Command == nil {
		return false
	}

	cmd := exec.Command(hook.Command.Cmd, hook.Command.Args...)
	if hook.Command.Cwd != "" {
		cmd.Dir = hook.Command.Cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout
	timeout := 60 * time.Second
	if hook.Command.Timeout > 0 {
		timeout = time.Duration(hook.Command.Timeout) * time.Second
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				fmt.Printf("[Command hook] Failed: %v\n", err)
				return false
			}
		}

		// Save response if requested
		if hook.SaveResponse != "" {
			ctx[hook.SaveResponse] = map[string]interface{}{
				"exitCode": exitCode,
				"stdout":   stdout.String(),
				"stderr":   stderr.String(),
			}
			fmt.Printf("[Command hook] Saved response to context: %s\n", hook.SaveResponse)
		}

		success := exitCode == 0
		if !success {
			fmt.Printf("[Command hook] Failed with exit code: %d\n", exitCode)
		}
		return success

	case <-time.After(timeout):
		cmd.Process.Kill()
		fmt.Printf("[Command hook] Timeout after %v\n", timeout)
		return false
	}
}


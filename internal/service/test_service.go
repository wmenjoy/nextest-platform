package service

import (
	"encoding/json"
	"fmt"
	"time"

	"test-management-service/internal/models"
	"test-management-service/internal/repository"
	"test-management-service/internal/testcase"
)

// TestService 测试服务接口
type TestService interface {
	// Test Case operations
	CreateTestCase(req *CreateTestCaseRequest) (*models.TestCase, error)
	UpdateTestCase(testID string, req *UpdateTestCaseRequest) (*models.TestCase, error)
	DeleteTestCase(testID string) error
	GetTestCase(testID string) (*models.TestCase, error)
	ListTestCases(limit, offset int) ([]models.TestCase, int64, error)
	SearchTestCases(query string) ([]models.TestCase, error)

	// Test Group operations
	CreateTestGroup(req *CreateTestGroupRequest) (*models.TestGroup, error)
	UpdateTestGroup(groupID string, req *UpdateTestGroupRequest) (*models.TestGroup, error)
	DeleteTestGroup(groupID string) error
	GetTestGroup(groupID string) (*models.TestGroup, error)
	GetTestGroupTree() ([]models.TestGroup, error)

	// Test execution
	ExecuteTest(testID string) (*models.TestResult, error)
	ExecuteTestGroup(groupID string) (*models.TestRun, error)

	// Test results
	GetTestResult(id uint) (*models.TestResult, error)
	GetTestHistory(testID string, limit int) ([]models.TestResult, error)

	// Test runs
	GetTestRun(runID string) (*models.TestRun, error)
	ListTestRuns(limit, offset int) ([]models.TestRun, int64, error)
}

type testService struct {
	caseRepo   repository.TestCaseRepository
	groupRepo  repository.TestGroupRepository
	resultRepo repository.TestResultRepository
	runRepo    repository.TestRunRepository
	executor   *testcase.UnifiedTestExecutor
}

// NewTestService creates a new test service
func NewTestService(
	caseRepo repository.TestCaseRepository,
	groupRepo repository.TestGroupRepository,
	resultRepo repository.TestResultRepository,
	runRepo repository.TestRunRepository,
	executor *testcase.UnifiedTestExecutor,
) TestService {
	return &testService{
		caseRepo:   caseRepo,
		groupRepo:  groupRepo,
		resultRepo: resultRepo,
		runRepo:    runRepo,
		executor:   executor,
	}
}

// ===== Request/Response DTOs =====

type CreateTestCaseRequest struct {
	TestID        string                 `json:"testId" binding:"required"`
	GroupID       string                 `json:"groupId" binding:"required"`
	Name          string                 `json:"name" binding:"required"`
	Type          string                 `json:"type" binding:"required"` // Now includes "workflow"
	Priority      string                 `json:"priority"`
	Status        string                 `json:"status"`
	Objective     string                 `json:"objective"`
	Timeout       int                    `json:"timeout"`

	// Workflow integration (NEW)
	WorkflowID    string                 `json:"workflowId,omitempty"`    // Mode 1: Reference workflow
	WorkflowDef   map[string]interface{} `json:"workflowDef,omitempty"`   // Mode 2: Embedded workflow

	// Existing fields
	HTTP          map[string]interface{} `json:"http"`
	Command       map[string]interface{} `json:"command"`
	Integration   map[string]interface{} `json:"integration"`
	Assertions    []interface{}          `json:"assertions"`
	Tags          []interface{}          `json:"tags"`
	SetupHooks    []interface{}          `json:"setupHooks"`
	TeardownHooks []interface{}          `json:"teardownHooks"`
}

type UpdateTestCaseRequest struct {
	Name          string                 `json:"name"`
	Priority      string                 `json:"priority"`
	Status        string                 `json:"status"`
	Objective     string                 `json:"objective"`
	Timeout       int                    `json:"timeout"`

	// Workflow integration (NEW)
	WorkflowID    string                 `json:"workflowId,omitempty"`
	WorkflowDef   map[string]interface{} `json:"workflowDef,omitempty"`

	HTTP          map[string]interface{} `json:"http"`
	Command       map[string]interface{} `json:"command"`
	Assertions    []interface{}          `json:"assertions"`
	Tags          []interface{}          `json:"tags"`
	SetupHooks    []interface{}          `json:"setupHooks"`
	TeardownHooks []interface{}          `json:"teardownHooks"`
}

type CreateTestGroupRequest struct {
	GroupID     string `json:"groupId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ParentID    string `json:"parentId"`
	Description string `json:"description"`
	TargetHost  string `json:"targetHost"` // 测试目标服务地址
}

type UpdateTestGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetHost  string `json:"targetHost"` // 测试目标服务地址
}

// ===== Test Case Operations =====

func (s *testService) CreateTestCase(req *CreateTestCaseRequest) (*models.TestCase, error) {
	// Validate workflow test configuration
	if req.Type == "workflow" {
		if req.WorkflowID == "" && req.WorkflowDef == nil {
			return nil, fmt.Errorf("workflow test must have either workflowId or workflowDef")
		}
		if req.WorkflowID != "" && req.WorkflowDef != nil {
			return nil, fmt.Errorf("workflow test cannot have both workflowId and workflowDef")
		}
	}

	tc := &models.TestCase{
		TestID:    req.TestID,
		GroupID:   req.GroupID,
		Name:      req.Name,
		Type:      req.Type,
		Priority:  req.Priority,
		Status:    req.Status,
		Objective: req.Objective,
		Timeout:   req.Timeout,
	}

	if req.HTTP != nil {
		tc.HTTPConfig = req.HTTP
	}
	if req.Command != nil {
		tc.CommandConfig = req.Command
	}
	if req.Integration != nil {
		tc.IntegrationConfig = req.Integration
	}
	if req.Assertions != nil {
		tc.Assertions = req.Assertions
	}
	if req.Tags != nil {
		tc.Tags = req.Tags
	}
	if req.SetupHooks != nil {
		tc.SetupHooks = req.SetupHooks
	}
	if req.TeardownHooks != nil {
		tc.TeardownHooks = req.TeardownHooks
	}

	// Workflow integration
	if req.WorkflowID != "" {
		tc.WorkflowID = req.WorkflowID
	}
	if req.WorkflowDef != nil {
		tc.WorkflowDef = models.JSONB(req.WorkflowDef)
	}

	if err := s.caseRepo.Create(tc); err != nil {
		return nil, fmt.Errorf("failed to create test case: %w", err)
	}

	return tc, nil
}

func (s *testService) UpdateTestCase(testID string, req *UpdateTestCaseRequest) (*models.TestCase, error) {
	tc, err := s.caseRepo.FindByID(testID)
	if err != nil {
		return nil, fmt.Errorf("failed to find test case: %w", err)
	}
	if tc == nil {
		return nil, fmt.Errorf("test case not found: %s", testID)
	}

	if req.Name != "" {
		tc.Name = req.Name
	}
	if req.Priority != "" {
		tc.Priority = req.Priority
	}
	if req.Status != "" {
		tc.Status = req.Status
	}
	if req.Objective != "" {
		tc.Objective = req.Objective
	}
	if req.Timeout > 0 {
		tc.Timeout = req.Timeout
	}
	if req.HTTP != nil {
		tc.HTTPConfig = req.HTTP
	}
	if req.Command != nil {
		tc.CommandConfig = req.Command
	}
	if req.Assertions != nil {
		tc.Assertions = req.Assertions
	}
	if req.Tags != nil {
		tc.Tags = req.Tags
	}
	if req.SetupHooks != nil {
		tc.SetupHooks = req.SetupHooks
	}
	if req.TeardownHooks != nil {
		tc.TeardownHooks = req.TeardownHooks
	}

	// Workflow integration
	if req.WorkflowID != "" {
		tc.WorkflowID = req.WorkflowID
	}
	if req.WorkflowDef != nil {
		tc.WorkflowDef = models.JSONB(req.WorkflowDef)
	}

	if err := s.caseRepo.Update(tc); err != nil {
		return nil, fmt.Errorf("failed to update test case: %w", err)
	}

	return tc, nil
}

func (s *testService) DeleteTestCase(testID string) error {
	return s.caseRepo.Delete(testID)
}

func (s *testService) GetTestCase(testID string) (*models.TestCase, error) {
	return s.caseRepo.FindByID(testID)
}

func (s *testService) ListTestCases(limit, offset int) ([]models.TestCase, int64, error) {
	return s.caseRepo.FindAll(limit, offset)
}

func (s *testService) SearchTestCases(query string) ([]models.TestCase, error) {
	return s.caseRepo.Search(query)
}

// ===== Test Group Operations =====

func (s *testService) CreateTestGroup(req *CreateTestGroupRequest) (*models.TestGroup, error) {
	group := &models.TestGroup{
		GroupID:     req.GroupID,
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		TargetHost:  req.TargetHost,
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, fmt.Errorf("failed to create test group: %w", err)
	}

	return group, nil
}

func (s *testService) UpdateTestGroup(groupID string, req *UpdateTestGroupRequest) (*models.TestGroup, error) {
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find test group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("test group not found: %s", groupID)
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	// Allow clearing targetHost by setting to empty string
	group.TargetHost = req.TargetHost

	if err := s.groupRepo.Update(group); err != nil {
		return nil, fmt.Errorf("failed to update test group: %w", err)
	}

	return group, nil
}

func (s *testService) DeleteTestGroup(groupID string) error {
	return s.groupRepo.Delete(groupID)
}

func (s *testService) GetTestGroup(groupID string) (*models.TestGroup, error) {
	return s.groupRepo.FindByID(groupID)
}

func (s *testService) GetTestGroupTree() ([]models.TestGroup, error) {
	return s.groupRepo.GetTree()
}

// ===== Test Execution =====

func (s *testService) ExecuteTest(testID string) (*models.TestResult, error) {
	// Get test case from database
	tc, err := s.caseRepo.FindByID(testID)
	if err != nil {
		return nil, fmt.Errorf("failed to find test case: %w", err)
	}
	if tc == nil {
		return nil, fmt.Errorf("test case not found: %s", testID)
	}

	// Get the test group to check for custom target host
	executor := s.executor
	if tc.GroupID != "" {
		group, err := s.groupRepo.FindByID(tc.GroupID)
		if err == nil && group != nil && group.TargetHost != "" {
			// Use group-specific target host
			executor = testcase.NewExecutor(group.TargetHost)
		}
	}

	// Convert to executor format
	execTC := s.convertToExecutorTestCase(tc)

	// Execute test
	result := executor.Execute(execTC)

	// Convert result to model and save
	dbResult := s.convertToModelResult(result)
	if err := s.resultRepo.Create(dbResult); err != nil {
		return nil, fmt.Errorf("failed to save test result: %w", err)
	}

	return dbResult, nil
}

func (s *testService) ExecuteTestGroup(groupID string) (*models.TestRun, error) {
	// Get all tests in group
	tests, err := s.caseRepo.FindByGroupID(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find tests in group: %w", err)
	}

	// Get the test group to check for custom target host
	executor := s.executor
	group, err := s.groupRepo.FindByID(groupID)
	if err == nil && group != nil && group.TargetHost != "" {
		// Use group-specific target host
		executor = testcase.NewExecutor(group.TargetHost)
	}

	// Create test run
	runID := fmt.Sprintf("run-%d", time.Now().Unix())
	run := &models.TestRun{
		RunID:     runID,
		Total:     len(tests),
		StartTime: time.Now(),
		Status:    "running",
	}

	if err := s.runRepo.Create(run); err != nil {
		return nil, fmt.Errorf("failed to create test run: %w", err)
	}

	// Execute each test
	for _, tc := range tests {
		execTC := s.convertToExecutorTestCase(&tc)
		result := executor.Execute(execTC)

		dbResult := s.convertToModelResult(result)
		dbResult.RunID = runID

		if err := s.resultRepo.Create(dbResult); err != nil {
			fmt.Printf("failed to save result for test %s: %v\n", tc.TestID, err)
			continue
		}

		// Update run statistics
		switch result.Status {
		case "passed":
			run.Passed++
		case "failed":
			run.Failed++
		case "error":
			run.Errors++
		}
	}

	// Update run status
	run.EndTime = time.Now()
	run.Duration = int(run.EndTime.Sub(run.StartTime).Milliseconds())
	run.Status = "completed"

	if err := s.runRepo.Update(run); err != nil {
		return nil, fmt.Errorf("failed to update test run: %w", err)
	}

	return run, nil
}

// ===== Test Results =====

func (s *testService) GetTestResult(id uint) (*models.TestResult, error) {
	return s.resultRepo.FindByID(id)
}

func (s *testService) GetTestHistory(testID string, limit int) ([]models.TestResult, error) {
	return s.resultRepo.FindByTestID(testID, limit)
}

// ===== Test Runs =====

func (s *testService) GetTestRun(runID string) (*models.TestRun, error) {
	return s.runRepo.FindByID(runID)
}

func (s *testService) ListTestRuns(limit, offset int) ([]models.TestRun, int64, error) {
	return s.runRepo.FindAll(limit, offset)
}

// ===== Helper Methods =====

func (s *testService) convertToExecutorTestCase(tc *models.TestCase) *testcase.TestCase {
	execTC := &testcase.TestCase{
		ID:      tc.TestID,
		Name:    tc.Name,
		Type:    tc.Type,
		GroupID: tc.GroupID,
	}

	// Workflow integration
	execTC.WorkflowID = tc.WorkflowID
	if tc.WorkflowDef != nil {
		execTC.WorkflowDef = tc.WorkflowDef
	}

	// Convert HTTP config
	if tc.HTTPConfig != nil {
		execTC.HTTP = &testcase.HTTPTest{}
		if method, ok := tc.HTTPConfig["method"].(string); ok {
			execTC.HTTP.Method = method
		}
		if path, ok := tc.HTTPConfig["path"].(string); ok {
			execTC.HTTP.Path = path
		}
		if headers, ok := tc.HTTPConfig["headers"].(map[string]interface{}); ok {
			execTC.HTTP.Headers = make(map[string]string)
			for k, v := range headers {
				if str, ok := v.(string); ok {
					execTC.HTTP.Headers[k] = str
				}
			}
		}
		if body, ok := tc.HTTPConfig["body"].(map[string]interface{}); ok {
			execTC.HTTP.Body = body
		}
	}

	// Convert Command config
	if tc.CommandConfig != nil {
		execTC.Command = &testcase.CommandTest{}
		if cmd, ok := tc.CommandConfig["cmd"].(string); ok {
			execTC.Command.Cmd = cmd
		}
		if args, ok := tc.CommandConfig["args"].([]interface{}); ok {
			for _, arg := range args {
				if str, ok := arg.(string); ok {
					execTC.Command.Args = append(execTC.Command.Args, str)
				}
			}
		}
		if timeout, ok := tc.CommandConfig["timeout"].(float64); ok {
			execTC.Command.Timeout = int(timeout)
		}
	}

	// Convert Assertions
	if tc.Assertions != nil {
		for _, a := range tc.Assertions {
			if assertMap, ok := a.(map[string]interface{}); ok {
				assertion := testcase.Assertion{}
				if aType, ok := assertMap["type"].(string); ok {
					assertion.Type = aType
				}
				if path, ok := assertMap["path"].(string); ok {
					assertion.Path = path
				}
				if expected := assertMap["expected"]; expected != nil {
					assertion.Expected = expected
				}
				if operator, ok := assertMap["operator"].(string); ok {
					assertion.Operator = operator
				}
				execTC.Assertions = append(execTC.Assertions, assertion)
			}
		}
	}

	// Convert Setup Hooks
	if tc.SetupHooks != nil {
		for _, h := range tc.SetupHooks {
			if hookMap, ok := h.(map[string]interface{}); ok {
				hook := testcase.Hook{}
				if hType, ok := hookMap["type"].(string); ok {
					hook.Type = hType
				}
				if name, ok := hookMap["name"].(string); ok {
					hook.Name = name
				}
				if saveResponse, ok := hookMap["saveResponse"].(string); ok {
					hook.SaveResponse = saveResponse
				}
				if runOnFailure, ok := hookMap["runOnFailure"].(bool); ok {
					hook.RunOnFailure = runOnFailure
				}
				if continueOnError, ok := hookMap["continueOnError"].(bool); ok {
					hook.ContinueOnError = continueOnError
				}

				// Convert HTTP config for hook
				if httpConfig, ok := hookMap["http"].(map[string]interface{}); ok {
					hook.HTTP = &testcase.HTTPTest{}
					if method, ok := httpConfig["method"].(string); ok {
						hook.HTTP.Method = method
					}
					if path, ok := httpConfig["path"].(string); ok {
						hook.HTTP.Path = path
					}
					if headers, ok := httpConfig["headers"].(map[string]interface{}); ok {
						hook.HTTP.Headers = make(map[string]string)
						for k, v := range headers {
							if str, ok := v.(string); ok {
								hook.HTTP.Headers[k] = str
							}
						}
					}
					if body, ok := httpConfig["body"].(map[string]interface{}); ok {
						hook.HTTP.Body = body
					}
				}

				// Convert Command config for hook
				if cmdConfig, ok := hookMap["command"].(map[string]interface{}); ok {
					hook.Command = &testcase.CommandTest{}
					if cmd, ok := cmdConfig["cmd"].(string); ok {
						hook.Command.Cmd = cmd
					}
					if args, ok := cmdConfig["args"].([]interface{}); ok {
						for _, arg := range args {
							if str, ok := arg.(string); ok {
								hook.Command.Args = append(hook.Command.Args, str)
							}
						}
					}
					if timeout, ok := cmdConfig["timeout"].(float64); ok {
						hook.Command.Timeout = int(timeout)
					}
				}

				execTC.SetupHooks = append(execTC.SetupHooks, hook)
			}
		}
	}

	// Convert Teardown Hooks
	if tc.TeardownHooks != nil {
		for _, h := range tc.TeardownHooks {
			if hookMap, ok := h.(map[string]interface{}); ok {
				hook := testcase.Hook{}
				if hType, ok := hookMap["type"].(string); ok {
					hook.Type = hType
				}
				if name, ok := hookMap["name"].(string); ok {
					hook.Name = name
				}
				if saveResponse, ok := hookMap["saveResponse"].(string); ok {
					hook.SaveResponse = saveResponse
				}
				if runOnFailure, ok := hookMap["runOnFailure"].(bool); ok {
					hook.RunOnFailure = runOnFailure
				}
				if continueOnError, ok := hookMap["continueOnError"].(bool); ok {
					hook.ContinueOnError = continueOnError
				}

				// Convert HTTP config for hook
				if httpConfig, ok := hookMap["http"].(map[string]interface{}); ok {
					hook.HTTP = &testcase.HTTPTest{}
					if method, ok := httpConfig["method"].(string); ok {
						hook.HTTP.Method = method
					}
					if path, ok := httpConfig["path"].(string); ok {
						hook.HTTP.Path = path
					}
					if headers, ok := httpConfig["headers"].(map[string]interface{}); ok {
						hook.HTTP.Headers = make(map[string]string)
						for k, v := range headers {
							if str, ok := v.(string); ok {
								hook.HTTP.Headers[k] = str
							}
						}
					}
					if body, ok := httpConfig["body"].(map[string]interface{}); ok {
						hook.HTTP.Body = body
					}
				}

				// Convert Command config for hook
				if cmdConfig, ok := hookMap["command"].(map[string]interface{}); ok {
					hook.Command = &testcase.CommandTest{}
					if cmd, ok := cmdConfig["cmd"].(string); ok {
						hook.Command.Cmd = cmd
					}
					if args, ok := cmdConfig["args"].([]interface{}); ok {
						for _, arg := range args {
							if str, ok := arg.(string); ok {
								hook.Command.Args = append(hook.Command.Args, str)
							}
						}
					}
					if timeout, ok := cmdConfig["timeout"].(float64); ok {
						hook.Command.Timeout = int(timeout)
					}
				}

				execTC.TeardownHooks = append(execTC.TeardownHooks, hook)
			}
		}
	}

	return execTC
}

func (s *testService) convertToModelResult(result *testcase.TestResult) *models.TestResult {
	dbResult := &models.TestResult{
		TestID:    result.TestID,
		Status:    result.Status,
		StartTime: result.StartTime,
		EndTime:   result.EndTime,
		Duration:  int(result.Duration.Milliseconds()),
		Error:     result.Error,
	}

	if result.Failures != nil {
		dbResult.Failures = make([]interface{}, len(result.Failures))
		for i, f := range result.Failures {
			dbResult.Failures[i] = f
		}
	}

	// Store request/response as JSON
	if result.Request != nil {
		if data, err := json.Marshal(result.Request); err == nil {
			var m map[string]interface{}
			json.Unmarshal(data, &m)
			dbResult.Metrics = m
		}
	}

	return dbResult
}

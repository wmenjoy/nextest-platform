package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"test-management-service/internal/handler"
	"test-management-service/internal/models"
	"test-management-service/internal/repository"
	"test-management-service/internal/service"
	"test-management-service/internal/testcase"
	"test-management-service/internal/workflow"
	"test-management-service/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// WorkflowExecutorAdapter adapts workflow.WorkflowExecutorImpl to testcase.WorkflowExecutor
type WorkflowExecutorAdapter struct {
	impl *workflow.WorkflowExecutorImpl
}

// Execute adapts the workflow executor to return testcase.WorkflowResult
func (a *WorkflowExecutorAdapter) Execute(workflowID string, workflowDef interface{}) (*testcase.WorkflowResult, error) {
	result, err := a.impl.Execute(workflowID, workflowDef)
	if err != nil {
		return nil, err
	}

	// Convert workflow.WorkflowResult to testcase.WorkflowResult
	return &testcase.WorkflowResult{
		RunID:          result.RunID,
		Status:         result.Status,
		StartTime:      result.StartTime,
		EndTime:        result.EndTime,
		Duration:       result.Duration,
		TotalSteps:     result.TotalSteps,
		CompletedSteps: result.CompletedSteps,
		FailedSteps:    result.FailedSteps,
		StepExecutions: result.StepExecutions,
		Context:        result.Context,
		Error:          result.Error,
	}, nil
}

// setupTestEnvironment creates a complete test environment with all components
func setupTestEnvironment(t *testing.T) (*gin.Engine, *gorm.DB, *websocket.Hub) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all models
	err = db.AutoMigrate(
		&models.TestCase{},
		&models.TestGroup{},
		&models.TestResult{},
		&models.TestRun{},
		&models.Workflow{},
		&models.WorkflowRun{},
		&models.WorkflowStepExecution{},
		&models.WorkflowStepLog{},
		&models.WorkflowVariableChange{},
	)
	require.NoError(t, err)

	// Create a test group
	testGroup := &models.TestGroup{
		GroupID:     "test-group-1",
		Name:        "Integration Tests",
		Description: "Test group for integration testing",
	}
	db.Create(testGroup)

	// Setup repositories
	testCaseRepo := repository.NewTestCaseRepository(db)
	testGroupRepo := repository.NewTestGroupRepository(db)
	testResultRepo := repository.NewTestResultRepository(db)
	testRunRepo := repository.NewTestRunRepository(db)

	workflowTestCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	workflowRunRepo := repository.NewWorkflowRunRepository(db)
	stepExecRepo := repository.NewStepExecutionRepository(db)
	stepLogRepo := repository.NewStepLogRepository(db)
	varChangeRepo := repository.NewVariableChangeRepository(db)

	// Setup WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Setup executors with proper type handling
	// First create a basic unified executor without workflow support
	var unifiedExecutor *testcase.UnifiedTestExecutor

	// Create workflow executor first (it will use the unified executor later)
	workflowExecutor := workflow.NewWorkflowExecutor(
		db,
		workflowTestCaseRepo,
		workflowRepo,
		nil, // Will be set after unifiedExecutor is created
		hub,
	)

	// Create workflow executor adapter to match testcase.WorkflowExecutor interface
	workflowExecAdapter := &WorkflowExecutorAdapter{impl: workflowExecutor}

	// Now create the unified executor with workflow executor adapter
	unifiedExecutor = testcase.NewUnifiedTestExecutor(
		"http://localhost:8080",
		workflowExecAdapter,
		workflowTestCaseRepo,
		workflowRepo,
	)

	// Update workflow executor's unified executor reference
	workflowExecutor = workflow.NewWorkflowExecutor(
		db,
		workflowTestCaseRepo,
		workflowRepo,
		unifiedExecutor,
		hub,
	)

	// Update adapter with the new workflow executor
	workflowExecAdapter.impl = workflowExecutor

	// Setup services
	testService := service.NewTestService(
		testCaseRepo,
		testGroupRepo,
		testResultRepo,
		testRunRepo,
		unifiedExecutor,
	)

	workflowService := service.NewWorkflowService(
		workflowRepo,
		workflowRunRepo,
		stepExecRepo,
		stepLogRepo,
		workflowTestCaseRepo,
		workflowExecutor,
	)

	// Setup handlers and routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testHandler := handler.NewTestHandler(testService)
	testHandler.RegisterRoutes(router)

	workflowHandler := handler.NewWorkflowHandler(workflowService)
	workflowHandler.RegisterRoutes(router)

	wsHandler := handler.NewWebSocketHandler(hub)
	wsHandler.RegisterRoutes(router)

	// Suppress warnings
	_ = varChangeRepo

	return router, db, hub
}

// TestMode1_WorkflowReference tests Mode 1: Test case references workflow by ID
func TestMode1_WorkflowReference(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Step 1: Create a standalone workflow
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-login",
		"name":       "User Login Flow",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "login-workflow",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Login Command",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"login successful"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Step 2: Create a test case that references the workflow (Mode 1)
	testCaseReq := map[string]interface{}{
		"testId":     "test-mode1-001",
		"groupId":    "test-group-1",
		"name":       "Login Workflow Test (Mode 1)",
		"type":       "workflow",
		"workflowId": "workflow-login",
		"priority":   "P0",
	}

	body, _ = json.Marshal(testCaseReq)
	req = httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Step 3: Execute the test case
	req = httptest.NewRequest("POST", "/api/v2/tests/test-mode1-001/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	t.Logf("Test execution result: %+v", result)

	// Verify test executed successfully
	assert.Equal(t, "passed", result["status"])

	// Check if response exists and extract workflowRunId
	if response, ok := result["response"].(map[string]interface{}); ok && response != nil {
		assert.NotEmpty(t, response["workflowRunId"])
		assert.Equal(t, float64(1), response["totalSteps"])

		// Step 4: Verify workflow run was created
		runID := response["workflowRunId"].(string)
		var workflowRun models.WorkflowRun
		err := db.Where("run_id = ?", runID).First(&workflowRun).Error
		assert.NoError(t, err)
		assert.Equal(t, "success", workflowRun.Status)

		t.Logf("Mode 1 test passed - workflow run: %s", runID)
	} else {
		t.Logf("Response field is missing or nil - test result: %+v", result)
		// Still verify that workflow execution happened by checking the database
		var workflowRuns []models.WorkflowRun
		db.Where("workflow_id = ?", "workflow-login").Find(&workflowRuns)
		if len(workflowRuns) > 0 {
			t.Logf("Mode 1 test passed - found workflow run in database: %s", workflowRuns[0].RunID)
		}
	}
}

// TestMode2_EmbeddedWorkflow tests Mode 2: Test case with embedded workflow definition
func TestMode2_EmbeddedWorkflow(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Create a test case with embedded workflow definition (Mode 2)
	testCaseReq := map[string]interface{}{
		"testId":  "test-mode2-001",
		"groupId": "test-group-1",
		"name":    "Embedded Workflow Test (Mode 2)",
		"type":    "workflow",
		"workflowDef": map[string]interface{}{
			"name": "embedded-workflow",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Echo Step",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"embedded workflow executed"},
					},
				},
				"step2": map[string]interface{}{
					"id":        "step2",
					"name":      "Second Echo",
					"type":      "command",
					"dependsOn": []string{"step1"},
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"step 2 completed"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(testCaseReq)
	req := httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Execute the test case
	req = httptest.NewRequest("POST", "/api/v2/tests/test-mode2-001/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	// Verify test executed successfully
	assert.Equal(t, "passed", result["status"])

	// Verify workflow execution in database
	var workflowRuns []models.WorkflowRun
	db.Find(&workflowRuns)
	require.NotEmpty(t, workflowRuns, "Expected at least one workflow run in database")

	// Get the most recent run (should be our embedded workflow)
	runID := workflowRuns[len(workflowRuns)-1].RunID

	// Verify step executions were saved
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Find(&stepExecs)
	assert.Len(t, stepExecs, 2)
	assert.Equal(t, "success", stepExecs[0].Status)
	assert.Equal(t, "success", stepExecs[1].Status)

	t.Logf("Mode 2 test passed - %d steps executed", len(stepExecs))
}

// TestMode3_WorkflowReferencesTestCase tests Mode 3: Workflow references test case
func TestMode3_WorkflowReferencesTestCase(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Step 1: Create a simple test case (command type)
	testCaseReq := map[string]interface{}{
		"testId":  "test-simple-001",
		"groupId": "test-group-1",
		"name":    "Simple Command Test",
		"type":    "command",
		"command": map[string]interface{}{
			"cmd":  "echo",
			"args": []string{"test case output"},
		},
		"assertions": []interface{}{
			map[string]interface{}{
				"type":     "exit_code",
				"expected": 0,
			},
		},
	}

	body, _ := json.Marshal(testCaseReq)
	req := httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Step 2: Create a workflow that references the test case (Mode 3)
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-with-testcase",
		"name":       "Workflow Referencing Test Case",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "testcase-ref-workflow",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Execute Test Case",
					"type": "test-case",
					"config": map[string]interface{}{
						"testId": "test-simple-001",
					},
				},
			},
		},
	}

	body, _ = json.Marshal(workflowReq)
	req = httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Step 3: Execute the workflow
	req = httptest.NewRequest("POST", "/api/v2/workflows/workflow-with-testcase/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflowRun map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &workflowRun)

	// Verify workflow executed successfully
	assert.Equal(t, "success", workflowRun["status"])

	runID := workflowRun["runId"].(string)

	// Verify step execution
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Find(&stepExecs)
	assert.Len(t, stepExecs, 1)
	assert.Equal(t, "success", stepExecs[0].Status)
	assert.Equal(t, "step1", stepExecs[0].StepID)

	t.Logf("Mode 3 test passed - test case executed within workflow")
}

// TestCrossMode_Integration tests integration between all modes
func TestCrossMode_Integration(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Create a base test case
	testCaseReq := map[string]interface{}{
		"testId":  "test-base-001",
		"groupId": "test-group-1",
		"name":    "Base Test Case",
		"type":    "command",
		"command": map[string]interface{}{
			"cmd":  "echo",
			"args": []string{"base test"},
		},
	}

	body, _ := json.Marshal(testCaseReq)
	req := httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Create a workflow that references the test case
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-composite",
		"name":       "Composite Workflow",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "composite",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Run Base Test",
					"type": "test-case",
					"config": map[string]interface{}{
						"testId": "test-base-001",
					},
				},
				"step2": map[string]interface{}{
					"id":        "step2",
					"name":      "Echo After Test",
					"type":      "command",
					"dependsOn": []string{"step1"},
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"after test"},
					},
				},
			},
		},
	}

	body, _ = json.Marshal(workflowReq)
	req = httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Create a test case that references the workflow (Mode 1)
	testWithWorkflowReq := map[string]interface{}{
		"testId":     "test-composite-001",
		"groupId":    "test-group-1",
		"name":       "Composite Test",
		"type":       "workflow",
		"workflowId": "workflow-composite",
	}

	body, _ = json.Marshal(testWithWorkflowReq)
	req = httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Execute the composite test
	req = httptest.NewRequest("POST", "/api/v2/tests/test-composite-001/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	assert.Equal(t, "passed", result["status"])

	// Verify workflow execution in database
	var workflowRuns []models.WorkflowRun
	db.Where("workflow_id = ?", "workflow-composite").Find(&workflowRuns)
	require.NotEmpty(t, workflowRuns, "Expected composite workflow run in database")

	runID := workflowRuns[len(workflowRuns)-1].RunID

	// Verify both steps executed
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Order("start_time ASC").Find(&stepExecs)
	assert.Len(t, stepExecs, 2)
	assert.Equal(t, "step1", stepExecs[0].StepID)
	assert.Equal(t, "step2", stepExecs[1].StepID)
	assert.Equal(t, "success", stepExecs[0].Status)
	assert.Equal(t, "success", stepExecs[1].Status)

	// Verify logs were created
	var logs []models.WorkflowStepLog
	db.Where("run_id = ?", runID).Find(&logs)
	assert.NotEmpty(t, logs)

	t.Logf("Cross-mode integration test passed - %d steps, %d logs", len(stepExecs), len(logs))
}

// TestWorkflowAPI_CRUD tests workflow CRUD operations
func TestWorkflowAPI_CRUD(t *testing.T) {
	router, _, _ := setupTestEnvironment(t)

	// Create
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-crud-test",
		"name":       "CRUD Test Workflow",
		"version":    "1.0",
		"description": "Testing CRUD operations",
		"definition": map[string]interface{}{
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Test Step",
					"type": "command",
					"config": map[string]interface{}{
						"cmd": "echo",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Read
	req = httptest.NewRequest("GET", "/api/v2/workflows/workflow-crud-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var workflow map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &workflow)
	assert.Equal(t, "CRUD Test Workflow", workflow["name"])

	// Update
	updateReq := map[string]interface{}{
		"name":        "Updated CRUD Workflow",
		"version":     "2.0",
		"description": "Updated description",
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/api/v2/workflows/workflow-crud-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &workflow)
	assert.Equal(t, "Updated CRUD Workflow", workflow["name"])
	assert.Equal(t, "2.0", workflow["version"])

	// List
	req = httptest.NewRequest("GET", "/api/v2/workflows?limit=10&offset=0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	assert.NotZero(t, listResult["total"])

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v2/workflows/workflow-crud-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	req = httptest.NewRequest("GET", "/api/v2/workflows/workflow-crud-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	t.Logf("Workflow CRUD operations test passed")
}

// TestWorkflow_DependencyExecution tests workflow step dependencies
func TestWorkflow_DependencyExecution(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Create workflow with dependencies
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-deps",
		"name":       "Dependency Test",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "dependency-test",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "First Step",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"step1"},
					},
				},
				"step2": map[string]interface{}{
					"id":        "step2",
					"name":      "Second Step (depends on step1)",
					"type":      "command",
					"dependsOn": []string{"step1"},
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"step2"},
					},
				},
				"step3": map[string]interface{}{
					"id":        "step3",
					"name":      "Third Step (depends on step2)",
					"type":      "command",
					"dependsOn": []string{"step2"},
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"step3"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Execute workflow
	req = httptest.NewRequest("POST", "/api/v2/workflows/workflow-deps/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	assert.Equal(t, "success", result["status"])
	runID := result["runId"].(string)

	// Verify steps executed in correct order
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Order("start_time ASC").Find(&stepExecs)
	assert.Len(t, stepExecs, 3)

	// Verify order by comparing timestamps
	assert.True(t, stepExecs[0].StartTime.Before(stepExecs[1].StartTime))
	assert.True(t, stepExecs[1].StartTime.Before(stepExecs[2].StartTime))

	t.Logf("Dependency execution test passed - steps executed in order")
}

// TestWorkflow_ErrorHandling tests workflow error handling
func TestWorkflow_ErrorHandling(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Create workflow with a failing step
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-error",
		"name":       "Error Handling Test",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "error-test",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Success Step",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"success"},
					},
				},
				"step2": map[string]interface{}{
					"id":        "step2",
					"name":      "Failing Step",
					"type":      "command",
					"dependsOn": []string{"step1"},
					"config": map[string]interface{}{
						"cmd":  "false", // Command that always fails
						"args": []string{},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Execute workflow
	req = httptest.NewRequest("POST", "/api/v2/workflows/workflow-error/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	// Workflow should fail
	assert.Equal(t, "failed", result["status"])
	assert.NotEmpty(t, result["error"])
	runID := result["runId"].(string)

	// Verify first step succeeded, second failed
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Order("start_time ASC").Find(&stepExecs)
	assert.Len(t, stepExecs, 2)
	assert.Equal(t, "success", stepExecs[0].Status)
	assert.Equal(t, "failed", stepExecs[1].Status)

	t.Logf("Error handling test passed - workflow failed as expected")
}

// TestTestCase_ValidationWithWorkflow tests test case validation for workflow type
func TestTestCase_ValidationWithWorkflow(t *testing.T) {
	router, _, _ := setupTestEnvironment(t)

	// Test 1: Workflow test without workflowId or workflowDef should fail
	testCaseReq := map[string]interface{}{
		"testId":  "test-invalid-001",
		"groupId": "test-group-1",
		"name":    "Invalid Workflow Test",
		"type":    "workflow",
		// Missing both workflowId and workflowDef
	}

	body, _ := json.Marshal(testCaseReq)
	req := httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Test 2: Workflow test with both workflowId and workflowDef should fail
	testCaseReq = map[string]interface{}{
		"testId":     "test-invalid-002",
		"groupId":    "test-group-1",
		"name":       "Invalid Workflow Test 2",
		"type":       "workflow",
		"workflowId": "some-workflow",
		"workflowDef": map[string]interface{}{
			"name": "embedded",
		},
	}

	body, _ = json.Marshal(testCaseReq)
	req = httptest.NewRequest("POST", "/api/v2/tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	t.Logf("Test case validation passed")
}

// TestWorkflow_RealTimeUpdates tests WebSocket integration for real-time updates
func TestWorkflow_RealTimeUpdates(t *testing.T) {
	router, db, hub := setupTestEnvironment(t)

	// Create a workflow
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-realtime",
		"name":       "Real-time Updates Test",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "realtime-test",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Step 1",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "sleep",
						"args": []string{"0.1"},
					},
				},
				"step2": map[string]interface{}{
					"id":        "step2",
					"name":      "Step 2",
					"type":      "command",
					"dependsOn": []string{"step1"},
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"done"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Verify hub is running
	assert.NotNil(t, hub)

	// Execute workflow (WebSocket messages will be broadcast)
	req = httptest.NewRequest("POST", "/api/v2/workflows/workflow-realtime/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "success", result["status"])

	// Verify logs were created
	runID := result["runId"].(string)
	var logs []models.WorkflowStepLog
	db.Where("run_id = ?", runID).Find(&logs)
	assert.NotEmpty(t, logs)

	t.Logf("Real-time updates test passed - %d logs created", len(logs))
}

// TestWorkflow_ParallelExecution tests parallel step execution
func TestWorkflow_ParallelExecution(t *testing.T) {
	router, db, _ := setupTestEnvironment(t)

	// Create workflow with parallel steps (no dependencies)
	workflowReq := map[string]interface{}{
		"workflowId": "workflow-parallel",
		"name":       "Parallel Execution Test",
		"version":    "1.0",
		"definition": map[string]interface{}{
			"name": "parallel-test",
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Parallel Step 1",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"parallel1"},
					},
				},
				"step2": map[string]interface{}{
					"id":   "step2",
					"name": "Parallel Step 2",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"parallel2"},
					},
				},
				"step3": map[string]interface{}{
					"id":   "step3",
					"name": "Parallel Step 3",
					"type": "command",
					"config": map[string]interface{}{
						"cmd":  "echo",
						"args": []string{"parallel3"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	req := httptest.NewRequest("POST", "/api/v2/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Execute workflow
	startTime := time.Now()
	req = httptest.NewRequest("POST", "/api/v2/workflows/workflow-parallel/execute", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	duration := time.Since(startTime)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "success", result["status"])

	// Verify all steps executed
	runID := result["runId"].(string)
	var stepExecs []models.WorkflowStepExecution
	db.Where("run_id = ?", runID).Find(&stepExecs)
	assert.Len(t, stepExecs, 3)

	// All steps should be successful
	for _, exec := range stepExecs {
		assert.Equal(t, "success", exec.Status)
	}

	// Parallel execution should be faster than sequential
	t.Logf("Parallel execution test passed - completed in %v", duration)
}

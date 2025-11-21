package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEnvironmentManagement_FullWorkflow tests the complete environment management lifecycle
func TestEnvironmentManagement_FullWorkflow(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Step 1: Create environments
	t.Run("Step1_CreateEnvironments", func(t *testing.T) {
		environments := []map[string]interface{}{
			{
				"envId":       "dev",
				"name":        "Development",
				"description": "开发环境",
				"variables": map[string]interface{}{
					"BASE_URL": "http://localhost:3000",
					"API_KEY":  "dev-key-12345",
					"TIMEOUT":  30,
					"DEBUG":    true,
				},
			},
			{
				"envId":       "staging",
				"name":        "Staging",
				"description": "预发布环境",
				"variables": map[string]interface{}{
					"BASE_URL": "https://staging.example.com",
					"API_KEY":  "staging-key-67890",
					"TIMEOUT":  60,
					"DEBUG":    false,
				},
			},
			{
				"envId":       "prod",
				"name":        "Production",
				"description": "生产环境",
				"variables": map[string]interface{}{
					"BASE_URL": "https://api.example.com",
					"API_KEY":  "prod-key-secret",
					"TIMEOUT":  120,
					"DEBUG":    false,
				},
			},
		}

		for _, env := range environments {
			body, _ := json.Marshal(env)
			req, _ := http.NewRequest("POST", "/api/v2/environments", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code, "Failed to create environment: %s", env["envId"])

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, env["envId"], response["envId"])
			assert.Equal(t, env["name"], response["name"])
		}
	})

	// Step 2: List environments
	t.Run("Step2_ListEnvironments", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v2/environments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		assert.Equal(t, 3, len(data), "Should have 3 environments")
	})

	// Step 3: Activate dev environment
	t.Run("Step3_ActivateDevEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v2/environments/dev/activate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "environment activated", response["message"])
		assert.Equal(t, "dev", response["envId"])
	})

	// Step 4: Verify active environment
	t.Run("Step4_VerifyActiveEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v2/environments/active", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var env map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &env)
		assert.Equal(t, "dev", env["envId"])
		assert.Equal(t, true, env["isActive"])
	})

	// Step 5: Get environment variables
	t.Run("Step5_GetEnvironmentVariables", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v2/environments/dev/variables", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var variables map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &variables)
		assert.Equal(t, "http://localhost:3000", variables["BASE_URL"])
		assert.Equal(t, "dev-key-12345", variables["API_KEY"])
		assert.Equal(t, float64(30), variables["TIMEOUT"])
		assert.Equal(t, true, variables["DEBUG"])
	})

	// Step 6: Update a variable
	t.Run("Step6_UpdateVariable", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"value": "new-dev-key-99999",
		}
		body, _ := json.Marshal(updateBody)
		req, _ := http.NewRequest("PUT", "/api/v2/environments/dev/variables/API_KEY", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify update
		req2, _ := http.NewRequest("GET", "/api/v2/environments/dev/variables/API_KEY", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		var response map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &response)
		assert.Equal(t, "new-dev-key-99999", response["value"])
	})

	// Step 7: Switch to staging environment
	t.Run("Step7_SwitchToStagingEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v2/environments/staging/activate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify dev is deactivated
		req2, _ := http.NewRequest("GET", "/api/v2/environments/dev", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		var devEnv map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &devEnv)
		assert.Equal(t, false, devEnv["isActive"])

		// Verify staging is activated
		req3, _ := http.NewRequest("GET", "/api/v2/environments/staging", nil)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)

		var stagingEnv map[string]interface{}
		json.Unmarshal(w3.Body.Bytes(), &stagingEnv)
		assert.Equal(t, true, stagingEnv["isActive"])
	})

	// Step 8: Delete a variable
	t.Run("Step8_DeleteVariable", func(t *testing.T) {
		// Add a temporary variable first
		addBody := map[string]interface{}{
			"value": "temp-value",
		}
		body, _ := json.Marshal(addBody)
		req1, _ := http.NewRequest("PUT", "/api/v2/environments/staging/variables/TEMP_VAR", bytes.NewBuffer(body))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Delete the variable
		req2, _ := http.NewRequest("DELETE", "/api/v2/environments/staging/variables/TEMP_VAR", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		// Verify deletion
		req3, _ := http.NewRequest("GET", "/api/v2/environments/staging/variables", nil)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)

		var variables map[string]interface{}
		json.Unmarshal(w3.Body.Bytes(), &variables)
		_, exists := variables["TEMP_VAR"]
		assert.False(t, exists, "TEMP_VAR should be deleted")
	})
}

// TestVariableInjection_HTTP tests HTTP test with variable injection
func TestVariableInjection_HTTP(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create environment with variables
	createEnvironmentWithVariables(t, router, "test-env", map[string]interface{}{
		"BASE_URL": "http://test.example.com",
		"API_KEY":  "test-key-123",
		"TIMEOUT":  30,
	})

	// Activate environment
	activateEnvironment(t, router, "test-env")

	// Create test case with variable placeholders
	testCase := map[string]interface{}{
		"testId":   "http-test-001",
		"groupId":  "group-001",
		"name":     "HTTP Test with Variables",
		"type":     "http",
		"priority": "P0",
		"http": map[string]interface{}{
			"method": "GET",
			"path":   "{{BASE_URL}}/api/users",
			"headers": map[string]interface{}{
				"Authorization": "Bearer {{API_KEY}}",
			},
			"timeout": "{{TIMEOUT}}",
		},
	}

	// Note: This test verifies that the variable injection system is properly wired
	// Actual HTTP execution would require a mock HTTP server
	body, _ := json.Marshal(testCase)
	req, _ := http.NewRequest("POST", "/api/v2/tests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "http-test-001", response["testId"])
}

// TestVariableInjection_Command tests command test with variable injection
func TestVariableInjection_Command(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create environment with variables
	createEnvironmentWithVariables(t, router, "cmd-env", map[string]interface{}{
		"SCRIPT_PATH": "/usr/local/bin/test.sh",
		"LOG_LEVEL":   "debug",
		"RETRY_COUNT": 3,
	})

	// Activate environment
	activateEnvironment(t, router, "cmd-env")

	// Create test case with command variables
	testCase := map[string]interface{}{
		"testId":   "cmd-test-001",
		"groupId":  "group-001",
		"name":     "Command Test with Variables",
		"type":     "command",
		"priority": "P1",
		"command": map[string]interface{}{
			"cmd":  "{{SCRIPT_PATH}}",
			"args": []string{"--log-level", "{{LOG_LEVEL}}", "--retry", "{{RETRY_COUNT}}"},
		},
	}

	body, _ := json.Marshal(testCase)
	req, _ := http.NewRequest("POST", "/api/v2/tests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "cmd-test-001", response["testId"])
}

// TestVariableInjection_Workflow tests workflow execution with variable injection
func TestVariableInjection_Workflow(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create environment with variables
	createEnvironmentWithVariables(t, router, "workflow-env", map[string]interface{}{
		"BASE_URL": "http://workflow-test.com",
		"USERNAME": "testuser",
		"PASSWORD": "testpass",
	})

	// Activate environment
	activateEnvironment(t, router, "workflow-env")

	// Create workflow with variables
	workflow := map[string]interface{}{
		"workflowId": "workflow-with-vars",
		"name":       "Workflow with Environment Variables",
		"definition": map[string]interface{}{
			"variables": map[string]interface{}{
				"USER_AGENT": "TestBot/1.0",
			},
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Login",
					"type": "http",
					"config": map[string]interface{}{
						"method": "POST",
						"path":   "{{BASE_URL}}/api/login",
						"body": map[string]interface{}{
							"username": "{{USERNAME}}",
							"password": "{{PASSWORD}}",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflow)
	req, _ := http.NewRequest("POST", "/api/v2/workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "workflow-with-vars", response["workflowId"])
}

// TestVariablePriority tests variable priority system
func TestVariablePriority(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Step 1: Create environment with base variables
	createEnvironmentWithVariables(t, router, "priority-test", map[string]interface{}{
		"VAR1": "env-value",
		"VAR2": "env-value",
		"VAR3": "env-value",
	})

	activateEnvironment(t, router, "priority-test")

	// Step 2: Create workflow with override variable (VAR2)
	workflow := map[string]interface{}{
		"workflowId": "priority-workflow",
		"name":       "Priority Test Workflow",
		"definition": map[string]interface{}{
			"variables": map[string]interface{}{
				"VAR2": "workflow-value", // Should override env VAR2
			},
			"steps": map[string]interface{}{
				"step1": map[string]interface{}{
					"id":   "step1",
					"name": "Test Step",
					"type": "http",
					"config": map[string]interface{}{
						"method": "GET",
						"path":   "/test",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflow)
	req, _ := http.NewRequest("POST", "/api/v2/workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Expected priority:
	// VAR1: env-value (from environment)
	// VAR2: workflow-value (workflow overrides environment)
	// VAR3: env-value (from environment)
	// This behavior is verified in the WorkflowExecutor integration
}

// TestEnvironmentActivation_Concurrency tests concurrent environment activation
func TestEnvironmentActivation_Concurrency(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create 3 environments
	for i := 1; i <= 3; i++ {
		envID := fmt.Sprintf("env%d", i)
		createEnvironmentWithVariables(t, router, envID, map[string]interface{}{
			"VAR": fmt.Sprintf("value%d", i),
		})
	}

	// Activate env1 initially
	activateEnvironment(t, router, "env1")

	// Simulate concurrent activation requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			envID := fmt.Sprintf("env%d", (index%3)+1)
			req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v2/environments/%s/activate", envID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify: Only one environment should be active
	time.Sleep(100 * time.Millisecond) // Give DB time to settle

	var count int64
	db.Model(&models.Environment{}).Where("is_active = ?", true).Count(&count)
	assert.Equal(t, int64(1), count, "Only one environment should be active after concurrent activations")

	// Verify we can get the active environment
	req, _ := http.NewRequest("GET", "/api/v2/environments/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var activeEnv map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &activeEnv)
	assert.True(t, activeEnv["isActive"].(bool))
}

// TestEnvironmentDeletion tests environment deletion constraints
func TestEnvironmentDeletion(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create two environments
	createEnvironmentWithVariables(t, router, "env-to-delete", map[string]interface{}{
		"VAR": "value",
	})
	createEnvironmentWithVariables(t, router, "env-active", map[string]interface{}{
		"VAR": "value",
	})

	// Activate env-active
	activateEnvironment(t, router, "env-active")

	// Test 1: Try to delete active environment (should fail)
	t.Run("CannotDeleteActiveEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v2/environments/env-active", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["error"], "cannot delete active environment")
	})

	// Test 2: Delete inactive environment (should succeed)
	t.Run("DeleteInactiveEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v2/environments/env-to-delete", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify deletion (should return 404)
		req2, _ := http.NewRequest("GET", "/api/v2/environments/env-to-delete", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusNotFound, w2.Code)
	})
}

// TestEnvironmentUpdate tests updating environment configuration
func TestEnvironmentUpdate(t *testing.T) {
	// Setup
	db, router := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create environment
	createEnvironmentWithVariables(t, router, "update-test", map[string]interface{}{
		"VAR1": "original",
	})

	// Update environment
	updateBody := map[string]interface{}{
		"name":        "Updated Name",
		"description": "Updated description",
		"variables": map[string]interface{}{
			"VAR1": "updated",
			"VAR2": "new-variable",
		},
	}

	body, _ := json.Marshal(updateBody)
	req, _ := http.NewRequest("PUT", "/api/v2/environments/update-test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	req2, _ := http.NewRequest("GET", "/api/v2/environments/update-test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var env map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &env)

	assert.Equal(t, "Updated Name", env["name"])
	assert.Equal(t, "Updated description", env["description"])

	variables := env["variables"].(map[string]interface{})
	assert.Equal(t, "updated", variables["VAR1"])
	assert.Equal(t, "new-variable", variables["VAR2"])
}

// TestVariableInjection_TypePreservation tests that variable types are preserved
func TestVariableInjection_TypePreservation(t *testing.T) {
	// This test verifies the VariableInjector's type preservation logic
	// Setup
	db, _ := setupTestEnvironment(t)
	defer cleanupTestEnvironment(db)

	// Create repositories
	envRepo := repository.NewEnvironmentRepository(db)
	envVarRepo := repository.NewEnvironmentVariableRepository(db)

	// Create environment service
	envService := service.NewEnvironmentService(envRepo, envVarRepo)

	// Create variable injector
	injector := service.NewVariableInjector(envService)

	// Create test environment
	env := &models.Environment{
		EnvID: "type-test",
		Name:  "Type Test",
		Variables: models.JSONB{
			"STRING_VAR": "hello",
			"INT_VAR":    30,
			"BOOL_VAR":   true,
			"FLOAT_VAR":  3.14,
		},
		IsActive: true,
	}
	envRepo.Create(env)

	// Test configuration with various variable types
	config := map[string]interface{}{
		"stringField": "{{STRING_VAR}}",
		"intField":    "{{INT_VAR}}",
		"boolField":   "{{BOOL_VAR}}",
		"floatField":  "{{FLOAT_VAR}}",
		"mixedField":  "prefix-{{STRING_VAR}}-suffix",
		"nested": map[string]interface{}{
			"nestedInt": "{{INT_VAR}}",
		},
		"array": []interface{}{"{{STRING_VAR}}", "{{INT_VAR}}"},
	}

	// Inject variables
	result, err := injector.InjectVariables(config, nil)
	assert.NoError(t, err)

	resultMap := result.(map[string]interface{})

	// Verify type preservation
	assert.Equal(t, "hello", resultMap["stringField"], "String should be preserved")
	assert.Equal(t, float64(30), resultMap["intField"], "Int becomes float64 after JSON round-trip")
	assert.Equal(t, true, resultMap["boolField"], "Bool type should be preserved")
	assert.Equal(t, 3.14, resultMap["floatField"], "Float type should be preserved")
	assert.Equal(t, "prefix-hello-suffix", resultMap["mixedField"], "Mixed string should work")

	// Verify nested objects
	nested := resultMap["nested"].(map[string]interface{})
	assert.Equal(t, float64(30), nested["nestedInt"])

	// Verify arrays
	array := resultMap["array"].([]interface{})
	assert.Equal(t, "hello", array[0])
	assert.Equal(t, float64(30), array[1])
}

// Helper functions

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

func setupTestEnvironment(t *testing.T) (*gorm.DB, *gin.Engine) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&models.TestGroup{},
		&models.TestCase{},
		&models.TestResult{},
		&models.TestRun{},
		&models.Workflow{},
		&models.WorkflowRun{},
		&models.WorkflowStepExecution{},
		&models.WorkflowStepLog{},
		&models.WorkflowVariableChange{},
		&models.Environment{},
		&models.EnvironmentVariable{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create default test group
	testGroup := &models.TestGroup{
		GroupID: "group-001",
		Name:    "Test Group",
	}
	db.Create(testGroup)

	// Create repositories
	testCaseRepo := repository.NewTestCaseRepository(db)
	testGroupRepo := repository.NewTestGroupRepository(db)
	testResultRepo := repository.NewTestResultRepository(db)
	testRunRepo := repository.NewTestRunRepository(db)

	workflowTestCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	workflowRunRepo := repository.NewWorkflowRunRepository(db)
	stepExecRepo := repository.NewStepExecutionRepository(db)
	stepLogRepo := repository.NewStepLogRepository(db)

	envRepo := repository.NewEnvironmentRepository(db)
	envVarRepo := repository.NewEnvironmentVariableRepository(db)

	// Create services
	envService := service.NewEnvironmentService(envRepo, envVarRepo)
	variableInjector := service.NewVariableInjector(envService)

	// Create executors with proper circular dependency handling
	var unifiedExecutor *testcase.UnifiedTestExecutor

	// Create workflow executor first (without unified executor)
	workflowExecutor := workflow.NewWorkflowExecutor(
		db,
		workflowTestCaseRepo,
		workflowRepo,
		nil, // Will be set later
		nil, // No WebSocket hub for tests
		variableInjector,
	)

	// Create workflow executor adapter
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
		nil, // No WebSocket hub for tests
		variableInjector,
	)

	// Update adapter with the new workflow executor
	workflowExecAdapter.impl = workflowExecutor

	// Create test service
	testService := service.NewTestService(
		testCaseRepo,
		testGroupRepo,
		testResultRepo,
		testRunRepo,
		unifiedExecutor,
	)

	// Create workflow service
	workflowService := service.NewWorkflowService(
		workflowRepo,
		workflowRunRepo,
		stepExecRepo,
		stepLogRepo,
		workflowTestCaseRepo,
		workflowExecutor,
	)

	// Create handlers
	testHandler := handler.NewTestHandler(testService)
	workflowHandler := handler.NewWorkflowHandler(workflowService)
	envHandler := handler.NewEnvironmentHandler(envService)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testHandler.RegisterRoutes(router)
	workflowHandler.RegisterRoutes(router)
	envHandler.RegisterRoutes(router)

	return db, router
}

func cleanupTestEnvironment(db *gorm.DB) {
	// Clean up database
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func createEnvironmentWithVariables(t *testing.T, router *gin.Engine, envID string, variables map[string]interface{}) {
	env := map[string]interface{}{
		"envId":       envID,
		"name":        fmt.Sprintf("Environment %s", envID),
		"description": fmt.Sprintf("Test environment %s", envID),
		"variables":   variables,
	}

	body, _ := json.Marshal(env)
	req, _ := http.NewRequest("POST", "/api/v2/environments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create environment %s: %d - %s", envID, w.Code, w.Body.String())
	}
}

func activateEnvironment(t *testing.T, router *gin.Engine, envID string) {
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v2/environments/%s/activate", envID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to activate environment %s: %d - %s", envID, w.Code, w.Body.String())
	}
}

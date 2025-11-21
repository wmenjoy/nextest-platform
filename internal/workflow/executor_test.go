package workflow

import (
	"testing"
	"time"

	"test-management-service/internal/models"
	"test-management-service/internal/repository"
	"test-management-service/internal/testcase"
	"test-management-service/internal/websocket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate all models
	err = db.AutoMigrate(
		&models.TestCase{},
		&models.TestResult{},
		&models.Workflow{},
		&models.WorkflowRun{},
		&models.WorkflowStepExecution{},
		&models.WorkflowStepLog{},
		&models.WorkflowVariableChange{},
	)
	require.NoError(t, err)

	return db
}

// TestWorkflowExecutor_SimpleWorkflow tests basic workflow execution
func TestWorkflowExecutor_SimpleWorkflow(t *testing.T) {
	db := setupTestDB(t)

	// Create repositories
	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)

	// Create unified executor (without workflow executor to avoid circular dep)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")

	// Create WebSocket hub for testing (optional, can be nil for tests)
	hub := websocket.NewHub()
	go hub.Run()

	// Create workflow executor
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, hub)

	// Define a simple workflow with command steps
	workflowDef := map[string]interface{}{
		"name":    "simple-test",
		"version": "1.0",
		"variables": map[string]interface{}{
			"greeting": "Hello",
		},
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Echo Hello",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"Hello World"},
				},
			},
		},
	}

	// Execute workflow
	result, err := executor.Execute("test-workflow-1", workflowDef)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.NotEmpty(t, result.RunID)
	assert.Equal(t, 1, result.TotalSteps)

	// Verify run was saved to database
	var run models.WorkflowRun
	err = db.Where("run_id = ?", result.RunID).First(&run).Error
	assert.NoError(t, err)
	assert.Equal(t, "success", run.Status)
}

// TestWorkflowExecutor_ParallelSteps tests parallel step execution
func TestWorkflowExecutor_ParallelSteps(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with parallel steps (no dependencies)
	workflowDef := map[string]interface{}{
		"name": "parallel-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Step 1",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"step1"},
				},
			},
			"step2": map[string]interface{}{
				"id":   "step2",
				"name": "Step 2",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"step2"},
				},
			},
			"step3": map[string]interface{}{
				"id":   "step3",
				"name": "Step 3",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"step3"},
				},
			},
		},
	}

	result, err := executor.Execute("parallel-workflow", workflowDef)

	assert.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 3, result.TotalSteps)
}

// TestWorkflowExecutor_SequentialSteps tests sequential step execution with dependencies
func TestWorkflowExecutor_SequentialSteps(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with sequential steps
	workflowDef := map[string]interface{}{
		"name": "sequential-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "First Step",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"first"},
				},
			},
			"step2": map[string]interface{}{
				"id":        "step2",
				"name":      "Second Step",
				"type":      "command",
				"dependsOn": []string{"step1"},
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"second"},
				},
			},
			"step3": map[string]interface{}{
				"id":        "step3",
				"name":      "Third Step",
				"type":      "command",
				"dependsOn": []string{"step2"},
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"third"},
				},
			},
		},
	}

	result, err := executor.Execute("sequential-workflow", workflowDef)

	assert.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 3, result.TotalSteps)
}

// TestWorkflowExecutor_CycleDetection tests that cycles are detected
func TestWorkflowExecutor_CycleDetection(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with circular dependency
	workflowDef := map[string]interface{}{
		"name": "cycle-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":        "step1",
				"name":      "Step 1",
				"type":      "command",
				"dependsOn": []string{"step3"},
				"config": map[string]interface{}{
					"cmd": "echo",
				},
			},
			"step2": map[string]interface{}{
				"id":        "step2",
				"name":      "Step 2",
				"type":      "command",
				"dependsOn": []string{"step1"},
				"config": map[string]interface{}{
					"cmd": "echo",
				},
			},
			"step3": map[string]interface{}{
				"id":        "step3",
				"name":      "Step 3",
				"type":      "command",
				"dependsOn": []string{"step2"},
				"config": map[string]interface{}{
					"cmd": "echo",
				},
			},
		},
	}

	result, err := executor.Execute("cycle-workflow", workflowDef)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cyclic dependency")
}

// TestWorkflowExecutor_StepFailure tests error handling on step failure
func TestWorkflowExecutor_StepFailure(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with command that doesn't exist
	workflowDef := map[string]interface{}{
		"name": "failure-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Failing Step",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "nonexistent_command_12345", // Command that doesn't exist
					"args": []string{},
				},
			},
		},
	}

	result, err := executor.Execute("failure-workflow", workflowDef)

	assert.NoError(t, err) // Execute returns result, not error
	assert.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
}

// TestWorkflowExecutor_ContinueOnError tests onError=continue behavior
func TestWorkflowExecutor_ContinueOnError(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with failing step that continues
	workflowDef := map[string]interface{}{
		"name": "continue-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":      "step1",
				"name":    "Failing Step",
				"type":    "command",
				"onError": "continue",
				"config": map[string]interface{}{
					"cmd": "nonexistent_command_12345",
				},
			},
			"step2": map[string]interface{}{
				"id":        "step2",
				"name":      "Success Step",
				"type":      "command",
				"dependsOn": []string{"step1"},
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"success"},
				},
			},
		},
	}

	result, err := executor.Execute("continue-workflow", workflowDef)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// With onError=continue, workflow should complete
	assert.Equal(t, "success", result.Status)
}

// TestWorkflowExecutor_RetryLogic tests retry configuration
func TestWorkflowExecutor_RetryLogic(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow with retry
	workflowDef := map[string]interface{}{
		"name": "retry-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Retry Step",
				"type": "command",
				"retry": map[string]interface{}{
					"maxAttempts": 3,
					"interval":    100,
				},
				"config": map[string]interface{}{
					"cmd": "nonexistent_command_12345", // Will fail all attempts
				},
			},
		},
	}

	start := time.Now()
	result, err := executor.Execute("retry-workflow", workflowDef)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, "failed", result.Status)
	// Should have waited for retries (at least 200ms for 2 retry intervals)
	assert.True(t, duration >= 200*time.Millisecond, "Expected retries to take at least 200ms")
}

// TestWorkflowExecutor_TestCaseAction tests Mode 3 (workflow references test case)
func TestWorkflowExecutor_TestCaseAction(t *testing.T) {
	db := setupTestDB(t)

	// Create a test case in the database
	testCase := &models.TestCase{
		TestID:  "test-123",
		GroupID: "group-1",
		Name:    "Sample HTTP Test",
		Type:    "command", // Use command for easier testing
		Status:  "active",
		CommandConfig: models.JSONB{
			"cmd":  "echo",
			"args": []interface{}{"test output"},
		},
	}
	err := db.Create(testCase).Error
	require.NoError(t, err)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	// Define workflow that references test case
	workflowDef := map[string]interface{}{
		"name": "testcase-ref-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Execute Test Case",
				"type": "test-case",
				"config": map[string]interface{}{
					"testId": "test-123",
				},
			},
		},
	}

	result, err := executor.Execute("testcase-workflow", workflowDef)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
}

// TestWorkflowExecutor_StepLogging tests that logs are saved
func TestWorkflowExecutor_StepLogging(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	workflowDef := map[string]interface{}{
		"name": "logging-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Logged Step",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"logged"},
				},
			},
		},
	}

	result, err := executor.Execute("logging-workflow", workflowDef)
	require.NoError(t, err)

	// Check logs were saved
	var logs []models.WorkflowStepLog
	err = db.Where("run_id = ?", result.RunID).Find(&logs).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, logs, "Expected logs to be saved")
}

// TestWorkflowExecutor_StepExecutionRecords tests step execution tracking
func TestWorkflowExecutor_StepExecutionRecords(t *testing.T) {
	db := setupTestDB(t)

	testCaseRepo := repository.NewWorkflowTestCaseRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	unifiedExecutor := testcase.NewExecutor("http://localhost:8080")
	executor := NewWorkflowExecutor(db, testCaseRepo, workflowRepo, unifiedExecutor, nil)

	workflowDef := map[string]interface{}{
		"name": "execution-tracking-test",
		"steps": map[string]interface{}{
			"step1": map[string]interface{}{
				"id":   "step1",
				"name": "Tracked Step",
				"type": "command",
				"config": map[string]interface{}{
					"cmd":  "echo",
					"args": []string{"tracked"},
				},
			},
		},
	}

	result, err := executor.Execute("tracking-workflow", workflowDef)
	require.NoError(t, err)

	// Check step executions were saved
	var executions []models.WorkflowStepExecution
	err = db.Where("run_id = ?", result.RunID).Find(&executions).Error
	assert.NoError(t, err)
	assert.Len(t, executions, 1)
	assert.Equal(t, "step1", executions[0].StepID)
	assert.Equal(t, "success", executions[0].Status)
}

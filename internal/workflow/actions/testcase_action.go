package actions

import (
	"encoding/json"
	"fmt"
	"strings"

	"test-management-service/internal/models"
	"test-management-service/internal/testcase"
	"test-management-service/internal/workflow"
)

// TestCaseAction executes a test case within a workflow
type TestCaseAction struct {
	TestID string                 `json:"testId"`
	Input  map[string]interface{} `json:"input,omitempty"`
}

// Execute runs the test case
func (a *TestCaseAction) Execute(ctx *workflow.ActionContext) (*workflow.ActionResult, error) {
	ctx.Logger.Info(ctx.StepID, fmt.Sprintf("Executing test case: %s", a.TestID))

	// Step 1: Load test case
	testCase, err := ctx.TestCaseRepo.GetTestCase(a.TestID)
	if err != nil {
		return nil, fmt.Errorf("test case not found: %s", a.TestID)
	}

	// Step 2: Apply input variables
	testCaseWithInput := a.applyInputVariables(testCase, ctx.Variables, a.Input)

	// Step 3: Execute test case
	ctx.Logger.Debug(ctx.StepID, fmt.Sprintf("Invoking UnifiedTestExecutor for test: %s", a.TestID))
	result := ctx.UnifiedExecutor.Execute(testCaseWithInput)

	// Step 4: Convert result
	if result.Status != "passed" {
		return &workflow.ActionResult{
			Status: "failed",
			Error:  fmt.Errorf("test case failed: %s", result.Error),
		}, nil
	}

	// Step 5: Extract response as output
	output := map[string]interface{}{
		"testId":   result.TestID,
		"status":   result.Status,
		"duration": result.Duration.Milliseconds(),
		"response": result.Response,
	}

	ctx.Logger.Info(ctx.StepID, fmt.Sprintf("Test case %s completed with status: %s", a.TestID, result.Status))

	return &workflow.ActionResult{
		Status:   "success",
		Output:   output,
		Duration: int(result.Duration.Milliseconds()),
	}, nil
}

// Validate validates the action configuration
func (a *TestCaseAction) Validate() error {
	if a.TestID == "" {
		return fmt.Errorf("testId is required")
	}
	return nil
}

// applyInputVariables applies input variables to test case configuration
func (a *TestCaseAction) applyInputVariables(
	testCase *models.TestCase,
	contextVars map[string]interface{},
	inputMapping map[string]interface{},
) *testcase.TestCase {
	// Clone test case
	cloned := &testcase.TestCase{
		ID:         testCase.TestID,
		Name:       testCase.Name,
		Type:       testCase.Type,
		Assertions: convertAssertions(testCase.Assertions),
	}

	// Apply variable replacement based on type
	switch testCase.Type {
	case "http":
		cloned.HTTP = a.replaceHTTPVariables(testCase.HTTPConfig, contextVars, inputMapping)
	case "command":
		cloned.Command = a.replaceCommandVariables(testCase.CommandConfig, contextVars, inputMapping)
	}

	return cloned
}

// replaceHTTPVariables replaces variables in HTTP configuration
func (a *TestCaseAction) replaceHTTPVariables(
	config models.JSONB,
	contextVars map[string]interface{},
	inputMapping map[string]interface{},
) *testcase.HTTPTest {
	// Convert JSONB to JSON string
	configJSON, _ := json.Marshal(config)
	str := string(configJSON)

	// Replace {{variableName}} placeholders
	for key, value := range inputMapping {
		placeholder := fmt.Sprintf("{{%s}}", key)
		str = strings.ReplaceAll(str, placeholder, fmt.Sprint(value))
	}

	// Deserialize back to HTTP config
	var httpTest testcase.HTTPTest
	json.Unmarshal([]byte(str), &httpTest)

	return &httpTest
}

// replaceCommandVariables replaces variables in Command configuration
func (a *TestCaseAction) replaceCommandVariables(
	config models.JSONB,
	contextVars map[string]interface{},
	inputMapping map[string]interface{},
) *testcase.CommandTest {
	configJSON, _ := json.Marshal(config)
	str := string(configJSON)

	for key, value := range inputMapping {
		placeholder := fmt.Sprintf("{{%s}}", key)
		str = strings.ReplaceAll(str, placeholder, fmt.Sprint(value))
	}

	var cmdTest testcase.CommandTest
	json.Unmarshal([]byte(str), &cmdTest)

	return &cmdTest
}

// convertAssertions converts models.JSONArray to []testcase.Assertion
func convertAssertions(jsonArray models.JSONArray) []testcase.Assertion {
	if jsonArray == nil {
		return nil
	}

	var assertions []testcase.Assertion
	data, _ := json.Marshal(jsonArray)
	json.Unmarshal(data, &assertions)

	return assertions
}

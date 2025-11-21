// Package testcase provides test case type definitions
package testcase

import "time"

// TestCase represents a test case to be executed
type TestCase struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Type       string       `json:"type"` // http, command, workflow, integration, etc.
	GroupID    string       `json:"groupId,omitempty"`
	Priority   string       `json:"priority,omitempty"`
	HTTP       *HTTPTest    `json:"http,omitempty"`
	Command    *CommandTest `json:"command,omitempty"`
	Assertions []Assertion  `json:"assertions,omitempty"`

	// Workflow integration support
	WorkflowID  string      `json:"workflowId,omitempty"`  // Mode 1: Reference workflow ID
	WorkflowDef interface{} `json:"workflowDef,omitempty"` // Mode 2: Embedded workflow definition

	// Lifecycle hooks
	SetupHooks    []Hook `json:"setupHooks,omitempty"`
	TeardownHooks []Hook `json:"teardownHooks,omitempty"`
}

// HTTPTest represents an HTTP test configuration
type HTTPTest struct {
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
}

// CommandTest represents a command line test configuration
type CommandTest struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
	Timeout int      `json:"timeout,omitempty"` // seconds
}

// Assertion represents a test assertion
type Assertion struct {
	Type     string      `json:"type"`     // status_code, json_path, exit_code, stdout_contains, etc.
	Path     string      `json:"path,omitempty"`
	Expected interface{} `json:"expected,omitempty"`
	Operator string      `json:"operator,omitempty"` // equals, in, exists, contains, etc.
}

// Hook represents a lifecycle hook (setup or teardown)
type Hook struct {
	Type            string       `json:"type"`                      // http, command, sql
	Name            string       `json:"name"`                      // descriptive name
	HTTP            *HTTPTest    `json:"http,omitempty"`            // HTTP hook configuration
	Command         *CommandTest `json:"command,omitempty"`         // Command hook configuration
	SaveResponse    string       `json:"saveResponse,omitempty"`    // variable name to store response
	RunOnFailure    bool         `json:"runOnFailure,omitempty"`    // for teardown: run even if test fails
	ContinueOnError bool         `json:"continueOnError,omitempty"` // don't stop if hook fails
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestID    string                 `json:"testId"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"` // passed, failed, error
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Failures  []string               `json:"failures,omitempty"`
	Request   map[string]interface{} `json:"request,omitempty"`
	Response  map[string]interface{} `json:"response,omitempty"`
}

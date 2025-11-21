package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"gorm.io/gorm"
)

// TestCase 测试案例模型
type TestCase struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	TestID          string         `gorm:"uniqueIndex;size:255;not null" json:"testId"`
	GroupID         string         `gorm:"size:255;not null;index" json:"groupId"`
	Name            string         `gorm:"size:255;not null" json:"name"`
	Type            string         `gorm:"size:50;not null;index" json:"type"`  // http, command, integration, etc.
	Priority        string         `gorm:"size:10;index" json:"priority"`        // P0, P1, P2
	Status          string         `gorm:"size:50;default:'active';index" json:"status"` // active, inactive
	Objective       string         `gorm:"type:text" json:"objective,omitempty"`
	Timeout         int            `gorm:"default:300" json:"timeout,omitempty"` // seconds

	// Workflow integration support
	WorkflowID      string         `gorm:"size:255;index" json:"workflowId,omitempty"`       // Mode 1: Reference workflow ID
	WorkflowDef     JSONB          `gorm:"type:text;column:workflow_def" json:"workflowDef,omitempty"` // Mode 2: Embedded workflow definition

	// JSON字段 - 使用自定义类型自动序列化
	Preconditions     JSONArray  `gorm:"type:text" json:"preconditions,omitempty"`
	Steps             JSONArray  `gorm:"type:text" json:"steps,omitempty"`
	HTTPConfig        JSONB      `gorm:"type:text;column:http_config" json:"http,omitempty"`
	CommandConfig     JSONB      `gorm:"type:text;column:command_config" json:"command,omitempty"`
	IntegrationConfig JSONB      `gorm:"type:text;column:integration_config" json:"integration,omitempty"`
	PerformanceConfig JSONB      `gorm:"type:text;column:performance_config" json:"performance,omitempty"`
	DatabaseConfig    JSONB      `gorm:"type:text;column:database_config" json:"database,omitempty"`
	SecurityConfig    JSONB      `gorm:"type:text;column:security_config" json:"security,omitempty"`
	GRPCConfig        JSONB      `gorm:"type:text;column:grpc_config" json:"grpc,omitempty"`
	WebSocketConfig   JSONB      `gorm:"type:text;column:websocket_config" json:"websocket,omitempty"`
	E2EConfig         JSONB      `gorm:"type:text;column:e2e_config" json:"e2e,omitempty"`
	Assertions        JSONArray  `gorm:"type:text" json:"assertions,omitempty"`
	Tags              JSONArray  `gorm:"type:text" json:"tags,omitempty"`
	CustomConfig      JSONB      `gorm:"type:text;column:custom_config" json:"custom,omitempty"`

	// Lifecycle hooks
	SetupHooks    JSONArray `gorm:"type:text;column:setup_hooks" json:"setupHooks,omitempty"`
	TeardownHooks JSONArray `gorm:"type:text;column:teardown_hooks" json:"teardownHooks,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Group    *TestGroup    `gorm:"foreignKey:GroupID;references:GroupID" json:"-"`
	Workflow *Workflow     `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"` // Workflow association for Mode 1
	Results  []TestResult  `gorm:"foreignKey:TestID;references:TestID" json:"-"`
}

// TableName 指定表名
func (TestCase) TableName() string {
	return "test_cases"
}

// TestResult 测试执行结果模型
type TestResult struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TestID    string    `gorm:"size:255;not null;index" json:"testId"`
	RunID     string    `gorm:"size:255;index" json:"runId,omitempty"`
	Status    string    `gorm:"size:50;not null;index" json:"status"` // passed, failed, error, skipped
	StartTime time.Time `gorm:"not null;index" json:"startTime"`
	EndTime   time.Time `json:"endTime,omitempty"`
	Duration  int       `json:"duration,omitempty"` // milliseconds
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	Failures  JSONArray `gorm:"type:text" json:"failures,omitempty"`
	Metrics   JSONB     `gorm:"type:text" json:"metrics,omitempty"`
	Artifacts JSONArray `gorm:"type:text" json:"artifacts,omitempty"`
	Logs      JSONArray `gorm:"type:text" json:"logs,omitempty"`
	CreatedAt time.Time `json:"createdAt"`

	// 关联
	TestCase *TestCase `gorm:"foreignKey:TestID;references:TestID" json:"-"`
}

// TableName 指定表名
func (TestResult) TableName() string {
	return "test_results"
}

// TestRun 测试批次执行模型
type TestRun struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RunID     string    `gorm:"uniqueIndex;size:255;not null" json:"runId"`
	Name      string    `gorm:"size:255" json:"name,omitempty"`
	Total     int       `gorm:"default:0" json:"total"`
	Passed    int       `gorm:"default:0" json:"passed"`
	Failed    int       `gorm:"default:0" json:"failed"`
	Errors    int       `gorm:"default:0" json:"errors"`
	Skipped   int       `gorm:"default:0" json:"skipped"`
	StartTime time.Time `gorm:"index" json:"startTime,omitempty"`
	EndTime   time.Time `json:"endTime,omitempty"`
	Duration  int       `json:"duration,omitempty"` // milliseconds
	Status    string    `gorm:"size:50;default:'running';index" json:"status"` // running, completed, cancelled
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// 关联
	Results []TestResult `gorm:"foreignKey:RunID;references:RunID" json:"results,omitempty"`
}

// TableName 指定表名
func (TestRun) TableName() string {
	return "test_runs"
}

// ===== 自定义JSON类型 =====

// JSONB 自定义JSON类型（用于对象）
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}
	return json.Unmarshal(bytes, j)
}

// JSONArray 自定义JSON数组类型
type JSONArray []interface{}

func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSONArray value: unsupported type %T", value)
	}

	// Handle empty array case
	if len(bytes) == 0 || string(bytes) == "[]" {
		*j = JSONArray{}
		return nil
	}

	if err := json.Unmarshal(bytes, j); err != nil {
		return fmt.Errorf("failed to unmarshal JSONArray value: %w (input: %s)", err, string(bytes))
	}
	return nil
}

package models

import (
	"time"
	"gorm.io/gorm"
)

// Workflow 工作流模型
type Workflow struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkflowID  string    `gorm:"uniqueIndex;size:255;not null" json:"workflowId"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Version     string    `gorm:"size:32" json:"version"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Definition  JSONB     `gorm:"type:text;not null" json:"definition"`

	// === 新增字段：测试案例关联 ===
	IsTestCase  bool      `gorm:"default:false;index" json:"isTestCase"`  // 是否被测试案例引用

	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy   string         `gorm:"size:64" json:"createdBy,omitempty"`

	// 关联
	TestCases []TestCase    `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`  // 新增：反向关联
	Runs      []WorkflowRun `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`
}

// TableName 指定表名
func (Workflow) TableName() string {
	return "workflows"
}

// WorkflowRun 工作流执行记录模型
type WorkflowRun struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RunID      string    `gorm:"uniqueIndex;size:255;not null" json:"runId"`
	WorkflowID string    `gorm:"size:255;not null;index" json:"workflowId"`
	Status     string    `gorm:"size:32;not null;index" json:"status"`  // running, success, failed, cancelled
	StartTime  time.Time `gorm:"index" json:"startTime"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Duration   int       `json:"duration,omitempty"`  // milliseconds
	Context    JSONB     `gorm:"type:text" json:"context,omitempty"`  // 执行上下文（变量、步骤结果）
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`

	// 关联
	Workflow *Workflow `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`
}

// TableName 指定表名
func (WorkflowRun) TableName() string {
	return "workflow_runs"
}

// WorkflowStepExecution 工作流步骤执行记录（用于实时数据流追踪）
type WorkflowStepExecution struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RunID      string    `gorm:"size:255;not null;index" json:"runId"`
	StepID     string    `gorm:"size:255;not null;index" json:"stepId"`
	StepName   string    `gorm:"size:255" json:"stepName"`
	Status     string    `gorm:"size:32;not null" json:"status"`  // pending, running, success, failed, skipped
	StartTime  time.Time `json:"startTime,omitempty"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Duration   int       `json:"duration,omitempty"`  // milliseconds
	InputData  JSONB     `gorm:"type:text" json:"inputData,omitempty"`   // 输入数据快照
	OutputData JSONB     `gorm:"type:text" json:"outputData,omitempty"`  // 输出数据快照
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`

	// 关联
	Run *WorkflowRun `gorm:"foreignKey:RunID;references:RunID" json:"-"`
}

// TableName 指定表名
func (WorkflowStepExecution) TableName() string {
	return "workflow_step_executions"
}

// WorkflowStepLog 工作流步骤日志记录
type WorkflowStepLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RunID     string    `gorm:"size:255;not null;index" json:"runId"`
	StepID    string    `gorm:"size:255;not null;index" json:"stepId"`
	Level     string    `gorm:"size:16;not null" json:"level"`  // debug, info, warn, error
	Message   string    `gorm:"type:text;not null" json:"message"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`

	// 关联
	Run *WorkflowRun `gorm:"foreignKey:RunID;references:RunID" json:"-"`
}

// TableName 指定表名
func (WorkflowStepLog) TableName() string {
	return "workflow_step_logs"
}

// WorkflowVariableChange 工作流变量变更历史
type WorkflowVariableChange struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RunID      string    `gorm:"size:255;not null;index" json:"runId"`
	StepID     string    `gorm:"size:255;index" json:"stepId"`  // 触发变更的步骤ID
	VarName    string    `gorm:"size:255;not null;index" json:"varName"`
	OldValue   JSONB     `gorm:"type:text" json:"oldValue,omitempty"`
	NewValue   JSONB     `gorm:"type:text" json:"newValue,omitempty"`
	ChangeType string    `gorm:"size:16;not null" json:"changeType"`  // create, update, delete
	Timestamp  time.Time `gorm:"index" json:"timestamp"`

	// 关联
	Run *WorkflowRun `gorm:"foreignKey:RunID;references:RunID" json:"-"`
}

// TableName 指定表名
func (WorkflowVariableChange) TableName() string {
	return "workflow_variable_changes"
}

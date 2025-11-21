package models

import (
	"time"
	"gorm.io/gorm"
)

// Environment represents a test environment (Dev/Staging/Prod)
// Stores environment configuration and can be activated for test execution
type Environment struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	EnvID       string         `gorm:"uniqueIndex;size:50;not null" json:"envId"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	IsActive    bool           `gorm:"default:false;index" json:"isActive"`
	Variables   JSONB          `gorm:"type:text;column:variables" json:"variables,omitempty"` // Using existing JSONB type
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations
	EnvironmentVariables []EnvironmentVariable `gorm:"foreignKey:EnvID;references:EnvID" json:"environmentVariables,omitempty"`
}

// TableName specifies the table name for Environment model
func (Environment) TableName() string {
	return "environments"
}

// EnvironmentVariable represents a single environment variable
// Supports different types and secret masking for sensitive data
type EnvironmentVariable struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	EnvID     string    `gorm:"size:50;not null;index" json:"envId"`
	Key       string    `gorm:"size:255;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	ValueType string    `gorm:"size:50;default:'string'" json:"valueType"` // string, number, boolean, json
	IsSecret  bool      `gorm:"default:false" json:"isSecret"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Associations
	Environment *Environment `gorm:"foreignKey:EnvID;references:EnvID" json:"-"`
}

// TableName specifies the table name for EnvironmentVariable model
func (EnvironmentVariable) TableName() string {
	return "environment_variables"
}

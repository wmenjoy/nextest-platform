package models

import (
	"time"
	"gorm.io/gorm"
)

// TestGroup 测试分组模型
type TestGroup struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	GroupID     string         `gorm:"uniqueIndex;size:255;not null" json:"groupId"`
	ParentID    string         `gorm:"size:255;index" json:"parentId,omitempty"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	TargetHost  string         `gorm:"size:512" json:"targetHost,omitempty"` // 测试目标服务地址

	// Lifecycle hooks for group-level setup/teardown
	SetupHooks    JSONArray `gorm:"type:text;column:setup_hooks" json:"setupHooks,omitempty"`
	TeardownHooks JSONArray `gorm:"type:text;column:teardown_hooks" json:"teardownHooks,omitempty"`

	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Children  []TestGroup `gorm:"foreignKey:ParentID;references:GroupID" json:"children,omitempty"`
	TestCases []TestCase  `gorm:"foreignKey:GroupID;references:GroupID" json:"testCases,omitempty"`
}

// TableName 指定表名
func (TestGroup) TableName() string {
	return "test_groups"
}

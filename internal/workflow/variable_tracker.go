package workflow

import (
	"time"

	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// DatabaseVariableChangeTracker tracks variable changes to database
type DatabaseVariableChangeTracker struct {
	db    *gorm.DB
	runID string
}

// NewDatabaseVariableChangeTracker creates a tracker
func NewDatabaseVariableChangeTracker(db *gorm.DB, runID string) *DatabaseVariableChangeTracker {
	return &DatabaseVariableChangeTracker{
		db:    db,
		runID: runID,
	}
}

func (t *DatabaseVariableChangeTracker) Track(stepID, varName string, oldValue, newValue interface{}, changeType string) {
	change := &models.WorkflowVariableChange{
		RunID:      t.runID,
		StepID:     stepID,
		VarName:    varName,
		OldValue:   models.JSONB(map[string]interface{}{"value": oldValue}),
		NewValue:   models.JSONB(map[string]interface{}{"value": newValue}),
		ChangeType: changeType,
		Timestamp:  time.Now(),
	}

	t.db.Create(change)
}

package workflow

import (
	"fmt"
	"time"

	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// DatabaseStepLogger logs to database
type DatabaseStepLogger struct {
	db    *gorm.DB
	runID string
}

// NewDatabaseStepLogger creates a database logger
func NewDatabaseStepLogger(db *gorm.DB, runID string) *DatabaseStepLogger {
	return &DatabaseStepLogger{
		db:    db,
		runID: runID,
	}
}

func (l *DatabaseStepLogger) log(level, stepID, message string) {
	logEntry := &models.WorkflowStepLog{
		RunID:     l.runID,
		StepID:    stepID,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	}
	l.db.Create(logEntry)
}

func (l *DatabaseStepLogger) Debug(stepID, message string) {
	l.log("debug", stepID, message)
	fmt.Printf("[DEBUG] Step %s: %s\n", stepID, message)
}

func (l *DatabaseStepLogger) Info(stepID, message string) {
	l.log("info", stepID, message)
	fmt.Printf("[INFO] Step %s: %s\n", stepID, message)
}

func (l *DatabaseStepLogger) Warn(stepID, message string) {
	l.log("warn", stepID, message)
	fmt.Printf("[WARN] Step %s: %s\n", stepID, message)
}

func (l *DatabaseStepLogger) Error(stepID, message string) {
	l.log("error", stepID, message)
	fmt.Printf("[ERROR] Step %s: %s\n", stepID, message)
}

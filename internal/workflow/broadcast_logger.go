package workflow

import (
	"fmt"
	"time"

	"test-management-service/internal/models"
	ws "test-management-service/internal/websocket"

	"gorm.io/gorm"
)

// BroadcastStepLogger logs to database and broadcasts via WebSocket
type BroadcastStepLogger struct {
	db    *gorm.DB
	runID string
	hub   *ws.Hub
}

// NewBroadcastStepLogger creates a logger that broadcasts events
func NewBroadcastStepLogger(db *gorm.DB, runID string, hub *ws.Hub) *BroadcastStepLogger {
	return &BroadcastStepLogger{
		db:    db,
		runID: runID,
		hub:   hub,
	}
}

func (l *BroadcastStepLogger) log(level, stepID, message string) {
	logEntry := &models.WorkflowStepLog{
		RunID:     l.runID,
		StepID:    stepID,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	}
	l.db.Create(logEntry)

	// Broadcast to WebSocket clients
	if l.hub != nil {
		l.hub.Broadcast(l.runID, "step_log", map[string]interface{}{
			"stepId":    stepID,
			"level":     level,
			"message":   message,
			"timestamp": logEntry.Timestamp,
		})
	}

	// Console output
	fmt.Printf("[%s] Step %s: %s\n", level, stepID, message)
}

func (l *BroadcastStepLogger) Debug(stepID, message string) {
	l.log("debug", stepID, message)
}

func (l *BroadcastStepLogger) Info(stepID, message string) {
	l.log("info", stepID, message)
}

func (l *BroadcastStepLogger) Warn(stepID, message string) {
	l.log("warn", stepID, message)
}

func (l *BroadcastStepLogger) Error(stepID, message string) {
	l.log("error", stepID, message)
}

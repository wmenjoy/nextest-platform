package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// StepLogRepository handles step log data access
type StepLogRepository struct {
	db *gorm.DB
}

// NewStepLogRepository creates a new repository
func NewStepLogRepository(db *gorm.DB) *StepLogRepository {
	return &StepLogRepository{db: db}
}

// Create creates a new step log entry
func (r *StepLogRepository) Create(log *models.WorkflowStepLog) error {
	result := r.db.Create(log)
	if result.Error != nil {
		return fmt.Errorf("failed to create step log: %w", result.Error)
	}
	return nil
}

// ListByRunID retrieves all logs for a run
func (r *StepLogRepository) ListByRunID(runID string, level *string) ([]models.WorkflowStepLog, error) {
	var logs []models.WorkflowStepLog

	query := r.db.Where("run_id = ?", runID)
	if level != nil {
		query = query.Where("level = ?", *level)
	}

	result := query.Order("timestamp ASC").Find(&logs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list step logs: %w", result.Error)
	}

	return logs, nil
}

// ListByStepID retrieves all logs for a specific step
func (r *StepLogRepository) ListByStepID(runID, stepID string) ([]models.WorkflowStepLog, error) {
	var logs []models.WorkflowStepLog

	result := r.db.Where("run_id = ? AND step_id = ?", runID, stepID).
		Order("timestamp ASC").
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list step logs: %w", result.Error)
	}

	return logs, nil
}

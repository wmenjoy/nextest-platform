package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// StepExecutionRepository handles step execution data access
type StepExecutionRepository struct {
	db *gorm.DB
}

// NewStepExecutionRepository creates a new repository
func NewStepExecutionRepository(db *gorm.DB) *StepExecutionRepository {
	return &StepExecutionRepository{db: db}
}

// Create creates a new step execution record
func (r *StepExecutionRepository) Create(stepExec *models.WorkflowStepExecution) error {
	result := r.db.Create(stepExec)
	if result.Error != nil {
		return fmt.Errorf("failed to create step execution: %w", result.Error)
	}
	return nil
}

// Update updates a step execution record
func (r *StepExecutionRepository) Update(stepExec *models.WorkflowStepExecution) error {
	result := r.db.Save(stepExec)
	if result.Error != nil {
		return fmt.Errorf("failed to update step execution: %w", result.Error)
	}
	return nil
}

// ListByRunID retrieves all step executions for a run
func (r *StepExecutionRepository) ListByRunID(runID string) ([]models.WorkflowStepExecution, error) {
	var executions []models.WorkflowStepExecution

	result := r.db.Where("run_id = ?", runID).Order("start_time ASC").Find(&executions)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list step executions: %w", result.Error)
	}

	return executions, nil
}

// GetByStepID retrieves a specific step execution
func (r *StepExecutionRepository) GetByStepID(runID, stepID string) (*models.WorkflowStepExecution, error) {
	var execution models.WorkflowStepExecution

	result := r.db.Where("run_id = ? AND step_id = ?", runID, stepID).First(&execution)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("step execution not found: %s/%s", runID, stepID)
		}
		return nil, fmt.Errorf("failed to query step execution: %w", result.Error)
	}

	return &execution, nil
}

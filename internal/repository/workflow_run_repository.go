package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// WorkflowRunRepository handles workflow run data access
type WorkflowRunRepository struct {
	db *gorm.DB
}

// NewWorkflowRunRepository creates a new repository
func NewWorkflowRunRepository(db *gorm.DB) *WorkflowRunRepository {
	return &WorkflowRunRepository{db: db}
}

// Create creates a new workflow run record
func (r *WorkflowRunRepository) Create(run *models.WorkflowRun) error {
	result := r.db.Create(run)
	if result.Error != nil {
		return fmt.Errorf("failed to create workflow run: %w", result.Error)
	}
	return nil
}

// Update updates a workflow run record
func (r *WorkflowRunRepository) Update(run *models.WorkflowRun) error {
	result := r.db.Save(run)
	if result.Error != nil {
		return fmt.Errorf("failed to update workflow run: %w", result.Error)
	}
	return nil
}

// GetByRunID retrieves a workflow run by runID
func (r *WorkflowRunRepository) GetByRunID(runID string) (*models.WorkflowRun, error) {
	var run models.WorkflowRun

	result := r.db.Where("run_id = ?", runID).First(&run)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to query workflow run: %w", result.Error)
	}

	return &run, nil
}

// ListByWorkflowID retrieves all runs for a workflow
func (r *WorkflowRunRepository) ListByWorkflowID(workflowID string, limit int) ([]models.WorkflowRun, error) {
	var runs []models.WorkflowRun

	query := r.db.Where("workflow_id = ?", workflowID).Order("start_time DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	result := query.Find(&runs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", result.Error)
	}

	return runs, nil
}

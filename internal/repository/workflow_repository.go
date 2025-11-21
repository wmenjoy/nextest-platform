package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// WorkflowRepository handles workflow data access
type WorkflowRepository struct {
	db *gorm.DB
}

// NewWorkflowRepository creates a new repository
func NewWorkflowRepository(db *gorm.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// GetWorkflow retrieves a workflow by workflowID
func (r *WorkflowRepository) GetWorkflow(workflowID string) (*models.Workflow, error) {
	var workflow models.Workflow

	result := r.db.Where("workflow_id = ? AND deleted_at IS NULL", workflowID).First(&workflow)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to query workflow: %w", result.Error)
	}

	return &workflow, nil
}

// ListWorkflows retrieves all workflows with optional filters
func (r *WorkflowRepository) ListWorkflows(isTestCase *bool) ([]models.Workflow, error) {
	var workflows []models.Workflow

	query := r.db.Where("deleted_at IS NULL")
	if isTestCase != nil {
		query = query.Where("is_test_case = ?", *isTestCase)
	}

	result := query.Find(&workflows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", result.Error)
	}

	return workflows, nil
}

// CreateWorkflow creates a new workflow
func (r *WorkflowRepository) CreateWorkflow(workflow *models.Workflow) error {
	result := r.db.Create(workflow)
	if result.Error != nil {
		return fmt.Errorf("failed to create workflow: %w", result.Error)
	}
	return nil
}

// UpdateWorkflow updates an existing workflow
func (r *WorkflowRepository) UpdateWorkflow(workflow *models.Workflow) error {
	result := r.db.Save(workflow)
	if result.Error != nil {
		return fmt.Errorf("failed to update workflow: %w", result.Error)
	}
	return nil
}

// DeleteWorkflow soft deletes a workflow
func (r *WorkflowRepository) DeleteWorkflow(workflowID string) error {
	result := r.db.Where("workflow_id = ?", workflowID).Delete(&models.Workflow{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete workflow: %w", result.Error)
	}
	return nil
}

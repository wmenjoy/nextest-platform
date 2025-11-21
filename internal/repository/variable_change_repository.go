package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// VariableChangeRepository handles variable change tracking
type VariableChangeRepository struct {
	db *gorm.DB
}

// NewVariableChangeRepository creates a new repository
func NewVariableChangeRepository(db *gorm.DB) *VariableChangeRepository {
	return &VariableChangeRepository{db: db}
}

// Create creates a new variable change record
func (r *VariableChangeRepository) Create(change *models.WorkflowVariableChange) error {
	result := r.db.Create(change)
	if result.Error != nil {
		return fmt.Errorf("failed to create variable change: %w", result.Error)
	}
	return nil
}

// ListByRunID retrieves all variable changes for a run
func (r *VariableChangeRepository) ListByRunID(runID string) ([]models.WorkflowVariableChange, error) {
	var changes []models.WorkflowVariableChange

	result := r.db.Where("run_id = ?", runID).
		Order("timestamp ASC").
		Find(&changes)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list variable changes: %w", result.Error)
	}

	return changes, nil
}

// ListByVariableName retrieves all changes for a specific variable
func (r *VariableChangeRepository) ListByVariableName(runID, varName string) ([]models.WorkflowVariableChange, error) {
	var changes []models.WorkflowVariableChange

	result := r.db.Where("run_id = ? AND var_name = ?", runID, varName).
		Order("timestamp ASC").
		Find(&changes)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list variable changes: %w", result.Error)
	}

	return changes, nil
}

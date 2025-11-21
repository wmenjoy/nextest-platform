package repository

import (
	"fmt"
	"test-management-service/internal/models"
	"gorm.io/gorm"
)

// WorkflowTestCaseRepository handles test case data access for workflow execution
type WorkflowTestCaseRepository struct {
	db *gorm.DB
}

// NewWorkflowTestCaseRepository creates a new repository
func NewWorkflowTestCaseRepository(db *gorm.DB) *WorkflowTestCaseRepository {
	return &WorkflowTestCaseRepository{db: db}
}

// GetTestCase retrieves a test case by testID
func (r *WorkflowTestCaseRepository) GetTestCase(testID string) (*models.TestCase, error) {
	var testCase models.TestCase

	result := r.db.Where("test_id = ? AND deleted_at IS NULL", testID).First(&testCase)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("test case not found: %s", testID)
		}
		return nil, fmt.Errorf("failed to query test case: %w", result.Error)
	}

	return &testCase, nil
}

// GetTestCasesByWorkflowID retrieves all test cases referencing a workflow
func (r *WorkflowTestCaseRepository) GetTestCasesByWorkflowID(workflowID string) ([]models.TestCase, error) {
	var testCases []models.TestCase

	result := r.db.Where("workflow_id = ? AND deleted_at IS NULL", workflowID).Find(&testCases)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query test cases: %w", result.Error)
	}

	return testCases, nil
}

// CreateTestCase creates a new test case
func (r *WorkflowTestCaseRepository) CreateTestCase(testCase *models.TestCase) error {
	result := r.db.Create(testCase)
	if result.Error != nil {
		return fmt.Errorf("failed to create test case: %w", result.Error)
	}
	return nil
}

// UpdateTestCase updates an existing test case
func (r *WorkflowTestCaseRepository) UpdateTestCase(testCase *models.TestCase) error {
	result := r.db.Save(testCase)
	if result.Error != nil {
		return fmt.Errorf("failed to update test case: %w", result.Error)
	}
	return nil
}

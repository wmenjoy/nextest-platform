package service

import (
	"fmt"
	"time"

	"test-management-service/internal/models"
	"test-management-service/internal/repository"
	"test-management-service/internal/workflow"
)

// WorkflowService handles workflow operations
type WorkflowService interface {
	CreateWorkflow(req *CreateWorkflowRequest) (*models.Workflow, error)
	UpdateWorkflow(workflowID string, req *UpdateWorkflowRequest) (*models.Workflow, error)
	DeleteWorkflow(workflowID string) error
	GetWorkflow(workflowID string) (*models.Workflow, error)
	ListWorkflows(isTestCase *bool, limit, offset int) ([]models.Workflow, int64, error)

	ExecuteWorkflow(workflowID string, variables map[string]interface{}) (*models.WorkflowRun, error)
	GetWorkflowRun(runID string) (*models.WorkflowRun, error)
	ListWorkflowRuns(workflowID string, limit, offset int) ([]models.WorkflowRun, int64, error)

	GetWorkflowTestCases(workflowID string) ([]models.TestCase, error)
	GetStepExecutions(runID string) ([]models.WorkflowStepExecution, error)
	GetStepLogs(runID string, stepID *string, level *string) ([]models.WorkflowStepLog, error)
}

type workflowService struct {
	workflowRepo     *repository.WorkflowRepository
	workflowRunRepo  *repository.WorkflowRunRepository
	stepExecRepo     *repository.StepExecutionRepository
	stepLogRepo      *repository.StepLogRepository
	testCaseRepo     *repository.WorkflowTestCaseRepository
	executor         *workflow.WorkflowExecutorImpl
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	workflowRepo *repository.WorkflowRepository,
	workflowRunRepo *repository.WorkflowRunRepository,
	stepExecRepo *repository.StepExecutionRepository,
	stepLogRepo *repository.StepLogRepository,
	testCaseRepo *repository.WorkflowTestCaseRepository,
	executor *workflow.WorkflowExecutorImpl,
) WorkflowService {
	return &workflowService{
		workflowRepo:    workflowRepo,
		workflowRunRepo: workflowRunRepo,
		stepExecRepo:    stepExecRepo,
		stepLogRepo:     stepLogRepo,
		testCaseRepo:    testCaseRepo,
		executor:        executor,
	}
}

// ===== DTOs =====

type CreateWorkflowRequest struct {
	WorkflowID  string                 `json:"workflowId" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Definition  map[string]interface{} `json:"definition" binding:"required"`
	IsTestCase  bool                   `json:"isTestCase"`
	CreatedBy   string                 `json:"createdBy"`
}

type UpdateWorkflowRequest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	IsTestCase  *bool                  `json:"isTestCase"`
}

type ExecuteWorkflowRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

// ===== Implementation =====

func (s *workflowService) CreateWorkflow(req *CreateWorkflowRequest) (*models.Workflow, error) {
	workflow := &models.Workflow{
		WorkflowID:  req.WorkflowID,
		Name:        req.Name,
		Version:     req.Version,
		Description: req.Description,
		Definition:  models.JSONB(req.Definition),
		IsTestCase:  req.IsTestCase,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.workflowRepo.CreateWorkflow(workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

func (s *workflowService) UpdateWorkflow(workflowID string, req *UpdateWorkflowRequest) (*models.Workflow, error) {
	workflow, err := s.workflowRepo.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		workflow.Name = req.Name
	}
	if req.Version != "" {
		workflow.Version = req.Version
	}
	if req.Description != "" {
		workflow.Description = req.Description
	}
	if req.Definition != nil {
		workflow.Definition = models.JSONB(req.Definition)
	}
	if req.IsTestCase != nil {
		workflow.IsTestCase = *req.IsTestCase
	}

	workflow.UpdatedAt = time.Now()

	if err := s.workflowRepo.UpdateWorkflow(workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	return workflow, nil
}

func (s *workflowService) DeleteWorkflow(workflowID string) error {
	return s.workflowRepo.DeleteWorkflow(workflowID)
}

func (s *workflowService) GetWorkflow(workflowID string) (*models.Workflow, error) {
	return s.workflowRepo.GetWorkflow(workflowID)
}

func (s *workflowService) ListWorkflows(isTestCase *bool, limit, offset int) ([]models.Workflow, int64, error) {
	workflows, err := s.workflowRepo.ListWorkflows(isTestCase)
	if err != nil {
		return nil, 0, err
	}

	// Simple pagination
	total := int64(len(workflows))
	start := offset
	end := offset + limit
	if start > len(workflows) {
		start = len(workflows)
	}
	if end > len(workflows) {
		end = len(workflows)
	}

	return workflows[start:end], total, nil
}

func (s *workflowService) ExecuteWorkflow(workflowID string, variables map[string]interface{}) (*models.WorkflowRun, error) {
	// Get workflow definition from database
	workflow, err := s.workflowRepo.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Execute workflow via executor
	result, err := s.executor.Execute(workflowID, workflow.Definition)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Get the saved run record
	run, err := s.workflowRunRepo.GetByRunID(result.RunID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve run record: %w", err)
	}

	return run, nil
}

func (s *workflowService) GetWorkflowRun(runID string) (*models.WorkflowRun, error) {
	return s.workflowRunRepo.GetByRunID(runID)
}

func (s *workflowService) ListWorkflowRuns(workflowID string, limit, offset int) ([]models.WorkflowRun, int64, error) {
	runs, err := s.workflowRunRepo.ListByWorkflowID(workflowID, 0)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(runs))
	start := offset
	end := offset + limit
	if start > len(runs) {
		start = len(runs)
	}
	if end > len(runs) {
		end = len(runs)
	}

	return runs[start:end], total, nil
}

func (s *workflowService) GetWorkflowTestCases(workflowID string) ([]models.TestCase, error) {
	return s.testCaseRepo.GetTestCasesByWorkflowID(workflowID)
}

func (s *workflowService) GetStepExecutions(runID string) ([]models.WorkflowStepExecution, error) {
	return s.stepExecRepo.ListByRunID(runID)
}

func (s *workflowService) GetStepLogs(runID string, stepID *string, level *string) ([]models.WorkflowStepLog, error) {
	if stepID != nil {
		return s.stepLogRepo.ListByStepID(runID, *stepID)
	}
	return s.stepLogRepo.ListByRunID(runID, level)
}

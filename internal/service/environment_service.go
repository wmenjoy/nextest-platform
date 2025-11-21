package service

import (
	"fmt"
	"test-management-service/internal/models"
	"test-management-service/internal/repository"
)

// EnvironmentService 环境管理服务接口
type EnvironmentService interface {
	// Environment CRUD
	CreateEnvironment(req *CreateEnvironmentRequest) (*models.Environment, error)
	UpdateEnvironment(envID string, req *UpdateEnvironmentRequest) (*models.Environment, error)
	DeleteEnvironment(envID string) error
	GetEnvironment(envID string) (*models.Environment, error)
	ListEnvironments(limit, offset int) ([]models.Environment, int64, error)

	// Environment Activation
	GetActiveEnvironment() (*models.Environment, error)
	ActivateEnvironment(envID string) error

	// Variable Management
	GetVariables(envID string) (map[string]interface{}, error)
	GetVariable(envID, key string) (interface{}, error)
	SetVariable(envID, key string, value interface{}) error
	DeleteVariable(envID, key string) error
}

type environmentService struct {
	envRepo    repository.EnvironmentRepository
	envVarRepo repository.EnvironmentVariableRepository
}

// NewEnvironmentService 创建环境管理服务
func NewEnvironmentService(
	envRepo repository.EnvironmentRepository,
	envVarRepo repository.EnvironmentVariableRepository,
) EnvironmentService {
	return &environmentService{
		envRepo:    envRepo,
		envVarRepo: envVarRepo,
	}
}

// ===== Request/Response DTOs =====

type CreateEnvironmentRequest struct {
	EnvID       string                 `json:"envId" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Variables   map[string]interface{} `json:"variables"`
}

type UpdateEnvironmentRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Variables   map[string]interface{} `json:"variables"`
}

type SetVariableRequest struct {
	Value interface{} `json:"value" binding:"required"`
}

// ===== Implementation =====

func (s *environmentService) CreateEnvironment(req *CreateEnvironmentRequest) (*models.Environment, error) {
	// 检查 envId 是否已存在
	existing, _ := s.envRepo.FindByID(req.EnvID)
	if existing != nil {
		return nil, fmt.Errorf("environment with envId '%s' already exists", req.EnvID)
	}

	env := &models.Environment{
		EnvID:       req.EnvID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    false, // 新环境默认不激活
		Variables:   models.JSONB(req.Variables),
	}

	if err := s.envRepo.Create(env); err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	return env, nil
}

func (s *environmentService) UpdateEnvironment(envID string, req *UpdateEnvironmentRequest) (*models.Environment, error) {
	env, err := s.envRepo.FindByID(envID)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envID)
	}

	if req.Name != "" {
		env.Name = req.Name
	}
	if req.Description != "" {
		env.Description = req.Description
	}
	if req.Variables != nil {
		env.Variables = models.JSONB(req.Variables)
	}

	if err := s.envRepo.Update(env); err != nil {
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}

	return env, nil
}

func (s *environmentService) DeleteEnvironment(envID string) error {
	// 不允许删除激活的环境
	env, err := s.envRepo.FindByID(envID)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envID)
	}

	if env.IsActive {
		return fmt.Errorf("cannot delete active environment '%s'", envID)
	}

	return s.envRepo.Delete(envID)
}

func (s *environmentService) GetEnvironment(envID string) (*models.Environment, error) {
	return s.envRepo.FindByID(envID)
}

func (s *environmentService) ListEnvironments(limit, offset int) ([]models.Environment, int64, error) {
	return s.envRepo.FindAll(limit, offset)
}

func (s *environmentService) GetActiveEnvironment() (*models.Environment, error) {
	env, err := s.envRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("no active environment found")
	}
	return env, nil
}

func (s *environmentService) ActivateEnvironment(envID string) error {
	// 检查环境是否存在
	_, err := s.envRepo.FindByID(envID)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envID)
	}

	return s.envRepo.SetActive(envID)
}

func (s *environmentService) GetVariables(envID string) (map[string]interface{}, error) {
	env, err := s.envRepo.FindByID(envID)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envID)
	}

	return env.Variables, nil
}

func (s *environmentService) GetVariable(envID, key string) (interface{}, error) {
	vars, err := s.GetVariables(envID)
	if err != nil {
		return nil, err
	}

	value, exists := vars[key]
	if !exists {
		return nil, fmt.Errorf("variable '%s' not found in environment '%s'", key, envID)
	}

	return value, nil
}

func (s *environmentService) SetVariable(envID, key string, value interface{}) error {
	env, err := s.envRepo.FindByID(envID)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envID)
	}

	if env.Variables == nil {
		env.Variables = make(models.JSONB)
	}

	env.Variables[key] = value

	return s.envRepo.Update(env)
}

func (s *environmentService) DeleteVariable(envID, key string) error {
	env, err := s.envRepo.FindByID(envID)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envID)
	}

	if env.Variables == nil {
		return fmt.Errorf("no variables found in environment '%s'", envID)
	}

	delete(env.Variables, key)

	return s.envRepo.Update(env)
}

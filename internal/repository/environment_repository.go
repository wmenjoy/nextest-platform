package repository

import (
	"errors"
	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// EnvironmentRepository defines the interface for environment data access operations
// Provides methods for CRUD operations and environment activation management
type EnvironmentRepository interface {
	// Create creates a new environment
	Create(env *models.Environment) error

	// Update updates an existing environment
	Update(env *models.Environment) error

	// Delete soft-deletes an environment by envID
	Delete(envID string) error

	// FindByID retrieves an environment by envID with preloaded variables
	FindByID(envID string) (*models.Environment, error)

	// FindAll retrieves all environments with pagination
	// Returns environments slice, total count, and error
	FindAll(limit, offset int) ([]models.Environment, int64, error)

	// FindActive retrieves the currently active environment with preloaded variables
	FindActive() (*models.Environment, error)

	// SetActive sets the specified environment as active
	// Deactivates all other environments in a transaction
	SetActive(envID string) error
}

// environmentRepository implements EnvironmentRepository interface
type environmentRepository struct {
	db *gorm.DB
}

// NewEnvironmentRepository creates a new EnvironmentRepository instance
func NewEnvironmentRepository(db *gorm.DB) EnvironmentRepository {
	return &environmentRepository{db: db}
}

// Create creates a new environment in the database
func (r *environmentRepository) Create(env *models.Environment) error {
	return r.db.Create(env).Error
}

// Update updates an existing environment in the database
func (r *environmentRepository) Update(env *models.Environment) error {
	return r.db.Save(env).Error
}

// Delete soft-deletes an environment by envID
func (r *environmentRepository) Delete(envID string) error {
	return r.db.Where("env_id = ?", envID).Delete(&models.Environment{}).Error
}

// FindByID retrieves an environment by envID
// Preloads associated environment variables and filters soft-deleted records
func (r *environmentRepository) FindByID(envID string) (*models.Environment, error) {
	var env models.Environment
	err := r.db.Preload("EnvironmentVariables").
		Where("env_id = ?", envID).
		First(&env).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &env, nil
}

// FindAll retrieves all environments with pagination
// Returns environments, total count, and error
func (r *environmentRepository) FindAll(limit, offset int) ([]models.Environment, int64, error) {
	var environments []models.Environment
	var total int64

	// Count total records (excluding soft-deleted)
	if err := r.db.Model(&models.Environment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated results with preloaded variables
	err := r.db.Preload("EnvironmentVariables").
		Limit(limit).
		Offset(offset).
		Find(&environments).Error

	return environments, total, err
}

// FindActive retrieves the currently active environment
// Preloads associated environment variables
func (r *environmentRepository) FindActive() (*models.Environment, error) {
	var env models.Environment
	err := r.db.Preload("EnvironmentVariables").
		Where("is_active = ?", true).
		First(&env).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &env, nil
}

// SetActive sets the specified environment as active
// Uses transaction to ensure atomicity:
// 1. Deactivates all environments
// 2. Activates the specified environment
func (r *environmentRepository) SetActive(envID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate all environments
		if err := tx.Model(&models.Environment{}).
			Where("is_active = ?", true).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// Activate the specified environment
		result := tx.Model(&models.Environment{}).
			Where("env_id = ?", envID).
			Update("is_active", true)

		if result.Error != nil {
			return result.Error
		}

		// Check if the environment exists
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

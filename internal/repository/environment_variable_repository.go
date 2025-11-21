package repository

import (
	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// EnvironmentVariableRepository defines the interface for environment variable data access operations
// Provides methods for CRUD operations and batch operations for environment variables
type EnvironmentVariableRepository interface {
	// Create creates a new environment variable
	Create(envVar *models.EnvironmentVariable) error

	// Update updates an existing environment variable
	Update(envVar *models.EnvironmentVariable) error

	// Delete deletes an environment variable by ID
	Delete(id uint) error

	// FindByEnvID retrieves all variables for a specific environment
	FindByEnvID(envID string) ([]models.EnvironmentVariable, error)

	// FindByKey retrieves a specific variable by environment ID and key
	FindByKey(envID, key string) (*models.EnvironmentVariable, error)

	// BatchCreate creates multiple environment variables in a single transaction
	BatchCreate(envVars []models.EnvironmentVariable) error
}

// environmentVariableRepository implements EnvironmentVariableRepository interface
type environmentVariableRepository struct {
	db *gorm.DB
}

// NewEnvironmentVariableRepository creates a new EnvironmentVariableRepository instance
func NewEnvironmentVariableRepository(db *gorm.DB) EnvironmentVariableRepository {
	return &environmentVariableRepository{db: db}
}

// Create creates a new environment variable in the database
func (r *environmentVariableRepository) Create(envVar *models.EnvironmentVariable) error {
	return r.db.Create(envVar).Error
}

// Update updates an existing environment variable in the database
func (r *environmentVariableRepository) Update(envVar *models.EnvironmentVariable) error {
	return r.db.Save(envVar).Error
}

// Delete deletes an environment variable by ID
func (r *environmentVariableRepository) Delete(id uint) error {
	return r.db.Delete(&models.EnvironmentVariable{}, id).Error
}

// FindByEnvID retrieves all variables for a specific environment
// Orders results by key for consistent ordering
func (r *environmentVariableRepository) FindByEnvID(envID string) ([]models.EnvironmentVariable, error) {
	var variables []models.EnvironmentVariable
	err := r.db.Where("env_id = ?", envID).
		Order("key ASC").
		Find(&variables).Error
	return variables, err
}

// FindByKey retrieves a specific variable by environment ID and key
// Returns nil if not found (instead of error)
func (r *environmentVariableRepository) FindByKey(envID, key string) (*models.EnvironmentVariable, error) {
	var variable models.EnvironmentVariable
	err := r.db.Where("env_id = ? AND key = ?", envID, key).
		First(&variable).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &variable, nil
}

// BatchCreate creates multiple environment variables in a single transaction
// Provides atomic batch insertion for efficiency
func (r *environmentVariableRepository) BatchCreate(envVars []models.EnvironmentVariable) error {
	if len(envVars) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&envVars).Error
	})
}

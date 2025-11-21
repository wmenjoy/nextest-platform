package repository

import (
	"test-management-service/internal/models"
	"time"

	"gorm.io/gorm"
)

// TestResultRepository 测试结果数据访问接口
type TestResultRepository interface {
	Create(result *models.TestResult) error
	FindByID(id uint) (*models.TestResult, error)
	FindByTestID(testID string, limit int) ([]models.TestResult, error)
	FindByRunID(runID string) ([]models.TestResult, error)
	DeleteOlderThan(days int) error
}

type testResultRepo struct {
	db *gorm.DB
}

func NewTestResultRepository(db *gorm.DB) TestResultRepository {
	return &testResultRepo{db: db}
}

func (r *testResultRepo) Create(result *models.TestResult) error {
	return r.db.Create(result).Error
}

func (r *testResultRepo) FindByID(id uint) (*models.TestResult, error) {
	var result models.TestResult
	err := r.db.First(&result, id).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *testResultRepo) FindByTestID(testID string, limit int) ([]models.TestResult, error) {
	var results []models.TestResult
	query := r.db.Where("test_id = ?", testID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&results).Error
	return results, err
}

func (r *testResultRepo) FindByRunID(runID string) ([]models.TestResult, error) {
	var results []models.TestResult
	err := r.db.Where("run_id = ?", runID).Find(&results).Error
	return results, err
}

func (r *testResultRepo) DeleteOlderThan(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	return r.db.Where("created_at < ?", cutoff).Delete(&models.TestResult{}).Error
}

// TestRunRepository 测试批次数据访问接口
type TestRunRepository interface {
	Create(run *models.TestRun) error
	Update(run *models.TestRun) error
	FindByID(runID string) (*models.TestRun, error)
	FindAll(limit, offset int) ([]models.TestRun, int64, error)
}

type testRunRepo struct {
	db *gorm.DB
}

func NewTestRunRepository(db *gorm.DB) TestRunRepository {
	return &testRunRepo{db: db}
}

func (r *testRunRepo) Create(run *models.TestRun) error {
	return r.db.Create(run).Error
}

func (r *testRunRepo) Update(run *models.TestRun) error {
	return r.db.Save(run).Error
}

func (r *testRunRepo) FindByID(runID string) (*models.TestRun, error) {
	var run models.TestRun
	err := r.db.Preload("Results").Where("run_id = ?", runID).First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *testRunRepo) FindAll(limit, offset int) ([]models.TestRun, int64, error) {
	var runs []models.TestRun
	var total int64

	if err := r.db.Model(&models.TestRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&runs).Error
	return runs, total, err
}

package repository

import (
	"errors"
	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// TestCaseRepository 测试案例数据访问接口
type TestCaseRepository interface {
	Create(testCase *models.TestCase) error
	Update(testCase *models.TestCase) error
	Delete(testID string) error
	FindByID(testID string) (*models.TestCase, error)
	FindByGroupID(groupID string) ([]models.TestCase, error)
	FindAll(limit, offset int) ([]models.TestCase, int64, error)
	FindByType(testType string) ([]models.TestCase, error)
	FindByTags(tags []string) ([]models.TestCase, error)
	Search(query string) ([]models.TestCase, error)
}

// testCaseRepo 实现
type testCaseRepo struct {
	db *gorm.DB
}

// NewTestCaseRepository 创建Repository实例
func NewTestCaseRepository(db *gorm.DB) TestCaseRepository {
	return &testCaseRepo{db: db}
}

func (r *testCaseRepo) Create(testCase *models.TestCase) error {
	return r.db.Create(testCase).Error
}

func (r *testCaseRepo) Update(testCase *models.TestCase) error {
	return r.db.Save(testCase).Error
}

func (r *testCaseRepo) Delete(testID string) error {
	return r.db.Where("test_id = ?", testID).Delete(&models.TestCase{}).Error
}

func (r *testCaseRepo) FindByID(testID string) (*models.TestCase, error) {
	var testCase models.TestCase
	err := r.db.Where("test_id = ?", testID).First(&testCase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &testCase, nil
}

func (r *testCaseRepo) FindByGroupID(groupID string) ([]models.TestCase, error) {
	var testCases []models.TestCase
	err := r.db.Where("group_id = ?", groupID).Find(&testCases).Error
	return testCases, err
}

func (r *testCaseRepo) FindAll(limit, offset int) ([]models.TestCase, int64, error) {
	var testCases []models.TestCase
	var total int64

	if err := r.db.Model(&models.TestCase{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Limit(limit).Offset(offset).Find(&testCases).Error
	return testCases, total, err
}

func (r *testCaseRepo) FindByType(testType string) ([]models.TestCase, error) {
	var testCases []models.TestCase
	err := r.db.Where("type = ?", testType).Find(&testCases).Error
	return testCases, err
}

func (r *testCaseRepo) FindByTags(tags []string) ([]models.TestCase, error) {
	var testCases []models.TestCase
	// 简化实现：这里需要JSON查询，SQLite支持有限
	// 生产环境建议使用PostgreSQL的JSONB查询
	err := r.db.Find(&testCases).Error
	if err != nil {
		return nil, err
	}

	// 在内存中过滤（简化版本）
	var filtered []models.TestCase
	for _, tc := range testCases {
		if containsAny(tc.Tags, tags) {
			filtered = append(filtered, tc)
		}
	}
	return filtered, nil
}

func (r *testCaseRepo) Search(query string) ([]models.TestCase, error) {
	var testCases []models.TestCase
	err := r.db.Where("name LIKE ? OR objective LIKE ?", "%"+query+"%", "%"+query+"%").
		Find(&testCases).Error
	return testCases, err
}

// 辅助函数
func containsAny(slice []interface{}, items []string) bool {
	for _, item := range items {
		for _, s := range slice {
			if str, ok := s.(string); ok && str == item {
				return true
			}
		}
	}
	return false
}

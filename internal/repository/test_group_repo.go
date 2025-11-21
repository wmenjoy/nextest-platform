package repository

import (
	"errors"
	"test-management-service/internal/models"

	"gorm.io/gorm"
)

// TestGroupRepository 测试分组数据访问接口
type TestGroupRepository interface {
	Create(group *models.TestGroup) error
	Update(group *models.TestGroup) error
	Delete(groupID string) error
	FindByID(groupID string) (*models.TestGroup, error)
	FindByParentID(parentID string) ([]models.TestGroup, error)
	FindAll() ([]models.TestGroup, error)
	GetTree() ([]models.TestGroup, error)
}

// testGroupRepo 实现
type testGroupRepo struct {
	db *gorm.DB
}

// NewTestGroupRepository 创建Repository实例
func NewTestGroupRepository(db *gorm.DB) TestGroupRepository {
	return &testGroupRepo{db: db}
}

func (r *testGroupRepo) Create(group *models.TestGroup) error {
	return r.db.Create(group).Error
}

func (r *testGroupRepo) Update(group *models.TestGroup) error {
	return r.db.Save(group).Error
}

func (r *testGroupRepo) Delete(groupID string) error {
	return r.db.Where("group_id = ?", groupID).Delete(&models.TestGroup{}).Error
}

func (r *testGroupRepo) FindByID(groupID string) (*models.TestGroup, error) {
	var group models.TestGroup
	err := r.db.Where("group_id = ?", groupID).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

func (r *testGroupRepo) FindByParentID(parentID string) ([]models.TestGroup, error) {
	var groups []models.TestGroup
	err := r.db.Where("parent_id = ?", parentID).Find(&groups).Error
	return groups, err
}

func (r *testGroupRepo) FindAll() ([]models.TestGroup, error) {
	var groups []models.TestGroup
	err := r.db.Find(&groups).Error
	return groups, err
}

func (r *testGroupRepo) GetTree() ([]models.TestGroup, error) {
	var groups []models.TestGroup
	// 获取所有分组
	if err := r.db.Find(&groups).Error; err != nil {
		return nil, err
	}

	// 构建 map 用于快速查找
	groupMap := make(map[string]*models.TestGroup)
	childMap := make(map[string][]string) // parentID -> []childID

	for i := range groups {
		groupMap[groups[i].GroupID] = &groups[i]
		if groups[i].ParentID != "" && groups[i].ParentID != "root" {
			childMap[groups[i].ParentID] = append(childMap[groups[i].ParentID], groups[i].GroupID)
		}
	}

	// 递归构建节点及其子节点
	var buildNode func(groupID string) models.TestGroup
	buildNode = func(groupID string) models.TestGroup {
		group := *groupMap[groupID]
		group.Children = []models.TestGroup{}

		if childIDs, ok := childMap[groupID]; ok {
			for _, childID := range childIDs {
				group.Children = append(group.Children, buildNode(childID))
			}
		}

		return group
	}

	// 构建根节点
	var roots []models.TestGroup
	for i := range groups {
		if groups[i].ParentID == "" || groups[i].ParentID == "root" {
			roots = append(roots, buildNode(groups[i].GroupID))
		}
	}

	return roots, nil
}

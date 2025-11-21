# 环境管理功能实现计划

**版本**: 1.0
**创建日期**: 2025-11-21
**预计工期**: 8-12 天（1.5-2 周）
**实施模式**: 分阶段并行实施 + Subagent 协作

---

## 📋 目录

1. [总体设计](#总体设计)
2. [Phase 1: 数据模型和迁移](#phase-1-数据模型和迁移)
3. [Phase 2: Repository 层](#phase-2-repository-层)
4. [Phase 3: 环境管理服务](#phase-3-环境管理服务)
5. [Phase 4: 变量注入引擎](#phase-4-变量注入引擎)
6. [Phase 5: API 层](#phase-5-api-层)
7. [Phase 6: 执行器集成](#phase-6-执行器集成)
8. [Phase 7: 测试和文档](#phase-7-测试和文档)
9. [任务依赖关系图](#任务依赖关系图)
10. [Subagent 执行策略](#subagent-执行策略)

---

## 总体设计

### 核心概念

**Environment（环境）**:
- 代表一个完整的测试环境（Dev, Staging, Prod）
- 包含该环境的所有配置变量
- 同一时间只能有一个环境处于激活状态

**Variable Injection（变量注入）**:
- 在测试执行前自动替换配置中的占位符
- 支持 `{{VARIABLE_NAME}}` 语法
- 变量优先级: 环境变量 < 工作流变量 < 测试案例变量

### 架构设计

```
┌─────────────────────────────────────────────────────┐
│                    API 层                            │
│  - EnvironmentHandler (6 个端点)                    │
│  - GET/POST/PUT/DELETE /environments                │
│  - POST /environments/:id/activate                   │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                   Service 层                         │
│  - EnvironmentService (环境管理)                    │
│  - VariableInjector (变量注入)                      │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                  Repository 层                       │
│  - EnvironmentRepository                            │
│  - EnvironmentVariableRepository                    │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                   数据层                             │
│  - environments 表                                   │
│  - environment_variables 表                         │
└─────────────────────────────────────────────────────┘
```

### 变量优先级规则

```
最低优先级: Environment Variables (全局环境变量)
    ↓
中等优先级: Workflow Variables (工作流变量)
    ↓
最高优先级: Test Case Inline Variables (测试案例内联变量)
```

---

## Phase 1: 数据模型和迁移

**工期**: 1 天
**依赖**: 无
**并行度**: 可单独执行

### Task 1.1: 创建 Environment 模型

**文件**: `internal/models/environment.go`

**模型定义**:
```go
package models

import (
    "time"
    "gorm.io/gorm"
)

// Environment 环境模型
type Environment struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    EnvID       string         `gorm:"uniqueIndex;size:50;not null" json:"envId"`
    Name        string         `gorm:"size:255;not null" json:"name"`
    Description string         `gorm:"type:text" json:"description,omitempty"`
    IsActive    bool           `gorm:"default:false;index" json:"isActive"`
    Variables   JSONB          `gorm:"type:text" json:"variables,omitempty"`
    CreatedAt   time.Time      `json:"createdAt"`
    UpdatedAt   time.Time      `json:"updatedAt"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

    // 关联
    EnvironmentVariables []EnvironmentVariable `gorm:"foreignKey:EnvID;references:EnvID" json:"environmentVariables,omitempty"`
}

// TableName 指定表名
func (Environment) TableName() string {
    return "environments"
}

// EnvironmentVariable 环境变量模型（可选，用于结构化存储）
type EnvironmentVariable struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    EnvID     string    `gorm:"size:50;not null;index" json:"envId"`
    Key       string    `gorm:"size:255;not null" json:"key"`
    Value     string    `gorm:"type:text" json:"value"`
    ValueType string    `gorm:"size:20;default:'string'" json:"valueType"` // string, number, boolean, json
    IsSecret  bool      `gorm:"default:false" json:"isSecret"` // 标记敏感信息
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`

    // 关联
    Environment *Environment `gorm:"foreignKey:EnvID;references:EnvID" json:"-"`
}

// TableName 指定表名
func (EnvironmentVariable) TableName() string {
    return "environment_variables"
}
```

**验证点**:
- ✅ 模型定义完整
- ✅ GORM 标签正确
- ✅ 关联关系定义
- ✅ 唯一索引 (envId)
- ✅ 软删除支持

---

### Task 1.2: 创建数据库迁移脚本

**文件**: `migrations/004_add_environment_management.sql`

**迁移内容**:
```sql
-- ============================================
-- Migration 004: Add Environment Management
-- Date: 2025-11-21
-- Description: 添加环境管理功能
-- ============================================

-- 1. 创建 environments 表
CREATE TABLE IF NOT EXISTS environments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    env_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT 0,
    variables TEXT,  -- JSONB format
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- 索引
CREATE UNIQUE INDEX idx_environments_env_id ON environments(env_id);
CREATE INDEX idx_environments_is_active ON environments(is_active);
CREATE INDEX idx_environments_deleted_at ON environments(deleted_at);

-- 2. 创建 environment_variables 表（可选，用于结构化存储）
CREATE TABLE IF NOT EXISTS environment_variables (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    env_id VARCHAR(50) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT,
    value_type VARCHAR(20) DEFAULT 'string',
    is_secret BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (env_id) REFERENCES environments(env_id) ON DELETE CASCADE,
    UNIQUE(env_id, key)
);

-- 索引
CREATE INDEX idx_environment_variables_env_id ON environment_variables(env_id);
CREATE INDEX idx_environment_variables_key ON environment_variables(key);

-- 3. 插入默认环境
INSERT INTO environments (env_id, name, description, is_active, variables) VALUES
('dev', 'Development', '开发环境', 1, '{"BASE_URL": "http://localhost:3000", "TIMEOUT": 30}'),
('staging', 'Staging', '预发布环境', 0, '{"BASE_URL": "https://staging.example.com", "TIMEOUT": 60}'),
('prod', 'Production', '生产环境', 0, '{"BASE_URL": "https://api.example.com", "TIMEOUT": 120}');

-- 4. 插入默认环境变量（可选）
INSERT INTO environment_variables (env_id, key, value, value_type) VALUES
('dev', 'BASE_URL', 'http://localhost:3000', 'string'),
('dev', 'API_KEY', 'dev-key-12345', 'string'),
('dev', 'TIMEOUT', '30', 'number'),
('staging', 'BASE_URL', 'https://staging.example.com', 'string'),
('staging', 'API_KEY', 'staging-key-67890', 'string'),
('staging', 'TIMEOUT', '60', 'number'),
('prod', 'BASE_URL', 'https://api.example.com', 'string'),
('prod', 'API_KEY', 'prod-key-secret', 'string'),
('prod', 'TIMEOUT', '120', 'number');

-- ============================================
-- Rollback Instructions
-- ============================================
-- DROP TABLE IF EXISTS environment_variables;
-- DROP TABLE IF EXISTS environments;
```

**PostgreSQL/MySQL 兼容性**:
- 修改 `AUTOINCREMENT` → `SERIAL` (PostgreSQL) 或 `AUTO_INCREMENT` (MySQL)
- 修改 `DATETIME` → `TIMESTAMP` (PostgreSQL)
- 修改 `BOOLEAN` → `TINYINT(1)` (MySQL)

**验证点**:
- ✅ SQL 语法正确
- ✅ 外键约束定义
- ✅ 索引创建
- ✅ 默认数据插入
- ✅ 回滚脚本提供

---

## Phase 2: Repository 层

**工期**: 1 天
**依赖**: Phase 1 完成
**并行度**: 可与 Phase 3 部分并行

### Task 2.1: 创建 EnvironmentRepository

**文件**: `internal/repository/environment_repository.go`

**接口定义**:
```go
package repository

import (
    "test-management-service/internal/models"
    "gorm.io/gorm"
)

// EnvironmentRepository 环境仓库接口
type EnvironmentRepository interface {
    Create(env *models.Environment) error
    Update(env *models.Environment) error
    Delete(envID string) error
    FindByID(envID string) (*models.Environment, error)
    FindAll(limit, offset int) ([]models.Environment, int64, error)
    FindActive() (*models.Environment, error)
    SetActive(envID string) error
}

type environmentRepository struct {
    db *gorm.DB
}

// NewEnvironmentRepository 创建环境仓库
func NewEnvironmentRepository(db *gorm.DB) EnvironmentRepository {
    return &environmentRepository{db: db}
}

func (r *environmentRepository) Create(env *models.Environment) error {
    return r.db.Create(env).Error
}

func (r *environmentRepository) Update(env *models.Environment) error {
    return r.db.Save(env).Error
}

func (r *environmentRepository) Delete(envID string) error {
    return r.db.Where("env_id = ?", envID).Delete(&models.Environment{}).Error
}

func (r *environmentRepository) FindByID(envID string) (*models.Environment, error) {
    var env models.Environment
    err := r.db.Where("env_id = ? AND deleted_at IS NULL", envID).
        Preload("EnvironmentVariables").
        First(&env).Error
    if err != nil {
        return nil, err
    }
    return &env, nil
}

func (r *environmentRepository) FindAll(limit, offset int) ([]models.Environment, int64, error) {
    var envs []models.Environment
    var total int64

    query := r.db.Model(&models.Environment{}).Where("deleted_at IS NULL")
    query.Count(&total)

    err := query.Limit(limit).Offset(offset).Find(&envs).Error
    return envs, total, err
}

func (r *environmentRepository) FindActive() (*models.Environment, error) {
    var env models.Environment
    err := r.db.Where("is_active = ? AND deleted_at IS NULL", true).
        Preload("EnvironmentVariables").
        First(&env).Error
    if err != nil {
        return nil, err
    }
    return &env, nil
}

func (r *environmentRepository) SetActive(envID string) error {
    // 使用事务确保原子性
    return r.db.Transaction(func(tx *gorm.DB) error {
        // 1. 停用所有环境
        if err := tx.Model(&models.Environment{}).
            Where("is_active = ?", true).
            Update("is_active", false).Error; err != nil {
            return err
        }

        // 2. 激活指定环境
        if err := tx.Model(&models.Environment{}).
            Where("env_id = ?", envID).
            Update("is_active", true).Error; err != nil {
            return err
        }

        return nil
    })
}
```

**验证点**:
- ✅ CRUD 操作完整
- ✅ 软删除过滤
- ✅ 事务支持 (SetActive)
- ✅ 关联加载 (Preload)
- ✅ 错误处理

---

### Task 2.2: 创建 EnvironmentVariableRepository（可选）

**文件**: `internal/repository/environment_variable_repository.go`

**接口定义**:
```go
package repository

import (
    "test-management-service/internal/models"
    "gorm.io/gorm"
)

// EnvironmentVariableRepository 环境变量仓库接口
type EnvironmentVariableRepository interface {
    Create(envVar *models.EnvironmentVariable) error
    Update(envVar *models.EnvironmentVariable) error
    Delete(id uint) error
    FindByEnvID(envID string) ([]models.EnvironmentVariable, error)
    FindByKey(envID, key string) (*models.EnvironmentVariable, error)
    BatchCreate(envVars []models.EnvironmentVariable) error
}

type environmentVariableRepository struct {
    db *gorm.DB
}

func NewEnvironmentVariableRepository(db *gorm.DB) EnvironmentVariableRepository {
    return &environmentVariableRepository{db: db}
}

func (r *environmentVariableRepository) Create(envVar *models.EnvironmentVariable) error {
    return r.db.Create(envVar).Error
}

func (r *environmentVariableRepository) Update(envVar *models.EnvironmentVariable) error {
    return r.db.Save(envVar).Error
}

func (r *environmentVariableRepository) Delete(id uint) error {
    return r.db.Delete(&models.EnvironmentVariable{}, id).Error
}

func (r *environmentVariableRepository) FindByEnvID(envID string) ([]models.EnvironmentVariable, error) {
    var vars []models.EnvironmentVariable
    err := r.db.Where("env_id = ?", envID).Find(&vars).Error
    return vars, err
}

func (r *environmentVariableRepository) FindByKey(envID, key string) (*models.EnvironmentVariable, error) {
    var envVar models.EnvironmentVariable
    err := r.db.Where("env_id = ? AND key = ?", envID, key).First(&envVar).Error
    if err != nil {
        return nil, err
    }
    return &envVar, nil
}

func (r *environmentVariableRepository) BatchCreate(envVars []models.EnvironmentVariable) error {
    return r.db.Create(&envVars).Error
}
```

**验证点**:
- ✅ CRUD 操作
- ✅ 批量创建支持
- ✅ 按环境和键查询

---

## Phase 3: 环境管理服务

**工期**: 2 天
**依赖**: Phase 1, Phase 2 完成
**并行度**: 可与 Phase 4 部分并行

### Task 3.1: 创建 EnvironmentService

**文件**: `internal/service/environment_service.go`

**服务接口**:
```go
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
```

**验证点**:
- ✅ CRUD 完整实现
- ✅ 激活环境验证
- ✅ 变量管理功能
- ✅ 错误处理
- ✅ 业务逻辑验证

---

## Phase 4: 变量注入引擎

**工期**: 2-3 天
**依赖**: Phase 2, Phase 3 完成
**并行度**: 核心模块，建议独立完成

### Task 4.1: 创建变量注入器

**文件**: `internal/service/variable_injector.go`

**核心实现**:
```go
package service

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strconv"
    "strings"
)

// VariableInjector 变量注入器
type VariableInjector struct {
    envService EnvironmentService
}

// NewVariableInjector 创建变量注入器
func NewVariableInjector(envService EnvironmentService) *VariableInjector {
    return &VariableInjector{
        envService: envService,
    }
}

// InjectVariables 注入环境变量到配置中
// 支持三层变量优先级: envVars < workflowVars < inlineVars
func (vi *VariableInjector) InjectVariables(
    config interface{},
    workflowVars map[string]interface{},
) (interface{}, error) {
    // 1. 获取当前激活的环境变量
    activeEnv, err := vi.envService.GetActiveEnvironment()
    if err != nil {
        // 如果没有激活环境，使用空变量集
        return vi.injectWithVars(config, make(map[string]interface{}), workflowVars), nil
    }

    envVars := activeEnv.Variables
    if envVars == nil {
        envVars = make(map[string]interface{})
    }

    // 2. 执行变量注入
    return vi.injectWithVars(config, envVars, workflowVars), nil
}

// injectWithVars 使用指定的变量集进行注入
func (vi *VariableInjector) injectWithVars(
    config interface{},
    envVars map[string]interface{},
    workflowVars map[string]interface{},
) interface{} {
    // 合并变量（优先级: envVars < workflowVars）
    mergedVars := vi.mergeVariables(envVars, workflowVars)

    // 递归替换配置中的占位符
    return vi.replaceVariables(config, mergedVars)
}

// mergeVariables 合并变量，后者优先级更高
func (vi *VariableInjector) mergeVariables(
    base map[string]interface{},
    override map[string]interface{},
) map[string]interface{} {
    result := make(map[string]interface{})

    // 复制基础变量
    for k, v := range base {
        result[k] = v
    }

    // 覆盖变量
    for k, v := range override {
        result[k] = v
    }

    return result
}

// replaceVariables 递归替换所有变量占位符
func (vi *VariableInjector) replaceVariables(
    value interface{},
    vars map[string]interface{},
) interface{} {
    switch v := value.(type) {
    case string:
        return vi.replaceStringVariables(v, vars)

    case map[string]interface{}:
        result := make(map[string]interface{})
        for key, val := range v {
            result[key] = vi.replaceVariables(val, vars)
        }
        return result

    case []interface{}:
        result := make([]interface{}, len(v))
        for i, val := range v {
            result[i] = vi.replaceVariables(val, vars)
        }
        return result

    default:
        return value
    }
}

// replaceStringVariables 替换字符串中的变量占位符
// 支持格式: {{VAR_NAME}}
func (vi *VariableInjector) replaceStringVariables(
    str string,
    vars map[string]interface{},
) interface{} {
    // 正则匹配 {{VAR_NAME}}
    re := regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

    matches := re.FindAllStringSubmatch(str, -1)
    if len(matches) == 0 {
        return str
    }

    // 如果整个字符串就是一个变量引用，直接返回变量值（保持类型）
    if len(matches) == 1 && matches[0][0] == str {
        varName := matches[0][1]
        if val, exists := vars[varName]; exists {
            return val
        }
        return str // 变量不存在，返回原字符串
    }

    // 字符串包含多个变量或混合内容，替换为字符串
    result := str
    for _, match := range matches {
        placeholder := match[0] // {{VAR_NAME}}
        varName := match[1]     // VAR_NAME

        if val, exists := vars[varName]; exists {
            // 将变量值转换为字符串
            replacement := vi.valueToString(val)
            result = strings.ReplaceAll(result, placeholder, replacement)
        }
    }

    return result
}

// valueToString 将任意值转换为字符串
func (vi *VariableInjector) valueToString(value interface{}) string {
    switch v := value.(type) {
    case string:
        return v
    case int, int32, int64:
        return fmt.Sprintf("%d", v)
    case float32, float64:
        return strconv.FormatFloat(v.(float64), 'f', -1, 64)
    case bool:
        return strconv.FormatBool(v)
    default:
        // 复杂类型转为 JSON 字符串
        bytes, _ := json.Marshal(v)
        return string(bytes)
    }
}

// InjectIntoHTTPConfig 注入变量到 HTTP 配置
func (vi *VariableInjector) InjectIntoHTTPConfig(
    httpConfig map[string]interface{},
    workflowVars map[string]interface{},
) (map[string]interface{}, error) {
    injected, err := vi.InjectVariables(httpConfig, workflowVars)
    if err != nil {
        return nil, err
    }

    return injected.(map[string]interface{}), nil
}

// InjectIntoCommandConfig 注入变量到命令配置
func (vi *VariableInjector) InjectIntoCommandConfig(
    commandConfig map[string]interface{},
    workflowVars map[string]interface{},
) (map[string]interface{}, error) {
    injected, err := vi.InjectVariables(commandConfig, workflowVars)
    if err != nil {
        return nil, err
    }

    return injected.(map[string]interface{}), nil
}
```

**验证点**:
- ✅ 支持 `{{VAR}}` 语法
- ✅ 递归替换（嵌套对象、数组）
- ✅ 类型保持（整个字符串是变量时）
- ✅ 变量优先级处理
- ✅ 错误处理

---

### Task 4.2: 单元测试

**文件**: `internal/service/variable_injector_test.go`

**测试用例**:
```go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestVariableInjector_ReplaceStringVariables(t *testing.T) {
    injector := &VariableInjector{}

    vars := map[string]interface{}{
        "BASE_URL": "http://localhost:3000",
        "API_KEY":  "test-key-123",
        "TIMEOUT":  30,
        "DEBUG":    true,
    }

    tests := []struct {
        name     string
        input    interface{}
        expected interface{}
    }{
        {
            name:     "Simple string variable",
            input:    "{{BASE_URL}}",
            expected: "http://localhost:3000",
        },
        {
            name:     "Number variable (keeps type)",
            input:    "{{TIMEOUT}}",
            expected: 30,
        },
        {
            name:     "Boolean variable (keeps type)",
            input:    "{{DEBUG}}",
            expected: true,
        },
        {
            name:     "Mixed string",
            input:    "{{BASE_URL}}/api/users",
            expected: "http://localhost:3000/api/users",
        },
        {
            name:     "Multiple variables",
            input:    "URL: {{BASE_URL}}, Key: {{API_KEY}}",
            expected: "URL: http://localhost:3000, Key: test-key-123",
        },
        {
            name: "Nested object",
            input: map[string]interface{}{
                "url":    "{{BASE_URL}}/api",
                "apiKey": "{{API_KEY}}",
                "config": map[string]interface{}{
                    "timeout": "{{TIMEOUT}}",
                },
            },
            expected: map[string]interface{}{
                "url":    "http://localhost:3000/api",
                "apiKey": "test-key-123",
                "config": map[string]interface{}{
                    "timeout": 30,
                },
            },
        },
        {
            name:     "Array",
            input:    []interface{}{"{{BASE_URL}}", "{{API_KEY}}", "static"},
            expected: []interface{}{"http://localhost:3000", "test-key-123", "static"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := injector.replaceVariables(tt.input, vars)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestVariableInjector_MergeVariables(t *testing.T) {
    injector := &VariableInjector{}

    base := map[string]interface{}{
        "A": 1,
        "B": 2,
        "C": 3,
    }

    override := map[string]interface{}{
        "B": 20,
        "D": 4,
    }

    result := injector.mergeVariables(base, override)

    expected := map[string]interface{}{
        "A": 1,
        "B": 20, // Overridden
        "C": 3,
        "D": 4,
    }

    assert.Equal(t, expected, result)
}
```

**验证点**:
- ✅ 基础变量替换
- ✅ 类型保持
- ✅ 嵌套对象处理
- ✅ 数组处理
- ✅ 变量合并

---

## Phase 5: API 层

**工期**: 1-2 天
**依赖**: Phase 3 完成
**并行度**: 可与 Phase 6 并行

### Task 5.1: 创建 EnvironmentHandler

**文件**: `internal/handler/environment_handler.go`

**Handler 实现**:
```go
package handler

import (
    "net/http"
    "strconv"
    "test-management-service/internal/service"

    "github.com/gin-gonic/gin"
)

type EnvironmentHandler struct {
    envService service.EnvironmentService
}

func NewEnvironmentHandler(envService service.EnvironmentService) *EnvironmentHandler {
    return &EnvironmentHandler{
        envService: envService,
    }
}

// RegisterRoutes 注册路由
func (h *EnvironmentHandler) RegisterRoutes(r *gin.Engine) {
    envGroup := r.Group("/api/v2/environments")
    {
        envGroup.POST("", h.CreateEnvironment)           // 创建环境
        envGroup.GET("", h.ListEnvironments)             // 列出环境
        envGroup.GET("/active", h.GetActiveEnvironment)  // 获取激活环境
        envGroup.GET("/:id", h.GetEnvironment)           // 获取环境详情
        envGroup.PUT("/:id", h.UpdateEnvironment)        // 更新环境
        envGroup.DELETE("/:id", h.DeleteEnvironment)     // 删除环境
        envGroup.POST("/:id/activate", h.ActivateEnvironment) // 激活环境

        // 变量管理
        envGroup.GET("/:id/variables", h.GetVariables)          // 获取所有变量
        envGroup.GET("/:id/variables/:key", h.GetVariable)      // 获取单个变量
        envGroup.PUT("/:id/variables/:key", h.SetVariable)      // 设置变量
        envGroup.DELETE("/:id/variables/:key", h.DeleteVariable) // 删除变量
    }
}

// CreateEnvironment 创建环境
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
    var req service.CreateEnvironmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    env, err := h.envService.CreateEnvironment(&req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, env)
}

// ListEnvironments 列出环境
func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

    envs, total, err := h.envService.ListEnvironments(limit, offset)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":   envs,
        "total":  total,
        "limit":  limit,
        "offset": offset,
    })
}

// GetEnvironment 获取环境详情
func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
    envID := c.Param("id")

    env, err := h.envService.GetEnvironment(envID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, env)
}

// UpdateEnvironment 更新环境
func (h *EnvironmentHandler) UpdateEnvironment(c *gin.Context) {
    envID := c.Param("id")

    var req service.UpdateEnvironmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    env, err := h.envService.UpdateEnvironment(envID, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, env)
}

// DeleteEnvironment 删除环境
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
    envID := c.Param("id")

    if err := h.envService.DeleteEnvironment(envID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "environment deleted"})
}

// GetActiveEnvironment 获取当前激活的环境
func (h *EnvironmentHandler) GetActiveEnvironment(c *gin.Context) {
    env, err := h.envService.GetActiveEnvironment()
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "no active environment"})
        return
    }

    c.JSON(http.StatusOK, env)
}

// ActivateEnvironment 激活环境
func (h *EnvironmentHandler) ActivateEnvironment(c *gin.Context) {
    envID := c.Param("id")

    if err := h.envService.ActivateEnvironment(envID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "environment activated",
        "envId":   envID,
    })
}

// GetVariables 获取环境所有变量
func (h *EnvironmentHandler) GetVariables(c *gin.Context) {
    envID := c.Param("id")

    vars, err := h.envService.GetVariables(envID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, vars)
}

// GetVariable 获取单个变量
func (h *EnvironmentHandler) GetVariable(c *gin.Context) {
    envID := c.Param("id")
    key := c.Param("key")

    value, err := h.envService.GetVariable(envID, key)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "key":   key,
        "value": value,
    })
}

// SetVariable 设置变量
func (h *EnvironmentHandler) SetVariable(c *gin.Context) {
    envID := c.Param("id")
    key := c.Param("key")

    var req struct {
        Value interface{} `json:"value" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.envService.SetVariable(envID, key, req.Value); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "variable set",
        "key":     key,
        "value":   req.Value,
    })
}

// DeleteVariable 删除变量
func (h *EnvironmentHandler) DeleteVariable(c *gin.Context) {
    envID := c.Param("id")
    key := c.Param("key")

    if err := h.envService.DeleteVariable(envID, key); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "variable deleted",
        "key":     key,
    })
}
```

**API 端点总结**:
```
POST   /api/v2/environments                      - 创建环境
GET    /api/v2/environments                      - 列出环境
GET    /api/v2/environments/active               - 获取激活环境
GET    /api/v2/environments/:id                  - 获取环境详情
PUT    /api/v2/environments/:id                  - 更新环境
DELETE /api/v2/environments/:id                  - 删除环境
POST   /api/v2/environments/:id/activate         - 激活环境
GET    /api/v2/environments/:id/variables        - 获取所有变量
GET    /api/v2/environments/:id/variables/:key   - 获取单个变量
PUT    /api/v2/environments/:id/variables/:key   - 设置变量
DELETE /api/v2/environments/:id/variables/:key   - 删除变量
```

**验证点**:
- ✅ 11 个端点完整实现
- ✅ 请求验证
- ✅ 错误处理
- ✅ RESTful 规范
- ✅ HTTP 状态码正确

---

## Phase 6: 执行器集成

**工期**: 2 天
**依赖**: Phase 4 完成
**并行度**: 关键路径

### Task 6.1: 修改 UnifiedTestExecutor

**文件**: `internal/testcase/executor.go`

**修改点**:
```go
// 在 UnifiedTestExecutor 结构中添加 VariableInjector
type UnifiedTestExecutor struct {
    baseURL          string
    client           *http.Client
    workflowExecutor WorkflowExecutor
    testCaseRepo     TestCaseRepository
    workflowRepo     WorkflowRepository
    variableInjector *service.VariableInjector  // 新增
}

// 修改构造函数
func NewExecutorWithInjector(
    baseURL string,
    workflowExecutor WorkflowExecutor,
    testCaseRepo TestCaseRepository,
    workflowRepo WorkflowRepository,
    variableInjector *service.VariableInjector,
) *UnifiedTestExecutor {
    return &UnifiedTestExecutor{
        baseURL:          baseURL,
        client:           &http.Client{Timeout: 30 * time.Second},
        workflowExecutor: workflowExecutor,
        testCaseRepo:     testCaseRepo,
        workflowRepo:     workflowRepo,
        variableInjector: variableInjector,
    }
}

// 在 executeHTTP 方法开始时注入变量
func (e *UnifiedTestExecutor) executeHTTP(tc *TestCase, result *TestResult) {
    // 注入环境变量
    if e.variableInjector != nil {
        injectedConfig, err := e.variableInjector.InjectIntoHTTPConfig(
            tc.HTTP.toMap(),
            nil, // 没有工作流变量
        )
        if err != nil {
            result.Status = "error"
            result.Error = fmt.Sprintf("variable injection failed: %v", err)
            return
        }

        // 更新 HTTP 配置
        tc.HTTP = parseHTTPConfig(injectedConfig)
    }

    // 继续原有逻辑...
}

// 在 executeCommand 方法开始时注入变量
func (e *UnifiedTestExecutor) executeCommand(tc *TestCase, result *TestResult) {
    // 注入环境变量
    if e.variableInjector != nil {
        injectedConfig, err := e.variableInjector.InjectIntoCommandConfig(
            tc.Command.toMap(),
            nil,
        )
        if err != nil {
            result.Status = "error"
            result.Error = fmt.Sprintf("variable injection failed: %v", err)
            return
        }

        // 更新命令配置
        tc.Command = parseCommandConfig(injectedConfig)
    }

    // 继续原有逻辑...
}
```

**验证点**:
- ✅ 添加 VariableInjector 依赖
- ✅ HTTP 测试注入变量
- ✅ 命令测试注入变量
- ✅ 向下兼容（注入器可选）

---

### Task 6.2: 修改 WorkflowExecutor

**文件**: `internal/workflow/executor.go`

**修改点**:
```go
// 在 WorkflowExecutorImpl 结构中添加 VariableInjector
type WorkflowExecutorImpl struct {
    db               *gorm.DB
    actionRegistry   *ActionRegistry
    testCaseRepo     TestCaseRepository
    workflowRepo     WorkflowRepository
    unifiedExecutor  *testcase.UnifiedTestExecutor
    hub              *websocket.Hub
    variableInjector *service.VariableInjector  // 新增
}

// 修改 Execute 方法，在初始化上下文时合并环境变量
func (e *WorkflowExecutorImpl) Execute(workflowID string, workflowDef interface{}) (*WorkflowResult, error) {
    // ... 前面的代码保持不变 ...

    // Step 4: 初始化执行上下文（合并环境变量）
    mergedVariables := workflow.Variables
    if mergedVariables == nil {
        mergedVariables = make(map[string]interface{})
    }

    // 注入环境变量（环境变量优先级最低）
    if e.variableInjector != nil {
        activeEnv, err := e.variableInjector.envService.GetActiveEnvironment()
        if err == nil && activeEnv != nil {
            // 环境变量作为基础
            for k, v := range activeEnv.Variables {
                if _, exists := mergedVariables[k]; !exists {
                    mergedVariables[k] = v
                }
            }
        }
    }

    ctx := &ExecutionContext{
        RunID:       runID,
        Variables:   mergedVariables,  // 使用合并后的变量
        StepOutputs: make(map[string]interface{}),
        StepResults: make(map[string]*StepExecutionResult),
        Logger:      NewBroadcastStepLogger(e.db, runID, e.hub),
        VarTracker:  NewDatabaseVariableChangeTracker(e.db, runID),
    }

    // ... 后续代码保持不变 ...
}
```

**验证点**:
- ✅ 添加 VariableInjector 依赖
- ✅ 环境变量与工作流变量合并
- ✅ 优先级正确（环境 < 工作流）
- ✅ 向下兼容

---

### Task 6.3: 修改 main.go 集成

**文件**: `cmd/server/main.go`

**集成代码**:
```go
// 创建 repositories
envRepo := repository.NewEnvironmentRepository(db)
envVarRepo := repository.NewEnvironmentVariableRepository(db)

// 创建 services
envService := service.NewEnvironmentService(envRepo, envVarRepo)
variableInjector := service.NewVariableInjector(envService)

// 创建 UnifiedTestExecutor（带变量注入）
unifiedExecutor := testcase.NewExecutorWithInjector(
    cfg.Test.TargetHost,
    nil, // workflowExecutor 稍后设置
    testCaseRepo,
    workflowRepo,
    variableInjector,
)

// 创建 WorkflowExecutor（带变量注入）
workflowExecutor := workflow.NewWorkflowExecutorWithInjector(
    db,
    testCaseRepo,
    workflowRepo,
    unifiedExecutor,
    hub,
    variableInjector,
)

// 设置循环依赖
unifiedExecutor.SetWorkflowExecutor(workflowExecutor)

// 创建 handlers
envHandler := handler.NewEnvironmentHandler(envService)
envHandler.RegisterRoutes(router)
```

**验证点**:
- ✅ 依赖注入正确
- ✅ 循环依赖处理
- ✅ 路由注册

---

## Phase 7: 测试和文档

**工期**: 2 天
**依赖**: Phase 1-6 全部完成
**并行度**: 最后阶段

### Task 7.1: 集成测试

**文件**: `test/integration/environment_integration_test.go`

**测试用例**:
```go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestEnvironmentManagement_FullWorkflow(t *testing.T) {
    // 1. 创建环境
    // 2. 设置变量
    // 3. 激活环境
    // 4. 执行测试（验证变量注入）
    // 5. 切换环境
    // 6. 再次执行测试（验证不同变量）
}

func TestVariableInjection_HTTP(t *testing.T) {
    // 测试 HTTP 配置变量注入
}

func TestVariableInjection_Command(t *testing.T) {
    // 测试命令配置变量注入
}

func TestVariableInjection_Workflow(t *testing.T) {
    // 测试工作流变量注入和优先级
}

func TestEnvironmentActivation_Concurrency(t *testing.T) {
    // 测试并发激活环境
}
```

**验证点**:
- ✅ 端到端测试
- ✅ 变量注入验证
- ✅ 优先级验证
- ✅ 并发安全

---

### Task 7.2: API 文档更新

**文件**: `docs/API_DOCUMENTATION.md`

**新增章节**:
```markdown
## 环境管理 API (新增)

### 1. 创建环境
**端点**: `POST /api/v2/environments`

**请求体**:
\`\`\`json
{
  "envId": "dev",
  "name": "Development",
  "description": "开发环境",
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-123",
    "TIMEOUT": 30
  }
}
\`\`\`

**响应**: `201 Created`

### 2. 激活环境
**端点**: `POST /api/v2/environments/:id/activate`

**响应**: `200 OK`

### 3. 变量使用示例
\`\`\`json
{
  "testId": "test-001",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "{{BASE_URL}}/api/login",
    "headers": {
      "Authorization": "Bearer {{API_KEY}}"
    },
    "body": {
      "timeout": "{{TIMEOUT}}"
    }
  }
}
\`\`\`
```

**验证点**:
- ✅ 完整 API 文档
- ✅ 请求/响应示例
- ✅ 使用场景说明

---

### Task 7.3: 数据库文档更新

**文件**: `docs/DATABASE_DESIGN.md`

**新增表结构**:
```markdown
### environments 表

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| env_id | VARCHAR(50) | UNIQUE, NOT NULL | 环境 ID |
| name | VARCHAR(255) | NOT NULL | 环境名称 |
| is_active | BOOLEAN | DEFAULT 0 | 是否激活 |
| variables | TEXT | | 变量（JSONB）|

### environment_variables 表

...
```

**验证点**:
- ✅ 表结构完整
- ✅ ER 图更新
- ✅ 索引说明

---

### Task 7.4: 用户指南

**文件**: `docs/ENVIRONMENT_MANAGEMENT_GUIDE.md`

**内容**:
```markdown
# 环境管理使用指南

## 快速开始

### 1. 创建环境
### 2. 配置变量
### 3. 激活环境
### 4. 在测试中使用变量
### 5. 切换环境

## 最佳实践

### 变量命名规范
### 敏感信息处理
### 环境切换时机

## 常见问题

Q: 如何查看当前激活的环境？
A: GET /api/v2/environments/active

Q: 变量优先级是什么？
A: 环境变量 < 工作流变量 < 测试案例内联变量
```

**验证点**:
- ✅ 快速开始指南
- ✅ 最佳实践
- ✅ FAQ

---

## 任务依赖关系图

```
Phase 1: 数据模型
├─ Task 1.1: Environment Model ✅
└─ Task 1.2: Migration Script ✅
        ↓
Phase 2: Repository 层
├─ Task 2.1: EnvironmentRepository ✅
└─ Task 2.2: EnvironmentVariableRepository ✅
        ↓
        ├─────────────────────┐
        ↓                     ↓
Phase 3: Service 层    Phase 5: API 层
└─ Task 3.1: ✅        ├─ Task 5.1: ✅
        ↓               └─ (可并行)
Phase 4: 变量注入
├─ Task 4.1: VariableInjector ✅
└─ Task 4.2: Unit Tests ✅
        ↓
Phase 6: 执行器集成
├─ Task 6.1: UnifiedTestExecutor ✅
├─ Task 6.2: WorkflowExecutor ✅
└─ Task 6.3: main.go Integration ✅
        ↓
Phase 7: 测试和文档
├─ Task 7.1: Integration Tests ✅
├─ Task 7.2: API Docs ✅
├─ Task 7.3: Database Docs ✅
└─ Task 7.4: User Guide ✅
```

---

## Subagent 执行策略

### Subagent 1: 数据层专家
**负责**: Phase 1, Phase 2
**任务**:
- 创建数据模型
- 编写迁移脚本
- 实现 Repository 层
- 编写单元测试

**预计时间**: 1-2 天

---

### Subagent 2: 业务逻辑专家
**负责**: Phase 3, Phase 4
**任务**:
- 实现 EnvironmentService
- 实现 VariableInjector
- 编写变量注入单元测试
- 验证变量优先级

**预计时间**: 2-3 天

---

### Subagent 3: API 和集成专家
**负责**: Phase 5, Phase 6
**任务**:
- 实现 EnvironmentHandler
- 修改 UnifiedTestExecutor
- 修改 WorkflowExecutor
- 集成到 main.go

**预计时间**: 2-3 天

---

### 主 Agent: 协调和测试
**负责**: Phase 7, 整体协调
**任务**:
- 监控 Subagent 进度
- 解决跨模块依赖
- 编写集成测试
- 更新文档

**预计时间**: 2 天

---

## 里程碑和验收标准

### Milestone 1: 数据层完成
- ✅ 数据库迁移成功
- ✅ Repository 单元测试通过
- ✅ 可以 CRUD 环境和变量

### Milestone 2: 变量注入完成
- ✅ 变量注入器单元测试通过
- ✅ 支持所有占位符语法
- ✅ 变量优先级正确

### Milestone 3: API 可用
- ✅ 11 个 API 端点可访问
- ✅ 可以通过 API 管理环境
- ✅ Postman 测试通过

### Milestone 4: 执行器集成完成
- ✅ HTTP 测试自动注入变量
- ✅ 命令测试自动注入变量
- ✅ 工作流测试自动注入变量
- ✅ 环境切换生效

### Milestone 5: 功能完整
- ✅ 所有集成测试通过
- ✅ 文档完整更新
- ✅ 用户指南可用

---

## 风险和缓解措施

### 风险 1: 变量注入性能影响
**影响**: 每次测试执行都需要变量替换
**缓解**:
- 缓存激活环境
- 优化正则匹配
- 只在包含占位符时替换

### 风险 2: 向下兼容性
**影响**: 现有测试可能依赖硬编码值
**缓解**:
- 变量注入可选（注入器为 nil 时跳过）
- 渐进式迁移
- 提供迁移工具

### 风险 3: 并发环境切换
**影响**: 多用户同时切换环境
**缓解**:
- 数据库事务保证原子性
- 环境激活加锁
- 明确的环境切换策略

### 风险 4: 敏感信息泄露
**影响**: API Key 等敏感信息暴露
**缓解**:
- 添加 `is_secret` 标记
- API 返回时脱敏
- 日志中过滤敏感信息

---

## 总结

**总工期**: 8-12 天（1.5-2 周）
**总任务数**: 18 个子任务
**Subagent 数量**: 3 个并行
**预期成果**:
- ✅ 完整的环境管理系统
- ✅ 自动变量注入
- ✅ 11 个 API 端点
- ✅ 向下兼容
- ✅ 完整文档

**下一步**: 确认计划，创建 Subagent 执行任务。

# 已知问题与产品路线图

**文档版本**: 1.0
**当前系统版本**: 2.0
**最后更新**: 2025-11-21

---

## 目录

1. [已知限制与问题](#已知限制与问题)
2. [技术债务](#技术债务)
3. [产品路线图](#产品路线图)
4. [架构改进计划](#架构改进计划)

---

## 已知限制与问题

### 🔴 P0 - 严重问题（影响生产使用）

#### 1. 缺乏多租户/多项目隔离 ⚠️⚠️⚠️

**问题ID**: ISSUE-001
**优先级**: P0 (Critical)
**影响版本**: v2.0
**计划修复版本**: v3.0

**问题描述**:

当前系统**完全没有多租户和多项目隔离机制**，所有数据共享同一个命名空间。这在企业级应用中是一个严重的架构缺陷。

**具体问题**:

1. **全局命名冲突**:
   ```
   团队A创建: envId = "dev"
   团队B创建: envId = "dev"  ❌ 冲突！无法创建
   ```

2. **数据混合**:
   - 所有团队的环境配置存储在同一个 `environments` 表
   - 无法区分哪个环境属于哪个项目/团队
   - 列表 API 返回所有环境（包括其他团队的）

3. **权限缺失**:
   - 任何人可以访问/修改任何环境
   - 无法限制团队A访问团队B的配置
   - 存在误操作和安全风险

4. **资源竞争**:
   - 同时只能有一个激活环境（全局唯一）
   - 团队A激活环境会影响团队B的测试执行
   - 无法并行运行不同项目的测试

**影响范围**:

| 场景 | 影响 | 严重程度 |
|------|------|----------|
| 多团队使用 | 环境配置混乱、命名冲突 | 🔴 Critical |
| 企业级部署 | 无法满足隔离要求 | 🔴 Critical |
| SaaS 部署 | 完全不可用 | 🔴 Critical |
| 安全合规 | 不满足数据隔离要求 | 🔴 Critical |

**当前临时解决方案**:

**方案1: 命名空间前缀** (最简单)
```json
// 团队A
{
  "envId": "team-a-dev",
  "envId": "team-a-staging"
}

// 团队B
{
  "envId": "team-b-dev",
  "envId": "team-b-staging"
}
```

**缺点**:
- ❌ 仅靠命名规范，无强制约束
- ❌ 仍然可以互相访问和修改
- ❌ 无法解决激活状态冲突

**方案2: 独立部署** (最可靠)
```
Team A: test-platform-team-a:8080
Team B: test-platform-team-b:8081
```

**优点**:
- ✅ 完全隔离
- ✅ 独立的激活状态

**缺点**:
- ❌ 运维成本高
- ❌ 资源浪费
- ❌ 无法共享公共配置

**方案3: 应用层过滤** (中等复杂度)

在代码中添加项目过滤逻辑：

```go
// internal/handler/environment_handler.go
func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
    // 从请求头获取项目ID
    projectID := c.GetHeader("X-Project-Id")
    if projectID == "" {
        c.JSON(400, gin.H{"error": "X-Project-Id header required"})
        return
    }

    // 过滤环境（基于命名前缀）
    allEnvs, _ := h.envService.ListEnvironments(limit, offset)
    filteredEnvs := filterByProject(allEnvs, projectID)

    c.JSON(200, filteredEnvs)
}

func filterByProject(envs []models.Environment, projectID string) []models.Environment {
    prefix := projectID + "-"
    var result []models.Environment
    for _, env := range envs {
        if strings.HasPrefix(env.EnvID, prefix) {
            result = append(result, env)
        }
    }
    return result
}
```

**优点**:
- ✅ 不需要修改数据库结构
- ✅ 可以快速实现

**缺点**:
- ❌ 仍然是软隔离，不够安全
- ❌ 需要在所有 API 添加过滤逻辑
- ❌ 容易遗漏导致安全漏洞

**永久解决方案** (v3.0 计划):

**数据库结构变更**:

```sql
-- 添加租户表
CREATE TABLE tenants (
    id INTEGER PRIMARY KEY,
    tenant_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 添加项目表
CREATE TABLE projects (
    id INTEGER PRIMARY KEY,
    project_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
);

-- 修改 environments 表：添加 tenant_id 和 project_id
ALTER TABLE environments ADD COLUMN tenant_id VARCHAR(255);
ALTER TABLE environments ADD COLUMN project_id VARCHAR(255);
ALTER TABLE environments ADD FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id);
ALTER TABLE environments ADD FOREIGN KEY (project_id) REFERENCES projects(project_id);

-- 唯一约束改为租户+项目内唯一
CREATE UNIQUE INDEX idx_env_unique ON environments(tenant_id, project_id, env_id);

-- 激活状态改为项目内唯一
-- 应用层保证：同一 project_id 内只有一个 is_active = TRUE
```

**API 变更**:

```bash
# 需要携带租户和项目信息
curl -X POST http://localhost:8080/api/v2/environments \
  -H "X-Tenant-Id: company-a" \
  -H "X-Project-Id: mobile-app" \
  -d '{
    "envId": "dev",  # 项目内唯一即可
    "name": "Development"
  }'

# 列表 API 自动过滤
curl http://localhost:8080/api/v2/environments \
  -H "X-Tenant-Id: company-a" \
  -H "X-Project-Id: mobile-app"
```

**代码实现**:

```go
// internal/middleware/tenant_context.go
func TenantContext() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetHeader("X-Tenant-Id")
        projectID := c.GetHeader("X-Project-Id")

        if tenantID == "" || projectID == "" {
            c.JSON(400, gin.H{"error": "X-Tenant-Id and X-Project-Id required"})
            c.Abort()
            return
        }

        // 存储到上下文
        c.Set("tenant_id", tenantID)
        c.Set("project_id", projectID)
        c.Next()
    }
}

// internal/repository/environment_repository.go
func (r *environmentRepository) FindByID(envID string, tenantID string, projectID string) (*models.Environment, error) {
    var env models.Environment
    err := r.db.Where("env_id = ? AND tenant_id = ? AND project_id = ? AND deleted_at IS NULL",
        envID, tenantID, projectID).First(&env).Error
    return &env, err
}
```

**迁移策略**:

1. **Phase 1**: 添加 tenant_id/project_id 字段（可选，默认值 "default"）
2. **Phase 2**: 更新 API 支持多租户（向后兼容）
3. **Phase 3**: 强制要求租户信息
4. **Phase 4**: 添加 RBAC 权限控制

**相关 Issues**:
- ISSUE-002: 并发环境切换冲突
- ISSUE-003: 权限控制缺失

---

#### 2. 敏感信息明文存储 🔒

**问题ID**: ISSUE-004
**优先级**: P0 (Critical - 安全)
**影响版本**: v2.0
**计划修复版本**: v2.5

**问题描述**:

环境变量（包括 API Key、密码等敏感信息）以**明文**存储在数据库中。

**安全风险**:

1. **数据库泄露风险**:
   - 数据库备份文件包含明文密钥
   - 数据库管理员可以看到所有密钥
   - SQL 注入可能导致密钥泄露

2. **日志泄露风险**:
   - 调试日志可能打印完整环境变量
   - 错误日志可能包含敏感信息

3. **API 响应泄露**:
   - GET /environments 返回完整变量值
   - WebSocket 实时推送可能包含敏感信息

4. **不满足合规要求**:
   - ❌ SOC 2 Type II
   - ❌ ISO 27001
   - ❌ GDPR（如果包含个人信息）
   - ❌ PCI DSS（如果包含支付信息）

**当前临时解决方案**:

**方案1: 密钥引用** (推荐)
```json
{
  "API_KEY": "vault://secrets/mobile-app/api-key",
  "DB_PASSWORD": "aws-secrets://prod/db-password"
}
```

在测试执行时，从外部密钥管理系统获取实际值。

**方案2: 环境变量注入** (CI/CD)
```bash
# 不在平台存储密钥，在 CI/CD 中注入
export API_KEY="actual-secret-key"

curl -X POST .../environments/prod \
  -d "{\"variables\":{\"API_KEY\":\"$API_KEY\"}}"
```

**方案3: 访问控制** (应用层)
```go
// 限制谁可以查看敏感变量
func (h *EnvironmentHandler) GetVariables(c *gin.Context) {
    userRole := c.GetHeader("X-User-Role")
    variables := h.envService.GetVariables(envID)

    // 非管理员隐藏敏感信息
    if userRole != "admin" {
        variables = maskSensitiveVars(variables)
    }

    c.JSON(200, variables)
}

func maskSensitiveVars(vars map[string]interface{}) map[string]interface{} {
    sensitiveKeys := []string{"API_KEY", "PASSWORD", "SECRET", "TOKEN"}
    masked := make(map[string]interface{})

    for k, v := range vars {
        if containsSensitiveKeyword(k, sensitiveKeys) {
            masked[k] = "***MASKED***"
        } else {
            masked[k] = v
        }
    }
    return masked
}
```

**永久解决方案** (v2.5 计划):

**1. 数据库加密存储**:

```sql
-- 添加 is_secret 标记
ALTER TABLE environment_variables ADD COLUMN is_secret BOOLEAN DEFAULT FALSE;
ALTER TABLE environment_variables ADD COLUMN encrypted_value TEXT;

-- 加密字段示例
INSERT INTO environment_variables (env_id, key, encrypted_value, is_secret)
VALUES ('prod', 'API_KEY', 'AES256:base64encodedvalue', TRUE);
```

**2. 加密实现**:

```go
// internal/crypto/encryption.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
)

type Encryptor struct {
    key []byte // 从环境变量或密钥管理系统获取
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
    block, _ := aes.NewCipher(e.key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
    data, _ := base64.StdEncoding.DecodeString(ciphertext)
    block, _ := aes.NewCipher(e.key)
    gcm, _ := cipher.NewGCM(block)

    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, _ := gcm.Open(nil, nonce, ciphertext, nil)

    return string(plaintext), nil
}
```

**3. 密钥管理系统集成**:

```go
// internal/secrets/vault_client.go
type VaultClient struct {
    client *vault.Client
}

func (v *VaultClient) GetSecret(path string) (string, error) {
    secret, err := v.client.Logical().Read(path)
    if err != nil {
        return "", err
    }
    return secret.Data["value"].(string), nil
}

// 在变量注入时解析
func (vi *VariableInjector) resolveSecret(value string) string {
    if strings.HasPrefix(value, "vault://") {
        path := strings.TrimPrefix(value, "vault://")
        return vi.vaultClient.GetSecret(path)
    }
    return value
}
```

**4. API 响应脱敏**:

```go
type EnvironmentResponse struct {
    EnvID       string                 `json:"envId"`
    Name        string                 `json:"name"`
    Variables   map[string]interface{} `json:"variables"`
}

func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
    env := h.envService.GetEnvironment(envID)

    // 脱敏处理
    response := EnvironmentResponse{
        EnvID: env.EnvID,
        Name:  env.Name,
        Variables: maskSecrets(env.Variables),
    }

    c.JSON(200, response)
}

func maskSecrets(vars map[string]interface{}) map[string]interface{} {
    masked := make(map[string]interface{})
    for k, v := range vars {
        if isSecret(k) {
            masked[k] = "***"
        } else {
            masked[k] = v
        }
    }
    return masked
}
```

**相关 Issues**:
- ISSUE-005: 审计日志缺失
- ISSUE-010: RBAC 权限控制

---

### 🟡 P1 - 重要问题（影响功能完整性）

#### 3. 并发环境切换冲突

**问题ID**: ISSUE-002
**优先级**: P1 (High)
**影响版本**: v2.0
**计划修复版本**: v2.5

**问题描述**:

全局只能有一个激活环境，导致无法并行在多个环境执行测试。

**影响场景**:

```yaml
# GitLab CI - 并行任务会冲突
test:dev:
  parallel: 3
  script:
    - curl -X POST .../environments/dev/activate  # 都想激活 dev
    - curl -X POST .../tests/suite1/execute

test:staging:
  parallel: 3
  script:
    - curl -X POST .../environments/staging/activate  # 与 dev 冲突
    - curl -X POST .../tests/suite2/execute
```

**解决方案** (v2.5):

**方案1: 执行上下文隔离**

```go
// 不依赖全局激活状态，每次执行传递环境参数
POST /tests/:id/execute
{
  "envId": "dev"  // 指定使用的环境
}

// 在执行上下文中注入
func (e *Executor) Execute(testID string, envID string) {
    // 获取指定环境的变量
    vars := e.envService.GetVariables(envID)

    // 注入到测试配置
    config := e.injectVars(test.Config, vars)

    // 执行测试
    return e.run(config)
}
```

**方案2: 执行队列隔离**

```go
// 每个环境维护独立的执行队列
type EnvironmentExecutor struct {
    envID string
    queue chan *TestExecution
}

// 不同环境的测试可以并行执行
devExecutor := NewEnvironmentExecutor("dev")
stagingExecutor := NewEnvironmentExecutor("staging")

devExecutor.Execute(test1)  // 并行
stagingExecutor.Execute(test2)  // 并行
```

---

#### 4. 缺乏配置版本控制

**问题ID**: ISSUE-006
**优先级**: P1 (High)
**影响版本**: v2.0
**计划修复版本**: v3.0

**问题描述**:

无法追踪环境变量的历史变更，无法回滚到之前的配置。

**影响**:
- ❌ 无法审计谁在何时修改了配置
- ❌ 配置错误后无法快速回滚
- ❌ 无法比较不同版本的差异

**解决方案** (v3.0):

**1. 添加变更历史表**:

```sql
CREATE TABLE environment_change_history (
    id INTEGER PRIMARY KEY,
    env_id VARCHAR(255) NOT NULL,
    change_type VARCHAR(16) NOT NULL,  -- create/update/delete/activate
    changed_by VARCHAR(255),           -- 操作人
    old_value TEXT,                    -- 变更前的值（JSON）
    new_value TEXT,                    -- 变更后的值（JSON）
    timestamp DATETIME NOT NULL,
    FOREIGN KEY (env_id) REFERENCES environments(env_id)
);
```

**2. 记录变更**:

```go
func (s *environmentService) UpdateEnvironment(envID string, req *UpdateEnvironmentRequest) error {
    // 获取旧值
    oldEnv, _ := s.envRepo.FindByID(envID)

    // 更新
    newEnv := applyUpdate(oldEnv, req)
    s.envRepo.Update(newEnv)

    // 记录历史
    s.historyRepo.Record(&ChangeHistory{
        EnvID:      envID,
        ChangeType: "update",
        ChangedBy:  req.UserID,
        OldValue:   toJSON(oldEnv),
        NewValue:   toJSON(newEnv),
        Timestamp:  time.Now(),
    })

    return nil
}
```

**3. 回滚功能**:

```go
POST /environments/:id/rollback
{
  "toVersion": 5  // 回滚到历史版本 5
}
```

---

#### 5. 缺乏环境配置共享机制

**问题ID**: ISSUE-007
**优先级**: P1 (Medium)
**影响版本**: v2.0
**计划修复版本**: v3.0

**问题描述**:

无法在多个环境间共享公共配置，必须重复定义。

**示例**:

```json
// Dev
{
  "API_VERSION": "v1",
  "TIMEOUT": 30,
  "RETRY_COUNT": 3,
  "BASE_URL": "http://localhost:3000"
}

// Staging
{
  "API_VERSION": "v1",      // 重复
  "TIMEOUT": 60,
  "RETRY_COUNT": 3,         // 重复
  "BASE_URL": "https://staging.example.com"
}

// Prod
{
  "API_VERSION": "v1",      // 重复
  "TIMEOUT": 120,
  "RETRY_COUNT": 3,         // 重复
  "BASE_URL": "https://api.example.com"
}
```

**解决方案** (v3.0):

**方案1: 环境模板**

```sql
CREATE TABLE environment_templates (
    id INTEGER PRIMARY KEY,
    template_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    variables TEXT  -- 基础变量
);

-- environments 表添加 template_id
ALTER TABLE environments ADD COLUMN template_id VARCHAR(255);
```

使用：
```json
// 创建模板
POST /environment-templates
{
  "templateId": "api-service-base",
  "variables": {
    "API_VERSION": "v1",
    "RETRY_COUNT": 3
  }
}

// 基于模板创建环境
POST /environments
{
  "envId": "dev",
  "templateId": "api-service-base",
  "variables": {
    "BASE_URL": "http://localhost:3000",  // 只需定义差异
    "TIMEOUT": 30
  }
}
```

**方案2: 环境继承**

```json
POST /environments
{
  "envId": "staging",
  "parentEnvId": "dev",  // 继承 dev 的变量
  "variables": {
    "BASE_URL": "https://staging.example.com"  // 覆盖 BASE_URL
  }
}
```

---

### 🟢 P2 - 次要问题（影响用户体验）

#### 6. 性能优化

**问题ID**: ISSUE-008
**优先级**: P2 (Medium)
**影响版本**: v2.0
**计划修复版本**: v2.5

**问题**:
- 大量环境时列表查询慢
- 复杂变量注入性能开销大

**解决方案**:
- 添加缓存
- 优化查询索引
- 变量注入算法优化

---

#### 7. 缺乏搜索和过滤

**问题ID**: ISSUE-009
**优先级**: P2 (Low)
**影响版本**: v2.0
**计划修复版本**: v3.0

**需求**:
- 按变量名搜索环境
- 按激活状态过滤
- 按创建时间排序

---

## 技术债务

### 1. 测试覆盖率不足

**当前状态**:
- 单元测试覆盖率: ~40%
- 集成测试: 8/9 通过（1个并发测试失败）

**计划**:
- v2.5: 提升到 80%
- v3.0: 提升到 90%

### 2. 缺乏监控和可观测性

**缺失功能**:
- ❌ Prometheus metrics
- ❌ 分布式追踪
- ❌ 结构化日志

**计划**: v3.0 添加

### 3. 文档不完整

**已完成**:
- ✅ API 文档
- ✅ 数据库设计文档
- ✅ 用户指南

**缺失**:
- ❌ 架构设计文档
- ❌ 开发者指南
- ❌ 部署文档

---

## 产品路线图

### v2.5 (计划: 2026-01)

**重点**: 安全和并发

| 功能 | 优先级 | 状态 |
|------|--------|------|
| 变量加密存储 | P0 | 📝 设计中 |
| 执行上下文隔离 | P0 | 📝 设计中 |
| 敏感信息脱敏 | P0 | 📝 设计中 |
| 激活历史记录 | P1 | 🔜 待开始 |
| 性能优化 | P1 | 🔜 待开始 |
| 变量搜索 | P2 | 🔜 待开始 |

### v3.0 (计划: 2026-03)

**重点**: 多租户和权限

| 功能 | 优先级 | 状态 |
|------|--------|------|
| 多租户架构 | P0 | 📝 设计中 |
| 项目管理 | P0 | 📝 设计中 |
| RBAC 权限控制 | P0 | 📝 设计中 |
| 环境模板 | P1 | 🔜 待开始 |
| 配置版本控制 | P1 | 🔜 待开始 |
| 配置回滚 | P1 | 🔜 待开始 |
| 审计日志 | P1 | 🔜 待开始 |

### v4.0+ (计划: 2026-06+)

**重点**: 企业级特性

| 功能 | 优先级 | 状态 |
|------|--------|------|
| 密钥管理系统集成 | P0 | 🔮 规划中 |
| SSO/SAML 集成 | P0 | 🔮 规划中 |
| 多实例同步 | P1 | 🔮 规划中 |
| 配置市场 | P2 | 🔮 规划中 |
| AI 辅助配置 | P2 | 🔮 规划中 |

---

## 架构改进计划

### 1. 多租户架构重构

**目标**: 支持完整的多租户隔离

**变更**:

```
当前架构:
┌─────────────────────────────┐
│  Single Database            │
│  ├─ environments (global)   │
│  ├─ test_cases (global)     │
│  └─ workflows (global)      │
└─────────────────────────────┘

目标架构 (v3.0):
┌─────────────────────────────┐
│  Database with Tenant ID    │
│  ├─ tenants                 │
│  ├─ projects                │
│  ├─ environments (scoped)   │
│  ├─ test_cases (scoped)     │
│  └─ workflows (scoped)      │
└─────────────────────────────┘

每个查询自动添加租户过滤:
SELECT * FROM environments
WHERE tenant_id = ? AND project_id = ?
```

### 2. 微服务拆分 (v4.0)

**当前**: 单体应用

**目标**: 微服务架构

```
┌─────────────────┐
│  API Gateway    │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼───────┐
│ Test  │ │Environment│
│Service│ │  Service  │
└───┬───┘ └──┬───────┘
    │        │
    └────┬───┘
         │
    ┌────▼────┐
    │Database │
    └─────────┘
```

### 3. 事件驱动架构

**目标**: 异步处理和解耦

```
Environment Activated
    │
    ├─> Event Bus (Kafka)
    │       │
    │       ├─> Notification Service (Webhook)
    │       ├─> Audit Service (日志)
    │       └─> Monitoring Service (指标)
    │
    └─> Test Execution Queue
```

---

## 贡献指南

### 如何报告问题

1. 检查 [已知问题](#已知限制与问题) 列表
2. 在 GitHub 创建 Issue，使用模板:

```markdown
## 问题描述
[简要描述问题]

## 复现步骤
1. [步骤1]
2. [步骤2]

## 期望行为
[描述期望的行为]

## 实际行为
[描述实际发生的情况]

## 环境信息
- 系统版本: v2.0
- 操作系统: [OS]
- 数据库: [SQLite/PostgreSQL/MySQL]
```

### 如何贡献修复

1. Fork 仓库
2. 创建特性分支: `git checkout -b fix/issue-001`
3. 提交代码
4. 创建 Pull Request

---

## 联系方式

- **GitHub Issues**: https://github.com/your-org/test-management-service/issues
- **邮件**: support@example.com
- **文档**: [docs/](../docs/)

---

**文档版本**: 1.0
**最后更新**: 2025-11-21
**维护者**: Development Team

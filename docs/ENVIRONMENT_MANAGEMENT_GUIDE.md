# 环境管理用户指南

**版本**: 2.0
**最后更新**: 2025-11-21
**适用于**: 测试管理服务 v2.0+

---

## 目录

1. [概述](#概述)
2. [快速开始](#快速开始)
3. [核心概念](#核心概念)
4. [使用场景](#使用场景)
5. [变量注入详解](#变量注入详解)
6. [CI/CD 集成](#cicd-集成)
7. [最佳实践](#最佳实践)
8. [常见问题 (FAQ)](#常见问题-faq)
9. [已知限制与未来规划](#已知限制与未来规划)
10. [故障排查](#故障排查)

---

## 概述

### 什么是环境管理？

环境管理功能允许您为不同的测试环境（如 Dev、Staging、Prod）定义和管理配置变量，并通过变量注入机制在测试执行时动态替换配置。

### 核心价值

- ✅ **一套测试，多环境运行**: 同一个测试脚本可以在不同环境下执行
- ✅ **配置集中管理**: 所有环境配置集中存储，便于维护和审计
- ✅ **环境快速切换**: 一键切换测试环境，无需修改测试代码
- ✅ **CI/CD 友好**: 通过 API 与 Jenkins、GitLab CI 等工具无缝集成
- ✅ **类型安全**: 支持字符串、数字、布尔等多种数据类型

### 适用场景

| 场景 | 描述 | 示例 |
|------|------|------|
| **多环境测试** | 在 Dev/Staging/Prod 运行相同测试 | API 接口测试 |
| **CI/CD Pipeline** | 根据分支自动切换测试环境 | GitLab CI 自动化 |
| **回归测试** | 在多个环境验证修复 | Bug 修复验证 |
| **配置变更验证** | 验证配置变更影响 | 超时参数调整 |

---

## 快速开始

### 第一步：创建环境

```bash
# 创建开发环境
curl -X POST http://localhost:8080/api/v2/environments \
  -H "Content-Type: application/json" \
  -d '{
    "envId": "dev",
    "name": "开发环境",
    "description": "本地开发测试环境",
    "variables": {
      "BASE_URL": "http://localhost:3000",
      "API_KEY": "dev-key-12345",
      "TIMEOUT": 30,
      "DEBUG": true
    }
  }'

# 创建预发布环境
curl -X POST http://localhost:8080/api/v2/environments \
  -H "Content-Type: application/json" \
  -d '{
    "envId": "staging",
    "name": "预发布环境",
    "description": "预发布测试环境",
    "variables": {
      "BASE_URL": "https://staging.example.com",
      "API_KEY": "staging-key-67890",
      "TIMEOUT": 60,
      "DEBUG": false
    }
  }'
```

### 第二步：激活环境

```bash
# 激活开发环境
curl -X POST http://localhost:8080/api/v2/environments/dev/activate
```

### 第三步：创建测试（使用变量）

```bash
curl -X POST http://localhost:8080/api/v2/tests \
  -H "Content-Type: application/json" \
  -d '{
    "testId": "login-test",
    "name": "用户登录测试",
    "type": "http",
    "http": {
      "method": "POST",
      "path": "{{BASE_URL}}/api/login",
      "headers": {
        "Authorization": "Bearer {{API_KEY}}",
        "Content-Type": "application/json"
      },
      "body": {
        "username": "testuser",
        "password": "password123"
      }
    },
    "assertions": [
      {
        "type": "status_code",
        "expected": 200
      }
    ]
  }'
```

### 第四步：执行测试

```bash
# 在当前激活环境（dev）下执行测试
curl -X POST http://localhost:8080/api/v2/tests/login-test/execute
```

测试将自动使用 dev 环境的变量：
- `{{BASE_URL}}` → `http://localhost:3000`
- `{{API_KEY}}` → `dev-key-12345`

### 第五步：切换环境并重新执行

```bash
# 切换到预发布环境
curl -X POST http://localhost:8080/api/v2/environments/staging/activate

# 再次执行相同测试（自动使用 staging 环境变量）
curl -X POST http://localhost:8080/api/v2/tests/login-test/execute
```

---

## 核心概念

### 1. 环境 (Environment)

环境是一组配置变量的集合，代表一个测试环境（如 Dev、Staging、Prod）。

**环境属性**:
- `envId`: 环境唯一标识符（不可修改）
- `name`: 环境名称
- `description`: 环境描述
- `isActive`: 是否为当前激活环境（同时只能有一个）
- `variables`: 环境变量键值对

### 2. 激活状态

同一时间只能有一个环境处于激活状态。激活环境的变量会被自动注入到测试执行中。

**激活规则**:
- ✅ 激活新环境时，旧环境自动停用
- ✅ 事务安全，确保激活状态一致性
- ❌ 不能删除激活的环境

### 3. 变量注入

使用 `{{VARIABLE_NAME}}` 语法在测试配置中引用环境变量，执行时自动替换为实际值。

**支持的位置**:
- HTTP 测试的 URL、Headers、Body
- 命令测试的参数
- Workflow 的全局变量

### 4. 变量优先级

当多层存在同名变量时，按以下优先级覆盖：

```
Environment < Workflow < TestCase
```

**示例**:
```json
// Environment
{"TIMEOUT": 30}

// Workflow
{"TIMEOUT": 60}

// 最终结果: TIMEOUT = 60 (Workflow 覆盖 Environment)
```

---

## 使用场景

### 场景1: 多环境 API 测试

**需求**: 在 Dev、Staging、Prod 三个环境测试用户登录 API。

**步骤**:

1. **创建三个环境**:
```bash
# Dev
curl -X POST .../environments -d '{"envId":"dev","variables":{"BASE_URL":"http://localhost:3000"}}'

# Staging
curl -X POST .../environments -d '{"envId":"staging","variables":{"BASE_URL":"https://staging.api.com"}}'

# Prod
curl -X POST .../environments -d '{"envId":"prod","variables":{"BASE_URL":"https://api.example.com"}}'
```

2. **创建一个测试**:
```json
{
  "testId": "login-test",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "{{BASE_URL}}/api/login"
  }
}
```

3. **在不同环境执行**:
```bash
# Dev
curl -X POST .../environments/dev/activate
curl -X POST .../tests/login-test/execute

# Staging
curl -X POST .../environments/staging/activate
curl -X POST .../tests/login-test/execute

# Prod
curl -X POST .../environments/prod/activate
curl -X POST .../tests/login-test/execute
```

### 场景2: CI/CD Pipeline 自动化

**需求**: GitLab CI 根据分支自动选择环境执行测试。

**.gitlab-ci.yml**:
```yaml
variables:
  TEST_PLATFORM: "http://test-platform:8080"

test:dev:
  stage: test
  only:
    - develop
  script:
    - curl -X POST $TEST_PLATFORM/api/v2/environments/dev/activate
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute

test:staging:
  stage: test
  only:
    - /^release\/.*$/
  script:
    - curl -X POST $TEST_PLATFORM/api/v2/environments/staging/activate
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/regression-suite/execute

test:prod:
  stage: test
  only:
    - tags
  script:
    - curl -X POST $TEST_PLATFORM/api/v2/environments/prod/activate
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute
```

### 场景3: 配置参数调优

**需求**: 测试不同超时参数对 API 性能的影响。

**步骤**:

1. **创建测试环境**:
```bash
curl -X POST .../environments -d '{
  "envId": "timeout-30",
  "variables": {"TIMEOUT": 30}
}'

curl -X POST .../environments -d '{
  "envId": "timeout-60",
  "variables": {"TIMEOUT": 60}
}'
```

2. **创建性能测试**:
```json
{
  "testId": "perf-test",
  "type": "http",
  "http": {
    "method": "GET",
    "path": "{{BASE_URL}}/api/data",
    "timeout": "{{TIMEOUT}}"
  }
}
```

3. **依次测试不同配置**:
```bash
curl -X POST .../environments/timeout-30/activate
curl -X POST .../tests/perf-test/execute

curl -X POST .../environments/timeout-60/activate
curl -X POST .../tests/perf-test/execute
```

### 场景4: Workflow 中使用环境变量

**需求**: 在多步骤 Workflow 中共享环境配置。

**Workflow 定义**:
```json
{
  "workflowId": "user-registration-flow",
  "definition": {
    "variables": {
      "BASE_URL": "{{BASE_URL}}",
      "API_KEY": "{{API_KEY}}"
    },
    "steps": {
      "register": {
        "id": "register",
        "type": "http",
        "config": {
          "method": "POST",
          "path": "{{BASE_URL}}/api/register",
          "headers": {
            "Authorization": "Bearer {{API_KEY}}"
          }
        }
      },
      "verify": {
        "id": "verify",
        "type": "http",
        "config": {
          "method": "POST",
          "path": "{{BASE_URL}}/api/verify"
        }
      }
    }
  }
}
```

所有步骤都会使用当前激活环境的 `BASE_URL` 和 `API_KEY`。

---

## 变量注入详解

### 注入语法

使用双大括号包裹变量名：`{{VARIABLE_NAME}}`

**规则**:
- 变量名必须是字母、数字、下划线组合
- 大小写敏感
- 支持嵌套对象和数组

### 类型保持

变量注入会保持原始数据类型：

| 变量值 | 注入后类型 | 示例 |
|--------|-----------|------|
| `"http://..."` | string | `"{{BASE_URL}}"` → `"http://localhost:3000"` |
| `30` | number | `"{{TIMEOUT}}"` → `30` (不是 `"30"`) |
| `true` | boolean | `"{{DEBUG}}"` → `true` (不是 `"true"`) |
| `{"key":"value"}` | object | `"{{CONFIG}}"` → `{"key":"value"}` |
| `[1,2,3]` | array | `"{{IDS}}"` → `[1,2,3]` |

### 完整替换 vs 部分替换

**完整替换** (保持类型):
```json
// 环境变量
{"TIMEOUT": 30}

// 测试配置
{"timeout": "{{TIMEOUT}}"}

// 注入后
{"timeout": 30}  // 类型为 number
```

**部分替换** (转为字符串):
```json
// 环境变量
{"API_VERSION": "v1"}

// 测试配置
{"path": "/api/{{API_VERSION}}/users"}

// 注入后
{"path": "/api/v1/users"}  // 类型为 string
```

### 嵌套注入

支持在对象和数组中递归注入：

```json
// 环境变量
{
  "BASE_URL": "http://localhost:3000",
  "API_KEY": "key-123"
}

// 测试配置
{
  "http": {
    "method": "POST",
    "path": "{{BASE_URL}}/api/login",
    "headers": {
      "Authorization": "Bearer {{API_KEY}}",
      "X-Custom": "fixed-value"
    },
    "body": {
      "config": {
        "url": "{{BASE_URL}}/callback"
      }
    }
  }
}

// 注入后
{
  "http": {
    "method": "POST",
    "path": "http://localhost:3000/api/login",
    "headers": {
      "Authorization": "Bearer key-123",
      "X-Custom": "fixed-value"
    },
    "body": {
      "config": {
        "url": "http://localhost:3000/callback"
      }
    }
  }
}
```

### 变量未定义处理

如果引用的变量不存在，占位符保持不变：

```json
// 环境变量
{"BASE_URL": "http://localhost:3000"}

// 测试配置
{"path": "{{BASE_URL}}/{{UNKNOWN}}"}

// 注入后
{"path": "http://localhost:3000/{{UNKNOWN}}"}
```

---

## CI/CD 集成

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['dev', 'staging', 'prod'],
            description: '选择测试环境'
        )
    }

    environment {
        TEST_PLATFORM = 'http://test-platform:8080'
    }

    stages {
        stage('激活环境') {
            steps {
                script {
                    sh """
                        curl -X POST ${TEST_PLATFORM}/api/v2/environments/${params.ENVIRONMENT}/activate
                    """
                }
            }
        }

        stage('执行测试') {
            steps {
                script {
                    def response = sh(
                        script: "curl -X POST ${TEST_PLATFORM}/api/v2/workflows/regression-suite/execute",
                        returnStdout: true
                    ).trim()

                    echo "测试结果: ${response}"
                }
            }
        }

        stage('验证结果') {
            steps {
                script {
                    // 获取测试结果并验证
                    def result = sh(
                        script: "curl ${TEST_PLATFORM}/api/v2/test-runs/latest",
                        returnStdout: true
                    ).trim()

                    def json = readJSON text: result
                    if (json.status != 'success') {
                        error("测试失败: ${json.error}")
                    }
                }
            }
        }
    }

    post {
        always {
            echo "测试完成，环境: ${params.ENVIRONMENT}"
        }
    }
}
```

### GitLab CI (完整示例)

```yaml
stages:
  - test-dev
  - test-staging
  - test-prod

variables:
  TEST_PLATFORM: "http://test-platform:8080"

# Dev 环境测试
test:dev:
  stage: test-dev
  only:
    - develop
    - merge_requests
  script:
    - echo "激活 Dev 环境"
    - curl -X POST $TEST_PLATFORM/api/v2/environments/dev/activate
    - echo "执行冒烟测试"
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute
    - echo "执行单元测试"
    - curl -X POST $TEST_PLATFORM/api/v2/test-runs/create -d '{"groupId":"unit-tests"}'
  allow_failure: false

# Staging 环境测试
test:staging:
  stage: test-staging
  only:
    - /^release\/.*$/
    - main
  script:
    - echo "激活 Staging 环境"
    - curl -X POST $TEST_PLATFORM/api/v2/environments/staging/activate
    - echo "执行完整回归测试套件"
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/regression-suite/execute
    - echo "等待测试完成"
    - sleep 30
    - echo "获取测试结果"
    - RESULT=$(curl $TEST_PLATFORM/api/v2/test-runs/latest)
    - echo $RESULT
  allow_failure: false
  artifacts:
    reports:
      junit: test-results.xml

# Prod 环境测试（仅标签触发）
test:prod:
  stage: test-prod
  only:
    - tags
  script:
    - echo "激活 Prod 环境"
    - curl -X POST $TEST_PLATFORM/api/v2/environments/prod/activate
    - echo "执行生产环境冒烟测试"
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/prod-smoke-test/execute
  when: manual  # 需要手动触发
  allow_failure: false
```

### GitHub Actions

```yaml
name: Multi-Environment Tests

on:
  push:
    branches: [ develop, main ]
  pull_request:
    branches: [ develop ]
  release:
    types: [ published ]

env:
  TEST_PLATFORM: http://test-platform:8080

jobs:
  test-dev:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    steps:
      - name: 激活 Dev 环境
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/environments/dev/activate

      - name: 执行测试
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute

  test-staging:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - name: 激活 Staging 环境
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/environments/staging/activate

      - name: 执行回归测试
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/workflows/regression-suite/execute

  test-prod:
    runs-on: ubuntu-latest
    if: github.event_name == 'release'
    steps:
      - name: 激活 Prod 环境
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/environments/prod/activate

      - name: 执行生产冒烟测试
        run: |
          curl -X POST $TEST_PLATFORM/api/v2/workflows/prod-smoke-test/execute
```

---

## 最佳实践

### 1. 环境命名规范

**推荐**:
- ✅ 使用小写字母和连字符: `dev`, `staging`, `prod`
- ✅ 语义化命名: `perf-test`, `security-test`
- ✅ 简短易记: 避免过长的环境 ID

**不推荐**:
- ❌ 使用空格或特殊字符: `dev env`, `prod#1`
- ❌ 无意义命名: `env1`, `test123`

### 2. 变量命名规范

**推荐**:
- ✅ 全大写+下划线: `BASE_URL`, `API_KEY`, `MAX_TIMEOUT`
- ✅ 语义明确: 见名知意
- ✅ 分组前缀: `DB_HOST`, `DB_PORT`, `DB_NAME`

**示例**:
```json
{
  "BASE_URL": "http://localhost:3000",
  "API_KEY": "dev-key-12345",
  "API_SECRET": "dev-secret",
  "DB_HOST": "localhost",
  "DB_PORT": 5432,
  "DB_NAME": "test_db",
  "MAX_TIMEOUT": 60,
  "RETRY_COUNT": 3,
  "DEBUG_MODE": true
}
```

### 3. 敏感信息管理

⚠️ **安全警告**: 当前版本环境变量以明文存储，生产环境需要额外安全措施。

**临时方案**:
1. 使用环境变量注入（在 CI/CD 中）
2. 定期轮换密钥
3. 限制访问权限（应用层）

**未来版本计划**:
- 🔒 变量加密存储
- 🔐 敏感信息标记 (is_secret)
- 🔑 与密钥管理系统集成（如 HashiCorp Vault）

### 4. 环境变量版本管理

建议将环境配置导出为 JSON 文件进行版本控制：

```bash
# 导出环境配置
curl http://localhost:8080/api/v2/environments/dev > dev-env.json

# Git 版本控制
git add dev-env.json
git commit -m "更新 dev 环境配置"
```

**配置文件示例** (`environments.json`):
```json
{
  "environments": [
    {
      "envId": "dev",
      "name": "Development",
      "variables": {
        "BASE_URL": "http://localhost:3000",
        "API_KEY": "dev-key-12345"
      }
    },
    {
      "envId": "staging",
      "name": "Staging",
      "variables": {
        "BASE_URL": "https://staging.example.com",
        "API_KEY": "staging-key-67890"
      }
    }
  ]
}
```

### 5. 测试隔离

**单环境隔离**: 同一个测试在不同环境使用不同配置

```json
// Dev: 短超时，调试模式
{"TIMEOUT": 30, "DEBUG": true}

// Prod: 长超时，关闭调试
{"TIMEOUT": 120, "DEBUG": false}
```

**数据隔离**: 不同环境使用不同数据库

```json
// Dev
{"DB_NAME": "test_db_dev"}

// Staging
{"DB_NAME": "test_db_staging"}

// Prod
{"DB_NAME": "production_db"}
```

### 6. 渐进式发布

在多环境中渐进式验证变更：

1. **Dev** → 开发测试
2. **Staging** → 完整回归测试
3. **Prod** → 冒烟测试

```bash
# 1. Dev 环境快速验证
curl -X POST .../environments/dev/activate
curl -X POST .../tests/quick-test/execute

# 2. Staging 环境完整测试
curl -X POST .../environments/staging/activate
curl -X POST .../workflows/full-regression/execute

# 3. Prod 环境冒烟测试
curl -X POST .../environments/prod/activate
curl -X POST .../workflows/smoke-test/execute
```

---

## 常见问题 (FAQ)

### Q1: 如何查看当前激活的环境？

```bash
curl http://localhost:8080/api/v2/environments/active
```

### Q2: 可以同时激活多个环境吗？

❌ 不可以。系统设计为同一时间只能有一个环境激活。

**原因**: 保证变量注入的确定性和一致性。

### Q3: 如何删除环境？

```bash
# 1. 先停用环境（如果是激活状态）
curl -X POST http://localhost:8080/api/v2/environments/other-env/activate

# 2. 删除环境
curl -X DELETE http://localhost:8080/api/v2/environments/dev
```

⚠️ **注意**: 不能删除激活的环境。

### Q4: 变量注入失败了怎么办？

**检查步骤**:
1. 确认环境已激活: `GET /environments/active`
2. 确认变量存在: `GET /environments/{id}/variables`
3. 检查变量名拼写: `{{BASE_URL}}` (大小写敏感)
4. 查看测试执行日志

### Q5: 如何批量更新环境变量？

使用 `PUT /environments/{id}` 更新整个环境：

```bash
curl -X PUT http://localhost:8080/api/v2/environments/dev \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Development",
    "variables": {
      "BASE_URL": "http://localhost:4000",
      "API_KEY": "new-key",
      "TIMEOUT": 60
    }
  }'
```

⚠️ **注意**: 这会替换所有变量，确保包含所有需要的变量。

### Q6: 环境变量支持哪些数据类型？

支持所有 JSON 数据类型：
- ✅ String: `"http://localhost:3000"`
- ✅ Number: `30`, `3.14`
- ✅ Boolean: `true`, `false`
- ✅ Object: `{"key": "value"}`
- ✅ Array: `[1, 2, 3]`
- ✅ Null: `null`

### Q7: 如何在本地测试中使用环境管理？

```bash
# 1. 启动测试管理服务
go run cmd/server/main.go

# 2. 创建本地环境
curl -X POST http://localhost:8080/api/v2/environments \
  -d '{"envId":"local","variables":{"BASE_URL":"http://localhost:3000"}}'

# 3. 激活本地环境
curl -X POST http://localhost:8080/api/v2/environments/local/activate

# 4. 执行测试
curl -X POST http://localhost:8080/api/v2/tests/my-test/execute
```

### Q8: 可以导入/导出环境配置吗？

**导出**:
```bash
curl http://localhost:8080/api/v2/environments/dev > dev-env.json
```

**导入** (需要解析 JSON 并调用 API):
```bash
cat dev-env.json | jq -r '.variables' | \
  curl -X POST http://localhost:8080/api/v2/environments \
    -H "Content-Type: application/json" \
    -d @-
```

### Q9: 环境变量的最大大小限制是多少？

- 单个环境变量值: 无硬限制（受数据库 TEXT 字段限制）
- 整个环境的 variables: 建议不超过 64KB
- 变量数量: 无限制

**最佳实践**: 避免在环境变量中存储大量数据，建议存储配置引用。

### Q10: 如何实现环境间变量继承？

当前版本不支持环境继承。**临时方案**:

使用脚本生成环境配置：

```bash
# base-env.json
{
  "BASE_URL": "http://api.example.com",
  "TIMEOUT": 60
}

# 生成 dev 环境（覆盖部分变量）
jq '. + {"BASE_URL": "http://localhost:3000", "DEBUG": true}' base-env.json > dev-env.json
```

---

## 已知限制与未来规划

### 当前已知限制

#### 1. 多租户/多项目隔离问题 ⚠️

**问题描述**:
- 当前版本**不支持多租户隔离**
- 所有环境、测试、工作流共享同一个数据库
- 不同团队/项目的数据混在一起，存在冲突风险

**影响范围**:
- ❌ 无法为不同团队创建独立的环境空间
- ❌ 环境 ID 必须全局唯一（不同项目不能用相同的 envId）
- ❌ 无法实现基于租户的权限控制
- ❌ 测试结果和日志无法按租户隔离

**临时解决方案**:
1. **命名空间前缀**: 在 envId 中添加项目前缀
   ```
   项目A: project-a-dev, project-a-staging
   项目B: project-b-dev, project-b-staging
   ```

2. **应用层隔离**: 通过应用层逻辑限制访问
   ```go
   // 在 Handler 层添加项目过滤
   func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
       projectId := c.GetHeader("X-Project-Id")
       envs := h.envService.ListByProject(projectId)
       // ...
   }
   ```

3. **独立部署**: 为每个团队/项目部署独立的测试管理服务实例

**未来规划** (v3.0):
- 🎯 **多租户架构**: 添加 `tenant_id` 字段到所有表
- 🎯 **项目隔离**: 支持项目级别的数据隔离
- 🎯 **权限管理**: 基于角色的访问控制 (RBAC)
- 🎯 **数据隔离**: 租户间数据完全隔离

#### 2. 并发环境切换限制 ⚠️

**问题描述**:
- 同一时间只能有一个激活环境
- 无法并行在多个环境执行测试

**影响范围**:
- ❌ 不支持并行多环境测试
- ❌ CI/CD Pipeline 中的并行任务可能冲突

**临时解决方案**:
- 使用串行执行: 先执行 dev，再执行 staging
- 部署多个测试服务实例，每个实例管理一个环境

**未来规划** (v2.5):
- 🎯 **执行上下文隔离**: 每次执行携带环境参数，不依赖全局激活状态
- 🎯 **并行执行支持**: 支持同时在多个环境执行测试

#### 3. 敏感信息存储问题 🔒

**问题描述**:
- 环境变量以明文存储在数据库中
- 无法区分敏感信息（如密钥、密码）

**影响范围**:
- ⚠️ 生产环境存在安全风险
- ⚠️ 无法满足合规要求（如 SOC2、ISO 27001）

**临时解决方案**:
1. **外部密钥管理**: 在变量中存储密钥引用，实际密钥存储在外部系统
   ```json
   {
     "API_KEY": "vault://secrets/api-key",
     "DB_PASSWORD": "aws-secrets://db-password"
   }
   ```

2. **环境变量注入**: 在 CI/CD 中通过环境变量注入敏感信息
   ```bash
   export API_KEY="secret-key"
   curl -X POST .../environments/prod -d "{\"variables\":{\"API_KEY\":\"$API_KEY\"}}"
   ```

**未来规划** (v2.5):
- 🔒 **变量加密**: 敏感变量加密存储
- 🔐 **is_secret 标记**: 标记敏感变量，API 响应中自动脱敏
- 🔑 **密钥管理集成**: 与 HashiCorp Vault、AWS Secrets Manager 集成

#### 4. 环境版本控制问题 📝

**问题描述**:
- 无法追踪环境变量的历史变更
- 无法回滚到之前的配置

**影响范围**:
- ❌ 无法审计谁在何时修改了环境配置
- ❌ 配置错误后无法快速回滚

**临时解决方案**:
- 定期导出环境配置到 Git
- 手动记录配置变更日志

**未来规划** (v3.0):
- 📝 **变更历史**: 记录所有环境变量的修改历史
- 🔄 **配置回滚**: 支持一键回滚到历史版本
- 👤 **审计日志**: 记录操作人、时间、变更内容

#### 5. 环境配置共享问题 🔗

**问题描述**:
- 无法在多个环境间共享公共配置
- 必须在每个环境重复定义相同的变量

**影响范围**:
- ❌ 配置维护成本高
- ❌ 容易出现配置不一致

**临时解决方案**:
- 使用脚本生成环境配置（基于模板）
- 通过 CI/CD 统一管理配置

**未来规划** (v3.0):
- 🔗 **环境模板**: 定义环境模板，多个环境继承同一模板
- 📦 **变量组**: 将公共变量组织为变量组，多个环境引用
- 🌲 **环境继承**: 支持环境间继承关系（如 staging 继承 dev）

#### 6. 性能限制 ⚡

**问题描述**:
- 大量环境时列表查询性能下降
- 复杂变量注入（深层嵌套）性能开销大

**影响范围**:
- ⚠️ 环境数量超过 100 个时可能出现性能问题
- ⚠️ 单个测试有大量变量注入时执行变慢

**临时解决方案**:
- 定期清理不用的环境
- 优化测试配置，减少不必要的变量引用

**未来规划** (v2.5):
- ⚡ **缓存优化**: 缓存激活环境的变量
- 📊 **查询优化**: 添加分页、搜索、过滤功能
- 🚀 **注入优化**: 优化变量注入算法

### 未来功能路线图

#### 短期 (v2.5 - 1-2 个月)

| 功能 | 优先级 | 描述 |
|------|--------|------|
| 变量加密存储 | P0 | 敏感变量加密存储 |
| 执行上下文隔离 | P0 | 支持并行多环境执行 |
| 激活历史记录 | P1 | 记录环境激活历史 |
| 变量搜索 | P1 | 支持按变量名搜索环境 |
| Webhook 通知 | P2 | 环境切换时发送 Webhook |

#### 中期 (v3.0 - 3-6 个月)

| 功能 | 优先级 | 描述 |
|------|--------|------|
| 多租户架构 | P0 | 支持多租户隔离 |
| 项目管理 | P0 | 支持项目级别隔离 |
| RBAC 权限控制 | P0 | 基于角色的访问控制 |
| 环境模板 | P1 | 环境模板和继承 |
| 变更历史 | P1 | 完整的配置变更审计 |
| 配置回滚 | P1 | 一键回滚到历史版本 |

#### 长期 (v4.0+ - 6+ 个月)

| 功能 | 优先级 | 描述 |
|------|--------|------|
| 密钥管理集成 | P0 | 与 Vault、AWS Secrets 集成 |
| 环境同步 | P1 | 多实例间环境同步 |
| 配置市场 | P2 | 共享环境配置模板 |
| AI 辅助配置 | P2 | 智能推荐环境配置 |

---

## 故障排查

### 问题1: 变量未被替换

**症状**: 测试执行后，`{{VARIABLE}}` 占位符没有被替换。

**排查步骤**:

1. **检查环境是否激活**:
```bash
curl http://localhost:8080/api/v2/environments/active
```

如果返回 404，说明没有激活的环境：
```bash
curl -X POST http://localhost:8080/api/v2/environments/dev/activate
```

2. **检查变量是否存在**:
```bash
curl http://localhost:8080/api/v2/environments/dev/variables
```

3. **检查变量名拼写**:
   - 变量名大小写敏感
   - 确保 `{{BASE_URL}}` 和环境变量中的 `BASE_URL` 完全一致

4. **检查变量注入日志**:
```bash
# 查看服务日志
tail -f /var/log/test-management-service.log | grep "variable injection"
```

### 问题2: 环境激活失败

**症状**: 调用激活 API 返回错误。

**常见错误**:

1. **环境不存在**:
```json
{"error": "environment not found: staging"}
```

**解决**: 先创建环境：
```bash
curl -X POST .../environments -d '{"envId":"staging",...}'
```

2. **数据库事务失败**:
```json
{"error": "failed to activate environment: database locked"}
```

**解决**: 检查数据库连接和并发访问。

### 问题3: 无法删除环境

**症状**: 删除环境时返回错误。

**错误信息**:
```json
{"error": "cannot delete active environment 'dev'"}
```

**解决**: 先激活其他环境，再删除：
```bash
# 激活其他环境
curl -X POST .../environments/staging/activate

# 删除 dev 环境
curl -X DELETE .../environments/dev
```

### 问题4: CI/CD 中环境切换冲突

**症状**: 多个 CI 任务并行执行时，环境切换互相干扰。

**原因**: 系统同时只能有一个激活环境。

**解决方案**:

**方案1**: 串行执行
```yaml
# .gitlab-ci.yml
stages:
  - test-dev
  - test-staging  # 等 test-dev 完成后执行
```

**方案2**: 部署多个实例
```yaml
test:dev:
  services:
    - name: test-platform-dev:latest
  script:
    - curl http://test-platform-dev/api/v2/environments/dev/activate

test:staging:
  services:
    - name: test-platform-staging:latest
  script:
    - curl http://test-platform-staging/api/v2/environments/staging/activate
```

### 问题5: 性能问题

**症状**: 测试执行变慢。

**可能原因**:

1. **变量注入开销大**: 测试配置中有大量嵌套对象

**解决**: 减少不必要的变量引用，直接使用固定值。

2. **环境数量过多**: 数据库查询慢

**解决**: 清理不用的环境。

3. **数据库性能问题**

**解决**:
- 为 `is_active` 字段添加索引
- 使用数据库连接池
- 考虑使用 PostgreSQL 替代 SQLite

### 问题6: 日志中找不到变量注入信息

**症状**: 无法确认变量是否被正确注入。

**解决**: 启用调试日志：

```bash
# 设置环境变量
export LOG_LEVEL=debug

# 或在配置文件中
# config.toml
[logging]
level = "debug"
```

查看日志：
```bash
tail -f /var/log/test-management-service.log | grep "VariableInjector"
```

### 获取帮助

如遇到其他问题，请：

1. **查看日志**: `/var/log/test-management-service.log`
2. **检查文档**: [API 文档](./API_DOCUMENTATION.md)
3. **提交 Issue**: [GitHub Issues](https://github.com/your-org/test-management-service/issues)
4. **联系支持**: support@example.com

---

## 总结

环境管理功能为测试管理平台提供了强大的多环境配置能力，通过变量注入机制实现了测试脚本与环境配置的解耦。

**核心价值**:
- ✅ 一套测试，多环境运行
- ✅ 配置集中管理
- ✅ CI/CD 友好
- ✅ 类型安全

**适用场景**:
- 多环境 API 测试
- CI/CD Pipeline 自动化
- 配置参数调优
- 回归测试

**已知限制**:
- ⚠️ 无多租户/多项目隔离
- ⚠️ 不支持并发多环境执行
- 🔒 敏感信息明文存储

**未来增强**:
- 多租户架构
- 并行执行支持
- 变量加密
- 配置版本控制

---

**文档版本**: 2.0
**贡献者**: AI Assistant
**反馈**: 如有问题或建议，请提交 Issue

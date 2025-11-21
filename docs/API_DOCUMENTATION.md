# 测试管理服务 API 文档

**版本**: 2.0
**基础 URL**: `http://localhost:8080/api/v2`
**最后更新**: 2025-11-21
**协议**: HTTP/REST + WebSocket

---

## 目录

1. [概述](#概述)
2. [认证](#认证)
3. [测试案例 API](#测试案例-api)
4. [测试分组 API](#测试分组-api)
5. [工作流 API](#工作流-api-新增)
6. [环境管理 API](#环境管理-api-新增)
7. [测试执行 API](#测试执行-api)
8. [测试结果 API](#测试结果-api)
9. [WebSocket API](#websocket-api-新增)
10. [数据模型](#数据模型)
11. [错误码](#错误码)

---

## 概述

测试管理服务提供完整的测试案例管理、执行和监控能力，支持以下测试类型：
- **HTTP 测试**: RESTful API 测试
- **命令测试**: Shell 命令执行测试
- **工作流测试**: 多步骤编排测试（新增）

### 工作流集成模式

本服务支持三种工作流集成模式：

| 模式 | 使用场景 | API 端点 |
|------|---------|---------|
| **Mode 1** | 测试案例引用独立工作流 | `POST /tests` (workflowId) |
| **Mode 2** | 测试案例内嵌工作流定义 | `POST /tests` (workflowDef) |
| **Mode 3** | 工作流引用测试案例 | `POST /workflows` (type=test-case) |

---

## 认证

当前版本暂不需要认证。生产环境建议添加以下认证方式：
- Bearer Token (JWT)
- API Key
- OAuth 2.0

---

## 测试案例 API

### 1. 创建测试案例

**端点**: `POST /tests`

**请求体**:
```json
{
  "testId": "test-001",
  "groupId": "group-001",
  "name": "用户登录测试",
  "type": "http|command|workflow",
  "priority": "P0|P1|P2",
  "status": "active|inactive",
  "objective": "验证用户登录功能",
  "timeout": 300,

  // HTTP 测试配置（type=http 时）
  "http": {
    "method": "POST",
    "path": "/api/login",
    "headers": {"Content-Type": "application/json"},
    "body": {"username": "test", "password": "123456"}
  },

  // 命令测试配置（type=command 时）
  "command": {
    "cmd": "curl",
    "args": ["-X", "POST", "http://api.example.com"],
    "timeout": 30
  },

  // 工作流配置（type=workflow 时）- Mode 1
  "workflowId": "workflow-login",

  // 或 Mode 2 内嵌工作流定义
  "workflowDef": {
    "name": "登录流程",
    "steps": {
      "step1": {
        "id": "step1",
        "name": "登录请求",
        "type": "http",
        "config": {"method": "POST", "path": "/api/login"}
      }
    }
  },

  // 断言（可选）
  "assertions": [
    {
      "type": "status_code",
      "expected": 200
    },
    {
      "type": "json_path",
      "path": "$.token",
      "operator": "exists"
    }
  ],

  // 生命周期钩子（可选）
  "setupHooks": [],
  "teardownHooks": [],

  // 标签（可选）
  "tags": ["smoke", "regression"]
}
```

**响应**: `201 Created`
```json
{
  "id": 1,
  "testId": "test-001",
  "groupId": "group-001",
  "name": "用户登录测试",
  "type": "http",
  "priority": "P0",
  "status": "active",
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

**注意事项**:
- 工作流测试必须提供 `workflowId` 或 `workflowDef` 之一
- `workflowId` 和 `workflowDef` 不能同时存在
- `testId` 必须全局唯一

---

### 2. 更新测试案例

**端点**: `PUT /tests/:id`

**路径参数**:
- `id` (string): 测试案例 ID (testId)

**请求体**: 与创建接口相同，所有字段可选

**响应**: `200 OK`

---

### 3. 删除测试案例

**端点**: `DELETE /tests/:id`

**响应**: `200 OK`
```json
{
  "message": "test case deleted"
}
```

---

### 4. 获取测试案例详情

**端点**: `GET /tests/:id`

**响应**: `200 OK`
```json
{
  "id": 1,
  "testId": "test-001",
  "groupId": "group-001",
  "name": "用户登录测试",
  "type": "workflow",
  "workflowId": "workflow-login",
  "priority": "P0",
  "status": "active",
  "objective": "验证用户登录功能",
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

---

### 5. 列出测试案例

**端点**: `GET /tests`

**查询参数**:
- `limit` (integer, 默认 20): 每页数量
- `offset` (integer, 默认 0): 偏移量

**响应**: `200 OK`
```json
{
  "data": [
    {
      "id": 1,
      "testId": "test-001",
      "name": "用户登录测试",
      "type": "workflow",
      "priority": "P0",
      "status": "active"
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

---

### 6. 搜索测试案例

**端点**: `GET /tests/search`

**查询参数**:
- `q` (string, 必需): 搜索关键词

**响应**: `200 OK` - 返回匹配的测试案例数组

---

### 7. 获取测试统计

**端点**: `GET /tests/stats`

**响应**: `200 OK`
```json
{
  "success": true,
  "data": {
    "total": 150,
    "active": 120,
    "p0": 30,
    "p1": 60,
    "p2": 60
  }
}
```

---

## 测试分组 API

### 1. 创建测试分组

**端点**: `POST /groups`

**请求体**:
```json
{
  "groupId": "group-001",
  "name": "用户管理模块",
  "parentId": null,
  "description": "用户相关的测试用例"
}
```

**响应**: `201 Created`

---

### 2. 获取分组树

**端点**: `GET /groups/tree`

**响应**: `200 OK` - 返回树形结构的分组列表

---

### 3. 获取测试树

**端点**: `GET /test-tree`

**响应**: `200 OK` - 返回包含分组和测试案例的完整树

---

## 工作流 API (新增)

### 1. 创建工作流

**端点**: `POST /workflows`

**请求体**:
```json
{
  "workflowId": "workflow-login",
  "name": "用户登录流程",
  "version": "1.0",
  "description": "完整的用户登录验证流程",
  "isTestCase": true,
  "createdBy": "admin",
  "definition": {
    "name": "login-workflow",
    "version": "1.0",
    "variables": {
      "baseUrl": "http://api.example.com",
      "username": "testuser"
    },
    "steps": {
      "step1": {
        "id": "step1",
        "name": "登录请求",
        "type": "http",
        "config": {
          "method": "POST",
          "path": "/api/login",
          "headers": {"Content-Type": "application/json"},
          "body": {
            "username": "{{username}}",
            "password": "test123"
          }
        },
        "output": {
          "token": "token"
        }
      },
      "step2": {
        "id": "step2",
        "name": "验证令牌",
        "type": "http",
        "dependsOn": ["step1"],
        "config": {
          "method": "GET",
          "path": "/api/user/profile",
          "headers": {
            "Authorization": "Bearer {{token}}"
          }
        },
        "retry": {
          "maxAttempts": 3,
          "interval": 1000
        },
        "onError": "abort"
      },
      "step3": {
        "id": "step3",
        "name": "执行登录测试",
        "type": "test-case",
        "dependsOn": ["step2"],
        "config": {
          "testId": "test-login-validation"
        },
        "when": "{{token}}"
      }
    }
  }
}
```

**步骤类型说明**:
- `http`: HTTP 请求步骤
- `command`: Shell 命令步骤
- `test-case`: 引用测试案例步骤（Mode 3）

**响应**: `201 Created`
```json
{
  "id": 1,
  "workflowId": "workflow-login",
  "name": "用户登录流程",
  "version": "1.0",
  "isTestCase": true,
  "createdAt": "2025-11-21T10:00:00Z"
}
```

---

### 2. 更新工作流

**端点**: `PUT /workflows/:id`

**路径参数**:
- `id` (string): 工作流 ID (workflowId)

**请求体**:
```json
{
  "name": "更新的工作流名称",
  "version": "2.0",
  "description": "更新的描述",
  "definition": { /* 新的工作流定义 */ }
}
```

**响应**: `200 OK`

---

### 3. 删除工作流

**端点**: `DELETE /workflows/:id`

**响应**: `200 OK`
```json
{
  "message": "workflow deleted"
}
```

---

### 4. 获取工作流详情

**端点**: `GET /workflows/:id`

**响应**: `200 OK`
```json
{
  "id": 1,
  "workflowId": "workflow-login",
  "name": "用户登录流程",
  "version": "1.0",
  "description": "完整的用户登录验证流程",
  "definition": { /* 工作流定义 */ },
  "isTestCase": true,
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

---

### 5. 列出工作流

**端点**: `GET /workflows`

**查询参数**:
- `limit` (integer, 默认 20): 每页数量
- `offset` (integer, 默认 0): 偏移量
- `isTestCase` (boolean, 可选): 过滤是否被测试案例引用

**响应**: `200 OK`
```json
{
  "data": [
    {
      "id": 1,
      "workflowId": "workflow-login",
      "name": "用户登录流程",
      "version": "1.0",
      "isTestCase": true
    }
  ],
  "total": 50,
  "limit": 20,
  "offset": 0
}
```

---

### 6. 执行工作流

**端点**: `POST /workflows/:id/execute`

**请求体** (可选):
```json
{
  "variables": {
    "username": "testuser",
    "environment": "staging"
  }
}
```

**响应**: `200 OK`
```json
{
  "id": 1,
  "runId": "run-abc-123",
  "workflowId": "workflow-login",
  "status": "running|success|failed|cancelled",
  "startTime": "2025-11-21T10:00:00Z",
  "endTime": "2025-11-21T10:00:30Z",
  "duration": 30000,
  "context": {
    "variables": { /* 变量值 */ },
    "outputs": { /* 步骤输出 */ }
  },
  "error": null
}
```

---

### 7. 获取工作流执行详情

**端点**: `GET /workflows/runs/:runId`

**响应**: `200 OK` - 返回执行记录详情

---

### 8. 列出工作流执行历史

**端点**: `GET /workflows/:id/runs`

**查询参数**:
- `limit` (integer, 默认 20)
- `offset` (integer, 默认 0)

**响应**: `200 OK`
```json
{
  "data": [
    {
      "runId": "run-abc-123",
      "status": "success",
      "startTime": "2025-11-21T10:00:00Z",
      "duration": 30000
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

---

### 9. 获取步骤执行记录

**端点**: `GET /workflows/runs/:runId/steps`

**响应**: `200 OK`
```json
[
  {
    "id": 1,
    "runId": "run-abc-123",
    "stepId": "step1",
    "stepName": "登录请求",
    "status": "success",
    "startTime": "2025-11-21T10:00:00Z",
    "endTime": "2025-11-21T10:00:10Z",
    "duration": 10000,
    "inputData": { /* 输入数据快照 */ },
    "outputData": { /* 输出数据快照 */ },
    "error": null
  }
]
```

---

### 10. 获取步骤日志

**端点**: `GET /workflows/runs/:runId/logs`

**查询参数**:
- `stepId` (string, 可选): 过滤特定步骤
- `level` (string, 可选): 过滤日志级别 (debug|info|warn|error)

**响应**: `200 OK`
```json
[
  {
    "id": 1,
    "runId": "run-abc-123",
    "stepId": "step1",
    "level": "info",
    "message": "开始执行 HTTP 请求",
    "timestamp": "2025-11-21T10:00:01Z"
  },
  {
    "id": 2,
    "runId": "run-abc-123",
    "stepId": "step1",
    "level": "info",
    "message": "HTTP 请求成功，状态码: 200",
    "timestamp": "2025-11-21T10:00:05Z"
  }
]
```

---

### 11. 获取工作流关联的测试案例

**端点**: `GET /workflows/:id/test-cases`

**响应**: `200 OK` - 返回引用此工作流的测试案例列表

---

## 环境管理 API (新增)

环境管理API允许您管理多个测试环境（如Dev、Staging、Prod），并通过变量注入机制动态配置测试执行。

### 核心概念

- **环境 (Environment)**: 一组配置变量的集合，如 `BASE_URL`、`API_KEY` 等
- **激活状态**: 同一时间只能有一个环境处于激活状态
- **变量注入**: 使用 `{{VARIABLE_NAME}}` 语法在测试配置中引用环境变量
- **变量优先级**: Environment < Workflow < TestCase (后者覆盖前者)

### 1. 创建环境

**端点**: `POST /environments`

**请求体**:
```json
{
  "envId": "dev",
  "name": "Development Environment",
  "description": "开发环境配置",
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-12345",
    "TIMEOUT": 30,
    "DEBUG": true
  }
}
```

**响应**: `201 Created`
```json
{
  "id": 1,
  "envId": "dev",
  "name": "Development Environment",
  "description": "开发环境配置",
  "isActive": false,
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-12345",
    "TIMEOUT": 30,
    "DEBUG": true
  },
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

**字段说明**:
- `envId` (必填): 环境唯一标识符
- `name` (必填): 环境名称
- `description` (可选): 环境描述
- `variables` (可选): 环境变量键值对

---

### 2. 列出所有环境

**端点**: `GET /environments`

**查询参数**:
- `limit` (可选): 每页数量，默认10
- `offset` (可选): 偏移量，默认0

**响应**: `200 OK`
```json
{
  "data": [
    {
      "id": 1,
      "envId": "dev",
      "name": "Development",
      "description": "开发环境",
      "isActive": true,
      "variables": {...},
      "createdAt": "2025-11-21T10:00:00Z",
      "updatedAt": "2025-11-21T10:00:00Z"
    },
    {
      "id": 2,
      "envId": "staging",
      "name": "Staging",
      "description": "预发布环境",
      "isActive": false,
      "variables": {...},
      "createdAt": "2025-11-21T10:05:00Z",
      "updatedAt": "2025-11-21T10:05:00Z"
    }
  ],
  "total": 2,
  "limit": 10,
  "offset": 0
}
```

---

### 3. 获取环境详情

**端点**: `GET /environments/:id`

**路径参数**:
- `id`: 环境ID (envId)

**响应**: `200 OK`
```json
{
  "id": 1,
  "envId": "dev",
  "name": "Development",
  "description": "开发环境",
  "isActive": true,
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-12345",
    "TIMEOUT": 30,
    "DEBUG": true
  },
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

---

### 4. 更新环境

**端点**: `PUT /environments/:id`

**路径参数**:
- `id`: 环境ID (envId)

**请求体**:
```json
{
  "name": "Development (Updated)",
  "description": "更新后的开发环境",
  "variables": {
    "BASE_URL": "http://localhost:4000",
    "API_KEY": "new-dev-key",
    "TIMEOUT": 60,
    "DEBUG": false
  }
}
```

**响应**: `200 OK` - 返回更新后的环境对象

**注意**: 更新变量时会完全替换现有变量集，请确保包含所有需要的变量。

---

### 5. 删除环境

**端点**: `DELETE /environments/:id`

**路径参数**:
- `id`: 环境ID (envId)

**响应**: `200 OK`
```json
{
  "message": "environment deleted",
  "envId": "dev"
}
```

**约束**:
- ❌ 不能删除当前激活的环境
- ✅ 只能删除非激活状态的环境

---

### 6. 获取当前激活的环境

**端点**: `GET /environments/active`

**响应**: `200 OK`
```json
{
  "id": 1,
  "envId": "dev",
  "name": "Development",
  "description": "开发环境",
  "isActive": true,
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-12345"
  },
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

**错误响应**: `404 Not Found`
```json
{
  "error": "no active environment found"
}
```

---

### 7. 激活环境

**端点**: `POST /environments/:id/activate`

**路径参数**:
- `id`: 环境ID (envId)

**响应**: `200 OK`
```json
{
  "message": "environment activated",
  "envId": "dev"
}
```

**行为**:
- ✅ 将指定环境设置为激活状态
- ✅ 自动停用之前激活的环境
- ✅ 事务安全，确保同一时间只有一个环境激活

---

### 8. 获取环境的所有变量

**端点**: `GET /environments/:id/variables`

**路径参数**:
- `id`: 环境ID (envId)

**响应**: `200 OK`
```json
{
  "BASE_URL": "http://localhost:3000",
  "API_KEY": "dev-key-12345",
  "TIMEOUT": 30,
  "DEBUG": true
}
```

---

### 9. 获取单个环境变量

**端点**: `GET /environments/:id/variables/:key`

**路径参数**:
- `id`: 环境ID (envId)
- `key`: 变量名

**响应**: `200 OK`
```json
{
  "key": "API_KEY",
  "value": "dev-key-12345"
}
```

**错误响应**: `500 Internal Server Error`
```json
{
  "error": "variable 'UNKNOWN_VAR' not found in environment 'dev'"
}
```

---

### 10. 设置/更新环境变量

**端点**: `PUT /environments/:id/variables/:key`

**路径参数**:
- `id`: 环境ID (envId)
- `key`: 变量名

**请求体**:
```json
{
  "value": "new-api-key-67890"
}
```

**响应**: `200 OK`
```json
{
  "message": "variable updated",
  "key": "API_KEY",
  "value": "new-api-key-67890"
}
```

**说明**:
- 如果变量不存在，则创建新变量
- 如果变量已存在，则更新其值
- 支持任意JSON类型的值（字符串、数字、布尔、对象、数组）

---

### 11. 删除环境变量

**端点**: `DELETE /environments/:id/variables/:key`

**路径参数**:
- `id`: 环境ID (envId)
- `key`: 变量名

**响应**: `200 OK`
```json
{
  "message": "variable deleted",
  "key": "API_KEY"
}
```

---

### 变量注入示例

#### 在HTTP测试中使用环境变量

```json
{
  "testId": "api-test-001",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "{{BASE_URL}}/api/login",
    "headers": {
      "Authorization": "Bearer {{API_KEY}}",
      "Content-Type": "application/json"
    },
    "body": {
      "timeout": "{{TIMEOUT}}"
    }
  }
}
```

当激活Dev环境时，变量会自动替换为：
```json
{
  "method": "POST",
  "path": "http://localhost:3000/api/login",
  "headers": {
    "Authorization": "Bearer dev-key-12345",
    "Content-Type": "application/json"
  },
  "body": {
    "timeout": 30
  }
}
```

#### 在Workflow中使用环境变量

```json
{
  "workflowId": "user-flow",
  "definition": {
    "variables": {
      "USER_ID": "{{DEFAULT_USER_ID}}"
    },
    "steps": {
      "login": {
        "type": "http",
        "config": {
          "method": "POST",
          "path": "{{BASE_URL}}/api/login"
        }
      }
    }
  }
}
```

#### 变量优先级示例

假设有以下配置：

**Environment (dev)**:
```json
{
  "BASE_URL": "http://localhost:3000",
  "TIMEOUT": 30
}
```

**Workflow Variables**:
```json
{
  "TIMEOUT": 60
}
```

**最终注入结果**:
- `BASE_URL` = `"http://localhost:3000"` (来自环境)
- `TIMEOUT` = `60` (Workflow覆盖环境)

---

### CI/CD集成示例

#### GitLab CI 集成

```yaml
# .gitlab-ci.yml
test:dev:
  stage: test
  only:
    - develop
  script:
    - curl -X POST $TEST_PLATFORM/api/v2/environments/dev/activate
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute

test:prod:
  stage: test
  only:
    - tags
  script:
    - curl -X POST $TEST_PLATFORM/api/v2/environments/prod/activate
    - curl -X POST $TEST_PLATFORM/api/v2/workflows/smoke-test/execute
```

#### Jenkins Pipeline 集成

```groovy
pipeline {
    agent any

    stages {
        stage('Set Environment') {
            steps {
                script {
                    def envName = env.BRANCH_NAME == 'main' ? 'prod' : 'staging'
                    sh "curl -X POST http://test-platform/api/v2/environments/${envName}/activate"
                }
            }
        }

        stage('Run Tests') {
            steps {
                sh 'curl -X POST http://test-platform/api/v2/workflows/regression-suite/execute'
            }
        }
    }
}
```

---

## 测试执行 API

### 1. 执行单个测试

**端点**: `POST /tests/:id/execute`

**响应**: `200 OK`
```json
{
  "id": 1,
  "testId": "test-001",
  "status": "passed|failed|error|skipped",
  "startTime": "2025-11-21T10:00:00Z",
  "endTime": "2025-11-21T10:00:05Z",
  "duration": 5000,
  "error": null,
  "failures": [],
  "response": {
    // HTTP 测试响应
    "statusCode": 200,
    "body": { /* 响应体 */ },

    // 或工作流测试响应
    "workflowRunId": "run-abc-123",
    "totalSteps": 3,
    "completedSteps": 3,
    "failedSteps": 0,
    "stepExecutions": [ /* 步骤详情 */ ]
  }
}
```

**工作流测试特殊字段**:
- `response.workflowRunId`: 工作流执行记录 ID
- `response.totalSteps`: 总步骤数
- `response.completedSteps`: 完成步骤数
- `response.failedSteps`: 失败步骤数
- `response.stepExecutions`: 步骤执行详情数组

---

### 2. 执行测试分组

**端点**: `POST /groups/:id/execute`

**响应**: `200 OK` - 返回批量执行结果

---

## 测试结果 API

### 1. 获取测试结果

**端点**: `GET /results/:id`

**路径参数**:
- `id` (integer): 测试结果 ID

**响应**: `200 OK` - 返回测试结果详情

---

### 2. 获取测试历史

**端点**: `GET /tests/:id/history`

**查询参数**:
- `limit` (integer, 默认 10): 返回最近 N 次执行

**响应**: `200 OK` - 返回历史执行记录数组

---

### 3. 获取测试批次

**端点**: `GET /runs/:id`

**响应**: `200 OK` - 返回批次执行详情

---

### 4. 列出测试批次

**端点**: `GET /runs`

**查询参数**:
- `limit` (integer, 默认 20)
- `offset` (integer, 默认 0)

**响应**: `200 OK`

---

## WebSocket API (新增)

### 实时监控工作流执行

**端点**: `ws://localhost:8080/api/v2/workflows/runs/:runId/stream`

**协议**: WebSocket

**连接方式**:
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v2/workflows/runs/run-abc-123/stream');

ws.onopen = () => {
  console.log('已连接到工作流实时监控');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('收到消息:', message);
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('连接已关闭');
};
```

**消息格式**:
```json
{
  "runId": "run-abc-123",
  "type": "step_start|step_complete|step_log|variable_change",
  "payload": { /* 具体数据 */ }
}
```

**消息类型说明**:

#### 1. step_start - 步骤开始
```json
{
  "runId": "run-abc-123",
  "type": "step_start",
  "payload": {
    "stepId": "step1",
    "stepName": "登录请求"
  }
}
```

#### 2. step_complete - 步骤完成
```json
{
  "runId": "run-abc-123",
  "type": "step_complete",
  "payload": {
    "stepId": "step1",
    "stepName": "登录请求",
    "status": "success|failed",
    "duration": 10000
  }
}
```

#### 3. step_log - 步骤日志
```json
{
  "runId": "run-abc-123",
  "type": "step_log",
  "payload": {
    "stepId": "step1",
    "level": "debug|info|warn|error",
    "message": "正在执行 HTTP 请求...",
    "timestamp": "2025-11-21T10:00:01Z"
  }
}
```

#### 4. variable_change - 变量变更
```json
{
  "runId": "run-abc-123",
  "type": "variable_change",
  "payload": {
    "stepId": "step1",
    "varName": "token",
    "oldValue": null,
    "newValue": "eyJhbGci...",
    "changeType": "create|update|delete"
  }
}
```

**心跳机制**:
- 客户端每 54 秒收到一次 Ping 消息
- 60 秒无响应则连接超时
- 建议客户端实现自动重连

---

## 数据模型

### TestCase
```json
{
  "id": 1,
  "testId": "test-001",
  "groupId": "group-001",
  "name": "测试名称",
  "type": "http|command|workflow",
  "priority": "P0|P1|P2",
  "status": "active|inactive",
  "objective": "测试目标",
  "timeout": 300,

  // 工作流集成字段（新增）
  "workflowId": "workflow-login",        // Mode 1
  "workflowDef": { /* 工作流定义 */ },   // Mode 2

  // 配置字段
  "http": { /* HTTP 配置 */ },
  "command": { /* 命令配置 */ },
  "assertions": [ /* 断言列表 */ ],
  "setupHooks": [ /* 前置钩子 */ ],
  "teardownHooks": [ /* 后置钩子 */ ],
  "tags": [ /* 标签 */ ],

  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

### Workflow (新增)
```json
{
  "id": 1,
  "workflowId": "workflow-login",
  "name": "工作流名称",
  "version": "1.0",
  "description": "工作流描述",
  "definition": {
    "name": "workflow-name",
    "version": "1.0",
    "variables": { /* 全局变量 */ },
    "steps": { /* 步骤定义 */ }
  },
  "isTestCase": true,
  "createdBy": "admin",
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z"
}
```

### WorkflowRun (新增)
```json
{
  "id": 1,
  "runId": "run-abc-123",
  "workflowId": "workflow-login",
  "status": "running|success|failed|cancelled",
  "startTime": "2025-11-21T10:00:00Z",
  "endTime": "2025-11-21T10:00:30Z",
  "duration": 30000,
  "context": {
    "variables": { /* 变量状态 */ },
    "outputs": { /* 步骤输出 */ }
  },
  "error": "错误信息（如果失败）",
  "createdAt": "2025-11-21T10:00:00Z"
}
```

### WorkflowStepExecution (新增)
```json
{
  "id": 1,
  "runId": "run-abc-123",
  "stepId": "step1",
  "stepName": "步骤名称",
  "status": "pending|running|success|failed|skipped",
  "startTime": "2025-11-21T10:00:00Z",
  "endTime": "2025-11-21T10:00:10Z",
  "duration": 10000,
  "inputData": { /* 输入数据快照 */ },
  "outputData": { /* 输出数据快照 */ },
  "error": null,
  "createdAt": "2025-11-21T10:00:00Z"
}
```

### Environment (新增)
```json
{
  "id": 1,
  "envId": "dev",
  "name": "Development Environment",
  "description": "开发环境配置",
  "isActive": true,
  "variables": {
    "BASE_URL": "http://localhost:3000",
    "API_KEY": "dev-key-12345",
    "TIMEOUT": 30,
    "DEBUG": true
  },
  "createdAt": "2025-11-21T10:00:00Z",
  "updatedAt": "2025-11-21T10:00:00Z",
  "deletedAt": null
}
```

**字段说明**:
- `envId`: 环境唯一标识符
- `name`: 环境名称
- `description`: 环境描述
- `isActive`: 是否为当前激活环境（同一时间只能有一个为true）
- `variables`: 环境变量键值对（JSONB类型）

---

## 错误码

### HTTP 状态码

| 状态码 | 含义 | 说明 |
|--------|------|------|
| 200 | OK | 请求成功 |
| 201 | Created | 资源创建成功 |
| 400 | Bad Request | 请求参数错误 |
| 404 | Not Found | 资源不存在 |
| 500 | Internal Server Error | 服务器内部错误 |

### 错误响应格式

```json
{
  "error": "错误描述信息"
}
```

### 常见错误

#### 1. 工作流测试配置错误
```json
{
  "error": "workflow test must have either workflowId or workflowDef"
}
```

#### 2. 工作流不存在
```json
{
  "error": "workflow not found: workflow-xxx"
}
```

#### 3. 循环依赖错误
```json
{
  "error": "workflow validation failed: workflow contains cyclic dependency involving step 'step1'"
}
```

#### 4. 测试案例不存在
```json
{
  "error": "test case not found: test-xxx"
}
```

---

## 使用示例

### 示例 1: 创建并执行 Mode 1 工作流测试

```bash
# 1. 创建独立工作流
curl -X POST http://localhost:8080/api/v2/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "workflow-login",
    "name": "用户登录流程",
    "definition": {
      "steps": {
        "step1": {
          "id": "step1",
          "name": "登录",
          "type": "http",
          "config": {"method": "POST", "path": "/api/login"}
        }
      }
    }
  }'

# 2. 创建测试案例引用工作流
curl -X POST http://localhost:8080/api/v2/tests \
  -H "Content-Type: application/json" \
  -d '{
    "testId": "test-login",
    "groupId": "group-1",
    "name": "登录测试",
    "type": "workflow",
    "workflowId": "workflow-login"
  }'

# 3. 执行测试
curl -X POST http://localhost:8080/api/v2/tests/test-login/execute

# 4. 查看执行结果
curl http://localhost:8080/api/v2/tests/test-login/history?limit=1
```

### 示例 2: 创建并执行 Mode 2 内嵌工作流测试

```bash
# 创建带内嵌工作流的测试
curl -X POST http://localhost:8080/api/v2/tests \
  -H "Content-Type: application/json" \
  -d '{
    "testId": "test-checkout",
    "groupId": "group-1",
    "name": "结账流程测试",
    "type": "workflow",
    "workflowDef": {
      "steps": {
        "step1": {
          "id": "step1",
          "name": "添加商品",
          "type": "http",
          "config": {"method": "POST", "path": "/api/cart"}
        },
        "step2": {
          "id": "step2",
          "name": "结账",
          "type": "http",
          "dependsOn": ["step1"],
          "config": {"method": "POST", "path": "/api/checkout"}
        }
      }
    }
  }'

# 执行测试
curl -X POST http://localhost:8080/api/v2/tests/test-checkout/execute
```

### 示例 3: WebSocket 实时监控

```javascript
// 浏览器或 Node.js 客户端
const runId = 'run-abc-123';
const ws = new WebSocket(`ws://localhost:8080/api/v2/workflows/runs/${runId}/stream`);

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  switch(msg.type) {
    case 'step_start':
      console.log(`步骤开始: ${msg.payload.stepName}`);
      break;
    case 'step_log':
      console.log(`[${msg.payload.level}] ${msg.payload.message}`);
      break;
    case 'step_complete':
      console.log(`步骤完成: ${msg.payload.stepName} (${msg.payload.duration}ms)`);
      break;
  }
};
```

---

## 版本历史

### v2.0 (2025-11-21)
- ✨ 新增工作流 API (11 个端点)
- ✨ 新增 WebSocket 实时监控
- ✨ 测试案例支持工作流类型
- ✨ 支持三种工作流集成模式
- 📝 完善 API 文档

### v1.0
- 基础测试案例管理
- HTTP 和命令测试支持
- 测试分组管理
- 测试执行和结果查询

---

**文档维护**: 请在每次 API 变更后更新此文档
**反馈**: 如发现文档错误或需要补充，请提交 Issue

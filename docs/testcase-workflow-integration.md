# 测试案例与工作流集成设计

**文档版本**: 1.0
**创建日期**: 2025-11-21
**作者**: 测试框架团队

---

## 目录

1. [设计目标](#设计目标)
2. [核心设计原则](#核心设计原则)
3. [三种集成模式](#三种集成模式)
4. [数据模型设计](#数据模型设计)
5. [API设计](#api设计)
6. [执行引擎改造](#执行引擎改造)
7. [前端UI设计](#前端ui设计)
8. [迁移方案](#迁移方案)

---

## 设计目标

### 核心问题

**用户需求**: "测试案例的配置要能兼容工作流，测试管理工具不仅管理案例，案例还能真正的实施起来"

**设计目标**:
1. ✅ 测试案例可以是工作流（复杂的多步骤测试）
2. ✅ 工作流可以引用测试案例（复用已有测试）
3. ✅ 统一的测试管理界面（列表、执行、结果、历史）
4. ✅ 统一的执行引擎（无论是单一测试还是工作流）
5. ✅ 保持向后兼容（不影响现有测试案例）

### 业务价值

| 角色 | 价值 |
|------|------|
| **QA工程师** | 在测试案例列表中直接管理复杂的端到端测试流程 |
| **自动化工程师** | 可以将工作流封装为测试案例，复用在不同场景 |
| **测试负责人** | 统一的测试视图，无需区分"测试"和"工作流" |
| **开发工程师** | 更真实的业务流程测试，发现集成问题 |

---

## 核心设计原则

### 原则1：测试案例是统一入口

**理念**: 用户看到的是"测试案例"，而不是"测试案例"和"工作流"两个概念。

```
用户视角:
┌────────────────────────────────┐
│  测试案例列表                  │
│  - HTTP测试：获取用户信息      │
│  - 命令测试：检查服务状态      │
│  - 工作流测试：用户注册流程    │  ← 工作流也是一种测试
│  - 数据库测试：查询订单        │
└────────────────────────────────┘
```

### 原则2：工作流是测试类型之一

**扩展Type字段**:

```go
type TestCase struct {
    Type string `json:"type"`
    // 可选值：
    // - "http"          单一HTTP请求
    // - "command"       命令执行
    // - "workflow"      多步骤工作流  ← 新增
    // - "database"      数据库操作
    // - "grpc"          gRPC调用
    // ...
}
```

### 原则3：双向引用支持

```
TestCase ←→ Workflow

模式1: TestCase 引用 Workflow
{
  "type": "workflow",
  "workflowId": "wf-123"  // 引用独立的工作流
}

模式2: TestCase 内嵌 Workflow
{
  "type": "workflow",
  "workflowDef": { ... }  // 内嵌工作流定义
}

模式3: Workflow 引用 TestCase
workflow:
  steps:
    - type: test-case
      testId: "test-123"  // 引用测试案例作为步骤
```

### 原则4：执行引擎统一调度

**统一的执行入口**:

```go
// 执行任何测试案例（包括工作流测试）
result := executor.ExecuteTest(testCase)

// 内部根据Type路由到不同的执行器
switch testCase.Type {
case "http":
    return httpExecutor.Execute(testCase.HTTPConfig)
case "command":
    return commandExecutor.Execute(testCase.CommandConfig)
case "workflow":
    return workflowExecutor.Execute(testCase.WorkflowDef or testCase.WorkflowID)
}
```

---

## 三种集成模式

### 模式1: 测试案例引用工作流 (推荐用于复杂场景)

#### 使用场景

- 端到端的业务流程测试
- 需要复杂编排的测试场景
- 工作流需要独立管理和版本控制

#### 数据结构

**测试案例定义**:

```json
{
  "testId": "e2e-order-flow",
  "name": "完整订单流程测试",
  "type": "workflow",
  "priority": "P0",
  "tags": ["e2e", "critical", "order"],

  "workflowId": "workflow-order-v2.1",

  "timeout": 600,
  "assertions": [
    {
      "type": "workflow_success",
      "description": "工作流必须成功完成"
    },
    {
      "type": "step_count",
      "expected": 5,
      "description": "应该执行5个步骤"
    }
  ]
}
```

**引用的工作流定义** (独立存储在 `workflows` 表):

```yaml
workflow:
  id: workflow-order-v2.1
  name: 订单完整流程
  version: 2.1

  steps:
    - id: register
      name: 注册用户
      type: http
      http:
        method: POST
        path: /api/users/register
      output:
        userId: "response.data.id"

    - id: login
      name: 用户登录
      type: http
      dependsOn: [register]
      # ... 详细配置

    - id: create-order
      name: 创建订单
      type: http
      dependsOn: [login]
      # ...

    - id: payment
      name: 支付订单
      type: http
      dependsOn: [create-order]
      # ...

    - id: verify
      name: 验证订单状态
      type: database
      dependsOn: [payment]
      # ...
```

#### 优点

- 工作流可以独立管理（版本、修改、复用）
- 多个测试案例可以引用同一个工作流
- 清晰的职责分离（测试案例管理测试，工作流管理流程）

#### 执行流程

```
1. 用户点击执行测试 "e2e-order-flow"
   ↓
2. 系统识别 type=workflow，提取 workflowId
   ↓
3. 从 workflows 表加载工作流定义
   ↓
4. WorkflowExecutor 执行工作流
   ↓
5. 执行结果存储到 test_results 表
   ↓
6. 用户在测试结果列表中查看
```

---

### 模式2: 测试案例内嵌工作流 (推荐用于简单场景)

#### 使用场景

- 简单的多步骤测试（2-5步）
- 不需要复用的测试流程
- 快速创建测试，无需管理独立工作流

#### 数据结构

**测试案例定义** (工作流定义内嵌):

```json
{
  "testId": "login-and-query",
  "name": "登录后查询用户数据",
  "type": "workflow",
  "priority": "P1",
  "tags": ["auth", "integration"],

  "workflowDef": {
    "steps": [
      {
        "id": "login",
        "name": "用户登录",
        "type": "http",
        "http": {
          "method": "POST",
          "path": "/api/auth/login",
          "body": {
            "username": "testuser",
            "password": "testpass"
          }
        },
        "assertions": [
          {"type": "status_code", "expected": 200}
        ],
        "output": {
          "token": "response.data.token"
        }
      },
      {
        "id": "query-user",
        "name": "查询用户信息",
        "type": "http",
        "dependsOn": ["login"],
        "http": {
          "method": "GET",
          "path": "/api/users/me",
          "headers": {
            "Authorization": "Bearer ${login.token}"
          }
        },
        "assertions": [
          {"type": "status_code", "expected": 200},
          {"type": "json_path", "path": "data.username", "expected": "testuser"}
        ]
      }
    ]
  },

  "timeout": 60
}
```

#### 优点

- 测试定义完全自包含
- 创建简单，无需额外管理工作流
- 适合一次性的测试场景

#### 缺点

- 无法复用工作流定义
- 修改需要编辑测试案例
- 不适合复杂的工作流

#### 执行流程

```
1. 用户点击执行测试 "login-and-query"
   ↓
2. 系统识别 type=workflow，提取 workflowDef
   ↓
3. WorkflowExecutor 直接执行内嵌的工作流定义
   ↓
4. 执行结果存储到 test_results 表
```

---

### 模式3: 工作流引用测试案例 (复用已有测试)

#### 使用场景

- 在工作流中复用已有的测试案例
- 组合多个独立测试形成流程
- 测试案例需要独立维护

#### 工作流定义

```yaml
workflow:
  id: integration-test-flow
  name: 集成测试流程

  steps:
    # 引用已有的测试案例
    - id: test-user-create
      type: test-case
      testId: "api-user-create"  # 引用测试案例
      name: 执行用户创建测试
      output:
        userId: "response.data.id"

    - id: test-user-login
      type: test-case
      testId: "api-user-login"
      dependsOn: [test-user-create]
      input:
        username: "${test-user-create.username}"
      output:
        token: "response.data.token"

    - id: test-order-create
      type: test-case
      testId: "api-order-create"
      dependsOn: [test-user-login]
      input:
        authToken: "${test-user-login.token}"

    # 混合使用：直接定义的HTTP步骤
    - id: cleanup
      type: http
      dependsOn: [test-order-create]
      http:
        method: DELETE
        path: "/api/users/${test-user-create.userId}"
```

#### 引用的测试案例

**测试案例1: api-user-create**

```json
{
  "testId": "api-user-create",
  "name": "创建用户API测试",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "/api/users",
    "body": {
      "username": "{{username}}",
      "email": "{{email}}"
    }
  },
  "assertions": [
    {"type": "status_code", "expected": 201},
    {"type": "json_path", "path": "data.id", "exists": true}
  ]
}
```

**测试案例2: api-user-login**

```json
{
  "testId": "api-user-login",
  "name": "用户登录API测试",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "/api/auth/login",
    "body": {
      "username": "{{username}}",
      "password": "{{password}}"
    }
  },
  "assertions": [
    {"type": "status_code", "expected": 200},
    {"type": "json_path", "path": "data.token", "exists": true}
  ]
}
```

#### 优点

- 测试案例独立管理，可以单独执行
- 工作流复用已有的测试逻辑
- 测试案例修改自动反映到工作流中

#### Action实现

```go
// TestCaseAction - 执行测试案例的Action
type TestCaseAction struct {
    TestID string                 `json:"testId"`
    Input  map[string]interface{} `json:"input"`  // 输入变量
}

func (a *TestCaseAction) Execute(ctx *ActionContext) (*ActionResult, error) {
    // 1. 加载测试案例
    testCase, err := a.repo.GetTestCase(a.TestID)
    if err != nil {
        return nil, fmt.Errorf("failed to load test case: %w", err)
    }

    // 2. 应用输入变量（变量替换）
    testCaseWithInput := a.applyInputVariables(testCase, ctx, a.Input)

    // 3. 执行测试案例
    result, err := a.executor.ExecuteTest(testCaseWithInput)
    if err != nil {
        return nil, fmt.Errorf("test case execution failed: %w", err)
    }

    // 4. 返回结果
    return &ActionResult{
        Status: result.Status,
        Output: result.Data,
        Error:  result.Error,
    }, nil
}
```

---

## 数据模型设计

### 1. TestCase 模型扩展

```go
// TestCase 测试案例模型（扩展版本）
type TestCase struct {
    ID              uint           `gorm:"primaryKey" json:"id"`
    TestID          string         `gorm:"uniqueIndex;size:255;not null" json:"testId"`
    GroupID         string         `gorm:"size:255;not null;index" json:"groupId"`
    Name            string         `gorm:"size:255;not null" json:"name"`
    Type            string         `gorm:"size:50;not null;index" json:"type"`  // 扩展：增加 "workflow"
    Priority        string         `gorm:"size:10;index" json:"priority"`
    Status          string         `gorm:"size:50;default:'active';index" json:"status"`
    Objective       string         `gorm:"type:text" json:"objective,omitempty"`
    Timeout         int            `gorm:"default:300" json:"timeout,omitempty"`

    // === 新增字段：工作流支持 ===
    WorkflowID      string         `gorm:"size:255;index" json:"workflowId,omitempty"`  // 引用工作流ID（模式1）
    WorkflowDef     JSONB          `gorm:"type:text;column:workflow_def" json:"workflowDef,omitempty"`  // 内嵌工作流定义（模式2）

    // === 原有字段保持不变 ===
    Preconditions     JSONArray  `gorm:"type:text" json:"preconditions,omitempty"`
    Steps             JSONArray  `gorm:"type:text" json:"steps,omitempty"`
    HTTPConfig        JSONB      `gorm:"type:text;column:http_config" json:"http,omitempty"`
    CommandConfig     JSONB      `gorm:"type:text;column:command_config" json:"command,omitempty"`
    DatabaseConfig    JSONB      `gorm:"type:text;column:database_config" json:"database,omitempty"`
    // ... 其他配置字段 ...

    Assertions        JSONArray  `gorm:"type:text" json:"assertions,omitempty"`
    Tags              JSONArray  `gorm:"type:text" json:"tags,omitempty"`

    SetupHooks        JSONArray  `gorm:"type:text;column:setup_hooks" json:"setupHooks,omitempty"`
    TeardownHooks     JSONArray  `gorm:"type:text;column:teardown_hooks" json:"teardownHooks,omitempty"`

    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    // 关联
    Group    *TestGroup    `gorm:"foreignKey:GroupID;references:GroupID" json:"-"`
    Workflow *Workflow     `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`  // 新增
    Results  []TestResult  `gorm:"foreignKey:TestID;references:TestID" json:"-"`
}
```

### 2. Workflow 模型扩展

```go
// Workflow 工作流模型（扩展版本）
type Workflow struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    WorkflowID  string    `gorm:"uniqueIndex;size:255;not null" json:"workflowId"`
    Name        string    `gorm:"size:255;not null" json:"name"`
    Version     string    `gorm:"size:32" json:"version"`
    Description string    `gorm:"type:text" json:"description,omitempty"`
    Definition  JSONB     `gorm:"type:text;not null" json:"definition"`

    // === 新增字段：测试案例关联 ===
    IsTestCase  bool      `gorm:"default:false;index" json:"isTestCase"`  // 是否被测试案例引用

    CreatedAt   time.Time      `json:"createdAt"`
    UpdatedAt   time.Time      `json:"updatedAt"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
    CreatedBy   string         `gorm:"size:64" json:"createdBy,omitempty"`

    // 关联
    TestCases []TestCase    `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`  // 新增：反向关联
    Runs      []WorkflowRun `gorm:"foreignKey:WorkflowID;references:WorkflowID" json:"-"`
}
```

### 3. 数据库迁移脚本

```sql
-- 扩展 test_cases 表，添加工作流支持
ALTER TABLE test_cases
ADD COLUMN workflow_id VARCHAR(255) DEFAULT NULL,
ADD COLUMN workflow_def TEXT DEFAULT NULL;

-- 添加索引
CREATE INDEX idx_test_cases_workflow_id ON test_cases(workflow_id);

-- 扩展 workflows 表，添加测试案例关联标记
ALTER TABLE workflows
ADD COLUMN is_test_case BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_workflows_is_test_case ON workflows(is_test_case);
```

---

## API设计

### 1. 测试案例API扩展

#### 创建工作流类型的测试案例

**POST /api/v2/tests**

**模式1: 引用工作流**

```json
{
  "testId": "e2e-order-flow",
  "name": "订单完整流程测试",
  "type": "workflow",
  "workflowId": "workflow-order-v2.1",
  "priority": "P0",
  "tags": ["e2e", "order"],
  "timeout": 600,
  "assertions": [
    {"type": "workflow_success"}
  ]
}
```

**模式2: 内嵌工作流**

```json
{
  "testId": "login-query-test",
  "name": "登录并查询数据",
  "type": "workflow",
  "workflowDef": {
    "steps": [
      {
        "id": "login",
        "type": "http",
        "http": { "method": "POST", "path": "/api/auth/login" },
        "output": { "token": "response.data.token" }
      },
      {
        "id": "query",
        "type": "http",
        "dependsOn": ["login"],
        "http": {
          "method": "GET",
          "path": "/api/data",
          "headers": { "Authorization": "Bearer ${login.token}" }
        }
      }
    ]
  },
  "priority": "P1"
}
```

#### 执行工作流测试案例

**POST /api/v2/tests/:testId/execute**

```bash
POST /api/v2/tests/e2e-order-flow/execute
{
  "variables": {
    "env": "test",
    "baseUrl": "http://test-api.example.com"
  }
}
```

**响应**:

```json
{
  "runId": "run-20251121-001",
  "testId": "e2e-order-flow",
  "status": "running",
  "workflowRunId": "wf-run-123",  // 关联的工作流执行ID
  "startTime": "2025-11-21T10:30:00Z"
}
```

#### 查询工作流测试结果

**GET /api/v2/tests/:testId/results/:resultId**

**响应**:

```json
{
  "id": 123,
  "testId": "e2e-order-flow",
  "runId": "run-20251121-001",
  "status": "passed",
  "startTime": "2025-11-21T10:30:00Z",
  "endTime": "2025-11-21T10:35:30Z",
  "duration": 330000,

  "workflowRunId": "wf-run-123",

  "metrics": {
    "totalSteps": 5,
    "completedSteps": 5,
    "failedSteps": 0,
    "stepDetails": [
      {"stepId": "register", "status": "passed", "duration": 200},
      {"stepId": "login", "status": "passed", "duration": 150},
      {"stepId": "create-order", "status": "passed", "duration": 300},
      {"stepId": "payment", "status": "passed", "duration": 500},
      {"stepId": "verify", "status": "passed", "duration": 100}
    ]
  },

  "logs": [
    {"level": "info", "message": "Workflow execution started"},
    {"level": "info", "message": "Step 'register' completed successfully"},
    // ...
  ]
}
```

### 2. 工作流API扩展

#### 查询引用某个工作流的测试案例

**GET /api/v2/workflows/:workflowId/test-cases**

```bash
GET /api/v2/workflows/workflow-order-v2.1/test-cases
```

**响应**:

```json
{
  "workflowId": "workflow-order-v2.1",
  "testCases": [
    {
      "testId": "e2e-order-flow",
      "name": "订单完整流程测试",
      "priority": "P0",
      "status": "active"
    },
    {
      "testId": "smoke-order-test",
      "name": "订单冒烟测试",
      "priority": "P1",
      "status": "active"
    }
  ]
}
```

---

## 执行引擎改造

### 统一的测试执行器

```go
// UnifiedTestExecutor - 统一的测试执行器
type UnifiedTestExecutor struct {
    httpExecutor     *HTTPExecutor
    commandExecutor  *CommandExecutor
    workflowExecutor *WorkflowExecutor
    databaseExecutor *DatabaseExecutor
    // ...
}

func (e *UnifiedTestExecutor) ExecuteTest(testCase *models.TestCase) (*TestResult, error) {
    // 路由到不同的执行器
    switch testCase.Type {
    case "http":
        return e.executeHTTPTest(testCase)

    case "command":
        return e.executeCommandTest(testCase)

    case "workflow":
        return e.executeWorkflowTest(testCase)  // 新增

    case "database":
        return e.executeDatabaseTest(testCase)

    default:
        return nil, fmt.Errorf("unsupported test type: %s", testCase.Type)
    }
}

func (e *UnifiedTestExecutor) executeWorkflowTest(testCase *models.TestCase) (*TestResult, error) {
    var workflow *models.Workflow
    var err error

    // 模式1：引用工作流
    if testCase.WorkflowID != "" {
        workflow, err = e.workflowRepo.GetWorkflow(testCase.WorkflowID)
        if err != nil {
            return nil, fmt.Errorf("failed to load workflow: %w", err)
        }
    }

    // 模式2：内嵌工作流定义
    if testCase.WorkflowDef != nil {
        workflow = &models.Workflow{
            WorkflowID: fmt.Sprintf("inline-%s", testCase.TestID),
            Name:       testCase.Name,
            Definition: testCase.WorkflowDef,
        }
    }

    if workflow == nil {
        return nil, errors.New("no workflow definition found")
    }

    // 执行工作流
    workflowResult, err := e.workflowExecutor.Execute(workflow)
    if err != nil {
        return nil, err
    }

    // 转换为测试结果
    testResult := &TestResult{
        TestID:        testCase.TestID,
        Status:        convertWorkflowStatusToTestStatus(workflowResult.Status),
        StartTime:     workflowResult.StartTime,
        EndTime:       workflowResult.EndTime,
        Duration:      workflowResult.Duration,
        WorkflowRunID: workflowResult.RunID,  // 关联工作流执行ID
        Metrics: map[string]interface{}{
            "totalSteps":     workflowResult.TotalSteps,
            "completedSteps": workflowResult.CompletedSteps,
            "failedSteps":    workflowResult.FailedSteps,
            "stepDetails":    workflowResult.StepExecutions,
        },
        Logs: workflowResult.Logs,
    }

    return testResult, nil
}

func convertWorkflowStatusToTestStatus(workflowStatus string) string {
    switch workflowStatus {
    case "success":
        return "passed"
    case "failed":
        return "failed"
    case "cancelled":
        return "skipped"
    default:
        return "error"
    }
}
```

### TestCaseAction 实现（用于工作流引用测试案例）

```go
// TestCaseAction - 在工作流步骤中执行测试案例
type TestCaseAction struct {
    TestID string                 `json:"testId"`
    Input  map[string]interface{} `json:"input,omitempty"`
}

func (a *TestCaseAction) Execute(ctx *ActionContext) (*ActionResult, error) {
    // 1. 加载测试案例
    testCase, err := ctx.TestCaseRepo.GetTestCase(a.TestID)
    if err != nil {
        return nil, fmt.Errorf("test case not found: %s", a.TestID)
    }

    // 2. 应用输入变量
    testCaseWithInput := a.applyInputVariables(testCase, ctx.Variables, a.Input)

    // 3. 执行测试案例
    result, err := ctx.Executor.ExecuteTest(testCaseWithInput)
    if err != nil {
        return &ActionResult{
            Status: "failed",
            Error:  err,
        }, nil
    }

    // 4. 返回结果
    return &ActionResult{
        Status: result.Status,
        Output: map[string]interface{}{
            "testId":   result.TestID,
            "status":   result.Status,
            "duration": result.Duration,
            "response": result.Data,  // HTTP响应数据等
        },
        Duration: result.Duration,
    }, nil
}

func (a *TestCaseAction) applyInputVariables(
    testCase *models.TestCase,
    contextVars map[string]interface{},
    inputMapping map[string]interface{},
) *models.TestCase {
    // 克隆测试案例
    cloned := *testCase

    // 应用变量替换到配置中
    // 例如：将 {{authToken}} 替换为实际的token值
    switch testCase.Type {
    case "http":
        cloned.HTTPConfig = a.replaceVariables(testCase.HTTPConfig, contextVars, inputMapping)
    case "command":
        cloned.CommandConfig = a.replaceVariables(testCase.CommandConfig, contextVars, inputMapping)
    // ...
    }

    return &cloned
}

func (a *TestCaseAction) replaceVariables(
    config models.JSONB,
    contextVars map[string]interface{},
    inputMapping map[string]interface{},
) models.JSONB {
    // 将config序列化为字符串
    configStr, _ := json.Marshal(config)
    str := string(configStr)

    // 替换 {{variableName}} 形式的占位符
    for key, value := range inputMapping {
        placeholder := fmt.Sprintf("{{%s}}", key)
        str = strings.ReplaceAll(str, placeholder, fmt.Sprint(value))
    }

    // 反序列化回JSONB
    var result models.JSONB
    json.Unmarshal([]byte(str), &result)
    return result
}

func (a *TestCaseAction) Validate() error {
    if a.TestID == "" {
        return errors.New("testId is required")
    }
    return nil
}

func (a *TestCaseAction) Metadata() ActionMetadata {
    return ActionMetadata{
        Type:        "test-case",
        Description: "Execute an existing test case",
        Category:    "testing",
    }
}
```

---

## 前端UI设计

### 1. 测试案例列表 - 显示工作流类型

```
┌─────────────────────────────────────────────────────────────────┐
│  测试案例列表                                [新建测试 ▼]       │
├─────────────────────────────────────────────────────────────────┤
│  名称                    类型        优先级  状态    操作       │
│  ─────────────────────────────────────────────────────────────  │
│  📡 获取用户列表          HTTP        P1     ✅ 活跃   执行 编辑│
│  💻 检查服务状态          Command     P2     ✅ 活跃   执行 编辑│
│  🔀 用户注册流程         Workflow    P0     ✅ 活跃   执行 编辑│  ← 工作流测试
│  🔀 订单完整流程         Workflow    P0     ✅ 活跃   执行 编辑│  ← 工作流测试
│  🗄️ 查询订单数据          Database    P1     ✅ 活跃   执行 编辑│
└─────────────────────────────────────────────────────────────────┘
```

### 2. 新建测试 - 选择类型

```
┌─────────────────────────────────────────┐
│  新建测试案例                           │
├─────────────────────────────────────────┤
│  选择测试类型:                          │
│                                         │
│  ┌───────────────┐  ┌───────────────┐ │
│  │  📡 HTTP       │  │  💻 Command   │ │
│  │  API接口测试   │  │  命令执行     │ │
│  └───────────────┘  └───────────────┘ │
│                                         │
│  ┌───────────────┐  ┌───────────────┐ │
│  │  🔀 Workflow   │  │  🗄️ Database  │ │  ← 新增
│  │  工作流测试    │  │  数据库操作   │ │
│  └───────────────┘  └───────────────┘ │
│                                         │
│  ┌───────────────┐  ┌───────────────┐ │
│  │  🔐 Security   │  │  ⚡ Performance│ │
│  │  安全测试      │  │  性能测试     │ │
│  └───────────────┘  └───────────────┘ │
└─────────────────────────────────────────┘
```

### 3. 创建工作流测试 - 模式选择

```
┌─────────────────────────────────────────────────────────┐
│  创建工作流测试                                         │
├─────────────────────────────────────────────────────────┤
│  基本信息:                                              │
│  测试ID: [e2e-order-flow               ]                │
│  名称:   [订单完整流程测试             ]                │
│  优先级: [P0 ▼]                                         │
│  标签:   [e2e, order, critical         ]                │
│                                                         │
│  工作流配置:                                            │
│  ○ 引用已有工作流                                      │
│     选择工作流: [workflow-order-v2.1 ▼]                │
│                                                         │
│  ● 内嵌工作流定义                                      │
│     ┌─────────────────────────────────────────┐       │
│     │ 1  steps:                               │       │
│     │ 2    - id: login                        │       │
│     │ 3      type: http                       │       │
│     │ 4      http:                            │       │
│     │ 5        method: POST                   │       │
│     │ 6        path: /api/auth/login          │       │
│     │ 7      output:                          │       │
│     │ 8        token: "response.data.token"   │       │
│     │ 9                                       │       │
│     │10    - id: query                        │       │
│     │11      type: http                       │       │
│     │12      dependsOn: [login]               │       │
│     └─────────────────────────────────────────┘       │
│     [YAML编辑器] [可视化编辑器]                        │
│                                                         │
│  [取消]                              [保存并测试 →]    │
└─────────────────────────────────────────────────────────┘
```

### 4. 工作流测试结果展示

```
┌─────────────────────────────────────────────────────────┐
│  测试结果 - 订单完整流程测试                            │
│  状态: ✅ 通过  执行时间: 5分30秒                       │
├─────────────────────────────────────────────────────────┤
│  概览:                                                  │
│  ┌─────────────────────────────────────────────────┐   │
│  │  总步骤: 5个                                    │   │
│  │  ✅ 成功: 5个                                   │   │
│  │  ❌ 失败: 0个                                   │   │
│  │  ⏱️ 总耗时: 330秒                                │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  步骤详情:                                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │ ✅ 1. register - 注册用户      (200ms)         │   │
│  │    Input: {username: "test_user_123"}           │   │
│  │    Output: {userId: 12345}                      │   │
│  │                                                 │   │
│  │ ✅ 2. login - 用户登录         (150ms)         │   │
│  │    Input: {userId: 12345}                       │   │
│  │    Output: {token: "eyJ..."}                    │   │
│  │                                                 │   │
│  │ ✅ 3. create-order - 创建订单  (300ms)         │   │
│  │    Input: {token: "eyJ..."}                     │   │
│  │    Output: {orderId: 67890}                     │   │
│  │                                                 │   │
│  │ ✅ 4. payment - 支付订单       (500ms)         │   │
│  │    Input: {orderId: 67890, amount: 100}         │   │
│  │    Output: {paymentId: "PAY123"}                │   │
│  │                                                 │   │
│  │ ✅ 5. verify - 验证订单状态    (100ms)         │   │
│  │    Input: {orderId: 67890}                      │   │
│  │    Output: {status: "completed"}                │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  变量快照:                                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │ userId: 12345                                   │   │
│  │ token: "eyJ..."                                 │   │
│  │ orderId: 67890                                  │   │
│  │ paymentId: "PAY123"                             │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  [查看完整日志] [导出报告] [重新执行]                  │
└─────────────────────────────────────────────────────────┘
```

### 5. 工作流详情页 - 显示关联的测试案例

```
┌─────────────────────────────────────────────────────────┐
│  工作流详情 - 订单完整流程                              │
│  ID: workflow-order-v2.1  版本: 2.1                    │
├─────────────────────────────────────────────────────────┤
│  基本信息:                                              │
│  名称: 订单完整流程                                     │
│  步骤数: 5个                                            │
│  创建时间: 2025-11-20 10:30                             │
│                                                         │
│  关联的测试案例: (2个)                                  │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 📋 e2e-order-flow - 订单完整流程测试    [执行] │   │  ← 引用此工作流
│  │ 📋 smoke-order-test - 订单冒烟测试      [执行] │   │  ← 引用此工作流
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  工作流定义: [查看YAML] [可视化视图]                   │
│                                                         │
│  执行历史: [查看全部]                                   │
│  最近5次执行:                                           │
│  - 2025-11-21 14:30  ✅ 成功  (5分20秒)                │
│  - 2025-11-21 10:15  ✅ 成功  (5分30秒)                │
│  - 2025-11-20 16:40  ❌ 失败  (2分10秒) [查看]         │
│                                                         │
│  [编辑工作流] [删除] [导出]                             │
└─────────────────────────────────────────────────────────┘
```

---

## 迁移方案

### Phase 1: 数据模型扩展 (Week 1)

**目标**: 扩展数据模型，不影响现有功能

**任务**:
1. ✅ 数据库迁移脚本
   - 添加 `test_cases.workflow_id` 列
   - 添加 `test_cases.workflow_def` 列
   - 添加 `workflows.is_test_case` 列
   - 创建索引

2. ✅ 更新GORM模型
   - TestCase 添加 WorkflowID、WorkflowDef 字段
   - Workflow 添加 IsTestCase 字段、TestCases 关联

3. ✅ 向后兼容测试
   - 确保现有测试案例 CRUD 正常
   - 确保现有工作流 CRUD 正常

**验收标准**:
- 现有功能100%正常运行
- 新字段可读写，但尚未使用

---

### Phase 2: 执行引擎改造 (Week 2-3)

**目标**: 实现统一的测试执行器

**任务**:
1. ✅ 重构 TestExecutor
   - 实现 UnifiedTestExecutor
   - 路由到不同的执行器
   - 添加 executeWorkflowTest 方法

2. ✅ 实现 TestCaseAction
   - 在工作流中引用测试案例
   - 变量替换机制
   - 输出映射

3. ✅ 测试结果存储
   - WorkflowRunID 关联
   - 步骤详情存储
   - 变量快照存储

**验收标准**:
- 模式1（引用工作流）可以执行
- 模式2（内嵌工作流）可以执行
- 模式3（工作流引用测试）可以执行
- 测试结果完整记录

---

### Phase 3: API扩展 (Week 4)

**目标**: 提供完整的API支持

**任务**:
1. ✅ 测试案例API
   - POST /api/v2/tests - 支持 type=workflow
   - GET /api/v2/tests/:id - 返回工作流信息
   - POST /api/v2/tests/:id/execute - 执行工作流测试

2. ✅ 工作流API
   - GET /api/v2/workflows/:id/test-cases - 查询关联测试
   - GET /api/v2/workflows/:id/usage - 使用统计

3. ✅ API文档更新
   - Swagger/OpenAPI规范
   - 示例代码

**验收标准**:
- 所有API端点正常工作
- API文档完整
- Postman collection 可用

---

### Phase 4: 前端UI (Week 5-6)

**目标**: 提供完整的前端界面

**任务**:
1. ✅ 测试案例列表
   - 显示工作流类型图标
   - 工作流测试筛选

2. ✅ 创建工作流测试
   - 模式选择（引用/内嵌）
   - 工作流选择器
   - YAML编辑器

3. ✅ 工作流测试结果
   - 步骤详情展示
   - 变量快照展示
   - 数据流可视化

4. ✅ 工作流详情页
   - 关联测试案例列表
   - 一键执行关联测试

**验收标准**:
- 用户可以创建工作流测试
- 用户可以执行工作流测试
- 用户可以查看详细结果
- UI响应流畅，交互清晰

---

### Phase 5: 文档和培训 (Week 7)

**目标**: 完善文档，培训用户

**任务**:
1. ✅ 用户文档
   - 工作流测试快速开始
   - 三种模式的使用场景
   - 最佳实践

2. ✅ 示例和模板
   - 10个常见工作流测试示例
   - 模板库更新

3. ✅ 培训材料
   - 视频教程
   - 在线研讨会

**验收标准**:
- 文档完整且易懂
- 用户可以自助完成简单场景
- 培训覆盖50%以上用户

---

## 总结

### 核心设计思想

1. **统一入口**: 测试案例是用户的统一入口，工作流是实现方式之一
2. **双向引用**: 测试案例可以引用工作流，工作流也可以引用测试案例
3. **模式灵活**: 三种模式覆盖不同场景（简单/复杂/复用）
4. **执行统一**: 统一的执行引擎和结果存储
5. **渐进迁移**: 分5个阶段，每阶段都可独立交付

### 预期收益

| 指标 | 当前 | 目标 | 提升 |
|------|------|------|------|
| 复杂测试覆盖 | 无法覆盖多步骤流程 | 支持端到端测试 | 质的飞跃 |
| 测试复用率 | 低（每个测试独立） | 高（工作流+测试组合） | 50%+ |
| 用户学习成本 | 需要理解"测试"和"工作流"两个概念 | 只需理解"测试"一个概念 | 降低30% |
| 管理复杂度 | 分散管理 | 统一管理 | 降低40% |

---

**文档结束**

**变更历史**:
| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0 | 2025-11-21 | 初始版本 |

# 测试管理框架增强方案

**文档版本**: 1.0
**创建日期**: 2025-11-21
**作者**: 测试框架团队

---

## 📑 目录

1. [概述](#概述)
2. [需求分析](#需求分析)
3. [技术方案](#技术方案)
4. [架构设计](#架构设计)
5. [实施路线图](#实施路线图)
6. [风险评估](#风险评估)
7. [附录](#附录)

---

## 概述

### 背景

当前测试管理系统提供了基础的测试用例管理和执行能力，但在以下方面存在局限性：

- 测试用例需要逐个创建，批量管理效率低
- 缺乏对复杂测试场景的编排能力
- 测试逻辑固化在配置中，灵活性不足
- 前端工程化水平较低，难以支撑复杂交互

### 目标

将当前系统从**简单的测试管理工具**升级为**企业级测试平台**，支持：

✅ 测试用例的批量导入和数据驱动测试
✅ 复杂业务流程的工作流编排
✅ 灵活的脚本化测试能力
✅ 标准化的模板系统
✅ 现代化的前端用户体验

### 预期收益

| 指标 | 当前状态 | 目标状态 | 提升幅度 |
|------|---------|---------|---------|
| 测试创建效率 | 1个/分钟 | 100个/分钟 | **100倍** |
| 复杂场景支持 | 不支持 | 支持多步骤工作流 | **质的飞跃** |
| 测试灵活性 | 配置驱动 | 脚本 + 配置混合 | **10倍** |
| 用户学习成本 | 需熟悉JSON | 模板 + 可视化 | **降低70%** |

---

## 需求分析

### 1. 测试数据文件导入与批量测试

#### 痛点

- **效率低下**: 创建100个测试用例需要手动操作100次
- **数据耦合**: 测试数据内嵌在测试定义中，难以复用
- **协作困难**: 非技术人员（QA、产品）难以参与测试数据准备
- **版本管理**: 测试数据变更难以追踪

#### 用户故事

> 作为一名QA工程师，我需要从Excel表格批量导入100个API测试用例，这样我可以在10分钟内完成原本需要2小时的工作。

> 作为一名测试负责人，我需要用不同的测试数据集运行同一个测试逻辑，以覆盖边界条件、异常场景和大数据量情况。

#### 核心需求

1. **文件导入**
   - 支持格式: CSV, Excel (.xlsx), JSON, YAML
   - 导入策略: 增量/覆盖/校验
   - 错误处理: 详细的验证报告

2. **数据驱动测试**
   - 分离测试逻辑和测试数据
   - 参数化测试执行
   - 数据集版本管理

3. **批量操作**
   - 按标签批量执行
   - 按文件批量管理
   - 批量结果分析

#### 技术价值

- **测试效率**: 批量导入可节省 **90%** 的测试创建时间
- **数据复用**: 一个测试模板 + 多个数据集 = N个测试场景
- **团队协作**: QA可在Excel中准备数据，开发导入系统
- **持续集成**: 测试数据可纳入版本控制，CI/CD自动运行

---

### 2. 工作流引擎与组件化测试

#### 痛点

- **场景局限**: 当前测试是原子化的，无法表达真实业务流程
- **状态孤立**: 测试间无法共享状态（如登录token）
- **依赖缺失**: 无法定义测试间的前置/后置依赖
- **复用困难**: 通用步骤（如认证）需要在每个测试中重复定义

#### 用户故事

> 作为一名自动化测试工程师，我需要编排一个"注册→登录→下单→支付→退款"的完整业务流程测试，其中每一步都依赖前一步的输出。

> 作为一名API测试工程师，我需要将"获取认证token"封装为可复用组件，在所有需要认证的测试中引用它。

#### 核心需求

1. **工作流定义**
   - YAML/JSON格式的工作流描述
   - 步骤间的顺序/并行执行
   - 条件分支和循环

2. **状态管理**
   - 步骤间的数据传递 (变量、上下文)
   - 全局/局部变量作用域
   - 状态持久化和恢复

3. **组件化**
   - 可复用的测试片段
   - 工作流模板
   - 子工作流调用

4. **错误处理**
   - 失败重试策略
   - 回滚机制
   - 部分失败处理

#### 工作流示例

```yaml
workflow:
  name: "电商下单完整流程"
  version: "1.0"

  # 全局变量
  variables:
    baseUrl: "https://api.example.com"
    retryCount: 3

  # 工作流步骤
  steps:
    - id: "register"
      name: "注册新用户"
      type: "http-test"
      testId: "user-register"
      input:
        username: "test_${timestamp}"
        email: "test_${timestamp}@example.com"
      output:
        userId: "response.data.id"  # 保存用户ID到上下文
      retry:
        maxAttempts: 3
        backoff: "exponential"

    - id: "login"
      name: "用户登录"
      type: "http-test"
      testId: "user-login"
      dependsOn: ["register"]  # 依赖注册步骤
      input:
        username: "${register.username}"
        password: "${register.password}"
      output:
        authToken: "response.data.token"
      onError: "abort"  # 登录失败则中止整个流程

    - id: "create-order"
      name: "创建订单"
      type: "http-test"
      testId: "order-create"
      dependsOn: ["login"]
      headers:
        Authorization: "Bearer ${login.authToken}"
      input:
        productId: "P12345"
        quantity: 2
      output:
        orderId: "response.data.orderId"
      onError: "continue"  # 创建失败继续执行（为了清理资源）

    - id: "payment"
      name: "支付订单"
      type: "http-test"
      testId: "order-payment"
      when: "${create-order.status == 'success'}"  # 条件执行
      dependsOn: ["create-order"]
      input:
        orderId: "${create-order.orderId}"
        amount: 100.00
      output:
        paymentId: "response.data.paymentId"

    - id: "cleanup"
      name: "清理测试数据"
      type: "http-test"
      testId: "cleanup-test-user"
      runOnFailure: true  # 即使前面失败也执行
      input:
        userId: "${register.userId}"
```

#### 技术价值

- **场景覆盖**: 可模拟真实用户旅程，端到端测试
- **代码复用**: 通用步骤封装后，减少 **80%** 重复代码
- **维护性**: 业务流程变更只需修改工作流定义
- **可视化**: 工作流可生成流程图，便于理解和沟通

---

### 3. 前端现代化改造

#### 痛点

- **工程化缺失**: 当前使用CDN引入React，无构建流程
- **开发体验差**: 无TypeScript、无热重载、无组件复用
- **交互局限**: 难以实现复杂的工作流可视化编辑器
- **性能问题**: 全量加载，无代码分割和懒加载

#### 用户故事

> 作为一名测试工程师，我希望通过拖拽的方式编排测试工作流，而不是手写YAML配置文件。

> 作为一名开发者，我需要快速定位测试失败的原因，通过交互式的结果可视化（调用链、时序图）进行问题诊断。

#### 技术选型对比

| 方案 | 优势 | 劣势 | 推荐场景 |
|------|------|------|----------|
| **Next.js** | • SSR优化性能<br>• 全栈能力<br>• 生态成熟<br>• SEO友好 | • 过度设计（对当前需求）<br>• 部署需Node.js<br>• 学习曲线陡峭 | 需要SEO、复杂服务端逻辑 |
| **Vite + React** | • 轻量快速<br>• 开发体验好<br>• 灵活可控<br>• TypeScript支持 | • 需要自行配置路由等<br>• 无SSR | ✅ **推荐**: 纯前端SPA应用 |
| **保持现状** | • 零成本<br>• 稳定性高 | • 无工程化<br>• 难以扩展 | 功能简单、不追求体验 |

#### 推荐方案: Vite + React + TypeScript

**理由:**
1. 当前应用是纯前端SPA，不需要SSR
2. Vite开发体验优于Next.js（HMR更快）
3. 轻量化，避免过度工程
4. 生态完善，支持所需的所有UI组件

#### 核心功能

1. **工作流可视化编辑器**
   - 技术栈: React Flow / GoJS
   - 功能: 拖拽编排、实时预览、自动布局

2. **测试结果可视化**
   - 调用链图 (类似Jaeger)
   - 时序图
   - 性能火焰图

3. **模板市场**
   - 浏览、搜索、预览模板
   - 一键克隆和自定义

4. **实时监控**
   - WebSocket实时推送测试执行状态
   - 进度条、日志流

---

### 4. 测试脚本支持 (Lua + JSON)

#### 痛点

- **逻辑受限**: JSON配置是声明式的，无法表达复杂逻辑
- **动态能力弱**: 无法动态生成数据、循环、条件判断
- **扩展困难**: 新增断言类型需要修改后端代码

#### 用户故事

> 作为一名高级测试工程师，我需要在测试中动态生成随机数据，并编写复杂的业务逻辑验证（如计算订单总额是否正确）。

> 作为一名性能测试工程师，我需要编写循环脚本，模拟100个并发用户的行为。

#### 为什么选择 Lua?

| 特性 | Lua | Python | JavaScript |
|------|-----|--------|-----------|
| **体积** | 200KB | 30MB+ | 取决于运行时 |
| **嵌入性** | ✅ 专为嵌入设计 | ⚠️ 需要独立进程 | ⚠️ 需要V8引擎 |
| **安全性** | ✅ 易于沙箱化 | ❌ 难以限制 | ⚠️ 中等 |
| **性能** | ✅ 快速 | ⚠️ 较慢 | ✅ 快速 |
| **Go集成** | ✅ gopher-lua | ⚠️ exec调用 | ⚠️ goja |
| **成熟度** | ✅ Kong, Nginx, Redis | ✅ 广泛使用 | ✅ 广泛使用 |

**结论**: Lua是嵌入式脚本的最佳选择

#### 使用场景

**场景1: 动态数据生成**
```lua
-- 生成测试用户数据
function generateUsers(count)
    local users = {}
    for i = 1, count do
        table.insert(users, {
            username = "user_" .. math.random(10000, 99999),
            email = "user" .. i .. "@test.com",
            age = math.random(18, 60)
        })
    end
    return users
end
```

**场景2: 复杂业务验证**
```lua
-- 验证订单计算逻辑
function validateOrder(response)
    local order = response.data
    local calculatedTotal = 0

    -- 计算商品总价
    for _, item in ipairs(order.items) do
        calculatedTotal = calculatedTotal + (item.price * item.quantity)
    end

    -- 加上运费
    if calculatedTotal < 100 then
        calculatedTotal = calculatedTotal + order.shippingFee
    end

    -- 减去折扣
    calculatedTotal = calculatedTotal * (1 - order.discountRate)

    -- 验证
    if math.abs(calculatedTotal - order.totalAmount) > 0.01 then
        return false, string.format(
            "订单总额错误: 期望 %.2f, 实际 %.2f",
            calculatedTotal, order.totalAmount
        )
    end

    return true
end
```

**场景3: 条件逻辑**
```lua
-- 根据环境选择不同的配置
function getConfig()
    local env = os.getenv("ENV") or "test"

    if env == "prod" then
        return {
            baseUrl = "https://api.production.com",
            timeout = 30
        }
    elseif env == "staging" then
        return {
            baseUrl = "https://api.staging.com",
            timeout = 60
        }
    else
        return {
            baseUrl = "http://localhost:8080",
            timeout = 120
        }
    end
end
```

#### JSON + Lua 混合模式

```json
{
  "testId": "complex-order-test",
  "name": "复杂订单验证",
  "type": "http",
  "http": {
    "method": "POST",
    "path": "/api/orders"
  },
  "luaScript": {
    "beforeRequest": "generateOrderData()",
    "afterResponse": "validateOrder(response)",
    "customAssertion": "checkBusinessRules(data)"
  }
}
```

#### 安全机制

| 安全措施 | 说明 |
|---------|------|
| **沙箱执行** | 禁用 `os`, `io`, `debug` 等危险模块 |
| **执行超时** | 默认5秒，防止死循环 |
| **内存限制** | 限制脚本可用内存 |
| **代码审查** | 敏感操作需要人工审核 |
| **权限控制** | 普通用户只能运行预定义脚本 |

---

### 5. 模板系统

#### 痛点

- **重复劳动**: 相似的测试需要重复编写配置
- **标准缺失**: 团队成员编写风格不一致
- **学习成本**: 新手不知道如何编写规范的测试
- **维护困难**: 最佳实践变更需要批量修改测试

#### 用户故事

> 作为一名新入职的QA，我希望从模板库中选择"RESTful CRUD测试"模板，填入API地址和资源名称，就能生成完整的增删改查测试套件。

> 作为一名测试负责人，我需要创建团队标准模板，确保所有测试都包含必要的断言（如响应时间、错误处理）。

#### 模板类型

**1. 变量替换模板**

最简单的模板形式，支持变量占位符。

```json
{
  "name": "RESTful CRUD 模板",
  "description": "标准的增删改查测试套件",
  "category": "API测试",
  "variables": [
    {
      "name": "resource",
      "description": "资源名称（如 users, products）",
      "type": "string",
      "required": true
    },
    {
      "name": "baseUrl",
      "description": "API基础地址",
      "type": "string",
      "default": "http://localhost:8080/api"
    }
  ],
  "tests": [
    {
      "name": "创建${resource}",
      "type": "http",
      "http": {
        "method": "POST",
        "path": "${baseUrl}/${resource}",
        "body": {
          "name": "测试数据"
        }
      },
      "assertions": [
        {"type": "status_code", "expected": 201},
        {"type": "response_time", "lessThan": 1000}
      ],
      "saveResponse": "createdId:response.data.id"
    },
    {
      "name": "查询${resource}列表",
      "type": "http",
      "http": {
        "method": "GET",
        "path": "${baseUrl}/${resource}"
      },
      "assertions": [
        {"type": "status_code", "expected": 200},
        {"type": "json_schema", "schema": "array"}
      ]
    },
    {
      "name": "获取${resource}详情",
      "type": "http",
      "http": {
        "method": "GET",
        "path": "${baseUrl}/${resource}/${createdId}"
      },
      "assertions": [
        {"type": "status_code", "expected": 200}
      ]
    },
    {
      "name": "更新${resource}",
      "type": "http",
      "http": {
        "method": "PUT",
        "path": "${baseUrl}/${resource}/${createdId}",
        "body": {
          "name": "更新后的数据"
        }
      },
      "assertions": [
        {"type": "status_code", "expected": 200}
      ]
    },
    {
      "name": "删除${resource}",
      "type": "http",
      "http": {
        "method": "DELETE",
        "path": "${baseUrl}/${resource}/${createdId}"
      },
      "assertions": [
        {"type": "status_code", "expected": 204}
      ]
    }
  ]
}
```

**2. 模板继承**

支持基础模板 + 扩展的方式。

```yaml
# 基础HTTP测试模板
base-http-test:
  type: http
  timeout: 30
  assertions:
    - type: status_code
      expected: 200
    - type: response_time
      lessThan: 1000

# 认证HTTP测试模板（继承基础模板）
authenticated-http-test:
  extends: base-http-test
  headers:
    Authorization: "Bearer ${authToken}"
  setupHooks:
    - name: "获取认证token"
      type: http
      http:
        method: POST
        path: /api/auth/login
      saveResponse: "authToken"

# 具体测试继承认证模板
my-api-test:
  extends: authenticated-http-test
  name: "获取用户信息"
  http:
    method: GET
    path: /api/users/me
```

**3. 模板组合**

将多个模板片段组合成完整测试。

```yaml
test:
  name: "完整的用户注册流程"
  components:
    - template: "email-validation"
      input:
        email: "${testEmail}"

    - template: "password-strength-check"
      input:
        password: "${testPassword}"

    - template: "duplicate-check"
      input:
        username: "${testUsername}"

    - template: "create-user-api"
      input:
        email: "${testEmail}"
        password: "${testPassword}"
        username: "${testUsername}"
```

#### 模板市场

**官方模板库**:
- RESTful CRUD
- 认证授权 (OAuth, JWT, Session)
- 文件上传下载
- 分页查询
- 批量操作
- WebSocket连接
- GraphQL查询

**团队私有模板**:
- 业务特定流程
- 内部API规范
- 合规性检查

**模板评分系统**:
- ⭐️ 使用次数
- 📝 文档完整度
- ✅ 测试覆盖率
- 💬 用户评价

---

## 技术方案

### 技术栈选型

#### 后端

| 组件 | 当前 | 目标 | 理由 |
|------|------|------|------|
| **语言** | Go 1.24 | Go 1.24+ | 保持不变 |
| **框架** | Gin | Gin | 成熟稳定 |
| **数据库** | SQLite | PostgreSQL (可选) | 支持高并发 |
| **脚本引擎** | - | gopher-lua | 嵌入式Lua VM |
| **工作流引擎** | - | 自研 | 轻量可控 |

#### 前端

| 组件 | 当前 | 目标 | 理由 |
|------|------|------|------|
| **框架** | React (CDN) | Vite + React 18 | 工程化 |
| **语言** | JavaScript | TypeScript | 类型安全 |
| **UI库** | Ant Design 5 | Ant Design 5 | 保持一致 |
| **状态管理** | - | Zustand | 轻量简洁 |
| **可视化** | - | React Flow | 工作流编辑 |
| **图表** | - | ECharts | 结果可视化 |

### 核心模块设计

#### 1. 数据导入模块

**API设计**:
```go
// POST /api/v2/import/tests
type ImportRequest struct {
    Format      string          `json:"format"`      // csv, excel, json, yaml
    Data        string          `json:"data"`        // Base64编码的文件内容
    Strategy    string          `json:"strategy"`    // create, update, upsert
    DryRun      bool            `json:"dryRun"`      // 只验证不导入
    GroupID     string          `json:"groupId"`     // 目标分组
}

type ImportResponse struct {
    TotalRows   int             `json:"totalRows"`
    SuccessRows int             `json:"successRows"`
    FailedRows  int             `json:"failedRows"`
    Errors      []ImportError   `json:"errors"`
    TestIDs     []string        `json:"testIds"`     // 创建的测试ID列表
}
```

**Excel格式示例**:

| testId | groupId | name | type | method | path | assertions | tags |
|--------|---------|------|------|--------|------|------------|------|
| test-1 | api-tests | 获取用户列表 | http | GET | /api/users | [{"type":"status_code","expected":200}] | api,smoke |
| test-2 | api-tests | 创建用户 | http | POST | /api/users | [{"type":"status_code","expected":201}] | api,critical |

#### 2. 工作流引擎

**工作流定义**:
```go
type Workflow struct {
    ID          string              `json:"id"`
    Name        string              `json:"name"`
    Version     string              `json:"version"`
    Variables   map[string]string   `json:"variables"`
    Steps       []WorkflowStep      `json:"steps"`
}

type WorkflowStep struct {
    ID          string              `json:"id"`
    Name        string              `json:"name"`
    Type        string              `json:"type"`        // http-test, lua-script, sub-workflow
    TestID      string              `json:"testId"`      // 引用的测试ID
    DependsOn   []string            `json:"dependsOn"`   // 依赖的步骤ID
    Input       map[string]string   `json:"input"`       // 输入变量映射
    Output      map[string]string   `json:"output"`      // 输出变量映射
    When        string              `json:"when"`        // 条件表达式
    OnError     string              `json:"onError"`     // abort, continue, retry
    Retry       *RetryConfig        `json:"retry"`
}

type WorkflowContext struct {
    Variables   map[string]interface{}  // 全局变量
    StepResults map[string]*StepResult  // 步骤执行结果
}
```

**执行引擎**:
```go
type WorkflowEngine struct {
    executor    *testcase.Executor
    luaVM       *lua.LState
}

func (e *WorkflowEngine) Execute(workflow *Workflow) *WorkflowResult {
    ctx := NewWorkflowContext(workflow.Variables)

    // 拓扑排序，确定执行顺序
    executionOrder := e.topologicalSort(workflow.Steps)

    for _, step := range executionOrder {
        // 评估条件
        if !e.evaluateCondition(step.When, ctx) {
            continue
        }

        // 执行步骤
        result := e.executeStep(step, ctx)
        ctx.StepResults[step.ID] = result

        // 错误处理
        if result.Error != nil {
            switch step.OnError {
            case "abort":
                return &WorkflowResult{Status: "failed", Error: result.Error}
            case "retry":
                result = e.retryStep(step, ctx)
            case "continue":
                // 继续执行
            }
        }
    }

    return &WorkflowResult{Status: "success", Context: ctx}
}
```

#### 3. Lua脚本引擎

**集成gopher-lua**:
```go
import "github.com/yuin/gopher-lua"

type LuaScriptEngine struct {
    vm *lua.LState
}

func NewLuaScriptEngine() *LuaScriptEngine {
    L := lua.NewState()

    // 安全沙箱：禁用危险模块
    L.SetGlobal("os", lua.LNil)
    L.SetGlobal("io", lua.LNil)
    L.SetGlobal("debug", lua.LNil)

    // 注册自定义函数
    L.SetGlobal("httpRequest", L.NewFunction(luaHTTPRequest))
    L.SetGlobal("sleep", L.NewFunction(luaSleep))

    return &LuaScriptEngine{vm: L}
}

func (e *LuaScriptEngine) Execute(script string, context map[string]interface{}) (interface{}, error) {
    // 设置上下文变量
    for k, v := range context {
        e.vm.SetGlobal(k, convertToLuaValue(v))
    }

    // 执行脚本（带超时）
    done := make(chan error, 1)
    go func() {
        done <- e.vm.DoString(script)
    }()

    select {
    case err := <-done:
        if err != nil {
            return nil, err
        }
        return e.vm.Get(-1), nil
    case <-time.After(5 * time.Second):
        return nil, errors.New("script execution timeout")
    }
}
```

**预定义的Lua函数**:
```lua
-- HTTP请求
response = httpRequest({
    method = "GET",
    url = "https://api.example.com/users",
    headers = {["Authorization"] = "Bearer " .. token}
})

-- JSON解析
data = json.decode(response.body)

-- 日志输出
log.info("Found " .. #data .. " users")

-- 断言
assert(response.statusCode == 200, "Status code should be 200")
assert(#data > 0, "Should have at least one user")
```

#### 4. 模板管理

**模板存储**:
```go
type Template struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Category    string                 `json:"category"`
    Variables   []TemplateVariable     `json:"variables"`
    Content     interface{}            `json:"content"`  // 测试定义或工作流定义
    IsOfficial  bool                   `json:"isOfficial"`
    UsageCount  int                    `json:"usageCount"`
    Rating      float64                `json:"rating"`
}

type TemplateVariable struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Type        string      `json:"type"`        // string, number, boolean
    Required    bool        `json:"required"`
    Default     interface{} `json:"default"`
}
```

**变量替换**:
```go
func (t *Template) Render(values map[string]interface{}) (interface{}, error) {
    // 验证必需变量
    for _, v := range t.Variables {
        if v.Required && values[v.Name] == nil {
            return nil, fmt.Errorf("missing required variable: %s", v.Name)
        }
    }

    // 应用默认值
    for _, v := range t.Variables {
        if values[v.Name] == nil && v.Default != nil {
            values[v.Name] = v.Default
        }
    }

    // 变量替换
    contentJSON, _ := json.Marshal(t.Content)
    contentStr := string(contentJSON)

    for k, v := range values {
        placeholder := fmt.Sprintf("${%s}", k)
        contentStr = strings.ReplaceAll(contentStr, placeholder, fmt.Sprint(v))
    }

    var result interface{}
    json.Unmarshal([]byte(contentStr), &result)
    return result, nil
}
```

---

## 架构设计

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     Presentation Layer                      │
│              (Vite + React + TypeScript)                    │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │  工作流    │  │  测试结果  │  │  模板市场  │           │
│  │  编辑器    │  │  可视化    │  │           │           │
│  └────────────┘  └────────────┘  └────────────┘           │
└─────────────────────────────────────────────────────────────┘
                         ↕ REST API / WebSocket
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway (Gin)                      │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ 测试管理 │  │ 工作流   │  │ 模板管理 │  │ 数据导入 │  │
│  │ API      │  │ API      │  │ API      │  │ API      │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                          │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  Workflow    │  │  Template    │  │  Import/Export  │  │
│  │  Engine      │  │  Service     │  │  Service        │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                    Execution Layer                          │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  Test        │  │  Lua Script  │  │  Hook           │  │
│  │  Executor    │  │  Engine      │  │  Manager        │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                      Data Layer                             │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  PostgreSQL / SQLite                                 │  │
│  │  - Tests  - Workflows  - Templates  - Results       │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 数据库设计

**新增表**:

```sql
-- 工作流定义表
CREATE TABLE workflows (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(32),
    description TEXT,
    definition JSONB NOT NULL,  -- 工作流定义（步骤、变量等）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64)
);

-- 工作流执行历史
CREATE TABLE workflow_runs (
    id SERIAL PRIMARY KEY,
    workflow_id VARCHAR(64) REFERENCES workflows(id),
    status VARCHAR(32),  -- running, success, failed, cancelled
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    duration INTEGER,
    context JSONB,  -- 执行上下文（变量、步骤结果）
    error TEXT
);

-- 模板表
CREATE TABLE templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(64),
    description TEXT,
    variables JSONB,  -- 模板变量定义
    content JSONB NOT NULL,  -- 模板内容
    is_official BOOLEAN DEFAULT FALSE,
    usage_count INTEGER DEFAULT 0,
    rating DECIMAL(3, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64)
);

-- 导入历史
CREATE TABLE import_history (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255),
    format VARCHAR(32),
    total_rows INTEGER,
    success_rows INTEGER,
    failed_rows INTEGER,
    errors JSONB,
    created_test_ids JSONB,
    imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    imported_by VARCHAR(64)
);

-- Lua脚本管理表
CREATE TABLE scripts (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(64),  -- utility, validation, data-generation, conversion, cleanup
    content TEXT NOT NULL,  -- Lua脚本内容
    parameters JSONB,  -- 参数定义 [{name, type, default, description}]
    tags TEXT[],  -- 标签数组
    visibility VARCHAR(32) DEFAULT 'private',  -- public, team, private
    status VARCHAR(32) DEFAULT 'pending',  -- pending, approved, rejected
    approval_status_reason TEXT,  -- 审核拒绝原因
    current_version VARCHAR(32) DEFAULT 'v1.0',
    reference_count INTEGER DEFAULT 0,  -- 被引用次数
    execution_count INTEGER DEFAULT 0,  -- 执行次数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    approved_by VARCHAR(64),
    approved_at TIMESTAMP,
    deleted_at TIMESTAMP  -- 软删除
);

-- 脚本版本历史表
CREATE TABLE script_versions (
    id SERIAL PRIMARY KEY,
    script_id VARCHAR(64) REFERENCES scripts(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    parameters JSONB,
    change_note TEXT,  -- 版本备注
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    UNIQUE(script_id, version)
);

-- 脚本审核历史表
CREATE TABLE script_audit_log (
    id SERIAL PRIMARY KEY,
    script_id VARCHAR(64) REFERENCES scripts(id) ON DELETE CASCADE,
    action VARCHAR(32),  -- submit, approve, reject
    status VARCHAR(32),  -- pending, approved, rejected
    reason TEXT,
    performed_by VARCHAR(64),
    performed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 模板评分表
CREATE TABLE template_ratings (
    id SERIAL PRIMARY KEY,
    template_id VARCHAR(64) REFERENCES templates(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL,
    overall_rating INTEGER CHECK (overall_rating >= 1 AND overall_rating <= 5),
    usability_rating INTEGER CHECK (usability_rating >= 1 AND usability_rating <= 5),
    documentation_rating INTEGER CHECK (documentation_rating >= 1 AND documentation_rating <= 5),
    coverage_rating INTEGER CHECK (coverage_rating >= 1 AND coverage_rating <= 5),
    comment TEXT,
    helpful_count INTEGER DEFAULT 0,  -- 点赞数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(template_id, user_id)
);

-- 模板评论回复表
CREATE TABLE template_rating_replies (
    id SERIAL PRIMARY KEY,
    rating_id INTEGER REFERENCES template_ratings(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 模板版本表
CREATE TABLE template_versions (
    id SERIAL PRIMARY KEY,
    template_id VARCHAR(64) REFERENCES templates(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    variables JSONB,
    content JSONB NOT NULL,
    change_note TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    UNIQUE(template_id, version)
);

-- 模板使用统计表
CREATE TABLE template_usage_stats (
    id SERIAL PRIMARY KEY,
    template_id VARCHAR(64) REFERENCES templates(id) ON DELETE CASCADE,
    used_at DATE NOT NULL,
    usage_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    UNIQUE(template_id, used_at)
);

-- 模板Fork关系表
CREATE TABLE template_forks (
    id SERIAL PRIMARY KEY,
    source_template_id VARCHAR(64) REFERENCES templates(id) ON DELETE SET NULL,
    forked_template_id VARCHAR(64) REFERENCES templates(id) ON DELETE CASCADE,
    forked_by VARCHAR(64),
    forked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(forked_template_id)
);

-- 数据库连接配置表
CREATE TABLE database_connections (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,  -- mysql, postgresql, sqlite
    host VARCHAR(255),
    port INTEGER,
    database_name VARCHAR(255),
    username VARCHAR(255),
    password_encrypted TEXT,  -- 加密存储
    ssl_enabled BOOLEAN DEFAULT FALSE,
    read_only BOOLEAN DEFAULT FALSE,
    connection_timeout INTEGER DEFAULT 30,
    status VARCHAR(32) DEFAULT 'offline',  -- online, offline, error
    last_connected_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64)
);

-- 查询模板表
CREATE TABLE query_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    database_connection_id VARCHAR(64) REFERENCES database_connections(id) ON DELETE CASCADE,
    sql_content TEXT NOT NULL,
    category VARCHAR(64),
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64)
);

-- SQL查询历史表
CREATE TABLE query_history (
    id SERIAL PRIMARY KEY,
    database_connection_id VARCHAR(64) REFERENCES database_connections(id) ON DELETE CASCADE,
    sql_content TEXT NOT NULL,
    execution_time_ms INTEGER,
    rows_affected INTEGER,
    status VARCHAR(32),  -- success, error
    error_message TEXT,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    executed_by VARCHAR(64)
);

-- 数据库操作审计表
CREATE TABLE database_operation_audit (
    id SERIAL PRIMARY KEY,
    database_connection_id VARCHAR(64) REFERENCES database_connections(id) ON DELETE CASCADE,
    operation_type VARCHAR(32),  -- SELECT, INSERT, UPDATE, DELETE, DROP, etc.
    table_name VARCHAR(255),
    sql_content TEXT NOT NULL,
    rows_affected INTEGER,
    status VARCHAR(32),  -- success, blocked, error
    blocked_reason TEXT,  -- 只读模式阻止原因
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    executed_by VARCHAR(64)
);
```

-- 索引优化
```sql
-- 脚本管理索引
CREATE INDEX idx_scripts_category ON scripts(category);
CREATE INDEX idx_scripts_status ON scripts(status);
CREATE INDEX idx_scripts_visibility ON scripts(visibility);
CREATE INDEX idx_scripts_created_by ON scripts(created_by);
CREATE INDEX idx_script_versions_script_id ON script_versions(script_id);

-- 模板管理索引
CREATE INDEX idx_template_ratings_template_id ON template_ratings(template_id);
CREATE INDEX idx_template_versions_template_id ON template_versions(template_id);
CREATE INDEX idx_template_usage_stats_template_id_date ON template_usage_stats(template_id, used_at);
CREATE INDEX idx_template_forks_source ON template_forks(source_template_id);

-- 数据库管理索引
CREATE INDEX idx_database_connections_created_by ON database_connections(created_by);
CREATE INDEX idx_query_templates_db_id ON query_templates(database_connection_id);
CREATE INDEX idx_query_history_db_id ON query_history(database_connection_id);
CREATE INDEX idx_query_history_executed_at ON query_history(executed_at);
CREATE INDEX idx_db_audit_db_id ON database_operation_audit(database_connection_id);
CREATE INDEX idx_db_audit_executed_at ON database_operation_audit(executed_at);
```

### API设计

#### 工作流API

```
POST   /api/v2/workflows              # 创建工作流
GET    /api/v2/workflows              # 列出工作流
GET    /api/v2/workflows/:id          # 获取工作流详情
PUT    /api/v2/workflows/:id          # 更新工作流
DELETE /api/v2/workflows/:id          # 删除工作流

POST   /api/v2/workflows/:id/execute  # 执行工作流
GET    /api/v2/workflows/:id/runs     # 工作流执行历史
GET    /api/v2/workflow-runs/:id      # 执行详情
POST   /api/v2/workflow-runs/:id/cancel  # 取消执行
```

#### 模板API

```
POST   /api/v2/templates              # 创建模板
GET    /api/v2/templates              # 浏览模板（支持分类、搜索）
GET    /api/v2/templates/:id          # 获取模板详情
POST   /api/v2/templates/:id/render   # 渲染模板（填充变量）
POST   /api/v2/templates/:id/clone    # 克隆模板
PUT    /api/v2/templates/:id/rating   # 评分
```

#### 导入/导出API

```
POST   /api/v2/import/tests           # 导入测试
POST   /api/v2/import/workflows       # 导入工作流
GET    /api/v2/import/history         # 导入历史

GET    /api/v2/export/tests           # 导出测试（支持CSV/Excel/JSON）
GET    /api/v2/export/workflows/:id   # 导出工作流
```

---

## 实施路线图

### Phase 1: 快速价值交付（2-4周）

**目标**: 解决最大痛点，快速见效

#### 任务清单

- [ ] **数据导入功能**
  - [ ] CSV导入解析器
  - [ ] Excel导入解析器（使用excelize库）
  - [ ] JSON/YAML导入
  - [ ] 导入API端点
  - [ ] 验证与错误处理

- [ ] **批量执行**
  - [ ] 按标签批量执行
  - [ ] 批量执行API
  - [ ] 批量结果汇总

- [ ] **基础模板系统**
  - [ ] 模板定义格式
  - [ ] 变量替换引擎
  - [ ] 模板API
  - [ ] 3-5个官方模板

- [ ] **前端增强（当前技术栈）**
  - [ ] 文件上传组件
  - [ ] 导入进度显示
  - [ ] 模板选择器
  - [ ] 批量操作UI

#### 技术栈

- 后端: Go + Gin + SQLite（保持不变）
- 前端: 当前React（CDN），小幅增强
- 新增库: excelize (Excel), go-yaml (YAML)

#### 交付物

- 支持CSV/Excel/JSON导入测试用例
- 3个官方模板（CRUD、认证、分页）
- 批量执行和结果导出
- 使用文档和示例

#### 成功指标

- 批量导入100个测试用例 < 1分钟
- 使用模板创建测试 < 30秒
- 用户满意度调查 > 8/10

---

### Phase 2: 工作流基础（4-6周）

**目标**: 支持复杂测试场景编排

#### 任务清单

- [ ] **工作流引擎**
  - [ ] 工作流定义格式（YAML）
  - [ ] 工作流解析器
  - [ ] 顺序执行引擎
  - [ ] 状态上下文管理
  - [ ] 依赖解析（拓扑排序）

- [ ] **变量与传递**
  - [ ] 变量替换引擎
  - [ ] 步骤间数据传递
  - [ ] 表达式求值器

- [ ] **错误处理**
  - [ ] 失败策略（abort/continue/retry）
  - [ ] 重试机制

- [ ] **API开发**
  - [ ] 工作流CRUD API
  - [ ] 工作流执行API
  - [ ] 执行历史查询

- [ ] **数据库**
  - [ ] workflows表
  - [ ] workflow_runs表
  - [ ] 迁移脚本

- [ ] **前端（简单版）**
  - [ ] 工作流列表页
  - [ ] 工作流编辑器（YAML文本编辑器）
  - [ ] 执行监控页
  - [ ] 结果展示

#### 技术难点

- 工作流DAG的拓扑排序
- 变量作用域管理
- 长时间运行的工作流状态持久化

#### 交付物

- YAML格式的工作流定义
- 工作流执行引擎
- 简单的工作流编辑UI
- 5个示例工作流

#### 成功指标

- 支持10步骤以上的复杂工作流
- 工作流执行成功率 > 95%
- 步骤间数据传递正确率 100%

---

### Phase 3: 脚本化增强（3-4周）

**目标**: 赋能高级用户，提供编程能力

#### 任务清单

- [ ] **Lua集成**
  - [ ] 集成gopher-lua
  - [ ] 沙箱环境配置
  - [ ] 预定义函数库（http, json, log等）
  - [ ] 执行超时控制
  - [ ] 内存限制

- [ ] **脚本执行**
  - [ ] 脚本步骤类型
  - [ ] 脚本上下文注入
  - [ ] 结果提取

- [ ] **安全机制**
  - [ ] 模块黑名单
  - [ ] 代码审查流程
  - [ ] 权限控制

- [ ] **前端**
  - [ ] 脚本编辑器（Monaco Editor）
  - [ ] 语法高亮
  - [ ] 在线调试
  - [ ] 示例脚本库

#### 技术难点

- Lua沙箱安全性
- Go与Lua数据类型转换
- 脚本调试工具

#### 交付物

- Lua脚本引擎
- 20+预定义函数
- 脚本编辑器
- 10个脚本示例

#### 成功指标

- 脚本执行安全性：0漏洞
- 执行性能：简单脚本 < 100ms
- 用户采纳率 > 30%

---

### Phase 4: 前端现代化（6-8周）

**目标**: 提升用户体验，支持复杂可视化

#### 任务清单

- [ ] **项目搭建**
  - [ ] Vite项目初始化
  - [ ] TypeScript配置
  - [ ] ESLint/Prettier
  - [ ] 路由配置（React Router）
  - [ ] 状态管理（Zustand）

- [ ] **页面迁移**
  - [ ] 测试列表页
  - [ ] 测试编辑页
  - [ ] 分组管理页
  - [ ] 结果查看页

- [ ] **工作流可视化**
  - [ ] 集成React Flow
  - [ ] 拖拽式工作流编辑器
  - [ ] 节点自定义
  - [ ] 自动布局
  - [ ] 实时预览

- [ ] **结果可视化**
  - [ ] 集成ECharts
  - [ ] 测试趋势图
  - [ ] 性能分析图
  - [ ] 调用链图

- [ ] **模板市场**
  - [ ] 模板浏览页
  - [ ] 分类筛选
  - [ ] 搜索功能
  - [ ] 模板预览
  - [ ] 一键克隆

- [ ] **实时监控**
  - [ ] WebSocket集成
  - [ ] 实时日志流
  - [ ] 进度条
  - [ ] 通知中心

#### 技术栈

- Vite 5
- React 18 + TypeScript
- React Router v6
- Zustand (状态管理)
- Ant Design 5
- React Flow (工作流)
- ECharts (图表)
- Monaco Editor (代码编辑)

#### 交付物

- 完整的现代化前端应用
- 可视化工作流编辑器
- 丰富的数据可视化
- 响应式设计

#### 成功指标

- 页面加载时间 < 2秒
- 交互响应时间 < 100ms
- 可视化编辑器上手时间 < 5分钟
- 用户满意度 > 9/10

---

## 风险评估

### 技术风险

| 风险 | 影响 | 概率 | 应对措施 |
|------|------|------|----------|
| **Lua安全漏洞** | 高 | 中 | 严格沙箱、代码审查、限制权限 |
| **工作流执行稳定性** | 高 | 中 | 充分测试、重试机制、状态恢复 |
| **前端重构引入Bug** | 中 | 高 | 渐进式迁移、自动化测试、灰度发布 |
| **性能下降** | 中 | 中 | 性能测试、优化查询、缓存机制 |
| **向后兼容性** | 中 | 低 | 迁移工具、双版本并行、文档清晰 |

### 业务风险

| 风险 | 影响 | 概率 | 应对措施 |
|------|------|------|----------|
| **用户采纳率低** | 高 | 中 | 用户调研、试点验证、培训支持 |
| **过度工程** | 中 | 高 | MVP验证、需求优先级、渐进交付 |
| **资源不足** | 高 | 中 | 合理排期、外部支持、范围控制 |
| **竞品替代** | 中 | 低 | 差异化定位、成本优势、定制化 |

### 项目风险

| 风险 | 影响 | 概率 | 应对措施 |
|------|------|------|----------|
| **范围蔓延** | 高 | 高 | 严格需求管理、版本规划、定期评审 |
| **技术债务** | 中 | 中 | 代码审查、重构计划、文档完善 |
| **团队能力** | 中 | 中 | 技术培训、结对编程、外部咨询 |
| **时间延期** | 中 | 中 | 缓冲时间、里程碑检查、风险预警 |

---

## 附录

### A. 参考案例

#### 商业产品

- **Postman**: API测试工作流、集合管理、环境变量
- **Katalon Studio**: 记录回放、数据驱动、测试套件
- **TestRail**: 测试用例管理、报告、集成
- **Playwright**: 脚本化测试、多浏览器支持

#### 开源项目

- **JMeter**: 性能测试、CSV数据集、断言
- **Robot Framework**: 关键字驱动、扩展性
- **Cypress**: E2E测试、时间旅行调试
- **K6**: 脚本化性能测试、结果可视化

### B. 技术选型对比

#### 工作流引擎

| 方案 | 优势 | 劣势 | 选择 |
|------|------|------|------|
| **Temporal** | 成熟、强大、容错 | 重量级、复杂 | ❌ |
| **Cadence** | 高可用、状态机 | 学习曲线陡 | ❌ |
| **自研轻量引擎** | 可控、轻量、定制 | 功能有限 | ✅ |

#### 脚本语言

| 方案 | 优势 | 劣势 | 选择 |
|------|------|------|------|
| **Lua** | 轻量、嵌入性好、安全 | 生态较小 | ✅ |
| **JavaScript** | 生态丰富、熟悉 | 嵌入复杂 | ❌ |
| **Python** | 强大、库多 | 重量级、难隔离 | ❌ |

#### 前端框架

| 方案 | 优势 | 劣势 | 选择 |
|------|------|------|------|
| **Next.js** | SSR、全栈、优化 | 过度设计、复杂 | ❌ |
| **Vite + React** | 快速、灵活、轻量 | 需配置 | ✅ |
| **Vue 3** | 渐进式、简单 | 团队不熟悉 | ❌ |

### C. 估算与资源

#### 工作量估算

| 阶段 | 任务量（人天） | 测试（人天） | 文档（人天） | 总计 |
|------|--------------|-------------|-------------|------|
| Phase 1 | 12 | 3 | 2 | **17** |
| Phase 2 | 20 | 5 | 3 | **28** |
| Phase 3 | 15 | 4 | 2 | **21** |
| Phase 4 | 30 | 8 | 4 | **42** |
| **总计** | **77** | **20** | **11** | **108** |

按2人团队，约 **3-4个月** 完成全部阶段。

#### 资源需求

- **开发**: 2名全栈工程师（Go + React）
- **测试**: 1名QA工程师（兼职）
- **设计**: UI/UX设计师支持（Phase 4）
- **运维**: DevOps支持（部署、监控）

### D. 成功指标（KPI）

| 指标 | 基线 | 目标 | 测量方式 |
|------|------|------|---------|
| **测试创建效率** | 1个/分钟 | 100个/分钟 | 导入功能性能测试 |
| **工作流执行成功率** | - | > 95% | 执行日志统计 |
| **用户活跃度** | 100% | 120% | DAU/MAU |
| **功能采纳率** | - | > 50% | 新功能使用率 |
| **页面性能** | 5秒 | < 2秒 | Lighthouse评分 |
| **用户满意度** | 7/10 | > 9/10 | NPS调查 |

### E. 术语表

| 术语 | 定义 |
|------|------|
| **DAG** | 有向无环图，用于表示工作流步骤依赖关系 |
| **DSL** | 领域特定语言，如工作流定义YAML |
| **沙箱** | 隔离的执行环境，限制脚本权限 |
| **拓扑排序** | 对DAG节点排序，确保依赖关系满足 |
| **数据驱动测试** | 用不同数据集运行同一测试逻辑 |
| **TTI** | Time to Interactive，页面可交互时间 |
| **SSR** | Server-Side Rendering，服务端渲染 |

---

## 版本历史

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|---------|
| 1.0 | 2025-11-21 | 测试框架团队 | 初始版本 |

---

**文档结束**

# 测试案例与工作流集成 - 详细实现设计

**文档版本**: 1.0
**创建日期**: 2025-11-21
**状态**: 实施中
**相关文档**: PRD.md, USER-STORIES.md, testcase-workflow-integration.md

---

## 1. 设计概览

### 1.1 实施阶段

根据 testcase-workflow-integration.md 的规划，分为5个阶段：

- **Phase 1**: ✅ 数据模型扩展（Week 1）- **已完成**
- **Phase 2**: 执行引擎改造（Week 2-3）- **进行中**
- **Phase 3**: API扩展（Week 4）
- **Phase 4**: 前端UI（Week 5-6）
- **Phase 5**: 文档和培训（Week 7）

### 1.2 当前状态

**已完成**:
- ✅ TestCase model 扩展（workflow_id, workflow_def）
- ✅ Workflow model 创建（含5个关联模型）
- ✅ 数据库迁移脚本（003_add_workflow_integration.sql）
- ✅ UnifiedTestExecutor 基础结构（接口定义）

**进行中**:
- 🔄 UnifiedTestExecutor 完整实现
- 🔄 WorkflowExecutor 实现

**待开始**:
- ⏳ TestCaseAction 实现
- ⏳ API handlers
- ⏳ 前端组件

---

## 2. 架构设计

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        前端层 (Phase 4)                      │
│  - 测试列表（显示workflow类型）                              │
│  - 工作流测试创建表单                                        │
│  - 工作流执行监控页面                                        │
│  - 测试结果详情页（含步骤详情）                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     API层 (Phase 3)                          │
│  - POST /api/v2/tests (支持type=workflow)                   │
│  - POST /api/v2/tests/:id/execute (执行workflow测试)        │
│  - GET /api/v2/tests/:id/results/:rid (查询结果)            │
│  - GET /api/v2/workflows/:id/test-cases (关联查询)          │
│  - WebSocket /api/v2/workflows/runs/:id/stream (实时推送)   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  执行引擎层 (Phase 2)                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ UnifiedTestExecutor (统一执行入口)                   │   │
│  │  - Execute(testCase) → TestResult                   │   │
│  │  - executeHTTP()     ← 已有                         │   │
│  │  - executeCommand()  ← 已有                         │   │
│  │  - executeWorkflowTest() ← 新增                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                              │                               │
│                              ▼                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ WorkflowExecutor (工作流执行)                        │   │
│  │  - Execute(workflowID, workflowDef) → WorkflowResult│   │
│  │  - loadWorkflow() - 加载工作流定义                   │   │
│  │  - parseWorkflow() - 解析YAML/JSON                  │   │
│  │  - buildDAG() - 构建依赖图                          │   │
│  │  - executeSteps() - 执行步骤（支持并行）            │   │
│  │  - manageContext() - 变量管理                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                              │                               │
│                              ▼                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Action Registry (步骤执行器)                        │   │
│  │  - HTTPAction       ← 已有（复用executor.go）       │   │
│  │  - CommandAction    ← 已有（复用executor.go）       │   │
│  │  - TestCaseAction   ← 新增（Mode 3支持）            │   │
│  │  - DatabaseAction   ← 未来扩展                      │   │
│  │  - LuaScriptAction  ← 未来扩展                      │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    数据持久层 (Phase 1) ✅                   │
│  - test_cases (含workflow_id, workflow_def)                 │
│  - workflows                                                 │
│  - workflow_runs                                             │
│  - workflow_step_executions (实时数据流)                     │
│  - workflow_step_logs (实时日志)                             │
│  - workflow_variable_changes (变量历史)                      │
│  - test_results (含workflow_run_id)                          │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 数据流设计

#### 2.2.1 Mode 1: 测试案例引用工作流

```
1. 用户创建测试 (type=workflow, workflowId=xxx)
   ↓
2. 用户执行测试 POST /api/v2/tests/:testId/execute
   ↓
3. UnifiedTestExecutor.Execute(testCase)
   ↓
4. executeWorkflowTest(testCase)
   ├─ 加载: workflowRepo.GetWorkflow(testCase.WorkflowID)
   ├─ 执行: workflowExecutor.Execute(workflowID, nil)
   └─ 转换: WorkflowResult → TestResult
   ↓
5. 保存 test_results (关联workflow_run_id)
   ↓
6. 返回结果给用户
```

#### 2.2.2 Mode 2: 测试案例内嵌工作流

```
1. 用户创建测试 (type=workflow, workflowDef={...})
   ↓
2. 用户执行测试 POST /api/v2/tests/:testId/execute
   ↓
3. UnifiedTestExecutor.Execute(testCase)
   ↓
4. executeWorkflowTest(testCase)
   ├─ 解析: parseWorkflowDef(testCase.WorkflowDef)
   ├─ 执行: workflowExecutor.Execute("", workflowDef)
   └─ 转换: WorkflowResult → TestResult
   ↓
5. 保存 test_results
   ↓
6. 返回结果给用户
```

#### 2.2.3 Mode 3: 工作流引用测试案例

```
1. 用户创建工作流 (steps包含type=test-case)
   ↓
2. 用户执行工作流 POST /api/v2/workflows/:id/execute
   ↓
3. WorkflowExecutor.Execute(workflowID, nil)
   ↓
4. executeSteps() 遇到 type=test-case
   ├─ 调用: TestCaseAction.Execute()
   ├─ 加载: testCaseRepo.GetTestCase(testId)
   ├─ 变量替换: applyInputVariables()
   ├─ 执行: unifiedExecutor.Execute(testCase)
   └─ 提取输出: extractOutput(testResult)
   ↓
5. 保存 workflow_runs, workflow_step_executions
   ↓
6. 返回结果给用户
```

---

## 3. 详细模块设计

### 3.1 UnifiedTestExecutor 详细设计

**文件**: `internal/testcase/executor.go`

#### 3.1.1 结构定义

```go
type UnifiedTestExecutor struct {
    baseURL          string
    client           *http.Client
    workflowExecutor WorkflowExecutor
    testCaseRepo     TestCaseRepository  // 用于加载测试案例
    workflowRepo     WorkflowRepository  // 用于加载工作流
}

type TestCaseRepository interface {
    GetTestCase(testID string) (*models.TestCase, error)
}

type WorkflowRepository interface {
    GetWorkflow(workflowID string) (*models.Workflow, error)
}
```

#### 3.1.2 executeWorkflowTest 方法设计

```go
// executeWorkflowTest 执行工作流类型的测试案例
func (e *UnifiedTestExecutor) executeWorkflowTest(tc *TestCase, result *TestResult) {
    // Step 1: 判断是Mode 1还是Mode 2
    var workflowID string
    var workflowDef interface{}

    if tc.WorkflowID != "" {
        // Mode 1: 引用工作流
        workflowID = tc.WorkflowID

        // 从数据库加载工作流定义
        workflow, err := e.workflowRepo.GetWorkflow(workflowID)
        if err != nil {
            result.Status = "error"
            result.Error = fmt.Sprintf("failed to load workflow: %v", err)
            return
        }
        workflowDef = workflow.Definition
    } else if tc.WorkflowDef != nil {
        // Mode 2: 内嵌工作流定义
        workflowID = fmt.Sprintf("inline-%s", tc.ID)
        workflowDef = tc.WorkflowDef
    } else {
        result.Status = "error"
        result.Error = "no workflow definition found (missing workflowId or workflowDef)"
        return
    }

    // Step 2: 检查workflowExecutor是否可用
    if e.workflowExecutor == nil {
        result.Status = "error"
        result.Error = "workflow executor not configured"
        return
    }

    // Step 3: 执行工作流
    workflowResult, err := e.workflowExecutor.Execute(workflowID, workflowDef)
    if err != nil {
        result.Status = "error"
        result.Error = fmt.Sprintf("workflow execution failed: %v", err)
        return
    }

    // Step 4: 转换WorkflowResult为TestResult
    result.Status = convertWorkflowStatusToTestStatus(workflowResult.Status)
    result.Response = map[string]interface{}{
        "workflowRunId":   workflowResult.RunID,
        "totalSteps":      workflowResult.TotalSteps,
        "completedSteps":  workflowResult.CompletedSteps,
        "failedSteps":     workflowResult.FailedSteps,
        "stepExecutions":  workflowResult.StepExecutions,
        "context":         workflowResult.Context,
    }

    if workflowResult.Error != "" {
        result.Error = workflowResult.Error
    }
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

### 3.2 WorkflowExecutor 详细设计

**文件**: `internal/workflow/executor.go` (新建)

#### 3.2.1 结构定义

```go
type WorkflowExecutorImpl struct {
    actionRegistry   ActionRegistry
    testCaseRepo     TestCaseRepository
    workflowRepo     WorkflowRepository
    unifiedExecutor  *testcase.UnifiedTestExecutor  // 用于TestCaseAction

    // 数据库仓库
    runRepo          WorkflowRunRepository
    stepExecRepo     StepExecutionRepository
    logRepo          StepLogRepository
    varChangeRepo    VariableChangeRepository
}

type ActionRegistry interface {
    GetAction(actionType string) (Action, error)
    RegisterAction(actionType string, action Action)
}

type Action interface {
    Execute(ctx *ActionContext) (*ActionResult, error)
    Validate() error
}

type ActionContext struct {
    StepID          string
    Variables       map[string]interface{}  // 全局变量
    StepOutputs     map[string]interface{}  // 步骤输出
    TestCaseRepo    TestCaseRepository
    UnifiedExecutor *testcase.UnifiedTestExecutor
    Logger          StepLogger
}

type ActionResult struct {
    Status      string  // success, failed
    Output      map[string]interface{}
    Duration    int
    Error       error
}

type StepLogger interface {
    Debug(stepID, message string)
    Info(stepID, message string)
    Warn(stepID, message string)
    Error(stepID, message string)
}
```

#### 3.2.2 Execute 方法设计

```go
func (e *WorkflowExecutorImpl) Execute(workflowID string, workflowDef interface{}) (*WorkflowResult, error) {
    // Step 1: 解析工作流定义
    workflow, err := e.parseWorkflowDefinition(workflowID, workflowDef)
    if err != nil {
        return nil, fmt.Errorf("failed to parse workflow: %w", err)
    }

    // Step 2: 验证工作流（检查循环依赖等）
    if err := e.validateWorkflow(workflow); err != nil {
        return nil, fmt.Errorf("workflow validation failed: %w", err)
    }

    // Step 3: 创建执行记录
    runID := generateRunID()
    run := &models.WorkflowRun{
        RunID:      runID,
        WorkflowID: workflowID,
        Status:     "running",
        StartTime:  time.Now(),
    }
    if err := e.runRepo.Create(run); err != nil {
        return nil, fmt.Errorf("failed to create run record: %w", err)
    }

    // Step 4: 初始化执行上下文
    ctx := &ExecutionContext{
        RunID:         runID,
        Variables:     workflow.Variables,
        StepOutputs:   make(map[string]interface{}),
        StepResults:   make(map[string]*StepExecutionResult),
        Logger:        NewDatabaseStepLogger(e.logRepo, runID),
        VarTracker:    NewVariableChangeTracker(e.varChangeRepo, runID),
    }

    // Step 5: 构建DAG（拓扑排序）
    dag, err := e.buildDAG(workflow.Steps)
    if err != nil {
        e.updateRunStatus(runID, "failed", err.Error())
        return nil, fmt.Errorf("failed to build DAG: %w", err)
    }

    // Step 6: 按层级执行步骤（支持并行）
    for _, layer := range dag.Layers {
        if err := e.executeLayer(ctx, layer, workflow.Steps); err != nil {
            e.updateRunStatus(runID, "failed", err.Error())
            return e.buildWorkflowResult(ctx, run, "failed", err.Error())
        }
    }

    // Step 7: 更新执行记录
    run.Status = "success"
    run.EndTime = time.Now()
    run.Duration = int(run.EndTime.Sub(run.StartTime).Milliseconds())
    run.Context = ctx.Variables
    e.runRepo.Update(run)

    // Step 8: 构建返回结果
    return e.buildWorkflowResult(ctx, run, "success", "")
}
```

#### 3.2.3 executeLayer 方法设计（支持并行）

```go
// executeLayer 执行一个层级的所有步骤（并行执行）
func (e *WorkflowExecutorImpl) executeLayer(ctx *ExecutionContext, layer []string, steps map[string]*WorkflowStep) error {
    var wg sync.WaitGroup
    errorsChan := make(chan error, len(layer))

    for _, stepID := range layer {
        step := steps[stepID]

        // 检查条件执行
        if step.When != "" && !e.evaluateCondition(step.When, ctx) {
            ctx.Logger.Info(stepID, fmt.Sprintf("Step skipped due to condition: %s", step.When))
            continue
        }

        wg.Add(1)
        go func(s *WorkflowStep) {
            defer wg.Done()

            if err := e.executeStep(ctx, s); err != nil {
                errorsChan <- fmt.Errorf("step %s failed: %w", s.ID, err)
            }
        }(step)
    }

    wg.Wait()
    close(errorsChan)

    // 检查是否有错误
    for err := range errorsChan {
        return err  // 返回第一个错误
    }

    return nil
}
```

#### 3.2.4 executeStep 方法设计

```go
// executeStep 执行单个步骤
func (e *WorkflowExecutorImpl) executeStep(ctx *ExecutionContext, step *WorkflowStep) error {
    ctx.Logger.Info(step.ID, fmt.Sprintf("Starting step: %s", step.Name))

    // 创建步骤执行记录
    stepExec := &models.WorkflowStepExecution{
        RunID:     ctx.RunID,
        StepID:    step.ID,
        StepName:  step.Name,
        Status:    "running",
        StartTime: time.Now(),
    }

    // 准备输入数据（变量替换）
    inputData := e.prepareInputData(step, ctx)
    stepExec.InputData = inputData
    e.stepExecRepo.Create(stepExec)

    // 获取Action
    action, err := e.actionRegistry.GetAction(step.Type)
    if err != nil {
        stepExec.Status = "failed"
        stepExec.Error = fmt.Sprintf("unknown action type: %s", step.Type)
        e.stepExecRepo.Update(stepExec)
        return err
    }

    // 构建ActionContext
    actionCtx := &ActionContext{
        StepID:          step.ID,
        Variables:       ctx.Variables,
        StepOutputs:     ctx.StepOutputs,
        TestCaseRepo:    e.testCaseRepo,
        UnifiedExecutor: e.unifiedExecutor,
        Logger:          ctx.Logger,
    }

    // 执行Action
    result, err := e.executeActionWithRetry(action, actionCtx, step)

    // 更新步骤执行记录
    stepExec.EndTime = time.Now()
    stepExec.Duration = int(stepExec.EndTime.Sub(stepExec.StartTime).Milliseconds())

    if err != nil || result.Status == "failed" {
        stepExec.Status = "failed"
        stepExec.Error = fmt.Sprintf("%v", err)
        e.stepExecRepo.Update(stepExec)

        // 错误处理策略
        return e.handleStepError(step, err)
    }

    // 成功
    stepExec.Status = "success"
    stepExec.OutputData = result.Output
    e.stepExecRepo.Update(stepExec)

    // 保存输出变量
    if step.Output != nil {
        e.saveOutputVariables(step, result.Output, ctx)
    }

    ctx.Logger.Info(step.ID, fmt.Sprintf("Step completed successfully in %dms", stepExec.Duration))
    return nil
}
```

### 3.3 TestCaseAction 详细设计

**文件**: `internal/workflow/actions/testcase_action.go` (新建)

#### 3.3.1 结构定义

```go
type TestCaseAction struct {
    TestID string                 `json:"testId"`
    Input  map[string]interface{} `json:"input,omitempty"`
}

func (a *TestCaseAction) Execute(ctx *ActionContext) (*ActionResult, error) {
    ctx.Logger.Info(ctx.StepID, fmt.Sprintf("Executing test case: %s", a.TestID))

    // Step 1: 加载测试案例
    testCase, err := ctx.TestCaseRepo.GetTestCase(a.TestID)
    if err != nil {
        return nil, fmt.Errorf("test case not found: %s", a.TestID)
    }

    // Step 2: 应用输入变量（变量替换）
    testCaseWithInput := a.applyInputVariables(testCase, ctx.Variables, a.Input)

    // Step 3: 执行测试案例
    ctx.Logger.Debug(ctx.StepID, fmt.Sprintf("Invoking UnifiedTestExecutor for test: %s", a.TestID))
    result := ctx.UnifiedExecutor.Execute(testCaseWithInput)

    // Step 4: 转换结果
    if result.Status != "passed" {
        return &ActionResult{
            Status: "failed",
            Error:  fmt.Errorf("test case failed: %s", result.Error),
        }, nil
    }

    // Step 5: 提取响应数据作为输出
    output := map[string]interface{}{
        "testId":   result.TestID,
        "status":   result.Status,
        "duration": result.Duration.Milliseconds(),
        "response": result.Response,  // HTTP响应等
    }

    ctx.Logger.Info(ctx.StepID, fmt.Sprintf("Test case %s completed with status: %s", a.TestID, result.Status))

    return &ActionResult{
        Status:   "success",
        Output:   output,
        Duration: int(result.Duration.Milliseconds()),
    }, nil
}

// applyInputVariables 应用输入变量到测试案例配置
func (a *TestCaseAction) applyInputVariables(
    testCase *models.TestCase,
    contextVars map[string]interface{},
    inputMapping map[string]interface{},
) *testcase.TestCase {
    // 克隆测试案例（避免修改原始数据）
    cloned := &testcase.TestCase{
        ID:         testCase.TestID,
        Name:       testCase.Name,
        Type:       testCase.Type,
        Assertions: convertAssertions(testCase.Assertions),
    }

    // 根据类型应用变量替换
    switch testCase.Type {
    case "http":
        cloned.HTTP = a.replaceHTTPVariables(testCase.HTTPConfig, contextVars, inputMapping)
    case "command":
        cloned.Command = a.replaceCommandVariables(testCase.CommandConfig, contextVars, inputMapping)
    }

    return cloned
}

// replaceHTTPVariables 替换HTTP配置中的变量
func (a *TestCaseAction) replaceHTTPVariables(
    config models.JSONB,
    contextVars map[string]interface{},
    inputMapping map[string]interface{},
) *testcase.HTTPTest {
    // 将JSONB转换为map
    configMap := map[string]interface{}(config)

    // 序列化为JSON字符串
    configJSON, _ := json.Marshal(configMap)
    str := string(configJSON)

    // 替换 {{variableName}} 占位符
    for key, value := range inputMapping {
        placeholder := fmt.Sprintf("{{%s}}", key)
        str = strings.ReplaceAll(str, placeholder, fmt.Sprint(value))
    }

    // 反序列化回HTTP配置
    var httpTest testcase.HTTPTest
    json.Unmarshal([]byte(str), &httpTest)

    return &httpTest
}
```

---

## 4. 实现子任务分解

### 4.1 Phase 2: 执行引擎改造（当前阶段）

#### Task 2.1: 完成UnifiedTestExecutor重构
- **依赖**: 无
- **文件**: `internal/testcase/executor.go`
- **子任务**:
  - ✅ 2.1.1: 重命名Executor → UnifiedTestExecutor
  - ✅ 2.1.2: 添加WorkflowExecutor接口定义
  - ✅ 2.1.3: 添加WorkflowResult和StepExecution结构
  - 🔄 2.1.4: 实现executeWorkflowTest方法
  - ⏳ 2.1.5: 更新所有方法接收者为*UnifiedTestExecutor
  - ⏳ 2.1.6: 添加Repository接口依赖注入

#### Task 2.2: 创建Workflow执行引擎基础架构
- **依赖**: Task 2.1
- **文件**: `internal/workflow/` (新目录)
- **子任务**:
  - ⏳ 2.2.1: 创建workflow包结构
  - ⏳ 2.2.2: 定义WorkflowExecutorImpl结构
  - ⏳ 2.2.3: 定义Action接口和ActionContext
  - ⏳ 2.2.4: 创建ActionRegistry实现
  - ⏳ 2.2.5: 创建StepLogger实现（数据库日志）
  - ⏳ 2.2.6: 创建VariableChangeTracker实现

#### Task 2.3: 实现WorkflowExecutor核心逻辑
- **依赖**: Task 2.2
- **文件**: `internal/workflow/executor.go`
- **子任务**:
  - ⏳ 2.3.1: 实现parseWorkflowDefinition（YAML/JSON解析）
  - ⏳ 2.3.2: 实现validateWorkflow（循环依赖检测）
  - ⏳ 2.3.3: 实现buildDAG（拓扑排序）
  - ⏳ 2.3.4: 实现executeLayer（并行执行）
  - ⏳ 2.3.5: 实现executeStep（单步执行）
  - ⏳ 2.3.6: 实现变量替换引擎
  - ⏳ 2.3.7: 实现条件执行（when表达式）
  - ⏳ 2.3.8: 实现错误处理策略（abort/continue/retry）

#### Task 2.4: 实现TestCaseAction
- **依赖**: Task 2.3
- **文件**: `internal/workflow/actions/testcase_action.go`
- **子任务**:
  - ⏳ 2.4.1: 创建TestCaseAction结构
  - ⏳ 2.4.2: 实现Execute方法
  - ⏳ 2.4.3: 实现applyInputVariables（变量替换）
  - ⏳ 2.4.4: 实现replaceHTTPVariables
  - ⏳ 2.4.5: 实现replaceCommandVariables
  - ⏳ 2.4.6: 注册到ActionRegistry

#### Task 2.5: 实现Repository层
- **依赖**: 无（可并行）
- **文件**: `internal/repository/`
- **子任务**:
  - ⏳ 2.5.1: 创建WorkflowRepository实现
  - ⏳ 2.5.2: 创建TestCaseRepository实现
  - ⏳ 2.5.3: 创建WorkflowRunRepository实现
  - ⏳ 2.5.4: 创建StepExecutionRepository实现
  - ⏳ 2.5.5: 创建StepLogRepository实现
  - ⏳ 2.5.6: 创建VariableChangeRepository实现

#### Task 2.6: 集成测试
- **依赖**: Task 2.1-2.5全部完成
- **子任务**:
  - ⏳ 2.6.1: 编写Mode 1集成测试（引用工作流）
  - ⏳ 2.6.2: 编写Mode 2集成测试（内嵌工作流）
  - ⏳ 2.6.3: 编写Mode 3集成测试（工作流引用测试）
  - ⏳ 2.6.4: 验证并行执行逻辑
  - ⏳ 2.6.5: 验证错误处理策略
  - ⏳ 2.6.6: 验证数据流追踪（输入输出记录）

### 4.2 Phase 3: API扩展

#### Task 3.1: 扩展测试案例API
- **依赖**: Phase 2完成
- **文件**: `internal/api/handlers/testcase_handler.go`
- **子任务**:
  - ⏳ 3.1.1: 更新CreateTestCase支持workflow类型
  - ⏳ 3.1.2: 更新ExecuteTestCase调用UnifiedTestExecutor
  - ⏳ 3.1.3: 更新GetTestResult返回workflow执行详情
  - ⏳ 3.1.4: 添加输入验证（workflowId或workflowDef二选一）

#### Task 3.2: 创建工作流API
- **依赖**: Phase 2完成
- **文件**: `internal/api/handlers/workflow_handler.go`
- **子任务**:
  - ⏳ 3.2.1: 实现CreateWorkflow
  - ⏳ 3.2.2: 实现GetWorkflow
  - ⏳ 3.2.3: 实现ListWorkflows
  - ⏳ 3.2.4: 实现ExecuteWorkflow
  - ⏳ 3.2.5: 实现GetWorkflowRun
  - ⏳ 3.2.6: 实现GetWorkflowTestCases（关联查询）

#### Task 3.3: WebSocket实时推送
- **依赖**: Phase 2完成
- **文件**: `internal/api/websocket/workflow_stream.go`
- **子任务**:
  - ⏳ 3.3.1: 实现WebSocketHub
  - ⏳ 3.3.2: 实现客户端连接管理
  - ⏳ 3.3.3: 实现按runID路由消息
  - ⏳ 3.3.4: 集成WorkflowExecutor事件推送
  - ⏳ 3.3.5: 实现断线重连和历史事件推送

### 4.3 Phase 4: 前端UI

#### Task 4.1: 测试列表页增强
- **依赖**: Phase 3完成
- **文件**: `web/src/components/TestList.tsx`
- **子任务**:
  - ⏳ 4.1.1: 显示workflow类型图标
  - ⏳ 4.1.2: 添加workflow类型筛选
  - ⏳ 4.1.3: 测试卡片显示步骤数

#### Task 4.2: 创建工作流测试表单
- **依赖**: Phase 3完成
- **文件**: `web/src/components/CreateWorkflowTest.tsx`
- **子任务**:
  - ⏳ 4.2.1: 模式选择UI（引用/内嵌）
  - ⏳ 4.2.2: 工作流选择器（下拉列表）
  - ⏳ 4.2.3: YAML编辑器集成（Monaco Editor）
  - ⏳ 4.2.4: 实时语法验证
  - ⏳ 4.2.5: 提交创建

#### Task 4.3: 工作流执行监控页
- **依赖**: Phase 3完成
- **文件**: `web/src/components/WorkflowMonitor.tsx`
- **子任务**:
  - ⏳ 4.3.1: WebSocket连接管理
  - ⏳ 4.3.2: 步骤列表实时更新
  - ⏳ 4.3.3: 变量监控面板
  - ⏳ 4.3.4: 实时日志流
  - ⏳ 4.3.5: 数据流可视化（输入输出）

#### Task 4.4: 工作流测试结果页
- **依赖**: Phase 3完成
- **文件**: `web/src/components/WorkflowTestResult.tsx`
- **子任务**:
  - ⏳ 4.4.1: 步骤详情展示
  - ⏳ 4.4.2: 输入输出数据格式化显示
  - ⏳ 4.4.3: 变量快照展示
  - ⏳ 4.4.4: 完整日志查看
  - ⏳ 4.4.5: 导出报告功能

### 4.4 Phase 5: 文档和培训

#### Task 5.1: 用户文档
- **依赖**: Phase 4完成
- **子任务**:
  - ⏳ 5.1.1: 快速开始指南
  - ⏳ 5.1.2: 三种模式使用说明
  - ⏳ 5.1.3: API文档更新
  - ⏳ 5.1.4: 最佳实践文档

#### Task 5.2: 示例和模板
- **依赖**: Phase 4完成
- **子任务**:
  - ⏳ 5.2.1: 创建10个工作流测试示例
  - ⏳ 5.2.2: 更新模板库

#### Task 5.3: 培训材料
- **依赖**: Phase 4完成
- **子任务**:
  - ⏳ 5.3.1: 录制视频教程
  - ⏳ 5.3.2: 组织在线研讨会

---

## 5. 任务依赖关系图

```
Phase 1 (已完成) ✅
└─ 数据模型扩展
   └─ 数据库迁移脚本

Phase 2 (进行中)
├─ Task 2.1: UnifiedTestExecutor重构 ✅ 部分完成
│  └─ Task 2.2: Workflow执行引擎基础架构
│     ├─ Task 2.3: WorkflowExecutor核心逻辑
│     │  └─ Task 2.4: TestCaseAction实现
│     └─ Task 2.5: Repository层实现（可并行）
│        └─ Task 2.6: 集成测试

Phase 3 (待开始)
└─ Task 3.1: 测试案例API扩展
   ├─ Task 3.2: 工作流API创建
   └─ Task 3.3: WebSocket实时推送

Phase 4 (待开始)
└─ Task 4.1: 测试列表页增强
   ├─ Task 4.2: 创建工作流测试表单
   ├─ Task 4.3: 工作流执行监控页
   └─ Task 4.4: 工作流测试结果页

Phase 5 (待开始)
└─ Task 5.1: 用户文档
   ├─ Task 5.2: 示例和模板
   └─ Task 5.3: 培训材料
```

---

## 6. 实施建议

### 6.1 使用Subagent执行策略

建议将Phase 2的任务分配给subagent并行执行：

**Subagent 1: 执行引擎核心**
- Task 2.1.4-2.1.6: 完成UnifiedTestExecutor
- Task 2.3: 实现WorkflowExecutor核心逻辑

**Subagent 2: Action系统**
- Task 2.2: 创建基础架构
- Task 2.4: 实现TestCaseAction

**Subagent 3: 数据层**
- Task 2.5: 实现所有Repository

**主Agent: 集成和协调**
- 监控各subagent进度
- Task 2.6: 执行集成测试
- 解决跨模块依赖问题

### 6.2 上下文连贯性保证

1. **统一数据模型**: 所有subagent使用相同的models定义
2. **接口契约**: 先定义接口，再并行实现
3. **集成测试**: 每个Phase结束必须通过集成测试
4. **代码审查**: subagent完成后主agent进行代码审查

### 6.3 关键里程碑

- **Milestone 1**: Phase 2完成 → 可执行Mode 1和Mode 2
- **Milestone 2**: Phase 3完成 → API可用，支持Mode 3
- **Milestone 3**: Phase 4完成 → 完整用户体验
- **Milestone 4**: Phase 5完成 → 可推广使用

---

**下一步行动**:
1. 确认设计方案
2. 创建subagent执行计划
3. 开始并行实施Phase 2任务

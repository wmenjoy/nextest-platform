# 测试管理服务 - 数据库设计文档

**版本**: 2.0
**数据库**: SQLite / PostgreSQL / MySQL
**最后更新**: 2025-11-21
**迁移版本**: 004

---

## 目录

1. [概述](#概述)
2. [ER 图](#er-图)
3. [表结构详细说明](#表结构详细说明)
4. [索引策略](#索引策略)
5. [外键约束](#外键约束)
6. [数据迁移](#数据迁移)
7. [查询优化建议](#查询优化建议)
8. [备份策略](#备份策略)

---

## 概述

### 数据库架构

测试管理服务数据库包含以下核心模块：

| 模块 | 表数量 | 说明 |
|------|--------|------|
| **测试管理** | 4 | 测试案例、分组、执行结果、批次 |
| **工作流管理** | 5 | 工作流定义、执行、步骤、日志、变量 |
| **环境管理** | 2 | 环境配置、环境变量 (新增) |
| **总计** | 11 | - |

### 数据库版本

| 版本 | 迁移文件 | 说明 | 日期 |
|------|---------|------|------|
| 001 | `001_initial_schema.sql` | 初始表结构 | - |
| 002 | `002_add_hooks.sql` | 添加生命周期钩子 | - |
| 003 | `003_add_workflow_integration.sql` | 工作流集成 | 2025-11-21 |
| **004** | `004_add_environment_management.sql` | **环境管理** | 2025-11-21 |

---

## ER 图

### 核心实体关系

```
┌──────────────┐         ┌──────────────┐
│  test_groups │◄───────┤  test_cases  │
│              │  1:N    │              │
└──────────────┘         └───────┬──────┘
                                 │ 1:N
                                 ▼
                         ┌──────────────┐
                         │ test_results │
                         └──────────────┘

┌──────────────┐         ┌──────────────┐
│  workflows   │◄───────┤  test_cases  │
│              │  1:N    │ (workflowId) │
└──────┬───────┘         └──────────────┘
       │ 1:N
       ▼
┌──────────────┐
│workflow_runs │
└──────┬───────┘
       │ 1:N
       ▼
┌────────────────────────┐
│workflow_step_executions│
└────────────────────────┘

┌──────────────┐         ┌──────────────────┐
│workflow_runs │◄───────┤workflow_step_logs │
│              │  1:N    └──────────────────┘
│              │         ┌──────────────────────────┐
│              │◄───────┤workflow_variable_changes  │
└──────────────┘  1:N    └──────────────────────────┘

┌──────────────┐         ┌────────────────────────┐
│ environments │◄───────┤environment_variables    │
│              │  1:N    │ (未来扩展)              │
└──────────────┘         └────────────────────────┘

                ┌──────────────┐
                │ environments │ ← 激活环境注入变量
                │ (is_active)  │
                └──────┬───────┘
                       │
                       │ 变量注入
                       ▼
        ┌──────────────────────────────┐
        │  test_cases / workflows      │
        │  ({{VARIABLE_NAME}})         │
        └──────────────────────────────┘
```

### 环境管理与测试执行的关系 (新增)

```
CI/CD Pipeline
      │
      │ 1. 激活环境
      ▼
┌─────────────┐
│ Environment │ (is_active = TRUE)
│   Dev       │ {BASE_URL, API_KEY, ...}
└──────┬──────┘
       │
       │ 2. 获取激活环境变量
       ▼
┌──────────────────┐
│VariableInjector  │
│  InjectVariables │
└──────┬───────────┘
       │
       │ 3. 替换占位符 {{VAR}}
       ▼
┌──────────────────┐
│ Test Execution   │
│  HTTP/Command    │
│  Workflow        │
└──────────────────┘
```


---

## 表结构详细说明

### 1. test_groups - 测试分组表

**用途**: 组织测试案例的层次结构

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| group_id | VARCHAR(255) | UNIQUE, NOT NULL | 业务 ID |
| name | VARCHAR(255) | NOT NULL | 分组名称 |
| parent_id | VARCHAR(255) | | 父分组 ID（支持树形结构）|
| description | TEXT | | 分组描述 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | DATETIME | | 软删除时间 |

**索引**:
- `UNIQUE INDEX idx_test_groups_group_id ON test_groups(group_id)`
- `INDEX idx_test_groups_parent_id ON test_groups(parent_id)`
- `INDEX idx_test_groups_deleted_at ON test_groups(deleted_at)`

**示例数据**:
```sql
INSERT INTO test_groups (group_id, name, parent_id, description) VALUES
('group-root', '根分组', NULL, '顶层分组'),
('group-auth', '认证模块', 'group-root', '用户认证相关测试'),
('group-payment', '支付模块', 'group-root', '支付流程测试');
```

---

### 2. test_cases - 测试案例表

**用途**: 存储所有类型的测试案例（HTTP、命令、工作流）

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| test_id | VARCHAR(255) | UNIQUE, NOT NULL | 业务 ID |
| group_id | VARCHAR(255) | NOT NULL | 所属分组 ID |
| name | VARCHAR(255) | NOT NULL | 测试名称 |
| type | VARCHAR(50) | NOT NULL | 测试类型: http/command/workflow |
| priority | VARCHAR(10) | | 优先级: P0/P1/P2 |
| status | VARCHAR(50) | DEFAULT 'active' | 状态: active/inactive |
| objective | TEXT | | 测试目标描述 |
| timeout | INTEGER | DEFAULT 300 | 超时时间（秒）|
| **workflow_id** | **VARCHAR(255)** | **索引** | **工作流 ID（Mode 1）** |
| **workflow_def** | **TEXT** | | **内嵌工作流定义（Mode 2）** |
| preconditions | TEXT | | 前置条件（JSON 数组）|
| steps | TEXT | | 测试步骤（JSON 数组）|
| http_config | TEXT | | HTTP 配置（JSONB）|
| command_config | TEXT | | 命令配置（JSONB）|
| integration_config | TEXT | | 集成测试配置 |
| performance_config | TEXT | | 性能测试配置 |
| database_config | TEXT | | 数据库测试配置 |
| security_config | TEXT | | 安全测试配置 |
| grpc_config | TEXT | | gRPC 测试配置 |
| websocket_config | TEXT | | WebSocket 测试配置 |
| e2e_config | TEXT | | E2E 测试配置 |
| assertions | TEXT | | 断言列表（JSON 数组）|
| tags | TEXT | | 标签（JSON 数组）|
| custom_config | TEXT | | 自定义配置 |
| setup_hooks | TEXT | | 前置钩子（JSON 数组）|
| teardown_hooks | TEXT | | 后置钩子（JSON 数组）|
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | DATETIME | | 软删除时间 |

**索引**:
- `UNIQUE INDEX idx_test_cases_test_id ON test_cases(test_id)`
- `INDEX idx_test_cases_group_id ON test_cases(group_id)`
- `INDEX idx_test_cases_type ON test_cases(type)`
- `INDEX idx_test_cases_status ON test_cases(status)`
- `INDEX idx_test_cases_deleted_at ON test_cases(deleted_at)`
- **`INDEX idx_test_cases_workflow_id ON test_cases(workflow_id)`** (新增)

**外键**:
- `FOREIGN KEY (group_id) REFERENCES test_groups(group_id)`
- **`FOREIGN KEY (workflow_id) REFERENCES workflows(workflow_id)`** (新增)

**示例数据 - HTTP 测试**:
```sql
INSERT INTO test_cases (
  test_id, group_id, name, type, priority, http_config, assertions
) VALUES (
  'test-login-001',
  'group-auth',
  '用户登录测试',
  'http',
  'P0',
  '{"method":"POST","path":"/api/login","body":{"username":"test","password":"123"}}',
  '[{"type":"status_code","expected":200}]'
);
```

**示例数据 - 工作流测试 (Mode 1)**:
```sql
INSERT INTO test_cases (
  test_id, group_id, name, type, priority, workflow_id
) VALUES (
  'test-workflow-001',
  'group-auth',
  '登录流程工作流测试',
  'workflow',
  'P0',
  'workflow-login'
);
```

**示例数据 - 工作流测试 (Mode 2)**:
```sql
INSERT INTO test_cases (
  test_id, group_id, name, type, priority, workflow_def
) VALUES (
  'test-workflow-002',
  'group-auth',
  '快速登录测试',
  'workflow',
  'P1',
  '{"steps":{"step1":{"id":"step1","type":"http","config":{"method":"POST","path":"/api/login"}}}}'
);
```

---

### 3. test_results - 测试结果表

**用途**: 存储测试执行结果

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| test_id | VARCHAR(255) | NOT NULL | 测试案例 ID |
| run_id | VARCHAR(255) | | 批次执行 ID |
| **workflow_run_id** | **VARCHAR(255)** | **索引** | **工作流执行 ID（新增）** |
| status | VARCHAR(50) | NOT NULL | 执行状态: passed/failed/error/skipped |
| start_time | DATETIME | NOT NULL | 开始时间 |
| end_time | DATETIME | | 结束时间 |
| duration | INTEGER | | 执行时长（毫秒）|
| error | TEXT | | 错误信息 |
| failures | TEXT | | 失败详情（JSON 数组）|
| metrics | TEXT | | 性能指标（JSONB）|
| artifacts | TEXT | | 附件列表（JSON 数组）|
| logs | TEXT | | 日志信息（JSON 数组）|
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引**:
- `INDEX idx_test_results_test_id ON test_results(test_id)`
- `INDEX idx_test_results_run_id ON test_results(run_id)`
- `INDEX idx_test_results_status ON test_results(status)`
- `INDEX idx_test_results_start_time ON test_results(start_time)`
- **`INDEX idx_test_results_workflow_run_id ON test_results(workflow_run_id)`** (新增)

**外键**:
- `FOREIGN KEY (test_id) REFERENCES test_cases(test_id)`
- `FOREIGN KEY (run_id) REFERENCES test_runs(run_id)`

---

### 4. test_runs - 测试批次表

**用途**: 批量测试执行记录

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| run_id | VARCHAR(255) | UNIQUE, NOT NULL | 批次 ID |
| name | VARCHAR(255) | | 批次名称 |
| total | INTEGER | DEFAULT 0 | 总测试数 |
| passed | INTEGER | DEFAULT 0 | 通过数 |
| failed | INTEGER | DEFAULT 0 | 失败数 |
| errors | INTEGER | DEFAULT 0 | 错误数 |
| skipped | INTEGER | DEFAULT 0 | 跳过数 |
| start_time | DATETIME | | 开始时间 |
| end_time | DATETIME | | 结束时间 |
| duration | INTEGER | | 总时长（毫秒）|
| status | VARCHAR(50) | DEFAULT 'running' | running/completed/cancelled |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |

**索引**:
- `UNIQUE INDEX idx_test_runs_run_id ON test_runs(run_id)`
- `INDEX idx_test_runs_status ON test_runs(status)`
- `INDEX idx_test_runs_start_time ON test_runs(start_time)`

---

## 工作流相关表（新增）

### 5. workflows - 工作流定义表

**用途**: 存储工作流定义和元数据

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| workflow_id | VARCHAR(255) | UNIQUE, NOT NULL | 业务 ID |
| name | VARCHAR(255) | NOT NULL | 工作流名称 |
| version | VARCHAR(32) | | 版本号 |
| description | TEXT | | 描述信息 |
| definition | TEXT | NOT NULL | 工作流定义（JSONB）|
| is_test_case | BOOLEAN | DEFAULT 0 | 是否被测试案例引用 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | DATETIME | | 软删除时间 |
| created_by | VARCHAR(64) | | 创建人 |

**索引**:
- `UNIQUE INDEX idx_workflows_workflow_id ON workflows(workflow_id)`
- `INDEX idx_workflows_is_test_case ON workflows(is_test_case)`
- `INDEX idx_workflows_deleted_at ON workflows(deleted_at)`

**Definition 字段结构** (JSONB):
```json
{
  "name": "workflow-name",
  "version": "1.0",
  "variables": {
    "baseUrl": "http://api.example.com",
    "timeout": 30
  },
  "steps": {
    "step1": {
      "id": "step1",
      "name": "步骤名称",
      "type": "http|command|test-case",
      "config": { /* 步骤配置 */ },
      "input": { /* 输入映射 */ },
      "output": { /* 输出映射 */ },
      "dependsOn": ["step0"],
      "when": "{{condition}}",
      "retry": {
        "maxAttempts": 3,
        "interval": 1000
      },
      "onError": "abort|continue"
    }
  }
}
```

**示例数据**:
```sql
INSERT INTO workflows (workflow_id, name, version, definition, is_test_case) VALUES (
  'workflow-login',
  '用户登录流程',
  '1.0',
  '{
    "name": "login-flow",
    "steps": {
      "step1": {
        "id": "step1",
        "name": "登录请求",
        "type": "http",
        "config": {"method": "POST", "path": "/api/login"}
      },
      "step2": {
        "id": "step2",
        "name": "获取用户信息",
        "type": "http",
        "dependsOn": ["step1"],
        "config": {"method": "GET", "path": "/api/user"}
      }
    }
  }',
  1
);
```

---

### 6. workflow_runs - 工作流执行记录表

**用途**: 记录每次工作流执行的顶层信息

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| run_id | VARCHAR(255) | UNIQUE, NOT NULL | 执行 ID（UUID）|
| workflow_id | VARCHAR(255) | NOT NULL | 工作流 ID |
| status | VARCHAR(32) | NOT NULL | running/success/failed/cancelled |
| start_time | DATETIME | NOT NULL | 开始时间 |
| end_time | DATETIME | | 结束时间 |
| duration | INTEGER | | 执行时长（毫秒）|
| context | TEXT | | 执行上下文（JSONB）|
| error | TEXT | | 错误信息 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引**:
- `UNIQUE INDEX idx_workflow_runs_run_id ON workflow_runs(run_id)`
- `INDEX idx_workflow_runs_workflow_id ON workflow_runs(workflow_id)`
- `INDEX idx_workflow_runs_status ON workflow_runs(status)`
- `INDEX idx_workflow_runs_start_time ON workflow_runs(start_time)`

**外键**:
- `FOREIGN KEY (workflow_id) REFERENCES workflows(workflow_id) ON DELETE CASCADE`

**Context 字段结构** (JSONB):
```json
{
  "variables": {
    "token": "eyJhbGci...",
    "userId": 12345
  },
  "outputs": {
    "step1": {
      "status": 200,
      "token": "eyJhbGci..."
    },
    "step2": {
      "status": 200,
      "user": { /* 用户对象 */ }
    }
  }
}
```

**示例数据**:
```sql
INSERT INTO workflow_runs (
  run_id, workflow_id, status, start_time, end_time, duration, context
) VALUES (
  'run-abc-123',
  'workflow-login',
  'success',
  '2025-11-21 10:00:00',
  '2025-11-21 10:00:30',
  30000,
  '{"variables":{"token":"xyz"},"outputs":{"step1":{"status":200}}}'
);
```

---

### 7. workflow_step_executions - 步骤执行记录表

**用途**: 记录每个步骤的执行详情（用于实时数据流追踪）

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| run_id | VARCHAR(255) | NOT NULL | 执行 ID |
| step_id | VARCHAR(255) | NOT NULL | 步骤 ID |
| step_name | VARCHAR(255) | | 步骤名称 |
| status | VARCHAR(32) | NOT NULL | pending/running/success/failed/skipped |
| start_time | DATETIME | | 开始时间 |
| end_time | DATETIME | | 结束时间 |
| duration | INTEGER | | 执行时长（毫秒）|
| input_data | TEXT | | 输入数据快照（JSONB）|
| output_data | TEXT | | 输出数据快照（JSONB）|
| error | TEXT | | 错误信息 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引**:
- `INDEX idx_workflow_step_executions_run_id ON workflow_step_executions(run_id)`
- `INDEX idx_workflow_step_executions_step_id ON workflow_step_executions(step_id)`
- `INDEX idx_workflow_step_executions_status ON workflow_step_executions(status)`

**外键**:
- `FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE`

**用途说明**:
- **实时数据流追踪**: 前端可通过此表查看每个步骤的输入输出
- **调试支持**: 完整保留步骤执行过程中的数据快照
- **审计日志**: 记录步骤级别的执行历史

**示例数据**:
```sql
INSERT INTO workflow_step_executions (
  run_id, step_id, step_name, status, start_time, end_time, duration, input_data, output_data
) VALUES (
  'run-abc-123',
  'step1',
  '登录请求',
  'success',
  '2025-11-21 10:00:00',
  '2025-11-21 10:00:10',
  10000,
  '{"username":"test","password":"***"}',
  '{"status":200,"token":"eyJhbGci..."}'
);
```

---

### 8. workflow_step_logs - 步骤日志表

**用途**: 记录步骤执行过程中的结构化日志

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| run_id | VARCHAR(255) | NOT NULL | 执行 ID |
| step_id | VARCHAR(255) | NOT NULL | 步骤 ID |
| level | VARCHAR(16) | NOT NULL | 日志级别: debug/info/warn/error |
| message | TEXT | NOT NULL | 日志消息 |
| timestamp | DATETIME | NOT NULL | 日志时间 |

**索引**:
- `INDEX idx_workflow_step_logs_run_id ON workflow_step_logs(run_id)`
- `INDEX idx_workflow_step_logs_step_id ON workflow_step_logs(step_id)`
- `INDEX idx_workflow_step_logs_timestamp ON workflow_step_logs(timestamp)`
- `INDEX idx_workflow_step_logs_level ON workflow_step_logs(level)`

**外键**:
- `FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE`

**用途说明**:
- **实时日志流**: 通过 WebSocket 推送给前端
- **调试工具**: 开发人员可查看步骤执行细节
- **监控告警**: 可基于 error 级别日志触发告警

**示例数据**:
```sql
INSERT INTO workflow_step_logs (run_id, step_id, level, message, timestamp) VALUES
('run-abc-123', 'step1', 'info', '开始执行 HTTP 请求', '2025-11-21 10:00:01'),
('run-abc-123', 'step1', 'debug', '请求 URL: POST /api/login', '2025-11-21 10:00:02'),
('run-abc-123', 'step1', 'info', 'HTTP 响应: 200 OK', '2025-11-21 10:00:05'),
('run-abc-123', 'step1', 'info', '步骤完成，耗时: 10000ms', '2025-11-21 10:00:10');
```

---

### 9. workflow_variable_changes - 变量变更历史表

**用途**: 记录工作流执行过程中的变量变化（用于调试和审计）

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| run_id | VARCHAR(255) | NOT NULL | 执行 ID |
| step_id | VARCHAR(255) | | 触发变更的步骤 ID |
| var_name | VARCHAR(255) | NOT NULL | 变量名 |
| old_value | TEXT | | 旧值（JSONB）|
| new_value | TEXT | | 新值（JSONB）|
| change_type | VARCHAR(16) | NOT NULL | 变更类型: create/update/delete |
| timestamp | DATETIME | NOT NULL | 变更时间 |

**索引**:
- `INDEX idx_workflow_variable_changes_run_id ON workflow_variable_changes(run_id)`
- `INDEX idx_workflow_variable_changes_var_name ON workflow_variable_changes(var_name)`
- `INDEX idx_workflow_variable_changes_timestamp ON workflow_variable_changes(timestamp)`

**外键**:
- `FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE`

**用途说明**:
- **调试工具**: 追踪变量何时被修改
- **审计日志**: 完整记录变量生命周期
- **时间旅行**: 可回溯到任意时间点的变量状态

**示例数据**:
```sql
INSERT INTO workflow_variable_changes (
  run_id, step_id, var_name, old_value, new_value, change_type, timestamp
) VALUES
('run-abc-123', 'step1', 'token', NULL, '"eyJhbGci..."', 'create', '2025-11-21 10:00:10'),
('run-abc-123', 'step2', 'userId', NULL, '12345', 'create', '2025-11-21 10:00:20'),
('run-abc-123', 'step3', 'token', '"eyJhbGci..."', '"newToken..."', 'update', '2025-11-21 10:00:30');
```

---

### 10. environments - 环境配置表 (新增)

**用途**: 管理多环境配置（Dev/Staging/Prod），支持环境变量管理和切换

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| env_id | VARCHAR(255) | UNIQUE, NOT NULL | 环境唯一标识符 |
| name | VARCHAR(255) | NOT NULL | 环境名称 |
| description | TEXT | | 环境描述 |
| is_active | BOOLEAN | DEFAULT FALSE | 是否激活（同时只能有一个为TRUE）|
| variables | TEXT | | 环境变量（JSONB格式）|
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | DATETIME | | 软删除时间 |

**索引**:
- `UNIQUE INDEX idx_environments_env_id ON environments(env_id)`
- `INDEX idx_environments_is_active ON environments(is_active)`
- `INDEX idx_environments_deleted_at ON environments(deleted_at)`

**约束**:
- `CHECK (env_id != '')` - 环境ID不能为空字符串
- 应用层保证同一时间只有一个环境 `is_active = TRUE`

**用途说明**:
- **多环境管理**: 支持 Dev、Staging、Prod 等多个环境
- **变量注入**: 通过 `{{VARIABLE_NAME}}` 语法在测试中引用环境变量
- **环境切换**: 激活不同环境以在不同配置下执行测试
- **CI/CD集成**: 支持 Jenkins、GitLab CI 等 CI 工具集成

**示例数据**:
```sql
INSERT INTO environments (
  env_id, name, description, is_active, variables
) VALUES
('dev', 'Development', '开发环境', TRUE,
 '{"BASE_URL":"http://localhost:3000","API_KEY":"dev-key-12345","TIMEOUT":30,"DEBUG":true}'),
('staging', 'Staging', '预发布环境', FALSE,
 '{"BASE_URL":"https://staging.example.com","API_KEY":"staging-key-67890","TIMEOUT":60,"DEBUG":false}'),
('prod', 'Production', '生产环境', FALSE,
 '{"BASE_URL":"https://api.example.com","API_KEY":"prod-key-secret","TIMEOUT":120,"DEBUG":false}');
```

**变量注入示例**:
```json
// 测试配置中使用变量占位符
{
  "http": {
    "method": "POST",
    "path": "{{BASE_URL}}/api/login",
    "headers": {
      "Authorization": "Bearer {{API_KEY}}"
    }
  }
}

// 激活 dev 环境后自动替换为
{
  "http": {
    "method": "POST",
    "path": "http://localhost:3000/api/login",
    "headers": {
      "Authorization": "Bearer dev-key-12345"
    }
  }
}
```

---

### 11. environment_variables - 环境变量表 (新增)

**用途**: 存储环境变量的详细信息（可选，当前版本使用 JSONB 存储在 environments 表中）

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INTEGER | PRIMARY KEY | 主键 |
| env_id | VARCHAR(255) | NOT NULL | 关联的环境 ID |
| key | VARCHAR(255) | NOT NULL | 变量名 |
| value | TEXT | NOT NULL | 变量值 |
| is_secret | BOOLEAN | DEFAULT FALSE | 是否为敏感信息 |
| description | TEXT | | 变量描述 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新时间 |

**索引**:
- `UNIQUE INDEX idx_environment_variables_env_key ON environment_variables(env_id, key)`
- `INDEX idx_environment_variables_env_id ON environment_variables(env_id)`

**外键**:
- `FOREIGN KEY (env_id) REFERENCES environments(env_id) ON DELETE CASCADE`

**用途说明**:
- **扩展存储**: 未来可用于更复杂的变量管理需求
- **敏感信息标记**: 通过 `is_secret` 字段标记敏感变量
- **变量审计**: 记录变量的创建和更新历史
- **变量描述**: 为每个变量添加说明文档

**注意**: 当前版本（v2.0）环境变量直接存储在 `environments.variables` JSONB 字段中，此表为未来扩展预留。

---

## 索引策略

### 索引设计原则

1. **主键自动索引**: 所有表的 `id` 字段自动创建主键索引
2. **唯一索引**: 业务 ID 字段（test_id, workflow_id, run_id）
3. **外键索引**: 所有外键字段创建索引以优化 JOIN 查询
4. **查询优化索引**: 基于常见查询模式创建的复合索引
5. **软删除索引**: deleted_at 字段索引支持软删除过滤

### 索引使用建议

**高频查询场景**:
1. **按状态查询**: `WHERE status = 'active'` → 使用 idx_test_cases_status
2. **按时间范围查询**: `WHERE start_time BETWEEN ... AND ...` → 使用 idx_workflow_runs_start_time
3. **关联查询**: `JOIN workflows ON test_cases.workflow_id = workflows.workflow_id` → 使用 idx_test_cases_workflow_id

**避免索引失效**:
- 不要对索引列使用函数: `WHERE LOWER(name) = 'test'` ❌
- 使用覆盖索引: `SELECT status, name WHERE status = 'active'` ✅
- 避免 `SELECT *`: 指定需要的列 ✅

---

## 外键约束

### 外键关系图

```
test_cases.group_id ──► test_groups.group_id
test_cases.workflow_id ──► workflows.workflow_id (新增)
test_results.test_id ──► test_cases.test_id
test_results.run_id ──► test_runs.run_id

workflow_runs.workflow_id ──► workflows.workflow_id
workflow_step_executions.run_id ──► workflow_runs.run_id
workflow_step_logs.run_id ──► workflow_runs.run_id
workflow_variable_changes.run_id ──► workflow_runs.run_id
```

### 级联删除策略

| 父表 | 子表 | 删除策略 | 原因 |
|------|------|---------|------|
| workflows | workflow_runs | CASCADE | 工作流删除时清理所有执行记录 |
| workflow_runs | workflow_step_executions | CASCADE | 执行删除时清理步骤记录 |
| workflow_runs | workflow_step_logs | CASCADE | 执行删除时清理日志 |
| workflow_runs | workflow_variable_changes | CASCADE | 执行删除时清理变量历史 |
| test_groups | test_cases | RESTRICT | 防止删除有测试的分组 |
| workflows | test_cases | RESTRICT | 防止删除被引用的工作流 |

---

## 数据迁移

### 迁移文件列表

#### 003_add_workflow_integration.sql

**执行顺序**: 在 002 之后

**包含操作**:
1. 扩展 test_cases 表
   ```sql
   ALTER TABLE test_cases ADD COLUMN workflow_id VARCHAR(255) DEFAULT NULL;
   ALTER TABLE test_cases ADD COLUMN workflow_def TEXT DEFAULT NULL;
   CREATE INDEX idx_test_cases_workflow_id ON test_cases(workflow_id);
   ```

2. 创建 workflows 表
3. 创建 workflow_runs 表
4. 创建 workflow_step_executions 表
5. 创建 workflow_step_logs 表
6. 创建 workflow_variable_changes 表
7. 扩展 test_results 表
   ```sql
   ALTER TABLE test_results ADD COLUMN workflow_run_id VARCHAR(255) DEFAULT NULL;
   CREATE INDEX idx_test_results_workflow_run_id ON test_results(workflow_run_id);
   ```

### 迁移执行

**SQLite**:
```bash
sqlite3 test-management.db < migrations/003_add_workflow_integration.sql
```

**PostgreSQL**:
```bash
psql -d test_management -f migrations/003_add_workflow_integration.sql
```

**MySQL**:
```bash
mysql -u root -p test_management < migrations/003_add_workflow_integration.sql
```

### 回滚策略

如需回滚迁移 003:
```sql
-- 删除新增的表
DROP TABLE IF EXISTS workflow_variable_changes;
DROP TABLE IF EXISTS workflow_step_logs;
DROP TABLE IF EXISTS workflow_step_executions;
DROP TABLE IF EXISTS workflow_runs;
DROP TABLE IF EXISTS workflows;

-- 删除新增的列
ALTER TABLE test_cases DROP COLUMN workflow_id;
ALTER TABLE test_cases DROP COLUMN workflow_def;
ALTER TABLE test_results DROP COLUMN workflow_run_id;
```

---

## 查询优化建议

### 1. 常见查询模式

#### 查询工作流测试案例
```sql
-- 优化前
SELECT * FROM test_cases WHERE type = 'workflow';

-- 优化后（使用覆盖索引）
SELECT test_id, name, workflow_id, workflow_def
FROM test_cases
WHERE type = 'workflow' AND deleted_at IS NULL;
```

#### 查询工作流执行历史
```sql
-- 使用索引优化
SELECT run_id, status, start_time, duration
FROM workflow_runs
WHERE workflow_id = 'workflow-login'
ORDER BY start_time DESC
LIMIT 10;

-- 使用索引: idx_workflow_runs_workflow_id, idx_workflow_runs_start_time
```

#### 查询步骤执行记录
```sql
SELECT step_id, step_name, status, duration, input_data, output_data
FROM workflow_step_executions
WHERE run_id = 'run-abc-123'
ORDER BY start_time ASC;

-- 使用索引: idx_workflow_step_executions_run_id
```

### 2. JOIN 查询优化

```sql
-- 查询测试案例及其关联的工作流
SELECT
  tc.test_id,
  tc.name AS test_name,
  w.workflow_id,
  w.name AS workflow_name
FROM test_cases tc
LEFT JOIN workflows w ON tc.workflow_id = w.workflow_id
WHERE tc.type = 'workflow' AND tc.deleted_at IS NULL;

-- 使用索引: idx_test_cases_type, idx_test_cases_workflow_id
```

### 3. 聚合查询优化

```sql
-- 统计工作流执行状态分布
SELECT
  status,
  COUNT(*) AS count,
  AVG(duration) AS avg_duration
FROM workflow_runs
WHERE workflow_id = 'workflow-login'
  AND start_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY status;

-- 使用索引: idx_workflow_runs_workflow_id, idx_workflow_runs_start_time
```

### 4. 分页查询优化

```sql
-- 使用 LIMIT OFFSET 分页
SELECT run_id, status, start_time, duration
FROM workflow_runs
WHERE workflow_id = 'workflow-login'
ORDER BY start_time DESC
LIMIT 20 OFFSET 40;

-- 更优的分页方式（游标分页）
SELECT run_id, status, start_time, duration
FROM workflow_runs
WHERE workflow_id = 'workflow-login'
  AND start_time < '2025-11-21 10:00:00'
ORDER BY start_time DESC
LIMIT 20;
```

---

## 备份策略

### 1. 定期备份

**每日备份**:
```bash
# SQLite
sqlite3 test-management.db ".backup 'backup/test-management-$(date +%Y%m%d).db'"

# PostgreSQL
pg_dump -U postgres test_management > backup/test-management-$(date +%Y%m%d).sql
```

**每周全量备份**:
```bash
# 保留最近 4 周的备份
find backup/ -name "*.db" -mtime +28 -delete
```

### 2. 关键表备份

**工作流定义表**（重要）:
```sql
-- 导出工作流定义
SELECT * FROM workflows WHERE deleted_at IS NULL
INTO OUTFILE 'backup/workflows_$(date +%Y%m%d).csv';
```

### 3. 灾难恢复

**恢复步骤**:
1. 停止应用服务
2. 恢复数据库备份
3. 验证数据完整性
4. 重新启动服务

```bash
# SQLite 恢复
cp backup/test-management-20251121.db test-management.db

# PostgreSQL 恢复
psql -U postgres test_management < backup/test-management-20251121.sql
```

---

## 数据字典

### 测试类型 (test_cases.type)

| 值 | 说明 | 配置字段 |
|---|------|---------|
| http | HTTP/REST API 测试 | http_config |
| command | Shell 命令测试 | command_config |
| **workflow** | **工作流测试（新增）** | **workflow_id 或 workflow_def** |
| integration | 集成测试 | integration_config |
| performance | 性能测试 | performance_config |

### 执行状态 (workflow_runs.status)

| 值 | 说明 | 终态 |
|---|------|------|
| running | 执行中 | ❌ |
| success | 成功 | ✅ |
| failed | 失败 | ✅ |
| cancelled | 取消 | ✅ |

### 步骤状态 (workflow_step_executions.status)

| 值 | 说明 | 终态 |
|---|------|------|
| pending | 等待执行 | ❌ |
| running | 执行中 | ❌ |
| success | 成功 | ✅ |
| failed | 失败 | ✅ |
| skipped | 跳过 | ✅ |

### 日志级别 (workflow_step_logs.level)

| 值 | 说明 | 用途 |
|---|------|------|
| debug | 调试 | 详细执行信息 |
| info | 信息 | 关键步骤记录 |
| warn | 警告 | 潜在问题 |
| error | 错误 | 错误详情 |

---

## 容量规划

### 数据增长预估

假设：
- 测试案例数: 1,000
- 每日工作流执行: 500 次
- 每个工作流平均步骤: 5 个
- 每个步骤平均日志: 10 条

**每日新增数据**:
- workflow_runs: 500 行
- workflow_step_executions: 2,500 行
- workflow_step_logs: 25,000 行
- workflow_variable_changes: ~1,000 行

**每月数据量**:
- ~15,000 工作流执行记录
- ~75,000 步骤执行记录
- ~750,000 日志记录

**存储估算**:
- 每行平均 1KB
- 每月新增: ~820 MB
- 每年新增: ~10 GB

### 数据清理策略

```sql
-- 清理 30 天前的日志
DELETE FROM workflow_step_logs
WHERE timestamp < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- 清理 90 天前的执行记录
DELETE FROM workflow_runs
WHERE end_time < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 归档重要数据到历史表
INSERT INTO workflow_runs_archive
SELECT * FROM workflow_runs
WHERE end_time < DATE_SUB(NOW(), INTERVAL 180 DAY);
```

---

## 维护建议

### 1. 定期维护任务

**每日**:
- 检查数据库连接数
- 监控慢查询日志

**每周**:
- 分析索引使用情况
- 检查表碎片

**每月**:
- 执行 VACUUM (SQLite) 或 VACUUM ANALYZE (PostgreSQL)
- 更新表统计信息
- 检查数据增长趋势

### 2. 监控指标

- 数据库大小
- 表行数增长
- 慢查询数量
- 索引命中率
- 缓存命中率

### 3. 性能调优

**SQLite**:
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB cache
```

**PostgreSQL**:
```sql
-- 定期分析表
ANALYZE workflows;
ANALYZE workflow_runs;
ANALYZE workflow_step_executions;
```

---

## 版本历史

### v2.0 (2025-11-21)
- ✨ 新增 5 个工作流相关表
- ✨ 扩展 test_cases 表支持工作流
- ✨ 扩展 test_results 表关联工作流执行
- 📝 完善数据库文档

### v1.0
- 初始数据库设计
- 测试管理核心表
- HTTP 和命令测试支持

---

**文档维护**: 请在每次数据库 schema 变更后更新此文档
**反馈**: 如发现文档错误或需要补充，请提交 Issue

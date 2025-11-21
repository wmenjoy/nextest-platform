# Test Management Service

独立的测试管理服务，提供完整的测试用例管理、执行和结果跟踪功能。

## 特性

- ✅ 完整的 CRUD 操作（测试用例和分组）
- ✅ 数据库持久化（SQLite）
- ✅ 层次化测试分组
- ✅ 测试执行引擎（HTTP、Command）
- ✅ 测试结果历史记录
- ✅ 批量测试执行
- ✅ RESTful API
- ✅ 可配置的目标服务地址
- ✅ **现代化 Web UI（React + Ant Design）**

## 架构

```
test-management-service/
├── cmd/
│   └── server/          # 服务入口
│       └── main.go
├── internal/
│   ├── config/          # 配置管理
│   ├── models/          # GORM 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   ├── handler/         # HTTP 处理层
│   └── testcase/        # 测试执行器
├── migrations/          # 数据库迁移
├── data/                # SQLite 数据库文件（自动创建）
├── config.toml          # 配置文件
└── go.mod
```

## 快速开始

### 1. 安装依赖

```bash
cd test-management-service
go mod tidy
```

### 2. 配置服务

编辑 `config.toml`:

```toml
[server]
host = "0.0.0.0"
port = 8090

[database]
type = "sqlite"
dsn = "./data/test_management.db"

[test]
target_host = "http://127.0.0.1:9095"  # 被测试服务的地址
```

### 3. 使用 Makefile 快速初始化

```bash
# 一键初始化（安装依赖 + 构建 + 导入示例数据）
make init

# 或者分步执行
make build         # 构建服务
make build-import  # 构建导入工具
make import        # 导入示例测试数据
```

### 4. 运行服务

```bash
make run
# 或者
./test-service
```

服务将在 `http://localhost:8090` 上启动。

### 5. 访问 Web UI

打开浏览器访问：`http://localhost:8090`

Web UI 提供：
- 📊 测试用例可视化管理
- 🗂️ 层次化分组视图
- ▶️ 一键执行测试
- 📈 测试结果实时展示
- 📝 测试历史记录查看

## API 文档

### 测试分组 (Test Groups)

- `POST /api/v2/groups` - 创建分组
- `GET /api/v2/groups/:id` - 获取分组
- `PUT /api/v2/groups/:id` - 更新分组
- `DELETE /api/v2/groups/:id` - 删除分组
- `GET /api/v2/groups/tree` - 获取分组树

### 测试用例 (Test Cases)

- `POST /api/v2/tests` - 创建测试用例
- `GET /api/v2/tests/:id` - 获取测试用例
- `PUT /api/v2/tests/:id` - 更新测试用例
- `DELETE /api/v2/tests/:id` - 删除测试用例
- `GET /api/v2/tests` - 列出测试用例（支持分页）
- `GET /api/v2/tests/search?q=keyword` - 搜索测试用例
- `GET /api/v2/tests/stats` - 获取测试统计信息

### Web UI 专用

- `GET /api/v2/test-tree` - 获取完整测试树（分组+用例）

### 测试执行 (Test Execution)

- `POST /api/v2/tests/:id/execute` - 执行单个测试
- `POST /api/v2/groups/:id/execute` - 执行分组下所有测试

### 测试结果 (Test Results)

- `GET /api/v2/results/:id` - 获取测试结果
- `GET /api/v2/tests/:id/history` - 获取测试历史记录

### 测试批次 (Test Runs)

- `GET /api/v2/runs/:id` - 获取测试批次
- `GET /api/v2/runs` - 列出测试批次

### 健康检查

- `GET /health` - 服务健康检查

## 示例

### 创建测试分组

```bash
curl -X POST http://localhost:8090/api/v2/groups \
  -H "Content-Type: application/json" \
  -d '{
    "groupId": "api-tests",
    "name": "API Tests",
    "description": "API integration tests"
  }'
```

### 创建 HTTP 测试用例

```bash
curl -X POST http://localhost:8090/api/v2/tests \
  -H "Content-Type: application/json" \
  -d '{
    "testId": "test-health-check",
    "groupId": "api-tests",
    "name": "Health Check Test",
    "type": "http",
    "priority": "P0",
    "http": {
      "method": "GET",
      "path": "/health"
    },
    "assertions": [
      {
        "type": "status_code",
        "expected": 200
      }
    ]
  }'
```

### 执行测试

```bash
curl -X POST http://localhost:8090/api/v2/tests/test-health-check/execute
```

### 查看测试历史

```bash
curl http://localhost:8090/api/v2/tests/test-health-check/history
```

## 支持的测试类型

当前实现：
- ✅ HTTP 测试
- ✅ Command 测试

扩展支持（数据库已预留字段）：
- 🔄 Integration 测试
- 🔄 Performance 测试
- 🔄 Database 测试
- 🔄 Security 测试
- 🔄 gRPC 测试
- 🔄 WebSocket 测试
- 🔄 E2E 测试

## 技术栈

- **语言**: Go 1.24
- **Web框架**: Gin
- **ORM**: GORM
- **数据库**: SQLite（支持扩展到 MySQL/PostgreSQL）
- **配置**: TOML

## 开发说明

### 项目特点

1. **清晰的分层架构**：Models → Repository → Service → Handler
2. **数据库持久化**：使用 GORM 和 SQLite
3. **灵活的配置**：TOML 配置文件支持
4. **扩展性设计**：支持多种测试类型的执行
5. **独立部署**：与业务逻辑完全分离

### 数据库

数据库文件存储在 `./data/test_management.db`（可在配置中修改）。

GORM 会自动创建表结构，无需手动运行迁移脚本。

### 自定义 JSON 类型

项目使用自定义的 `JSONB` 和 `JSONArray` 类型来存储复杂的测试配置，支持：
- 自动 JSON 序列化/反序列化
- SQLite TEXT 字段存储
- 类型安全的数据操作

## 下一步计划

- [ ] 添加更多测试类型执行器（Integration, Performance, etc.）
- [ ] Web UI 界面迁移
- [ ] 认证和权限控制
- [ ] 测试报告生成
- [ ] 定时任务执行
- [ ] Webhook 通知
- [ ] 测试数据管理
- [ ] 并发测试执行

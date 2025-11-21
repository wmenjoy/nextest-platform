# 快速入门指南

## 1分钟快速开始

```bash
# 进入项目目录
cd test-management-service

# 一键初始化（构建 + 导入数据）
make init

# 启动服务
make run
```

然后访问 http://localhost:8090

就这么简单！🎉

---

## 详细步骤

### 步骤 1: 配置（可选）

编辑 `config.toml`:

```toml
[server]
port = 8090  # 修改端口

[test]
target_host = "http://your-service:port"  # 修改被测试服务地址
```

### 步骤 2: 构建

```bash
make build
```

这会生成 `test-service` 可执行文件。

### 步骤 3: 导入测试数据（可选）

```bash
make import
```

这会导入 `examples/sample-tests.json` 中的示例测试用例。

### 步骤 4: 启动服务

```bash
./test-service
```

或使用 Make:

```bash
make run
```

### 步骤 5: 使用服务

#### 方式1：Web UI（推荐）

打开浏览器访问：http://localhost:8090

#### 方式2：API

```bash
# 健康检查
curl http://localhost:8090/health

# 获取测试分组树
curl http://localhost:8090/api/v2/groups/tree

# 获取测试用例列表
curl http://localhost:8090/api/v2/tests

# 执行测试
curl -X POST http://localhost:8090/api/v2/tests/health-check/execute
```

---

## 常用命令

```bash
# 显示所有可用命令
make help

# 开发模式（自动重载）
make dev

# 查看服务状态
make health

# 查看测试分组
make api-groups

# 查看测试用例
make api-tests

# 清理并重新开始
make clean-db && make init
```

---

## 创建第一个测试用例

### 方式1：通过 API

```bash
# 1. 创建分组
curl -X POST http://localhost:8090/api/v2/groups \
  -H "Content-Type: application/json" \
  -d '{
    "groupId": "my-tests",
    "name": "My Tests",
    "description": "My first test group"
  }'

# 2. 创建测试用例
curl -X POST http://localhost:8090/api/v2/tests \
  -H "Content-Type: application/json" \
  -d '{
    "testId": "my-first-test",
    "groupId": "my-tests",
    "name": "My First Test",
    "type": "http",
    "priority": "P0",
    "http": {
      "method": "GET",
      "path": "/api/v1/status"
    },
    "assertions": [
      {
        "type": "status_code",
        "expected": 200
      }
    ]
  }'

# 3. 执行测试
curl -X POST http://localhost:8090/api/v2/tests/my-first-test/execute

# 4. 查看结果
curl http://localhost:8090/api/v2/tests/my-first-test/history
```

### 方式2：通过 JSON 文件

1. 创建 `my-tests.json`:

```json
{
  "groups": [
    {
      "groupId": "my-tests",
      "name": "My Tests",
      "description": "My first test group"
    }
  ],
  "tests": [
    {
      "testId": "my-first-test",
      "groupId": "my-tests",
      "name": "My First Test",
      "type": "http",
      "priority": "P0",
      "http": {
        "method": "GET",
        "path": "/api/v1/status"
      },
      "assertions": [
        {
          "type": "status_code",
          "expected": 200
        }
      ]
    }
  ]
}
```

2. 导入：

```bash
./import-tool -config config.toml -data my-tests.json
```

---

## 故障排查

### 问题：端口被占用

```
Error: listen tcp 0.0.0.0:8090: bind: address already in use
```

**解决方案**：修改 `config.toml` 中的端口号。

### 问题：找不到数据库

**解决方案**：数据库会自动创建在 `./data/test_management.db`。确保有写入权限。

### 问题：测试执行失败

```
Error: dial tcp 127.0.0.1:9095: connect: connection refused
```

**解决方案**：
1. 检查被测试服务是否运行
2. 修改 `config.toml` 中的 `target_host`

---

## 下一步

- 📖 阅读 [完整文档](README.md)
- 🔍 查看 [API 文档](README.md#api-文档)
- 🎯 查看 [项目总结](PROJECT_SUMMARY.md)
- 📝 查看 [示例测试用例](examples/sample-tests.json)

---

**需要帮助？** 查看 README.md 或 PROJECT_SUMMARY.md

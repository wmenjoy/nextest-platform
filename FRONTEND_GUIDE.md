# 前端使用指南

## 访问地址

启动服务后，在浏览器中访问：

```
http://localhost:8090
```

自动跳转到基于 Ant Design 的测试管理界面。

---

## 功能介绍

### 1. 首页概览

**统计卡片**:
- 总测试数
- 活跃测试数
- P0 优先级测试数
- P1 优先级测试数

**侧边栏**:
- 显示测试分组树
- 点击分组可筛选测试用例

---

### 2. 测试用例管理

#### 查看测试列表

- 表格显示所有测试用例
- 包含：ID、名称、类型、优先级、状态
- 支持分页（每页10条）

#### 创建测试用例

点击 **"➕ 新建测试"** 按钮：

**基本信息**：
- 测试ID：唯一标识符（必填）
- 分组：选择所属分组（必填）
- 名称：测试名称（必填）
- 类型：HTTP 或 Command（必填）
- 优先级：P0、P1、P2
- 目标：测试目标描述

**HTTP类型配置**：
- HTTP方法：GET、POST、PUT、DELETE
- 请求路径：如 `/api/v1/endpoint`
- 请求头：JSON格式，如 `{"Content-Type": "application/json"}`
- 请求体：JSON格式

**Command类型配置**：
- 命令：如 `echo`
- 参数：空格分隔，如 `hello world`
- 超时：默认60秒

**断言配置**：
JSON数组格式：
```json
[
  {"type": "status_code", "expected": 200},
  {"type": "json_path", "path": "$.status", "expected": "ok"}
]
```

**标签**：
逗号分隔，如：`api, smoke, p0`

#### 编辑测试用例

点击测试行的 **"✏️ 编辑"** 按钮：

- 可修改名称、优先级、目标、配置等
- 测试ID和类型不可修改

#### 删除测试用例

点击 **"🗑️"** 按钮，确认后删除。

---

### 3. 执行测试

#### 单个测试执行

点击 **"▶ 执行"** 按钮：

- 按钮显示loading状态
- 执行完成后显示提示
- 自动刷新列表

#### 查看执行历史

点击 **"📊 历史"** 按钮：

右侧抽屉显示：
- 最近10次执行记录
- 每次执行的状态（PASSED/FAILED/ERROR）
- 执行时间和耗时
- 错误信息（如有）
- 失败原因（如有）

---

### 4. 按分组筛选

在左侧边栏点击分组：
- 表格自动过滤显示该分组的测试
- 标题显示当前分组名称

点击空白区域：
- 显示所有测试用例

---

### 5. 刷新数据

点击右上角 **"🔄 刷新"** 按钮：
- 重新加载分组数据
- 重新加载测试列表
- 重新加载统计信息

---

## 示例操作流程

### 创建HTTP测试用例

1. 点击 "➕ 新建测试"
2. 填写基本信息：
   - 测试ID: `test-api-health`
   - 分组: `integration`
   - 名称: `Health Check API`
   - 类型: `HTTP`
   - 优先级: `P0`

3. 配置HTTP请求：
   - 方法: `GET`
   - 路径: `/health`

4. 添加断言：
   ```json
   [{"type": "status_code", "expected": 200}]
   ```

5. 点击"确定"创建

### 执行并查看结果

1. 在测试列表中找到刚创建的测试
2. 点击 "▶ 执行" 按钮
3. 等待执行完成（按钮loading消失）
4. 点击 "📊 历史" 查看执行结果
5. 查看状态、耗时和详细信息

---

## 断言类型

### HTTP测试断言

```json
[
  {
    "type": "status_code",
    "expected": 200
  },
  {
    "type": "status_code",
    "operator": "in",
    "expected": [200, 201]
  },
  {
    "type": "json_path",
    "path": "$.data.id",
    "operator": "exists"
  },
  {
    "type": "json_path",
    "path": "$.status",
    "expected": "ok"
  }
]
```

### Command测试断言

```json
[
  {
    "type": "exit_code",
    "expected": 0
  },
  {
    "type": "stdout_contains",
    "expected": "success"
  }
]
```

---

## 技术栈

- **React 18.2**：前端框架
- **Ant Design 5.12**：UI组件库
- **Babel Standalone**：浏览器内JSX编译
- **无需构建工具**：直接在浏览器中运行

---

## 浏览器要求

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

推荐使用最新版本的 Chrome 浏览器以获得最佳体验。

---

## 常见问题

### Q: 页面加载慢？
A: 首次加载需要从CDN下载React和Ant Design，请稍等片刻。后续访问会使用浏览器缓存。

### Q: 执行测试后没有反应？
A: 检查被测试服务是否运行，查看 `config.toml` 中的 `target_host` 配置。

### Q: 如何批量执行测试？
A: 当前版本不支持批量执行，可以在API层面调用 `POST /api/v2/groups/:id/execute`。

### Q: 断言JSON格式错误？
A: 使用JSON验证工具检查格式，确保双引号和逗号正确。

### Q: 如何查看更多执行历史？
A: 默认显示最近10条，可以调用API `GET /api/v2/tests/:id/history?limit=50` 查看更多。

---

## 快捷键

当前版本不支持键盘快捷键，所有操作通过鼠标点击完成。

---

## 反馈与建议

如有问题或建议，请查看：
- [README.md](README.md) - 完整文档
- [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) - 项目总结
- [QUICKSTART.md](QUICKSTART.md) - 快速开始

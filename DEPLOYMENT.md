# 部署指南

## 本地开发部署

### 前置要求
- Go 1.24+
- Make（可选，推荐）

### 部署步骤

```bash
# 1. 克隆/进入项目目录
cd test-management-service

# 2. 一键初始化
make init

# 3. 启动服务
make run

# 4. 验证服务
curl http://localhost:8090/health
```

---

## 生产环境部署

### 方式1：直接部署

```bash
# 1. 构建
go build -o test-service ./cmd/server

# 2. 准备配置文件
cat > config.toml <<EOF
[server]
host = "0.0.0.0"
port = 8090

[database]
type = "sqlite"
dsn = "./data/test_management.db"

[test]
target_host = "http://your-production-service:port"
EOF

# 3. 启动服务（后台运行）
nohup ./test-service > app.log 2>&1 &

# 4. 验证
curl http://localhost:8090/health
```

### 方式2：使用 systemd（推荐）

1. 创建服务文件 `/etc/systemd/system/test-management.service`:

```ini
[Unit]
Description=Test Management Service
After=network.target

[Service]
Type=simple
User=app
WorkingDirectory=/opt/test-management-service
Environment="CONFIG_FILE=/opt/test-management-service/config.toml"
ExecStart=/opt/test-management-service/test-service
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

2. 部署：

```bash
# 复制文件
sudo mkdir -p /opt/test-management-service
sudo cp test-service config.toml /opt/test-management-service/
sudo cp -r web /opt/test-management-service/

# 设置权限
sudo useradd -r -s /bin/false app
sudo chown -R app:app /opt/test-management-service

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable test-management
sudo systemctl start test-management

# 查看状态
sudo systemctl status test-management
sudo journalctl -u test-management -f
```

### 方式3：Docker 部署

1. 创建 `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o test-service ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app
COPY --from=builder /app/test-service .
COPY config.toml .
COPY web ./web

EXPOSE 8090

CMD ["./test-service"]
```

2. 构建和运行：

```bash
# 构建镜像
docker build -t test-management-service:latest .

# 运行容器
docker run -d \
  --name test-management \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.toml:/app/config.toml \
  test-management-service:latest

# 查看日志
docker logs -f test-management
```

### 方式4：Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  test-management:
    build: .
    container_name: test-management
    ports:
      - "8090:8090"
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/config.toml
    environment:
      - GIN_MODE=release
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

运行：

```bash
docker-compose up -d
docker-compose logs -f
```

---

## 配置说明

### 基础配置

```toml
[server]
host = "0.0.0.0"  # 监听地址，0.0.0.0 表示所有接口
port = 8090        # 监听端口

[database]
type = "sqlite"                          # 数据库类型
dsn = "./data/test_management.db"       # 数据库连接字符串

[test]
target_host = "http://127.0.0.1:9095"   # 被测试服务的地址
registry_path = ""                       # 测试用例注册路径（可选）
```

### 环境变量

```bash
# 指定配置文件路径
export CONFIG_FILE=/path/to/config.toml

# Gin 运行模式
export GIN_MODE=release  # 生产环境设置为 release
```

---

## 反向代理配置

### Nginx

```nginx
upstream test_management {
    server 127.0.0.1:8090;
}

server {
    listen 80;
    server_name test-management.example.com;

    location / {
        proxy_pass http://test_management;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持（如需要）
    location /ws {
        proxy_pass http://test_management;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Caddy

```caddy
test-management.example.com {
    reverse_proxy localhost:8090
}
```

---

## 数据备份

### SQLite 数据库备份

```bash
# 在线备份
sqlite3 data/test_management.db ".backup 'backup.db'"

# 或简单复制
cp data/test_management.db data/test_management_$(date +%Y%m%d).db

# 定时备份（crontab）
0 2 * * * cd /opt/test-management-service && sqlite3 data/test_management.db ".backup 'backups/backup_$(date +\%Y\%m\%d).db'"
```

---

## 监控和日志

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8090/health

# 预期响应
{"service":"test-management-service","status":"ok"}
```

### 日志查看

```bash
# systemd
sudo journalctl -u test-management -f

# Docker
docker logs -f test-management

# 文件日志（如果使用 nohup）
tail -f app.log
```

### 性能监控

推荐使用：
- Prometheus + Grafana
- ELK Stack
- Datadog

---

## 故障排查

### 端口占用

```bash
# 查看端口占用
lsof -i :8090

# 杀死进程
kill -9 <PID>
```

### 数据库锁定

```bash
# SQLite 被锁定时，找到占用进程
lsof data/test_management.db

# 重启服务
sudo systemctl restart test-management
```

### 权限问题

```bash
# 检查文件权限
ls -la /opt/test-management-service

# 修复权限
sudo chown -R app:app /opt/test-management-service
sudo chmod -R 755 /opt/test-management-service
```

---

## 性能优化

### 数据库优化

对于高负载场景，建议：
1. 迁移到 PostgreSQL 或 MySQL
2. 启用连接池
3. 添加适当的索引

### 服务优化

```bash
# 生产模式运行
export GIN_MODE=release

# 增加文件描述符限制
ulimit -n 65535
```

---

## 安全建议

1. **HTTPS**: 使用 Nginx/Caddy 提供 HTTPS
2. **认证**: 添加 API 认证（待实现）
3. **防火墙**: 限制访问来源
4. **备份**: 定期备份数据库
5. **更新**: 及时更新依赖包

---

## 升级指南

```bash
# 1. 备份数据
cp -r data data.backup

# 2. 停止服务
sudo systemctl stop test-management

# 3. 替换二进制文件
sudo cp test-service /opt/test-management-service/

# 4. 启动服务
sudo systemctl start test-management

# 5. 验证
curl http://localhost:8090/health
```

---

## 完整部署检查清单

- [ ] Go 1.24+ 已安装
- [ ] 配置文件已准备
- [ ] 目标服务地址已配置
- [ ] 数据目录有写入权限
- [ ] 端口未被占用
- [ ] 防火墙规则已配置
- [ ] 反向代理已配置（如需要）
- [ ] 健康检查通过
- [ ] Web UI 可访问
- [ ] API 可正常调用
- [ ] 日志正常输出
- [ ] 备份脚本已配置

---

**需要帮助？** 查看 [README.md](README.md) 或 [QUICKSTART.md](QUICKSTART.md)

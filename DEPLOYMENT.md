# AlertBot 部署指南

## 🚀 快速部署

### 使用 Docker Compose（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/company/alertbot.git
cd alertbot

# 2. 启动所有服务
docker-compose up -d

# 3. 检查服务状态
docker-compose ps
```

### 手动部署

#### 1. 环境准备

**系统要求:**
- Linux/macOS
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Docker (可选)

**安装依赖:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql-14 postgresql-client-14

# macOS
brew install postgresql@14
brew install go node
```

#### 2. 数据库设置

```bash
# 创建数据库用户和数据库
sudo -u postgres psql
CREATE USER alertbot WITH PASSWORD 'your-secure-password';
CREATE DATABASE alertbot OWNER alertbot;
GRANT ALL PRIVILEGES ON DATABASE alertbot TO alertbot;
\q
```

#### 3. 后端部署

```bash
# 构建后端
go mod tidy
go build -o bin/alertbot cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go

# 配置环境变量
export DATABASE_HOST=localhost
export DATABASE_USER=alertbot
export DATABASE_PASSWORD=your-secure-password
export DATABASE_NAME=alertbot
export JWT_SECRET=your-super-secret-jwt-key

# 运行数据库迁移
./bin/migrate

# 启动后端服务
./bin/alertbot
```

#### 4. 前端部署

```bash
# 构建前端
cd web
npm install
npm run build

# 使用 nginx 提供静态文件服务
sudo cp -r dist/* /var/www/html/
```

## 🔧 配置说明

### 环境变量

| 变量名 | 描述 | 默认值 | 必填 |
|--------|------|--------|------|
| `ENV` | 运行环境 | `development` | 否 |
| `SERVER_PORT` | 服务端口 | `8080` | 否 |
| `DATABASE_HOST` | 数据库地址 | `localhost` | 是 |
| `DATABASE_PORT` | 数据库端口 | `5432` | 否 |
| `DATABASE_USER` | 数据库用户名 | `alertbot` | 是 |
| `DATABASE_PASSWORD` | 数据库密码 | - | 是 |
| `DATABASE_NAME` | 数据库名称 | `alertbot` | 是 |
| `JWT_SECRET` | JWT密钥 | - | 是 |

### 配置文件

创建 `configs/config.yaml`:

```yaml
env: production

server:
  port: 8080
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  host: localhost
  port: 5432
  user: alertbot
  password: your-secure-password
  dbname: alertbot
  sslmode: require
  timezone: UTC

logger:
  level: info
  format: json

jwt:
  secret: your-super-secret-jwt-key
  expiration: 24

rate_limit:
  enabled: true
  rps: 1000
```

## 🐳 Docker 部署

### 单机部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: alertbot
      POSTGRES_USER: alertbot
      POSTGRES_PASSWORD: secure-password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  alertbot:
    image: alertbot:latest
    environment:
      DATABASE_HOST: postgres
      DATABASE_PASSWORD: secure-password
      JWT_SECRET: your-jwt-secret
    ports:
      - "8080:8080"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Kubernetes 部署

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertbot
spec:
  replicas: 3
  selector:
    matchLabels:
      app: alertbot
  template:
    metadata:
      labels:
        app: alertbot
    spec:
      containers:
      - name: alertbot
        image: alertbot:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_HOST
          value: "postgresql-service"
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: alertbot-secrets
              key: db-password
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "1000m"

---
apiVersion: v1
kind: Service
metadata:
  name: alertbot-service
spec:
  selector:
    app: alertbot
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 🔍 监控和日志

### Prometheus 监控

AlertBot 内置 Prometheus 指标暴露:

```bash
curl http://localhost:8080/metrics
```

主要指标:
- `alertbot_alerts_received_total` - 接收的告警总数
- `alertbot_alerts_processed_total` - 处理的告警总数
- `alertbot_http_requests_total` - HTTP 请求总数
- `alertbot_http_request_duration_seconds` - 请求耗时

### 日志配置

生产环境建议使用结构化日志:

```yaml
logger:
  level: warn  # debug, info, warn, error
  format: json
```

日志收集示例:
```bash
# 使用 journalctl
sudo journalctl -u alertbot -f

# 使用 Docker
docker-compose logs -f alertbot
```

## 🚦 健康检查

```bash
# 服务健康检查
curl http://localhost:8080/health

# 数据库连接检查
curl http://localhost:8080/api/v1/alerts?size=1
```

## 🔒 安全建议

### 1. 网络安全
- 使用 HTTPS (TLS 1.3)
- 配置防火墙规则
- 使用反向代理 (nginx/traefik)

### 2. 数据库安全
- 启用 SSL 连接
- 定期备份数据
- 限制数据库访问权限

### 3. 应用安全
- 定期更新依赖
- 使用强密码和密钥
- 启用访问日志

### 4. Nginx 配置示例

```nginx
server {
    listen 443 ssl http2;
    server_name alertbot.company.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        root /var/www/alertbot;
        try_files $uri $uri/ /index.html;
    }
}
```

## 📊 性能优化

### 数据库优化

```sql
-- 创建必要的索引
CREATE INDEX CONCURRENTLY idx_alerts_labels_gin ON alerts USING GIN(labels);
CREATE INDEX CONCURRENTLY idx_alerts_created_at_desc ON alerts(created_at DESC);

-- 定期清理历史数据
DELETE FROM alert_history WHERE created_at < NOW() - INTERVAL '30 days';
```

### 应用优化

```yaml
# 增加连接池配置
database:
  max_idle_conns: 10
  max_open_conns: 20
  conn_max_lifetime: 3600

# 启用压缩
server:
  enable_gzip: true
```

## 🔄 备份和恢复

### 数据库备份

```bash
# 每日备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump -h localhost -U alertbot alertbot > /backup/alertbot_$DATE.sql
find /backup -name "alertbot_*.sql" -mtime +7 -delete
```

### 恢复数据

```bash
# 恢复数据库
psql -h localhost -U alertbot alertbot < /backup/alertbot_20250805_120000.sql
```

## 🔍 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查端口占用
   netstat -tlnp | grep :8080
   
   # 检查日志
   journalctl -u alertbot -n 50
   ```

2. **数据库连接失败**
   ```bash
   # 测试数据库连接
   psql -h localhost -U alertbot -d alertbot -c "SELECT 1;"
   ```

3. **前端无法加载**
   ```bash
   # 检查 nginx 配置
   nginx -t
   systemctl reload nginx
   ```

## 📞 技术支持

- 项目仓库: https://github.com/company/alertbot
- 问题反馈: https://github.com/company/alertbot/issues
- 技术文档: https://docs.company.com/alertbot
# AlertBot 告警管理平台

> ⚠️ **此项目正在开发中，功能未完全通过，请勿部署到生产环境**

## 项目概述

AlertBot 是一个现代化的告警管理平台，旨在替代 Prometheus Alertmanager，提供更友好的 Web UI 和强大的告警处理能力。

## 告警处理流程

AlertBot 作为 Prometheus 告警的智能路由系统，工作流程如下：

```
Prometheus 告警 → AlertBot 接收 → 匹配路由规则 → 发送到对应通知渠道
```

### 路由规则示例

- **规则1**: `severity=critical` → 发送到钉钉群 + 短信通知
- **规则2**: `alertname=DatabaseDown` → 发送到 DBA 邮件组
- **规则3**: `instance` 包含 "prod" → 发送到生产环境告警群
- **规则4**: `team=frontend` → 发送到前端团队企业微信群

### 与 Prometheus 的区别

- **Prometheus 规则**: 定义什么时候产生告警（alerting rules）
- **AlertBot 规则**: 定义收到告警后如何路由分发（routing rules）

## 架构特点

- **后端**: Go + Gin + PostgreSQL + GORM
- **前端**: React 18 + TypeScript + Ant Design + Vite
- **容器化**: Docker + Docker Compose
- **简化架构**: 去除Redis依赖，仅使用PostgreSQL

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose

### 启动开发环境

1. **启动数据库**
```bash
docker-compose up -d postgres
```

2. **运行数据库迁移**
```bash
go run cmd/migrate/main.go
```

3. **启动后端服务**
```bash
go run cmd/server/main.go
```

4. **启动前端开发服务器**
```bash
cd web
npm install
npm run dev
```

### 容器化部署

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f alertbot
```

## API 测试

### 健康检查
```bash
curl http://localhost:8080/health
```

### 发送测试告警
```bash
curl -X POST http://localhost:8080/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '[
    {
      "labels": {
        "alertname": "HighCPUUsage",
        "instance": "server1:9100",
        "severity": "warning"
      },
      "annotations": {
        "description": "CPU usage is above 80%",
        "summary": "High CPU usage detected"
      },
      "startsAt": "2025-08-05T10:30:00Z",
      "endsAt": "0001-01-01T00:00:00Z"
    }
  ]'
```

### 查询告警列表
```bash
curl http://localhost:8080/api/v1/alerts
```

## 开发进度

### Phase 1: 基础架构 ✅
- [x] Go项目结构和配置
- [x] React项目结构  
- [x] Docker开发环境
- [x] 数据库设计和迁移
- [x] 基础中间件(日志、CORS、认证)

### Phase 2: 核心功能 🚧
- [x] 告警接收API(兼容Prometheus格式)
- [ ] 规则引擎实现
- [ ] 通知系统集成
- [ ] 前端界面开发

### Phase 3: 高级功能 📋
- [ ] WebSocket实时推送
- [ ] 性能优化
- [ ] 用户权限管理
- [ ] 监控指标暴露

## 项目结构

```
.
├── cmd/                    # 可执行文件
│   ├── server/            # 主服务器
│   └── migrate/           # 数据库迁移
├── internal/              # 内部包
│   ├── api/              # HTTP处理器
│   ├── service/          # 业务逻辑
│   ├── repository/       # 数据访问层
│   ├── models/           # 数据模型
│   ├── middleware/       # 中间件
│   └── config/           # 配置管理
├── pkg/                   # 公共包
│   ├── logger/           # 日志工具
│   └── utils/            # 工具函数
├── web/                   # 前端项目
│   ├── src/              # 源代码
│   └── public/           # 静态资源
├── configs/               # 配置文件
├── migrations/            # 数据库迁移文件
└── docker-compose.yml     # Docker编排文件
```

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
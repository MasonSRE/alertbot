# AlertBot 项目文档

欢迎来到 AlertBot 告警平台项目！这里是项目的完整文档目录。

## 📋 文档目录

### 🚀 [开发计划](./development-plan.md)
详细的 12 周开发排期计划，包含：
- 项目目标与技术选型
- 分阶段开发计划
- 团队分工建议
- 质量保证标准
- 风险评估与成功指标

### 🏗️ [技术规格说明](./technical-specification.md) 
完整的技术架构设计，包含：
- 系统整体架构图
- 核心模块设计
- 数据库设计与索引策略
- 前后端技术规范
- 性能指标与安全规范
- 监控日志与部署方案

### 📡 [API 接口规范](./api-specification.md)
详细的 API 接口文档，包含：
- RESTful API 设计规范
- 告警管理接口
- 规则与通知渠道管理
- WebSocket 实时接口
- 认证授权机制
- 错误码与限流规则

## 🎯 项目概述

AlertBot 是一个高性能的告警管理平台，旨在替代 Prometheus Alertmanager，提供：

- ✅ **现代化 Web UI** - 基于 React + TypeScript + Ant Design
- ✅ **高性能后端** - Go 语言实现，支持 10000+ QPS
- ✅ **智能告警处理** - 去重、聚合、路由、静默
- ✅ **多样化通知** - 钉钉、企微、邮件、短信
- ✅ **可视化配置** - 拖拽式规则编辑器
- ✅ **实时同步** - WebSocket 推送告警状态

## 🏃‍♂️ 快速开始

### 环境要求
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 7+
- Docker & Docker Compose

### 开发环境搭建
```bash
# 克隆项目
git clone https://github.com/company/alertbot.git
cd alertbot

# 启动开发环境
docker-compose up -d

# 初始化数据库
go run cmd/migrate/main.go

# 启动后端服务
go run cmd/server/main.go

# 启动前端开发服务器
cd web && npm install && npm run dev
```

## 🔗 相关链接

- **项目仓库**: https://github.com/company/alertbot
- **演示环境**: https://alertbot-demo.company.com
- **监控面板**: https://grafana.company.com/d/alertbot
- **问题反馈**: https://github.com/company/alertbot/issues

## 📞 联系方式

- **项目负责人**: 待定
- **技术咨询**: dev@company.com
- **紧急联系**: ops@company.com

---

**最后更新**: 2025-08-05  
**文档版本**: v1.0
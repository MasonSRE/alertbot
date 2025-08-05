# AlertBot API 接口规范

## 概述

AlertBot 提供 RESTful API 和 WebSocket 接口，支持告警管理、规则配置、通知渠道管理等功能。

### 基础信息
- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **字符编码**: UTF-8

### 通用响应格式

#### 成功响应
```json
{
  "success": true,
  "data": {...},
  "message": "Success"
}
```

#### 分页响应
```json
{
  "success": true,
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "size": 20,
    "pages": 5
  }
}
```

#### 错误响应
```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid parameters",
    "details": {...}
  }
}
```

## 1. 告警管理接口

### 1.1 接收告警

**接口**: `POST /alerts`  
**描述**: 接收外部告警数据，兼容 Prometheus Alertmanager 格式

#### 请求示例
```http
POST /api/v1/alerts
Content-Type: application/json
Authorization: Bearer <token>

[
  {
    "labels": {
      "alertname": "HighCPUUsage",
      "instance": "server1:9100",
      "job": "node-exporter",
      "severity": "warning"
    },
    "annotations": {
      "description": "CPU usage is above 80% for more than 5 minutes",
      "summary": "High CPU usage detected on server1",
      "runbook_url": "https://wiki.company.com/runbooks/high-cpu"
    },
    "startsAt": "2025-08-05T10:30:00Z",
    "endsAt": "0001-01-01T00:00:00Z",
    "generatorURL": "http://prometheus:9090/graph?g0.expr=..."
  }
]
```

#### 响应示例
```json
{
  "success": true,
  "data": {
    "received": 1,
    "processed": 1,
    "duplicates": 0
  },
  "message": "Alerts processed successfully"
}
```

### 1.2 查询告警列表

**接口**: `GET /alerts`  
**描述**: 获取告警列表，支持筛选和分页

#### 查询参数
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | string | 否 | 告警状态: firing, resolved, silenced |
| severity | string | 否 | 严重程度: critical, warning, info |
| alertname | string | 否 | 告警名称模糊匹配 |
| instance | string | 否 | 实例名称 |
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20，最大 100 |
| sort | string | 否 | 排序字段: created_at, severity, status |
| order | string | 否 | 排序方向: asc, desc，默认 desc |

#### 请求示例
```http
GET /api/v1/alerts?status=firing&severity=critical&page=1&size=20&sort=created_at&order=desc
Authorization: Bearer <token>
```

#### 响应示例
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1001,
        "fingerprint": "f1a2b3c4d5e6f7g8",
        "labels": {
          "alertname": "HighCPUUsage",
          "instance": "server1:9100",
          "severity": "critical"
        },
        "annotations": {
          "description": "CPU usage is above 90%",
          "summary": "Critical CPU usage"
        },
        "status": "firing",
        "severity": "critical",
        "starts_at": "2025-08-05T10:30:00Z",
        "ends_at": null,
        "created_at": "2025-08-05T10:30:15Z",
        "updated_at": "2025-08-05T10:35:20Z"
      }
    ],
    "total": 156,
    "page": 1,
    "size": 20,
    "pages": 8
  }
}
```

### 1.3 获取告警详情

**接口**: `GET /alerts/{fingerprint}`  
**描述**: 根据指纹获取告警详细信息

#### 响应示例
```json
{
  "success": true,
  "data": {
    "id": 1001,
    "fingerprint": "f1a2b3c4d5e6f7g8",
    "labels": {...},
    "annotations": {...},
    "status": "firing",
    "severity": "critical",
    "starts_at": "2025-08-05T10:30:00Z",
    "ends_at": null,
    "created_at": "2025-08-05T10:30:15Z",
    "updated_at": "2025-08-05T10:35:20Z",
    "history": [
      {
        "id": 501,
        "action": "created",
        "details": {},
        "created_at": "2025-08-05T10:30:15Z"
      }
    ]
  }
}
```

### 1.4 告警操作

#### 静默告警
**接口**: `PUT /alerts/{fingerprint}/silence`

```json
{
  "duration": "1h",
  "comment": "Maintenance window"
}
```

#### 确认告警
**接口**: `PUT /alerts/{fingerprint}/ack`

```json
{
  "comment": "Investigating the issue"
}
```

#### 关闭告警
**接口**: `DELETE /alerts/{fingerprint}`

```json
{
  "comment": "Issue resolved"
}
```

## 2. 规则管理接口

### 2.1 获取规则列表

**接口**: `GET /rules`

#### 响应示例
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Critical Alerts",
        "description": "Route critical alerts to on-call team",
        "conditions": {
          "severity": "critical"
        },
        "receivers": [
          {
            "channel_id": 1,
            "template": "critical_template"
          }
        ],
        "priority": 100,
        "enabled": true,
        "created_at": "2025-08-05T09:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "size": 20,
    "pages": 1
  }
}
```

### 2.2 创建规则

**接口**: `POST /rules`

#### 请求示例
```json
{
  "name": "Database Alerts",
  "description": "Route database alerts to DBA team",
  "conditions": {
    "job": "mysql-exporter",
    "severity": ["warning", "critical"]
  },
  "receivers": [
    {
      "channel_id": 2,
      "template": "database_template"
    }
  ],
  "priority": 80,
  "enabled": true
}
```

### 2.3 更新规则

**接口**: `PUT /rules/{id}`

### 2.4 删除规则

**接口**: `DELETE /rules/{id}`

### 2.5 测试规则

**接口**: `POST /rules/test`

#### 请求示例
```json
{
  "conditions": {
    "severity": "critical",
    "alertname": "HighCPUUsage"
  },
  "sample_alert": {
    "labels": {
      "alertname": "HighCPUUsage",
      "severity": "critical",
      "instance": "server1"
    }
  }
}
```

#### 响应示例
```json
{
  "success": true,
  "data": {
    "matched": true,
    "matched_rules": [
      {
        "id": 1,
        "name": "Critical Alerts",
        "priority": 100
      }
    ]
  }
}
```

## 3. 通知渠道接口

### 3.1 获取渠道列表

**接口**: `GET /channels`

#### 响应示例
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "DingTalk On-Call",
        "type": "dingtalk",
        "config": {
          "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=***",
          "secret": "***"
        },
        "enabled": true,
        "created_at": "2025-08-05T08:00:00Z"
      }
    ]
  }
}
```

### 3.2 创建通知渠道

**接口**: `POST /channels`

#### DingTalk 渠道示例
```json
{
  "name": "DingTalk Dev Team",
  "type": "dingtalk",
  "config": {
    "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=abc123",
    "secret": "SEC123456789",
    "at_mobiles": ["13800138000"],
    "at_all": false
  },
  "enabled": true
}
```

#### 企业微信渠道示例
```json
{
  "name": "WeChat Work Ops",
  "type": "wechat_work",
  "config": {
    "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xyz789",
    "mentioned_list": ["@all"]
  },
  "enabled": true
}
```

#### 邮件渠道示例
```json
{
  "name": "Email Notifications",
  "type": "email",
  "config": {
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "username": "alerts@company.com",
    "password": "app_password",
    "from": "alerts@company.com",
    "to": ["admin@company.com", "ops@company.com"],
    "cc": [],
    "bcc": []
  },
  "enabled": true
}
```

### 3.3 测试通知渠道

**接口**: `POST /channels/{id}/test`

#### 请求示例
```json
{
  "message": "This is a test notification from AlertBot"
}
```

#### 响应示例
```json
{
  "success": true,
  "data": {
    "sent": true,
    "response_time": "1.2s",
    "message": "Test notification sent successfully"
  }
}
```

## 4. 静默管理接口

### 4.1 创建静默规则

**接口**: `POST /silences`

#### 请求示例
```json
{
  "matchers": [
    {
      "name": "alertname",
      "value": "HighCPUUsage",
      "is_regex": false
    },
    {
      "name": "instance",
      "value": "server[1-3]",
      "is_regex": true
    }
  ],
  "starts_at": "2025-08-05T12:00:00Z",
  "ends_at": "2025-08-05T18:00:00Z",
  "comment": "Scheduled maintenance"
}
```

### 4.2 获取静默列表

**接口**: `GET /silences`

### 4.3 删除静默规则

**接口**: `DELETE /silences/{id}`

## 5. 统计分析接口

### 5.1 告警统计

**接口**: `GET /stats/alerts`

#### 查询参数
| 参数 | 类型 | 说明 |
|------|------|------|
| start_time | string | 开始时间 (RFC3339) |
| end_time | string | 结束时间 (RFC3339) |
| group_by | string | 分组字段: severity, status, alertname |

#### 响应示例
```json
{
  "success": true,
  "data": {
    "total_alerts": 1205,
    "firing_alerts": 45,
    "resolved_alerts": 1160,
    "groups": [
      {
        "key": "critical",
        "count": 120,
        "percentage": 10.0
      },
      {
        "key": "warning", 
        "count": 800,
        "percentage": 66.4
      }
    ],
    "timeline": [
      {
        "timestamp": "2025-08-05T10:00:00Z",
        "count": 25
      }
    ]
  }
}
```

### 5.2 通知统计

**接口**: `GET /stats/notifications`

#### 响应示例
```json
{
  "success": true,
  "data": {
    "total_sent": 856,
    "success_rate": 98.5,
    "failed_count": 13,
    "channels": [
      {
        "channel_id": 1,
        "channel_name": "DingTalk On-Call",
        "sent": 450,
        "success": 448,
        "failed": 2
      }
    ]
  }
}
```

## 6. WebSocket 实时接口

### 6.1 实时告警推送

**接口**: `WS /ws/alerts`  
**描述**: 实时推送告警状态变化

#### 连接示例
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/alerts?token=jwt_token');

ws.onopen = function() {
    console.log('Connected to AlertBot WebSocket');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```

#### 消息格式
```json
{
  "type": "alert_update",
  "data": {
    "action": "created",
    "alert": {
      "id": 1001,
      "fingerprint": "f1a2b3c4d5e6f7g8",
      "labels": {...},
      "status": "firing",
      "severity": "critical"
    }
  },
  "timestamp": "2025-08-05T10:30:15Z"
}
```

#### 消息类型
- `alert_created`: 新告警创建
- `alert_updated`: 告警状态更新  
- `alert_resolved`: 告警解决
- `alert_silenced`: 告警静默
- `alert_acked`: 告警确认

## 7. 错误码说明

| 错误码 | HTTP状态码 | 说明 |
|--------|------------|------|
| INVALID_REQUEST | 400 | 请求参数无效 |
| UNAUTHORIZED | 401 | 未授权访问 |
| FORBIDDEN | 403 | 权限不足 |
| NOT_FOUND | 404 | 资源不存在 |
| CONFLICT | 409 | 资源冲突 |
| RATE_LIMITED | 429 | 请求频率超限 |
| INTERNAL_ERROR | 500 | 服务器内部错误 |
| SERVICE_UNAVAILABLE | 503 | 服务不可用 |

## 8. 认证授权

### 8.1 获取 Token

**接口**: `POST /auth/login`

#### 请求示例
```json
{
  "username": "admin",
  "password": "password123"
}
```

#### 响应示例
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-08-06T10:30:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 8.2 刷新 Token

**接口**: `POST /auth/refresh`

### 8.3 注销

**接口**: `POST /auth/logout`

## 9. 限流规则

- **普通用户**: 100 请求/分钟
- **管理员**: 500 请求/分钟  
- **告警接收**: 1000 请求/分钟
- **WebSocket 连接**: 10 个/用户

---

**文档版本**: v1.0  
**最后更新**: 2025-08-05  
**联系方式**: dev@company.com
# AlertBot Monitoring and Metrics Guide

## Overview

AlertBot provides comprehensive monitoring and metrics collection capabilities to ensure system health, performance tracking, and operational visibility. This guide covers all monitoring features, metrics, and health checks available in the system.

## Monitoring Features

### 1. Health Checks

#### System Health
- **Endpoint**: `GET /api/v1/monitoring/health`
- **Purpose**: Comprehensive system health status
- **Components Monitored**:
  - Database connectivity and performance
  - Alert service functionality
  - Notification service status
  - WebSocket connections
  - System resources (memory, goroutines)

#### Simple Health Check
- **Endpoint**: `GET /api/v1/monitoring/health/simple`
- **Purpose**: Basic health check for load balancers
- **Response**: Simple HTTP 200/503 status

#### Kubernetes Health Checks
- **Readiness**: `GET /api/v1/monitoring/health/ready`
- **Liveness**: `GET /api/v1/monitoring/health/live`
- **Purpose**: Container orchestration health probes

#### Component Health
- **Endpoint**: `GET /api/v1/monitoring/health/component/{component}`
- **Purpose**: Individual component health status
- **Supported Components**:
  - `database`
  - `alert_service`
  - `notification_service`
  - `websocket`
  - `system`

### 2. Metrics Collection

#### Prometheus Metrics
- **Endpoint**: `GET /metrics`
- **Format**: Prometheus exposition format
- **Categories**:
  - HTTP request metrics
  - Alert processing metrics
  - Database performance metrics
  - System resource metrics
  - Deduplication metrics
  - Security and validation metrics

#### Metrics Summary
- **Endpoint**: `GET /api/v1/monitoring/metrics/summary`
- **Purpose**: Key metrics overview in JSON format

#### Performance Metrics
- **Endpoint**: `GET /api/v1/monitoring/metrics/performance`
- **Purpose**: Detailed performance statistics

### 3. System Information

#### System Info
- **Endpoint**: `GET /api/v1/monitoring/system/info`
- **Returns**:
  - Version information
  - Uptime
  - Go version
  - CPU count
  - Current goroutine count

#### System Metrics
- **Endpoint**: `GET /api/v1/monitoring/metrics/system`
- **Returns**:
  - Memory usage details
  - Database connection stats
  - Alert statistics
  - API performance metrics

## Available Metrics

### Alert Processing Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_alerts_received_total` | Counter | Total alerts received | `status`, `severity` |
| `alertbot_alerts_processed_total` | Counter | Total alerts processed | `action`, `status` |
| `alertbot_alert_processing_duration_seconds` | Histogram | Alert processing time | `operation` |
| `alertbot_active_alerts` | Gauge | Current active alerts | `severity`, `status` |

### Deduplication Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_deduplication_processed_total` | Counter | Deduplication operations | `action` |
| `alertbot_deduplication_duplicates_found_total` | Counter | Duplicate alerts found | `deduplication_type` |
| `alertbot_deduplication_correlations_found_total` | Counter | Alert correlations found | - |
| `alertbot_deduplication_processing_duration_seconds` | Histogram | Deduplication processing time | - |
| `alertbot_active_deduplication_windows` | Gauge | Active deduplication windows | - |

### HTTP/API Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_http_requests_total` | Counter | Total HTTP requests | `method`, `endpoint`, `status_code` |
| `alertbot_http_request_duration_seconds` | Histogram | HTTP request duration | `method`, `endpoint` |
| `alertbot_api_endpoint_requests_total` | Counter | API endpoint requests | `endpoint`, `method`, `status` |
| `alertbot_api_endpoint_duration_seconds` | Histogram | API endpoint response time | `endpoint`, `method` |

### Database Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_database_connections` | Gauge | Database connections | `state` |
| `alertbot_database_query_duration_seconds` | Histogram | Database query duration | `operation`, `table` |
| `alertbot_database_errors_total` | Counter | Database errors | `operation`, `error_type` |

### System Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_memory_bytes` | Gauge | Memory usage | `type` |
| `alertbot_goroutines` | Gauge | Current goroutines | - |
| `alertbot_uptime_seconds_total` | Counter | Total uptime | - |
| `alertbot_service_health` | Gauge | Service health status | `service` |
| `alertbot_service_response_time_seconds` | Histogram | Service response time | `service` |

### Notification Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_notifications_sent_total` | Counter | Notifications sent | `channel_type`, `status` |
| `alertbot_notification_duration_seconds` | Histogram | Notification send time | `channel_type` |
| `alertbot_notification_errors_total` | Counter | Notification errors | `channel_type`, `error_type` |

### Security Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_rate_limited_requests_total` | Counter | Rate limited requests | `client_ip` |
| `alertbot_validation_errors_total` | Counter | Validation errors | `type`, `field` |
| `alertbot_security_events_total` | Counter | Security events | `event_type` |

### Background Job Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `alertbot_background_jobs_total` | Counter | Background jobs executed | `job_name`, `status` |
| `alertbot_background_job_duration_seconds` | Histogram | Background job duration | `job_name` |

## Health Check Components

### Database Health Checker
- **Checks**:
  - Basic connectivity (ping)
  - Query execution capability
  - Connection pool status
  - Connection usage percentage
- **Status Levels**:
  - `healthy`: All checks pass
  - `degraded`: High connection usage (>90%)
  - `unhealthy`: Connectivity or query failures

### System Health Checker
- **Checks**:
  - Memory usage
  - Goroutine count
  - GC statistics
- **Thresholds**:
  - Memory: Warning at 1GB heap usage
  - Goroutines: Warning at 10,000 goroutines

### Alert Service Health Checker
- **Checks**:
  - Repository availability
  - Query functionality
  - Recent alert activity
  - Query performance
- **Performance Threshold**: Query time >1 second = degraded

### Notification Health Checker
- **Checks**:
  - Notification channels configuration
  - Enabled channels count
  - Routing rules status
- **Requirements**:
  - At least one enabled channel
  - At least one enabled routing rule

## Background Monitoring

### Automatic Metrics Collection
- **System Metrics**: Every 30 seconds
- **Database Metrics**: Every 60 seconds
- **Alert Metrics**: Every 30 seconds
- **Performance Checks**: Every 15 seconds

### Cleanup Tasks
- **Frequency**: Every 24 hours
- **Operations**:
  - Old metrics cleanup
  - Alert history cleanup
  - Garbage collection

### Performance Monitoring
- **Memory Threshold**: 1GB heap usage
- **Goroutine Threshold**: 10,000 goroutines
- **Database Connection Threshold**: 80% of max connections
- **Query Performance Threshold**: 1 second

## Configuration

### Monitoring Configuration
```json
{
  "health_check_interval": "30s",
  "metrics_collection_interval": "15s",
  "alert_thresholds": {
    "high_memory_usage": 500,
    "high_cpu_usage": 80,
    "high_database_connections": 90,
    "high_response_time": "5s",
    "low_disk_space": 10
  },
  "enable_system_metrics": true,
  "enable_database_metrics": true,
  "enable_service_metrics": true
}
```

### Endpoints for Configuration
- **Get Config**: `GET /api/v1/monitoring/config`
- **Update Config**: `PUT /api/v1/monitoring/config`

## Integration Examples

### Prometheus Configuration
```yaml
scrape_configs:
  - job_name: 'alertbot'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Kubernetes Deployment
```yaml
apiVersion: v1
kind: Service
metadata:
  name: alertbot
spec:
  ports:
  - port: 8080
    name: http
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertbot
spec:
  template:
    spec:
      containers:
      - name: alertbot
        image: alertbot:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /api/v1/monitoring/health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/v1/monitoring/health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Grafana Dashboard Queries
```promql
# Alert processing rate
rate(alertbot_alerts_processed_total[5m])

# Memory usage
alertbot_memory_bytes{type="heap_alloc"}

# Database connection usage
alertbot_database_connections{state="open"} / alertbot_database_connections{state="max"} * 100

# API response time
histogram_quantile(0.95, rate(alertbot_api_endpoint_duration_seconds_bucket[5m]))

# Deduplication efficiency
rate(alertbot_deduplication_duplicates_found_total[5m]) / rate(alertbot_deduplication_processed_total[5m]) * 100
```

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Check `alertbot_memory_bytes{type="heap_alloc"}`
   - Monitor goroutine count
   - Review alert processing volume

2. **Database Performance**
   - Monitor `alertbot_database_query_duration_seconds`
   - Check connection pool usage
   - Review slow query logs

3. **API Latency**
   - Monitor `alertbot_api_endpoint_duration_seconds`
   - Check system resource usage
   - Review rate limiting configuration

4. **Deduplication Issues**
   - Monitor `alertbot_deduplication_processing_duration_seconds`
   - Check duplicate rates
   - Review deduplication configuration

### Health Check Failures

- **Database**: Check connection string, network connectivity
- **Alert Service**: Verify repository initialization
- **System**: Check memory and goroutine thresholds
- **Notifications**: Verify channel configuration

## Best Practices

1. **Monitoring Setup**
   - Use Prometheus for metrics collection
   - Set up Grafana for visualization
   - Configure alerting rules for critical metrics

2. **Performance Tuning**
   - Monitor deduplication efficiency
   - Optimize database queries
   - Tune connection pool settings

3. **Health Checks**
   - Use different health check endpoints appropriately
   - Set reasonable timeouts for health probes
   - Monitor health check performance

4. **Alerting**
   - Set up alerts for high error rates
   - Monitor resource usage trends
   - Alert on service health degradation

This comprehensive monitoring system ensures operational visibility and helps maintain high availability and performance of the AlertBot system.
#!/bin/bash

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "=== AlertBot API 测试脚本 ==="
echo "测试地址: $BASE_URL"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -n "测试: $description ... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "%{http_code}" -o /tmp/response.json "$API_URL$endpoint")
    elif [ "$method" = "POST" ]; then
        response=$(curl -s -w "%{http_code}" -o /tmp/response.json -X POST -H "Content-Type: application/json" -d "$data" "$API_URL$endpoint")
    fi
    
    http_code="${response: -3}"
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}✓ 成功 ($http_code)${NC}"
        return 0
    else
        echo -e "${RED}✗ 失败 ($http_code)${NC}"
        echo "响应内容:"
        cat /tmp/response.json
        echo ""
        return 1
    fi
}

# 1. 健康检查
echo -e "${YELLOW}=== 1. 基础健康检查 ===${NC}"
test_endpoint "GET" "" "" "健康检查" "$BASE_URL/health"

# 健康检查
echo -n "测试: 健康检查 ... "
response=$(curl -s -w "%{http_code}" -o /tmp/health.json "$BASE_URL/health")
http_code="${response: -3}"

if [ "$http_code" = "200" ]; then
    echo -e "${GREEN}✓ 成功${NC}"
    cat /tmp/health.json | python3 -m json.tool 2>/dev/null || cat /tmp/health.json
    echo ""
else
    echo -e "${RED}✗ 失败 ($http_code)${NC}"
    echo "无法连接到服务器，请确保服务正在运行"
    exit 1
fi

echo ""

# 2. 发送测试告警
echo -e "${YELLOW}=== 2. 发送测试告警 ===${NC}"

test_alert='{
  "labels": {
    "alertname": "TestAPIAlert",
    "instance": "test-server:9100",
    "severity": "warning",
    "job": "api-test",
    "service": "alertbot"
  },
  "annotations": {
    "description": "这是通过API测试脚本发送的测试告警",
    "summary": "API测试告警",
    "runbook_url": "https://github.com/company/alertbot"
  },
  "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
  "endsAt": "0001-01-01T00:00:00Z",
  "generatorURL": "http://test-script"
}'

echo "发送告警数据:"
echo "$test_alert" | python3 -m json.tool 2>/dev/null || echo "$test_alert"
echo ""

test_endpoint "POST" "/alerts" "[$test_alert]" "发送测试告警"
echo ""

# 3. 获取告警列表
echo -e "${YELLOW}=== 3. 获取告警列表 ===${NC}"
test_endpoint "GET" "/alerts?size=5" "" "获取告警列表"

if [ $? -eq 0 ]; then
    echo "返回数据:"
    cat /tmp/response.json | python3 -m json.tool 2>/dev/null || cat /tmp/response.json
fi
echo ""

# 4. 测试其他API端点
echo -e "${YELLOW}=== 4. 测试其他API端点 ===${NC}"
test_endpoint "GET" "/rules" "" "获取规则列表"
test_endpoint "GET" "/channels" "" "获取通知渠道列表"
test_endpoint "GET" "/silences" "" "获取静默列表"
test_endpoint "GET" "/stats/alerts" "" "获取告警统计"
echo ""

# 5. 发送更多测试告警
echo -e "${YELLOW}=== 5. 发送多种类型测试告警 ===${NC}"

# 严重告警
critical_alert='{
  "labels": {
    "alertname": "CriticalTestAlert",
    "instance": "prod-server:9100",
    "severity": "critical",
    "job": "api-test"
  },
  "annotations": {
    "description": "严重级别的测试告警",
    "summary": "严重告警测试"
  },
  "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
  "endsAt": "0001-01-01T00:00:00Z"
}'

test_endpoint "POST" "/alerts" "[$critical_alert]" "发送严重告警"

# 信息告警
info_alert='{
  "labels": {
    "alertname": "InfoTestAlert", 
    "instance": "dev-server:9100",
    "severity": "info",
    "job": "api-test"
  },
  "annotations": {
    "description": "信息级别的测试告警",
    "summary": "信息告警测试"
  },
  "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
  "endsAt": "0001-01-01T00:00:00Z"
}'

test_endpoint "POST" "/alerts" "[$info_alert]" "发送信息告警"
echo ""

# 6. 验证告警创建
echo -e "${YELLOW}=== 6. 验证告警创建结果 ===${NC}"
test_endpoint "GET" "/alerts?size=10&sort=created_at&order=desc" "" "获取最新告警"

if [ $? -eq 0 ]; then
    echo "最新告警列表:"
    cat /tmp/response.json | python3 -m json.tool 2>/dev/null || cat /tmp/response.json
fi
echo ""

# 清理临时文件
rm -f /tmp/response.json /tmp/health.json

echo -e "${GREEN}=== API 测试完成 ===${NC}"
echo ""
echo "后续测试建议:"
echo "1. 访问前端界面: http://localhost:3000"
echo "2. 查看告警管理页面确认告警已创建"
echo "3. 测试告警操作 (静默、确认、解决)"
echo "4. 检查数据库中的数据"
echo ""
echo "数据库查询示例:"
echo "  docker-compose exec postgres psql -U alertbot -d alertbot -c \"SELECT id, labels->>'alertname' as name, severity, status, created_at FROM alerts ORDER BY created_at DESC LIMIT 5;\""
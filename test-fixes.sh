#!/bin/bash

echo "=== AlertBot 修复验证脚本 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 基础URL
BASE_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"

echo -e "${YELLOW}1. 检查后端API状态${NC}"
echo "检查 silences API..."
response=$(curl -s -w "%{http_code}" -o /tmp/test_silences.json "${BASE_URL}/api/v1/silences")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Silences API 工作正常${NC}"
else
    echo -e "${RED}✗ Silences API 返回状态码: $response${NC}"
fi

echo "检查 inhibitions API..."
response=$(curl -s -w "%{http_code}" -o /tmp/test_inhibitions.json "${BASE_URL}/api/v1/inhibitions")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Inhibitions API 工作正常${NC}"
else
    echo -e "${RED}✗ Inhibitions API 返回状态码: $response${NC}"
fi

echo "检查 alert-groups API..."
response=$(curl -s -w "%{http_code}" -o /tmp/test_alert_groups.json "${BASE_URL}/api/v1/alert-groups")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Alert Groups API 工作正常${NC}"
else
    echo -e "${RED}✗ Alert Groups API 返回状态码: $response${NC}"
fi

echo ""
echo -e "${YELLOW}2. 检查前端页面访问${NC}"

pages=("silences" "inhibitions" "alert-groups")
for page in "${pages[@]}"; do
    echo "检查 /$page 页面..."
    response=$(curl -s -w "%{http_code}" -o /dev/null "${FRONTEND_URL}/$page")
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ /$page 页面可访问${NC}"
    else
        echo -e "${RED}✗ /$page 页面返回状态码: $response${NC}"
    fi
done

echo ""
echo -e "${YELLOW}3. 检查API响应格式${NC}"
echo "Silences API 数据结构:"
if [ -f "/tmp/test_silences.json" ]; then
    jq -r '.data | keys[]' /tmp/test_silences.json 2>/dev/null || echo "JSON 解析失败"
fi

echo "Inhibitions API 数据结构:"
if [ -f "/tmp/test_inhibitions.json" ]; then
    jq -r '.data | type' /tmp/test_inhibitions.json 2>/dev/null || echo "JSON 解析失败"  
fi

echo "Alert Groups API 数据结构:"
if [ -f "/tmp/test_alert_groups.json" ]; then
    jq -r '.data | keys[]' /tmp/test_alert_groups.json 2>/dev/null || echo "JSON 解析失败"
fi

echo ""
echo -e "${GREEN}=== 验证完成 ===${NC}"
echo "如果所有检查都显示 ✓，则修复成功！"
echo "现在可以访问 http://localhost:3000 查看修复后的页面。"
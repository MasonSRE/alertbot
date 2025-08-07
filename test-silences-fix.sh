#!/bin/bash

echo "=== Silences页面修复验证 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"

echo -e "${YELLOW}1. 检查后端API数据结构${NC}"
echo "调用 silences API..."
response=$(curl -s "${BASE_URL}/api/v1/silences")
echo "API响应:"
echo "$response" | jq .

echo ""
echo -e "${YELLOW}2. 检查数据结构是否正确${NC}"
# 检查是否有 items 字段
items_exists=$(echo "$response" | jq -r '.data.items | type' 2>/dev/null)
if [ "$items_exists" = "array" ]; then
    echo -e "${GREEN}✓ API返回正确的数据结构 (items是数组)${NC}"
    
    # 检查 items 数量
    items_count=$(echo "$response" | jq -r '.data.items | length' 2>/dev/null)
    echo "数据项数量: $items_count"
    
    # 如果有数据，检查第一个项目的结构
    if [ "$items_count" -gt 0 ]; then
        echo "第一个silence项目结构:"
        echo "$response" | jq -r '.data.items[0] | keys[]' 2>/dev/null | sort
        
        # 检查matchers结构
        matchers_struct=$(echo "$response" | jq -r '.data.items[0].matchers | type' 2>/dev/null)
        echo "matchers字段类型: $matchers_struct"
        
        if [ "$matchers_struct" = "object" ]; then
            matchers_array=$(echo "$response" | jq -r '.data.items[0].matchers.matchers | type' 2>/dev/null)
            echo "matchers.matchers字段类型: $matchers_array"
            if [ "$matchers_array" = "array" ]; then
                echo -e "${GREEN}✓ Matchers结构正确${NC}"
            else
                echo -e "${RED}✗ Matchers结构错误${NC}"
            fi
        fi
    else
        echo "没有silence数据，创建测试数据..."
        test_data='{
          "matchers": {
            "matchers": [
              {
                "name": "alertname",
                "value": "TestAlert",
                "is_regex": false
              }
            ]
          },
          "starts_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
          "ends_at": "'$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ)'",
          "creator": "test-script",
          "comment": "Test silence for debugging"
        }'
        
        echo "创建测试silence..."
        create_response=$(curl -s -X POST "${BASE_URL}/api/v1/silences" \
          -H "Content-Type: application/json" \
          -d "$test_data")
        echo "创建结果: $create_response"
    fi
else
    echo -e "${RED}✗ API返回错误的数据结构${NC}"
    echo "预期: { data: { items: [...] } }"
    echo "实际: $response"
fi

echo ""
echo -e "${YELLOW}3. 检查前端页面${NC}"
echo "访问 silences 页面..."
frontend_response=$(curl -s -w "%{http_code}" -o /tmp/silences_page.html "${FRONTEND_URL}/silences")

if [ "$frontend_response" = "200" ]; then
    echo -e "${GREEN}✓ Silences页面可访问 (HTTP 200)${NC}"
    
    # 检查是否包含正确的JavaScript文件
    js_file=$(grep -o 'index-[a-f0-9]\+\.js' /tmp/silences_page.html | head -1)
    if [ -n "$js_file" ]; then
        echo "使用的JavaScript文件: $js_file"
        echo -e "${GREEN}✓ 页面使用了新的JavaScript构建${NC}"
    else
        echo -e "${RED}✗ 无法检测JavaScript文件${NC}"
    fi
    
    # 检查是否还有旧的缓存文件引用
    old_js=$(grep -o 'index-dc2ef0a4\.js' /tmp/silences_page.html)
    if [ -n "$old_js" ]; then
        echo -e "${RED}✗ 页面仍在使用旧的JavaScript文件: $old_js${NC}"
        echo "需要强制刷新浏览器缓存"
    else
        echo -e "${GREEN}✓ 页面没有引用旧的JavaScript文件${NC}"
    fi
    
else
    echo -e "${RED}✗ Silences页面访问失败 (HTTP $frontend_response)${NC}"
fi

echo ""
echo -e "${YELLOW}4. 浏览器缓存清理建议${NC}"
echo "如果页面仍然报错，请尝试以下操作："
echo "1. 按 Ctrl+Shift+R (或 Cmd+Shift+R) 强制刷新"
echo "2. 或者打开开发者工具，右键刷新按钮选择"空缓存并硬性重新加载""
echo "3. 或者使用无痕/隐私浏览模式访问"

echo ""
echo -e "${GREEN}=== 验证完成 ===${NC}"
echo "请访问: http://localhost:3000/silences"
echo "如果仍有问题，请按上述建议清除浏览器缓存"
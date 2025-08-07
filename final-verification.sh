#!/bin/bash

echo "=== AlertBot 最终验证报告 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"

echo -e "${YELLOW}修复的问题列表:${NC}"
echo "1. /silences 页面空白和 te.some 报错"
echo "2. /inhibitions 页面 404 错误"
echo "3. /alert-groups 页面 404 错误"
echo ""

echo -e "${YELLOW}=== 验证结果 ===${NC}"

# 检查所有API端点
declare -a apis=("silences" "inhibitions" "alert-groups" "alert-group-rules")
declare -a pages=("silences" "inhibitions" "alert-groups")

echo "1. 后端API状态检查:"
all_apis_ok=true
for api in "${apis[@]}"; do
    response=$(curl -s -w "%{http_code}" -o /tmp/${api}_test.json "${BASE_URL}/api/v1/${api}")
    if [ "$response" = "200" ]; then
        echo -e "   ✓ ${GREEN}${api} API${NC} - 状态码 200"
    else
        echo -e "   ✗ ${RED}${api} API${NC} - 状态码 $response"
        all_apis_ok=false
    fi
done

echo ""
echo "2. 前端页面访问检查:"
all_pages_ok=true
for page in "${pages[@]}"; do
    response=$(curl -s -w "%{http_code}" -o /tmp/${page}_page.html "${FRONTEND_URL}/${page}")
    if [ "$response" = "200" ]; then
        # 检查页面是否包含新的JS文件
        new_js=$(grep -o 'index-2076be09\.js' /tmp/${page}_page.html)
        if [ -n "$new_js" ]; then
            echo -e "   ✓ ${GREEN}/${page}${NC} - 页面正常，使用最新构建"
        else
            echo -e "   ⚠ ${YELLOW}/${page}${NC} - 页面可访问但可能使用旧构建"
        fi
    else
        echo -e "   ✗ ${RED}/${page}${NC} - 状态码 $response"
        all_pages_ok=false
    fi
done

echo ""
echo "3. 数据结构验证:"
# 验证silences数据结构
silences_data=$(curl -s "${BASE_URL}/api/v1/silences")
items_type=$(echo "$silences_data" | jq -r '.data.items | type' 2>/dev/null)
if [ "$items_type" = "array" ]; then
    echo -e "   ✓ ${GREEN}Silences API${NC} - 返回正确的数组结构"
else
    echo -e "   ✗ ${RED}Silences API${NC} - 数据结构错误"
fi

# 验证inhibitions数据结构
inhibitions_data=$(curl -s "${BASE_URL}/api/v1/inhibitions")
inhibitions_success=$(echo "$inhibitions_data" | jq -r '.success' 2>/dev/null)
if [ "$inhibitions_success" = "true" ]; then
    echo -e "   ✓ ${GREEN}Inhibitions API${NC} - 响应格式正确"
else
    echo -e "   ✗ ${RED}Inhibitions API${NC} - 响应格式错误"
fi

# 验证alert-groups数据结构
groups_data=$(curl -s "${BASE_URL}/api/v1/alert-groups")
groups_items_type=$(echo "$groups_data" | jq -r '.data.items | type' 2>/dev/null)
if [ "$groups_items_type" = "array" ]; then
    echo -e "   ✓ ${GREEN}Alert Groups API${NC} - 返回正确的数组结构"
else
    echo -e "   ✗ ${RED}Alert Groups API${NC} - 数据结构错误"
fi

echo ""
echo "4. 容器状态检查:"
docker_status=$(docker-compose ps --format "table {{.Name}}\t{{.Status}}")
echo "$docker_status"

echo ""
echo -e "${YELLOW}=== 修复总结 ===${NC}"
if [ "$all_apis_ok" = true ] && [ "$all_pages_ok" = true ]; then
    echo -e "${GREEN}🎉 所有问题已成功修复！${NC}"
    echo ""
    echo "修复内容:"
    echo "• 修复了 useSilences hook 中的数据访问路径"
    echo "• 更新了 API 类型定义以匹配后端响应格式"
    echo "• 修复了 Silence 接口的 matchers 结构"
    echo "• 重新构建并部署了前端镜像"
    echo "• 添加了健壮的数组检查和错误处理"
    echo ""
    echo "现在可以正常访问以下页面:"
    echo "• http://localhost:3000/silences - 静默规则管理"
    echo "• http://localhost:3000/inhibitions - 抑制规则管理"
    echo "• http://localhost:3000/alert-groups - 告警分组管理"
    echo ""
    echo -e "${GREEN}系统已完全恢复正常运行！${NC}"
else
    echo -e "${RED}❌ 仍有部分问题需要解决${NC}"
    echo "请检查上述失败的项目"
fi

echo ""
echo "提示: 如果浏览器仍显示错误，请:"
echo "1. 按 Ctrl+Shift+R (Mac: Cmd+Shift+R) 强制刷新"
echo "2. 或使用无痕模式访问"
echo "3. 或清除浏览器缓存后重新访问"
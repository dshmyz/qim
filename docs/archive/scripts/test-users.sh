#!/bin/bash
# test-users.sh
# 用户管理自动化测试

BASE_URL="${QIM_BASE_URL:-http://localhost:8080}"
TOKEN="${QIM_TOKEN:-}"
PASS=0
FAIL=0

assert_eq() {
    local expected=$1 actual=$2 test_name=$3
    if [ "$expected" = "$actual" ]; then
        echo "✅ PASS: $test_name"
        PASS=$((PASS+1))
    else
        echo "❌ FAIL: $test_name (期望=$expected, 实际=$actual)"
        FAIL=$((FAIL+1))
    fi
}

if [ -z "$TOKEN" ]; then
    echo "❌ 请设置 QIM_TOKEN 环境变量"
    echo "   示例: export QIM_TOKEN=<your_token>"
    exit 1
fi

echo "===== 用户管理测试 ====="

# USER-001: 获取当前用户
echo "--- USER-001: 获取当前用户 ---"
RESP=$(curl -s "$BASE_URL/api/v1/users/me" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取当前用户"

# USER-002: 更新用户资料
echo "--- USER-002: 更新用户资料 ---"
RESP=$(curl -s -X PUT "$BASE_URL/api/v1/users/me" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"nickname":"测试昵称","signature":"测试签名"}')
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "更新用户资料"

# USER-005: 搜索用户
echo "--- USER-005: 搜索用户 ---"
RESP=$(curl -s "$BASE_URL/api/v1/users/search?q=admin" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "搜索用户"

echo ""
echo "===== 用户管理测试结果: $PASS 通过, $FAIL 失败 ====="

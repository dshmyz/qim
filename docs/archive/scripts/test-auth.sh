#!/bin/bash
# test-auth.sh
# 认证模块自动化测试

BASE_URL="${QIM_BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

# 工具函数
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

echo "===== 认证模块测试 ====="

# AUTH-001: 用户登录
echo "--- AUTH-001: 用户登录 ---"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"123456"}')
CODE=$(echo "$LOGIN_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "用户登录"

# 提取 Token
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)

# AUTH-002: 错误密码登录
echo "--- AUTH-002: 错误密码登录 ---"
LOGIN_FAIL_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrong"}')
CODE=$(echo "$LOGIN_FAIL_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "401" "$CODE" "错误密码登录返回401"

# AUTH-003: 用户不存在
echo "--- AUTH-003: 用户不存在 ---"
LOGIN_NORESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"nonexistent","password":"123"}')
CODE=$(echo "$LOGIN_NORESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "401" "$CODE" "用户不存在返回401"

# AUTH-004: 用户注册
echo "--- AUTH-004: 用户注册 ---"
REG_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"test_'"$(date +%s)"'","password":"123456","nickname":"测试用户"}')
CODE=$(echo "$REG_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "用户注册"

# AUTH-006: 刷新 Token
echo "--- AUTH-006: 刷新Token ---"
REFRESH_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/refresh" \
    -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$REFRESH_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "刷新Token"

# AUTH-007: 用户登出
echo "--- AUTH-007: 用户登出 ---"
LOGOUT_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/logout" \
    -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$LOGOUT_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "用户登出"

echo ""
echo "===== 认证测试结果: $PASS 通过, $FAIL 失败 ====="

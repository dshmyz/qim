#!/bin/bash
# test-all.sh
# QIM 完整回归测试套件
# 用法: QIM_BASE_URL=http://localhost:8080 bash test-all.sh

BASE_URL="${QIM_BASE_URL:-http://localhost:8080}"
TOTAL_PASS=0
TOTAL_FAIL=0

# 工具函数
assert_eq() {
    local expected=$1 actual=$2 test_name=$3
    if [ "$expected" = "$actual" ]; then
        echo "  ✅ PASS: $test_name"
        TOTAL_PASS=$((TOTAL_PASS+1))
    else
        echo "  ❌ FAIL: $test_name (期望=$expected, 实际=$actual)"
        TOTAL_FAIL=$((TOTAL_FAIL+1))
    fi
}

check_server() {
    echo "检查服务器状态..."
    if curl -s "$BASE_URL" > /dev/null 2>&1; then
        echo "✅ 服务器运行正常"
    else
        echo "❌ 服务器未启动，请先运行: cd qim-server && go run main.go"
        exit 1
    fi
}

echo "========================================"
echo "  QIM 全量回归测试"
echo "  服务器: $BASE_URL"
echo "  日期: $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================"
echo ""

check_server

# ===== 1. 认证测试 =====
echo ">>> [1/15] 认证模块测试"

# 登录获取 Token
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"123456"}')
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('token',''))" 2>/dev/null)
USER_ID=$(echo "$LOGIN_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('user',{}).get('id',''))" 2>/dev/null)

if [ -n "$TOKEN" ]; then
    assert_eq "非空" "非空" "登录成功获取Token"
else
    echo "  ⚠️ 跳过认证测试（登录失败或无admin账号）"
    echo "  提示: 请先注册用户: curl -X POST $BASE_URL/api/v1/auth/register -H 'Content-Type: application/json' -d '{\"username\":\"admin\",\"password\":\"123456\",\"nickname\":\"管理员\"}'"
    exit 1
fi

# ===== 2. 用户管理 =====
echo ""
echo ">>> [2/15] 用户管理测试"

RESP=$(curl -s "$BASE_URL/api/v1/users/me" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取当前用户"

RESP=$(curl -s -X PUT "$BASE_URL/api/v1/users/me" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"nickname":"回归测试用户"}')
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "更新用户资料"

RESP=$(curl -s "$BASE_URL/api/v1/users/search?keyword=admin" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "搜索用户"

# ===== 3. 会话管理 =====
echo ""
echo ">>> [3/15] 会话管理测试"

# 创建单聊
RESP=$(curl -s -X POST "$BASE_URL/api/v1/conversations/single" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"user_id":2}' 2>/dev/null)
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code',''))" 2>/dev/null)
assert_eq "0" "$CODE" "创建单聊会话"

# 创建群聊
GROUP_RESP=$(curl -s -X POST "$BASE_URL/api/v1/conversations/group" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"回归测试群","member_ids":[2,3]}' 2>/dev/null)
GROUP_ID=$(echo "$GROUP_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
CODE=$(echo "$GROUP_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code',''))" 2>/dev/null)
assert_eq "0" "$CODE" "创建群聊会话"

# 获取会话列表
RESP=$(curl -s "$BASE_URL/api/v1/conversations" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取会话列表"

# 会话置顶（如果群聊创建成功）
if [ -n "$GROUP_ID" ]; then
    RESP=$(curl -s -X PUT "$BASE_URL/api/v1/conversations/$GROUP_ID/pin" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"is_pinned":true}')
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "会话置顶"
fi

# 获取会话详情
if [ -n "$GROUP_ID" ]; then
    RESP=$(curl -s "$BASE_URL/api/v1/conversations/$GROUP_ID" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "获取会话详情"
fi

# ===== 4. 消息功能 =====
echo ""
echo ">>> [4/15] 消息功能测试"

if [ -n "$GROUP_ID" ]; then
    # 发送文本消息
    MSG_RESP=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$GROUP_ID/messages" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"type":"text","content":"回归测试消息"}')
    MSG_ID=$(echo "$MSG_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
    CODE=$(echo "$MSG_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "发送文本消息"

    # 获取历史消息
    RESP=$(curl -s "$BASE_URL/api/v1/conversations/$GROUP_ID/messages" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "获取历史消息"

    # 撤回消息
    if [ -n "$MSG_ID" ]; then
        RESP=$(curl -s -X POST "$BASE_URL/api/v1/messages/$MSG_ID/recall" \
            -H "Authorization: Bearer $TOKEN")
        CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
        assert_eq "0" "$CODE" "撤回消息"
    fi
fi

# ===== 5. 组织架构 =====
echo ""
echo ">>> [5/15] 组织架构测试"

RESP=$(curl -s "$BASE_URL/api/v1/organization/tree" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取组织架构树"

# ===== 6. 文件管理 =====
echo ""
echo ">>> [6/15] 文件管理测试"

# 创建测试文件
echo "test content" > /tmp/qim_test_file.txt
RESP=$(curl -s -X POST "$BASE_URL/api/v1/upload" \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@/tmp/qim_test_file.txt")
FILE_ID=$(echo "$RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "上传文件"

# 获取文件列表
RESP=$(curl -s "$BASE_URL/api/v1/files" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取文件列表"

# 删除文件
if [ -n "$FILE_ID" ]; then
    RESP=$(curl -s -X DELETE "$BASE_URL/api/v1/files/$FILE_ID" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "删除文件"
fi

rm -f /tmp/qim_test_file.txt

# ===== 7. 笔记管理 =====
echo ""
echo ">>> [7/15] 笔记管理测试"

NOTE_RESP=$(curl -s -X POST "$BASE_URL/api/v1/notes" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"回归测试笔记","content":"# 测试内容","color":"blue"}')
NOTE_ID=$(echo "$NOTE_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
CODE=$(echo "$NOTE_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "创建笔记"

if [ -n "$NOTE_ID" ]; then
    # 获取笔记列表
    RESP=$(curl -s "$BASE_URL/api/v1/notes" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "获取笔记列表"

    # 更新笔记
    RESP=$(curl -s -X PUT "$BASE_URL/api/v1/notes/$NOTE_ID" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"title":"更新后的笔记"}')
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "更新笔记"

    # 删除笔记
    RESP=$(curl -s -X DELETE "$BASE_URL/api/v1/notes/$NOTE_ID" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "删除笔记"
fi

# ===== 8. 任务管理 =====
echo ""
echo ">>> [8/15] 任务管理测试"

TASK_RESP=$(curl -s -X POST "$BASE_URL/api/v1/tasks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"回归测试任务","description":"测试描述","priority":"high"}')
TASK_ID=$(echo "$TASK_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
CODE=$(echo "$TASK_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "创建任务"

if [ -n "$TASK_ID" ]; then
    # 获取任务列表
    RESP=$(curl -s "$BASE_URL/api/v1/tasks" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "获取任务列表"

    # 更新任务状态
    RESP=$(curl -s -X PATCH "$BASE_URL/api/v1/tasks/$TASK_ID/status" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"status":"completed"}')
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "更新任务状态"

    # 删除任务
    RESP=$(curl -s -X DELETE "$BASE_URL/api/v1/tasks/$TASK_ID" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "删除任务"
fi

# ===== 9. 日历事件 =====
echo ""
echo ">>> [9/15] 日历事件测试"

EVENT_RESP=$(curl -s -X POST "$BASE_URL/api/v1/events" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"回归测试事件","description":"测试","start":"2026-05-01T10:00:00Z","end":"2026-05-01T11:00:00Z","reminder":15}')
EVENT_ID=$(echo "$EVENT_RESP" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}).get('id',''))" 2>/dev/null)
CODE=$(echo "$EVENT_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "创建日历事件"

if [ -n "$EVENT_ID" ]; then
    RESP=$(curl -s "$BASE_URL/api/v1/events" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "获取事件列表"

    RESP=$(curl -s -X DELETE "$BASE_URL/api/v1/events/$EVENT_ID" -H "Authorization: Bearer $TOKEN")
    CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
    assert_eq "0" "$CODE" "删除日历事件"
fi

# ===== 10. 应用管理 =====
echo ""
echo ">>> [10/15] 应用管理测试"

RESP=$(curl -s "$BASE_URL/api/v1/apps" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取应用列表"

RESP=$(curl -s "$BASE_URL/api/v1/apps/all" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取所有应用"

# ===== 11. 小程序管理 =====
echo ""
echo ">>> [11/15] 小程序管理测试"

RESP=$(curl -s "$BASE_URL/api/v1/mini-apps" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取小程序列表"

# ===== 12. 通知管理 =====
echo ""
echo ">>> [12/15] 通知管理测试"

RESP=$(curl -s "$BASE_URL/api/v1/notifications" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取通知列表"

RESP=$(curl -s -X PUT "$BASE_URL/api/v1/notifications/read-all" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "标记所有通知已读"

# ===== 13. 频道管理 =====
echo ""
echo ">>> [13/15] 频道管理测试"

RESP=$(curl -s "$BASE_URL/api/v1/channels" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取频道列表"

# ===== 14. 系统消息 =====
echo ""
echo ">>> [14/15] 系统消息测试"

RESP=$(curl -s "$BASE_URL/api/v1/system-messages" -H "Authorization: Bearer $TOKEN")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "0" "$CODE" "获取系统消息"

# ===== 15. 安全测试 =====
echo ""
echo ">>> [15/15] 安全测试"

# 未认证访问
RESP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/users/me")
assert_eq "401" "$RESP" "未认证返回401"

# 无效 Token
RESP=$(curl -s "$BASE_URL/api/v1/users/me" -H "Authorization: Bearer invalid_token")
CODE=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['code'])" 2>/dev/null)
assert_eq "401" "$CODE" "无效Token返回401"

# CORS
RESP_HEADERS=$(curl -s -I -X OPTIONS "$BASE_URL/api/v1/users/me" \
    -H "Origin: http://localhost:5173" \
    -H "Access-Control-Request-Method: GET" 2>/dev/null)
if echo "$RESP_HEADERS" | grep -qi "access-control-allow-origin"; then
    echo "  ✅ PASS: CORS配置正确"
    TOTAL_PASS=$((TOTAL_PASS+1))
else
    echo "  ❌ FAIL: CORS配置缺失"
    TOTAL_FAIL=$((TOTAL_FAIL+1))
fi

# ===== 测试结果汇总 =====
echo ""
echo "========================================"
echo "  测试完成"
echo "  通过: $TOTAL_PASS"
echo "  失败: $TOTAL_FAIL"
echo "  总计: $((TOTAL_PASS + TOTAL_FAIL))"
echo "  成功率: $(echo "scale=1; $TOTAL_PASS * 100 / ($TOTAL_PASS + $TOTAL_FAIL)" | bc 2>/dev/null || echo "N/A")%"
echo "========================================"

if [ "$TOTAL_FAIL" -gt 0 ]; then
    exit 1
fi
exit 0

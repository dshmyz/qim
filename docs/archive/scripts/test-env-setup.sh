#!/bin/bash
# test-env-setup.sh
# 测试环境配置

BASE_URL="${QIM_BASE_URL:-http://localhost:8080}"
echo "测试服务器: $BASE_URL"

# 检查服务器是否运行
if ! curl -s "$BASE_URL/api/v1/auth/login" -X POST -H "Content-Type: application/json" -d '{"username":"test","password":"test"}' > /dev/null 2>&1; then
    echo "❌ 服务器未启动，请先运行: cd qim-server && go run main.go"
    exit 1
fi
echo "✅ 服务器运行正常"

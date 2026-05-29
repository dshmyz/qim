#!/bin/bash

echo "=========================================="
echo "  运行 LDAP 连接测试"
echo "=========================================="
echo ""

cd "$(dirname "$0")"

if ! command -v go &> /dev/null; then
    echo "错误: 未安装 Go"
    exit 1
fi

if ! docker ps | grep -q qim-ldap-test; then
    echo "LDAP 服务器未运行，正在启动..."
    ./start.sh
    sleep 3
fi

echo "运行测试程序..."
echo ""
go run test_ldap.go

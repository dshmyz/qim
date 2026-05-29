#!/bin/bash

echo "正在停止 LDAP 测试服务器..."
cd "$(dirname "$0")"
docker compose down

echo "已停止"

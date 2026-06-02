#!/bin/bash
# QIM 后端跨平台构建脚本
# 包含前端构建 + Go embed 打包
# 用法: ./scripts/build.sh [--arch amd64,arm64] [--output ./dist]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$PROJECT_DIR")"

cd "$PROJECT_DIR"

APP_NAME="qim-server"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_DIR/dist}"
BUILD_ARCH="${BUILD_ARCH:-amd64,arm64}"
LDFLAGS="-s -w"
SKIP_FRONTEND="${SKIP_FRONTEND:-}"

while [[ $# -gt 0 ]]; do
  case $1 in
    --arch) BUILD_ARCH="$2"; shift 2 ;;
    --output) OUTPUT_DIR="$2"; shift 2 ;;
    --skip-frontend) SKIP_FRONTEND="1"; shift ;;
    --help)
      echo "Usage: $0 [--arch amd64,arm64] [--output ./dist] [--skip-frontend]"
      exit 0 ;;
    *) echo "Unknown: $1"; exit 1 ;;
  esac
done

mkdir -p "$OUTPUT_DIR"

if [[ -z "$SKIP_FRONTEND" ]]; then
  echo "========================================"
  echo "  构建前端 (qim-admin)"
  echo "========================================"

  ADMIN_DIR="$ROOT_DIR/qim-admin"
  if [[ -d "$ADMIN_DIR" ]]; then
    cd "$ADMIN_DIR"
    echo "  npm install..."
    npm install --legacy-peer-deps 2>/dev/null || npm install

    echo "  构建管理后台 (base: /admin/)..."
    npm run build

    echo "  构建 Landing 页 (base: /)..."
    npx vite build --config vite.config.landing.ts

    echo "  复制产物到 embed 目录..."
    mkdir -p "$PROJECT_DIR/web/webroot/admin"
    mkdir -p "$PROJECT_DIR/web/webroot/landing"
    cp -r dist/* "$PROJECT_DIR/web/webroot/admin/"
    cp -r dist-landing/* "$PROJECT_DIR/web/webroot/landing/"
    # landing.html 重命名为 index.html（SPA embed 默认查找 index.html）
    if [[ -f "$PROJECT_DIR/web/webroot/landing/landing.html" ]]; then
      mv "$PROJECT_DIR/web/webroot/landing/landing.html" "$PROJECT_DIR/web/webroot/landing/index.html"
    fi

    echo "  前端构建完成"
    echo ""
    cd "$PROJECT_DIR"
  else
    echo "  警告: qim-admin 目录不存在，跳过前端构建"
    echo ""
  fi
else
  echo "========================================"
  echo "  跳过前端构建 (--skip-frontend)"
  echo "========================================"
  echo ""
fi

IFS=',' read -ra ARCH_LIST <<< "$BUILD_ARCH"

echo "========================================"
echo "  QIM Server 构建"
echo "  架构: $BUILD_ARCH"
echo "  输出: $OUTPUT_DIR"
echo "========================================"

BUILD_OK=0
BUILD_FAIL=0

for ARCH in "${ARCH_LIST[@]}"; do
  echo ""
  echo "[$ARCH] 编译中..."

  OUTPUT_FILE="$OUTPUT_DIR/${APP_NAME}-linux-${ARCH}"

  GOOS=linux GOARCH="$ARCH" \
    go build \
    -ldflags="$LDFLAGS" \
    -o "$OUTPUT_FILE" \
    .

  echo "  OK: $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" | cut -f1))"

  cp "$PROJECT_DIR/config.yaml" "$OUTPUT_DIR/config.yaml" 2>/dev/null || true

  BUILD_OK=$((BUILD_OK + 1))
done

echo ""
echo "========================================"
echo "  构建完成: ${BUILD_OK} 成功, ${BUILD_FAIL} 失败"
echo "  输出目录: $OUTPUT_DIR"
echo "========================================"

[ "$BUILD_FAIL" -eq 0 ] || exit 1

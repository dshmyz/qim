#!/bin/bash
# QIM 后端跨平台构建脚本
# 包含前端构建 + Go embed 打包
# 用法: ./scripts/build.sh [--arch amd64,arm64] [--output ./dist] [--skip-frontend]
#
# 前端产物布局（webroot）：
#   webroot/admin/   ← qim-admin 构建（管理后台 SPA，base=/admin/）
#   webroot/landing/ ← qim-landing 构建（VitePress 首页 + 共享资源 assets/fonts）
#   webroot/docs/    ← qim-landing 构建的 docs 子目录（CLI/MCP 文档，cleanUrls）

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

WEBROOT_DIR="$PROJECT_DIR/web/webroot"

if [[ -z "$SKIP_FRONTEND" ]]; then
  # ============================================================
  # 1) 构建管理后台 (qim-admin)
  # ============================================================
  echo "========================================"
  echo "  构建管理后台 (qim-admin, base: /admin/)"
  echo "========================================"

  ADMIN_DIR="$ROOT_DIR/qim-admin"
  if [[ -d "$ADMIN_DIR" ]]; then
    cd "$ADMIN_DIR"
    echo "  npm install..."
    npm install --legacy-peer-deps 2>/dev/null || npm install

    echo "  构建管理后台..."
    npm run build

    echo "  复制产物到 embed 目录 (webroot/admin)..."
    rm -rf "$WEBROOT_DIR/admin"
    mkdir -p "$WEBROOT_DIR/admin"
    cp -r dist/* "$WEBROOT_DIR/admin/"
    echo "  管理后台构建完成"
    echo ""
    cd "$PROJECT_DIR"
  else
    echo "  警告: qim-admin 目录不存在，跳过管理后台构建"
    echo ""
  fi

  # ============================================================
  # 2) 构建 Landing + 文档 (qim-landing，VitePress)
  # ============================================================
  echo "========================================"
  echo "  构建 Landing + 文档 (qim-landing, VitePress)"
  echo "========================================"

  LANDING_DIR="$ROOT_DIR/qim-landing"
  if [[ -d "$LANDING_DIR" ]]; then
    cd "$LANDING_DIR"
    echo "  npm install..."
    npm install 2>/dev/null || true

    echo "  构建 VitePress 站点..."
    npm run build

    echo "  复制产物到 embed 目录..."
    # landing：首页 index.html + 共享资源（assets/fonts/app-logo-v1.png/vp-icons.css）
    rm -rf "$WEBROOT_DIR/landing"
    mkdir -p "$WEBROOT_DIR/landing"
    cp -r dist/* "$WEBROOT_DIR/landing/"

    # docs：CLI/MCP 文档（独立 webroot/docs，由 ServeDocs 提供）
    # docs 子目录同时存在于 dist/docs，需分离到 webroot/docs 供 /docs/* 路由访问
    rm -rf "$WEBROOT_DIR/docs"
    if [[ -d "dist/docs" ]]; then
      mkdir -p "$WEBROOT_DIR/docs"
      cp -r dist/docs/* "$WEBROOT_DIR/docs/"
      # landing 下不再保留 docs 副本，避免混淆
      rm -rf "$WEBROOT_DIR/landing/docs"
    fi

    echo "  landing 文档构建完成"
    echo ""
    cd "$PROJECT_DIR"
  else
    echo "  警告: qim-landing 目录不存在，跳过 Landing 构建"
    echo ""
  fi
else
  echo "========================================"
  echo "  跳过前端构建 (--skip-frontend)"
  echo "========================================"
  echo ""
fi

# ============================================================
# 3) 编译 Go 后端（嵌入 webroot）
# ============================================================
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

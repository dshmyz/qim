#!/bin/bash
# QIM 后端跨平台构建脚本
# 纯 Go 编译，无需 CGO 工具链
# 用法: ./scripts/build.sh [--arch amd64,arm64] [--output ./dist]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

APP_NAME="qim-server"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_DIR/dist}"
BUILD_ARCH="${BUILD_ARCH:-amd64,arm64}"
LDFLAGS="-s -w"

while [[ $# -gt 0 ]]; do
  case $1 in
    --arch) BUILD_ARCH="$2"; shift 2 ;;
    --output) OUTPUT_DIR="$2"; shift 2 ;;
    --help)
      echo "Usage: $0 [--arch amd64,arm64] [--output ./dist]"
      exit 0 ;;
    *) echo "Unknown: $1"; exit 1 ;;
  esac
done

mkdir -p "$OUTPUT_DIR"

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

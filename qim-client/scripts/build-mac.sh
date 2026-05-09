#!/bin/bash
# ============================================
# macOS 版本构建脚本
# 使用 Electron 33.0.0（最新稳定版）
# 生成 dmg、zip 格式
# ============================================

set -e

echo "========================================"
echo "  开始构建 macOS 版本"
echo "  Electron: 33.0.0"
echo "  目标平台: macOS"
echo "  输出格式: dmg, zip"
echo "========================================"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 1. 检查操作系统
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ 此脚本只能在 macOS 系统上运行"
    echo "   当前系统: $OSTYPE"
    exit 1
fi

# 2. 检查 Node.js 版本
echo ""
echo "📦 检查 Node.js 版本..."
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "❌ Node.js 版本过低，需要 18 或以上"
    echo "   当前版本: $(node -v)"
    exit 1
fi
echo "✅ Node.js 版本: $(node -v)"

# 3. 检查依赖是否安装
echo ""
echo "📦 检查依赖..."
if [ ! -d "node_modules" ]; then
    echo "⚠️  node_modules 不存在，正在安装依赖..."
    npm install
else
    echo "✅ 依赖已安装"
fi

# 4. 构建前端资源
echo ""
echo "🔨 构建前端资源..."
npm run build

# 5. 检查代码签名证书（可选）
echo ""
echo "🔐 检查代码签名配置..."
if [ -n "$APPLE_ID" ] && [ -n "$APPLE_APP_SPECIFIC_PASSWORD" ]; then
    echo "✅ Apple ID 已配置（将启用代码签名和公证）"
else
    echo "⚠️  未配置 Apple ID，将跳过代码签名和公证"
    echo "   如需启用，请设置环境变量:"
    echo "   export APPLE_ID=your.apple.id@example.com"
    echo "   export APPLE_APP_SPECIFIC_PASSWORD=your-app-specific-password"
    echo "   export CSC_LINK=/path/to/certificate.p12"
    echo "   export CSC_KEY_PASSWORD=your-key-password"
fi

# 6. 设置国内镜像加速（可选）
echo ""
echo "🌐 设置 Electron 下载镜像..."
export ELECTRON_MIRROR="${ELECTRON_MIRROR:-https://npmmirror.com/mirrors/electron/}"
export ELECTRON_BUILDER_BINARIES_MIRROR="${ELECTRON_BUILDER_BINARIES_MIRROR:-https://npmmirror.com/mirrors/electron-builder-binaries/}"

# 7. 构建 macOS 版本
echo ""
echo "🚀 开始打包 macOS 版本..."
npx electron-builder --mac

# 8. 显示构建结果
echo ""
echo "========================================"
echo "  ✅ 构建完成！"
echo "========================================"
echo ""
echo "📁 输出目录: electron-dist/"
echo ""
if [ -d "electron-dist" ]; then
    echo "📦 生成的文件:"
    ls -lh electron-dist/ 2>/dev/null | grep -E '\.(dmg|zip)$' || echo "   未找到 macOS 相关文件"
fi
echo ""
echo "========================================"

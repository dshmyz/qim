#!/bin/bash
# ============================================
# Windows 7 版本构建脚本
# 使用 Electron 22.3.27（最后支持 Win7 的版本）
# 可在 macOS/Linux 上交叉编译
# ============================================

set -e

echo "========================================"
echo "  开始构建 Windows 7 版本"
echo "  Electron: 22.3.27"
echo "  目标平台: Windows x64"
echo "========================================"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 1. 检查 Node.js 版本
echo ""
echo "📦 检查 Node.js 版本..."
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "❌ Node.js 版本过低，需要 18 或以上"
    echo "   当前版本: $(node -v)"
    exit 1
fi
echo "✅ Node.js 版本: $(node -v)"

# 2. 检查依赖是否安装
echo ""
echo "📦 检查依赖..."
if [ ! -d "node_modules" ]; then
    echo "⚠️  node_modules 不存在，正在安装依赖..."
    npm install
else
    echo "✅ 依赖已安装"
fi

# 3. 构建前端资源
echo ""
echo "🔨 构建前端资源..."
npm run build

# 4. 检查 Wine（仅在 Linux 上需要）
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo ""
    echo "🍷 检查 Wine（Linux 交叉编译 Windows 需要）..."
    if ! command -v wine &> /dev/null; then
        echo "❌ Wine 未安装"
        echo "   请运行: sudo apt-get install wine64 wine32"
        exit 1
    fi
    echo "✅ Wine 版本: $(wine --version)"
    
    if ! command -v makensis &> /dev/null; then
        echo "❌ NSIS 未安装"
        echo "   请运行: sudo apt-get install nsis"
        exit 1
    fi
    echo "✅ NSIS 已安装"
fi

# 5. 设置国内镜像加速（可选）
echo ""
echo "🌐 设置 Electron 下载镜像..."
export ELECTRON_MIRROR="${ELECTRON_MIRROR:-https://npmmirror.com/mirrors/electron/}"
export ELECTRON_BUILDER_BINARIES_MIRROR="${ELECTRON_BUILDER_BINARIES_MIRROR:-https://npmmirror.com/mirrors/electron-builder-binaries/}"

# 6. 构建 Windows 7 版本
echo ""
echo "🚀 开始打包 Windows 7 版本..."
npx electron-builder --win --x64 -c.win7

# 7. 显示构建结果
echo ""
echo "========================================"
echo "  ✅ 构建完成！"
echo "========================================"
echo ""
echo "📁 输出目录: electron-dist/"
echo ""
if [ -d "electron-dist" ]; then
    ls -lh electron-dist/*Win7* 2>/dev/null || echo "   未找到 Win7 相关文件"
fi
echo ""
echo "========================================"

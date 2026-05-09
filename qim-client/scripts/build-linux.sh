#!/bin/bash
# ============================================
# Linux 版本构建脚本
# 使用 Electron 33.0.0（最新稳定版）
# 生成 deb、rpm、AppImage 格式
# ============================================

set -e

echo "========================================"
echo "  开始构建 Linux 版本"
echo "  Electron: 33.0.0"
echo "  目标平台: Linux x64"
echo "  输出格式: deb, rpm, AppImage"
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

# 4. 检查 Linux 打包依赖
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo ""
    echo "🐧 检查 Linux 打包依赖..."
    
    # 检查 dpkg（用于 deb 打包）
    if ! command -v dpkg &> /dev/null; then
        echo "⚠️  dpkg 未找到，将跳过 deb 打包"
    else
        echo "✅ dpkg 已安装"
    fi
    
    # 检查 rpmbuild（用于 rpm 打包）
    if ! command -v rpmbuild &> /dev/null; then
        echo "⚠️  rpmbuild 未找到"
        echo "   如需 rpm 打包，请运行: sudo apt-get install rpm"
    else
        echo "✅ rpmbuild 已安装"
    fi
    
    # 检查 AppImage 相关依赖
    if ! command -v fuse &> /dev/null && ! command -v fusermount &> /dev/null; then
        echo "⚠️  FUSE 未安装，AppImage 可能需要"
        echo "   请运行: sudo apt-get install libfuse2"
    else
        echo "✅ FUSE 已安装"
    fi
fi

# 5. 设置国内镜像加速（可选）
echo ""
echo "🌐 设置 Electron 下载镜像..."
export ELECTRON_MIRROR="${ELECTRON_MIRROR:-https://npmmirror.com/mirrors/electron/}"
export ELECTRON_BUILDER_BINARIES_MIRROR="${ELECTRON_BUILDER_BINARIES_MIRROR:-https://npmmirror.com/mirrors/electron-builder-binaries/}"

# 6. 构建 Linux 版本
echo ""
echo "🚀 开始打包 Linux 版本..."
npx electron-builder --linux

# 7. 显示构建结果
echo ""
echo "========================================"
echo "  ✅ 构建完成！"
echo "========================================"
echo ""
echo "📁 输出目录: electron-dist/"
echo ""
if [ -d "electron-dist" ]; then
    echo "📦 生成的文件:"
    ls -lh electron-dist/ 2>/dev/null | grep -E '\.(deb|rpm|AppImage|tar\.gz|zip)$' || echo "   未找到 Linux 相关文件"
fi
echo ""
echo "========================================"

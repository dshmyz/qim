#!/bin/bash
# ============================================
# Windows 10 版本构建脚本
# 使用 Electron 33.0.0（最新稳定版）
# 可在 macOS/Linux 上交叉编译
# ============================================

set -e

echo "========================================"
echo "  开始构建 Windows 10 版本"
echo "  Electron: 33.0.0"
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

# 5. 设置更新服务器地址（可通过环境变量 QIM_UPDATE_URL 覆盖）
#    默认从 .env.production 解析 VITE_API_URL，与渲染进程默认同源（只改一处）；
#    仍未取到时才回退 localhost 并告警。
if [ -z "$QIM_UPDATE_URL" ]; then
  QIM_UPDATE_URL="$(grep -E '^VITE_API_URL=' "$PROJECT_DIR/.env.production" 2>/dev/null | head -n1 | cut -d'=' -f2- | tr -d '\r')"
  if [ -z "$QIM_UPDATE_URL" ]; then
    QIM_UPDATE_URL="http://localhost:8080"
    echo "⚠️  .env.production 未配置 VITE_API_URL，更新地址回退到 $QIM_UPDATE_URL（仅开发）"
  fi
fi
export QIM_UPDATE_URL
echo "🔧 更新服务器地址: $QIM_UPDATE_URL"

# 6. 设置国内镜像加速（可选）
echo ""
echo "🌐 设置 Electron 下载镜像..."
export ELECTRON_MIRROR="${ELECTRON_MIRROR:-https://npmmirror.com/mirrors/electron/}"
export ELECTRON_BUILDER_BINARIES_MIRROR="${ELECTRON_BUILDER_BINARIES_MIRROR:-https://npmmirror.com/mirrors/electron-builder-binaries/}"

# 6. 构建 Windows 10 版本
echo ""
echo "🚀 开始打包 Windows 10 版本..."
npx electron-builder --win --x64 -c.win10

# 7. 显示构建结果
echo ""
echo "========================================"
echo "  ✅ 构建完成！"
echo "========================================"
echo ""
echo "📁 输出目录: electron-dist/"
echo ""
if [ -d "electron-dist" ]; then
    ls -lh electron-dist/*Win10* 2>/dev/null || echo "   未找到 Win10 相关文件"
fi
echo ""
echo "========================================"

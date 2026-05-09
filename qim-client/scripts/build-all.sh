#!/bin/bash
# ============================================
# 全平台一键构建脚本
# 自动检测当前系统并构建所有支持的平台
# ============================================

set -e

echo "========================================"
echo "  QIM 全平台构建工具"
echo "========================================"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 统计变量
SUCCESS_COUNT=0
FAIL_COUNT=0
TOTAL_COUNT=0

# 显示使用帮助
show_help() {
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --win7        仅构建 Windows 7 版本"
    echo "  --win10       仅构建 Windows 10 版本"
    echo "  --win         构建所有 Windows 版本 (win7 + win10)"
    echo "  --linux       仅构建 Linux 版本"
    echo "  --mac         仅构建 macOS 版本"
    echo "  --all         构建所有平台版本（默认）"
    echo "  --clean       构建前清理 dist 和 electron-dist 目录"
    echo "  --skip-frontend  跳过前端构建，仅打包 Electron"
    echo "  --help, -h    显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0              # 构建所有平台"
    echo "  $0 --win7       # 仅构建 Win7"
    echo "  $0 --win --linux  # 构建 Windows 和 Linux"
    echo "  $0 --all --clean  # 清理后构建所有平台"
    echo ""
}

# 解析参数
BUILD_WIN7=false
BUILD_WIN10=false
BUILD_LINUX=false
BUILD_MAC=false
CLEAN_BEFORE=false
SKIP_FRONTEND=false

# 如果没有参数，默认构建所有平台
if [ $# -eq 0 ]; then
    BUILD_WIN7=true
    BUILD_WIN10=true
    BUILD_LINUX=true
    # macOS 仅在 macOS 系统上构建
    if [[ "$OSTYPE" == "darwin"* ]]; then
        BUILD_MAC=true
    fi
else
    for arg in "$@"; do
        case $arg in
            --win7)
                BUILD_WIN7=true
                shift
                ;;
            --win10)
                BUILD_WIN10=true
                shift
                ;;
            --win)
                BUILD_WIN7=true
                BUILD_WIN10=true
                shift
                ;;
            --linux)
                BUILD_LINUX=true
                shift
                ;;
            --mac)
                BUILD_MAC=true
                shift
                ;;
            --all)
                BUILD_WIN7=true
                BUILD_WIN10=true
                BUILD_LINUX=true
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    BUILD_MAC=true
                fi
                shift
                ;;
            --clean)
                CLEAN_BEFORE=true
                shift
                ;;
            --skip-frontend)
                SKIP_FRONTEND=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}❌ 未知参数: $arg${NC}"
                show_help
                exit 1
                ;;
        esac
    done
fi

# 检查是否选择了任何平台
if [ "$BUILD_WIN7" = false ] && [ "$BUILD_WIN10" = false ] && [ "$BUILD_LINUX" = false ] && [ "$BUILD_MAC" = false ]; then
    echo -e "${RED}❌ 请至少选择一个平台进行构建${NC}"
    show_help
    exit 1
fi

# 显示构建配置
echo ""
echo "📋 构建配置:"
[ "$BUILD_WIN7" = true ] && echo "   ✅ Windows 7 (Electron 22.3.27)" || echo "   ⏭️  跳过 Windows 7"
[ "$BUILD_WIN10" = true ] && echo "   ✅ Windows 10 (Electron 33.0.0)" || echo "   ⏭️  跳过 Windows 10"
[ "$BUILD_LINUX" = true ] && echo "   ✅ Linux" || echo "   ⏭️  跳过 Linux"
[ "$BUILD_MAC" = true ] && echo "   ✅ macOS" || echo "   ⏭️  跳过 macOS"
[ "$CLEAN_BEFORE" = true ] && echo "   🧹 构建前清理"
[ "$SKIP_FRONTEND" = true ] && echo "   ⏭️  跳过前端构建"
echo ""

# 检查 Node.js 版本
echo "📦 检查 Node.js 版本..."
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo -e "${RED}❌ Node.js 版本过低，需要 18 或以上${NC}"
    echo "   当前版本: $(node -v)"
    exit 1
fi
echo -e "${GREEN}✅ Node.js 版本: $(node -v)${NC}"

# 检查依赖
echo ""
echo "📦 检查依赖..."
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}⚠️  node_modules 不存在，正在安装依赖...${NC}"
    npm install
else
    echo -e "${GREEN}✅ 依赖已安装${NC}"
fi

# 清理（如果需要）
if [ "$CLEAN_BEFORE" = true ]; then
    echo ""
    echo "🧹 清理旧的构建文件..."
    rm -rf dist electron-dist
    echo -e "${GREEN}✅ 清理完成${NC}"
fi

# 构建前端资源
if [ "$SKIP_FRONTEND" = false ]; then
    echo ""
    echo "🔨 构建前端资源..."
    npm run build
    echo -e "${GREEN}✅ 前端构建完成${NC}"
fi

# 设置国内镜像加速
echo ""
echo "🌐 设置 Electron 下载镜像..."
export ELECTRON_MIRROR="${ELECTRON_MIRROR:-https://npmmirror.com/mirrors/electron/}"
export ELECTRON_BUILDER_BINARIES_MIRROR="${ELECTRON_BUILDER_BINARIES_MIRROR:-https://npmmirror.com/mirrors/electron-builder-binaries/}"
echo -e "${GREEN}✅ 镜像配置完成${NC}"

# 检查 Wine（仅 Linux 且需要构建 Windows 时）
if [[ "$OSTYPE" == "linux-gnu"* ]] && ([ "$BUILD_WIN7" = true ] || [ "$BUILD_WIN10" = true ]); then
    echo ""
    echo "🍷 检查 Wine（Linux 交叉编译 Windows 需要）..."
    if ! command -v wine &> /dev/null; then
        echo -e "${RED}❌ Wine 未安装${NC}"
        echo "   请运行: sudo apt-get install wine64 wine32"
        exit 1
    fi
    echo -e "${GREEN}✅ Wine 版本: $(wine --version)${NC}"
    
    if ! command -v makensis &> /dev/null; then
        echo -e "${RED}❌ NSIS 未安装${NC}"
        echo "   请运行: sudo apt-get install nsis"
        exit 1
    fi
    echo -e "${GREEN}✅ NSIS 已安装${NC}"
fi

# 开始构建
echo ""
echo "========================================"
echo "  开始构建..."
echo "========================================"

# 构建 Windows 7
if [ "$BUILD_WIN7" = true ]; then
    TOTAL_COUNT=$((TOTAL_COUNT + 1))
    echo ""
    echo -e "${BLUE}[$TOTAL_COUNT] 构建 Windows 7 版本 (Electron 22.3.27)...${NC}"
    if npx electron-builder --win --x64 -c electron-builder-win7.yml; then
        echo -e "${GREEN}✅ Windows 7 构建成功${NC}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}❌ Windows 7 构建失败${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi

# 构建 Windows 10+
if [ "$BUILD_WIN10" = true ]; then
    TOTAL_COUNT=$((TOTAL_COUNT + 1))
    echo ""
    echo -e "${BLUE}[$TOTAL_COUNT] 构建 Windows 10+ 版本 (Electron 33)...${NC}"
    if npx electron-builder --win --x64; then
        echo -e "${GREEN}✅ Windows 10+ 构建成功${NC}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}❌ Windows 10+ 构建失败${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi

# 构建 Linux
if [ "$BUILD_LINUX" = true ]; then
    TOTAL_COUNT=$((TOTAL_COUNT + 1))
    echo ""
    echo -e "${BLUE}[$TOTAL_COUNT] 构建 Linux 版本...${NC}"
    if npx electron-builder --linux; then
        echo -e "${GREEN}✅ Linux 构建成功${NC}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}❌ Linux 构建失败${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi

# 构建 macOS
if [ "$BUILD_MAC" = true ]; then
    TOTAL_COUNT=$((TOTAL_COUNT + 1))
    echo ""
    echo -e "${BLUE}[$TOTAL_COUNT] 构建 macOS 版本...${NC}"
    if npx electron-builder --mac; then
        echo -e "${GREEN}✅ macOS 构建成功${NC}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}❌ macOS 构建失败${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi

# 显示最终结果
echo ""
echo "========================================"
echo "  🎉 构建完成！"
echo "========================================"
echo ""
echo "📊 构建统计:"
echo "   总计: $TOTAL_COUNT"
echo -e "   成功: ${GREEN}$SUCCESS_COUNT${NC}"
echo -e "   失败: ${RED}$FAIL_COUNT${NC}"
echo ""
echo "📁 输出目录: electron-dist/"
echo ""

if [ -d "electron-dist" ]; then
    echo "📦 生成的文件:"
    echo "----------------------------------------"
    ls -lh electron-dist/ 2>/dev/null | tail -n +2 || echo "   目录为空"
    echo "----------------------------------------"
fi

echo ""
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${RED}⚠️  有 $FAIL_COUNT 个平台构建失败，请检查上方错误信息${NC}"
    exit 1
else
    echo -e "${GREEN}✅ 所有平台构建成功！${NC}"
fi

echo "========================================"

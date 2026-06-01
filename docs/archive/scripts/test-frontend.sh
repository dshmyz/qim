#!/bin/bash
# test-frontend.sh
# QIM 前端完整回归测试套件
# 用法: bash test-frontend.sh

cd "$(dirname "$0")/qim-client"

TOTAL_PASS=0
TOTAL_FAIL=0

echo "========================================"
echo "  QIM 前端回归测试"
echo "  日期: $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================"
echo ""

# 检查依赖
echo ">>> [1/3] 检查依赖"
if [ ! -d "node_modules" ]; then
    echo "  ⚠️ node_modules 不存在，正在安装依赖..."
    npm install
    if [ $? -ne 0 ]; then
        echo "  ❌ 依赖安装失败"
        exit 1
    fi
fi
echo "  ✅ 依赖检查通过"

# 运行单元测试
echo ""
echo ">>> [2/3] 运行单元测试"
echo ""

UNIT_OUTPUT=$(npm run test:unit 2>&1)
UNIT_EXIT=$?

echo "$UNIT_OUTPUT" | grep -E "(passed|failed|Tests|PASS|FAIL)"

# 提取测试结果
PASS_COUNT=$(echo "$UNIT_OUTPUT" | grep -o '[0-9]* passed' | head -1 | grep -o '[0-9]*')
FAIL_COUNT=$(echo "$UNIT_OUTPUT" | grep -o '[0-9]* failed' | head -1 | grep -o '[0-9]*')

if [ -z "$PASS_COUNT" ]; then
    PASS_COUNT=0
fi
if [ -z "$FAIL_COUNT" ]; then
    FAIL_COUNT=0
fi

TOTAL_PASS=$((TOTAL_PASS + PASS_COUNT))
TOTAL_FAIL=$((TOTAL_FAIL + FAIL_COUNT))

if [ $UNIT_EXIT -eq 0 ]; then
    echo ""
    echo "  ✅ 单元测试全部通过"
else
    echo ""
    echo "  ❌ 单元测试有 $FAIL_COUNT 个失败"
fi

# 运行类型检查
echo ""
echo ">>> [3/3] 运行类型检查"
echo ""

TYPE_OUTPUT=$(npm run typecheck 2>&1)
TYPE_EXIT=$?

if [ $TYPE_EXIT -eq 0 ]; then
    echo "  ✅ TypeScript 类型检查通过"
    TOTAL_PASS=$((TOTAL_PASS + 1))
else
    echo "  ❌ TypeScript 类型检查失败"
    echo "$TYPE_OUTPUT" | tail -5
    TOTAL_FAIL=$((TOTAL_FAIL + 1))
fi

# 汇总
echo ""
echo "========================================"
echo "  测试完成"
echo "  通过: $TOTAL_PASS"
echo "  失败: $TOTAL_FAIL"
echo "  总计: $((TOTAL_PASS + TOTAL_FAIL))"
echo "========================================"

if [ "$TOTAL_FAIL" -gt 0 ]; then
    exit 1
fi
exit 0

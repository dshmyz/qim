#!/usr/bin/env node

const fs = require('fs');

// 根据之前的分析结果，这些是在模板和JS中确认未使用的class
// 我们只删除那些确定不需要的样式

const mainFile = '/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue';
let content = fs.readFileSync(mainFile, 'utf-8');

// 在 style scoped 部分，要删除的 class 列表
// 这些 class 都是旧的、已经被拆分到其他组件的样式
const classesToRemove = [
  // 组织架构相关（可能还有使用，需要保留）
  // 对话框相关（这些对话框仍然在 Main.vue 中使用）
  // 所以我们要保留正在使用的样式
  
  // 只删除确认未使用的：
  // 1. 重复的定义
  // 2. 旧的应用相关样式
  // 3. 频道相关样式
  
  // 应用相关样式（Main.vue 可能仍在使用）
  // 统计报表样式
  // 动画样式（这些可能被使用）
];

// 更好的策略是：删除那些包含硬编码颜色值而不是使用 CSS 变量的样式
// 这些通常是旧代码

// 删除重复的 keyframes 定义
const duplicateKeyframes = [
  // fadeIn 定义了多次
  // pulse 定义了多次
  // spin 定义了多次
];

console.log('开始分析可删除的样式...');

// 统计样式块
const scopedStyleMatch = content.match(/<style scoped>([\s\S]*?)<\/style>/);
if (scopedStyleMatch) {
  const scopedStyle = scopedStyleMatch[1];
  
  // 计算行数
  const lines = scopedStyle.split('\n');
  console.log(`Scoped style 总行数: ${lines.length}`);
  
  // 查找包含硬编码颜色的样式（这些通常是需要清理的）
  const hardcodedColorPattern = /#[0-9a-fA-F]{3,8}(?!\))/g;
  const hardcodedColors = scopedStyle.match(hardcodedColorPattern);
  
  if (hardcodedColors) {
    console.log(`发现 ${hardcodedColors.length} 个硬编码颜色值`);
    // 统计不同的颜色
    const uniqueColors = [...new Set(hardcodedColors)];
    console.log(`不同的颜色值: ${uniqueColors.length}`);
    console.log('颜色列表:', uniqueColors);
  }
  
  // 统计 !important 使用次数
  const importantCount = (scopedStyle.match(/!important/g) || []).length;
  console.log(`!important 使用次数: ${importantCount}`);
}

// 输出建议
console.log('\n建议清理策略:');
console.log('1. 删除所有包含硬编码颜色值的样式块（如果它们可以被 CSS 变量替代）');
console.log('2. 删除重复的 @keyframes 定义');
console.log('3. 删除注释掉的样式');
console.log('4. 合并相同选择器的样式定义');

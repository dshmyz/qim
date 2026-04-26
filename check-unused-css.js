#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const files = [
  '/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue',
  '/Users/gracegaoya/work/project/qim/qim-client/src/components/chat/ChatWindow.vue'
];

function extractTemplateClasses(templateContent) {
  const classes = new Set();
  const classAttrRegex = /class=["']([^"']*)["']/g;
  let match;
  while ((match = classAttrRegex.exec(templateContent)) !== null) {
    match[1].split(/\s+/).forEach(cls => cls.trim() && classes.add(cls.trim()));
  }
  
  const arrayClassRegex = /:class=\["([^"]*)"\]/g;
  while ((match = arrayClassRegex.exec(templateContent)) !== null) {
    match[1].split(/['",\s]+/).forEach(cls => cls.trim() && classes.add(cls.trim()));
  }
  
  const objectClassRegex = /:class=\{[^}]*\}/g;
  while ((match = objectClassRegex.exec(templateContent)) !== null) {
    const classNameMatches = match[0].match(/['"]([^'"]+)['"]\s*:/g);
    if (classNameMatches) {
      classNameMatches.forEach(m => {
        const cls = m.replace(/['":\s]/g, '');
        cls && classes.add(cls);
      });
    }
  }
  
  return classes;
}

function extractStyleClasses(styleContent) {
  const classes = new Set();
  const classRegex = /\.([a-zA-Z_][a-zA-Z0-9_-]*)/g;
  let match;
  while ((match = classRegex.exec(styleContent)) !== null) {
    classes.add(match[1].trim());
  }
  
  // 排除 keyframes 名称
  const keyframesRegex = /@keyframes\s+([a-zA-Z_][a-zA-Z0-9_-]*)/g;
  const keyframesNames = new Set();
  while ((match = keyframesRegex.exec(styleContent)) !== null) {
    keyframesNames.add(match[1]);
  }
  keyframesNames.forEach(name => classes.delete(name));
  
  return classes;
}

function isClassUsed(className, templateContent, scriptContent, dynamicReferences) {
  const directPattern = new RegExp(`class=["'][^"']*\\b${className}\\b[^"']*["']`, 'g');
  if (directPattern.test(templateContent)) return true;
  
  const arrayPattern = new RegExp(`:class=\\[[^\\]]*['"]\\b${className}\\b['"][^\\]]*\\]`, 'g');
  if (arrayPattern.test(templateContent)) return true;
  
  const objectPattern = new RegExp(`:class=\\{[^\\}]*['"]\\b${className}\\b['"]\\s*:[^\\}]*\\}`, 'g');
  if (objectPattern.test(templateContent)) return true;
  
  if (dynamicReferences.includes(className)) return true;
  
  // 检查是否在 JS 中通过字符串引用
  const jsPattern = new RegExp(`['"]\\b${className}\\b['"]`, 'g');
  if (jsPattern.test(scriptContent)) return true;
  
  return false;
}

function extractDynamicClasses(scriptContent) {
  const dynamicClasses = [];
  
  const classListPattern = /classList\.add\(['"]([^'"]+)['"]\)/g;
  let match;
  while ((match = classListPattern.exec(scriptContent)) !== null) {
    dynamicClasses.push(match[1]);
  }
  
  const classNamePattern = /\.className\s*=\s*['"]([^'"]+)['"]/g;
  while ((match = classNamePattern.exec(scriptContent)) !== null) {
    dynamicClasses.push(match[1]);
  }
  
  const queryPattern = /querySelector\(['"][^"']*\.([^'" [\]]+)['"]\)/g;
  while ((match = queryPattern.exec(scriptContent)) !== null) {
    dynamicClasses.push(match[1]);
  }
  
  const queryAllPattern = /querySelectorAll\(['"][^"']*\.([^'" [\]]+)['"]\)/g;
  while ((match = queryAllPattern.exec(scriptContent)) !== null) {
    dynamicClasses.push(match[1]);
  }
  
  return dynamicClasses;
}

function analyzeFile(filePath) {
  if (!fs.existsSync(filePath)) {
    console.log(`文件不存在: ${filePath}`);
    return [];
  }
  
  const content = fs.readFileSync(filePath, 'utf-8');
  const fileName = path.basename(filePath);
  
  const templateMatch = content.match(/<template[^>]*>([\s\S]*?)<\/template>/);
  const templateContent = templateMatch ? templateMatch[1] : '';
  
  const styleMatches = content.match(/<style[^>]*>([\s\S]*?)<\/style>/g);
  let allStyleContent = '';
  if (styleMatches) {
    styleMatches.forEach(match => {
      allStyleContent += match.replace(/<\/?style[^>]*>/g, '');
    });
  }
  
  const scriptMatch = content.match(/<script[^>]*>([\s\S]*?)<\/script>/);
  const scriptContent = scriptMatch ? scriptMatch[1] : '';
  
  const definedClasses = extractStyleClasses(allStyleContent);
  const usedClasses = extractTemplateClasses(templateContent);
  const dynamicClasses = extractDynamicClasses(scriptContent);
  
  const unusedClasses = [];
  definedClasses.forEach(cls => {
    if (!isClassUsed(cls, templateContent, scriptContent, dynamicClasses)) {
      unusedClasses.push(cls);
    }
  });
  
  console.log(`\n${'='.repeat(80)}`);
  console.log(`文件: ${fileName}`);
  console.log(`${'='.repeat(80)}`);
  console.log(`定义的 CSS class 总数: ${definedClasses.size}`);
  console.log(`模板中使用的 class 总数: ${usedClasses.size}`);
  console.log(`动态使用的 class 数量: ${dynamicClasses.length}`);
  console.log(`未使用的 class 数量: ${unusedClasses.length}`);
  
  if (unusedClasses.length > 0) {
    console.log('\n未使用的 CSS class 列表:');
    unusedClasses.sort().forEach(cls => {
      console.log(`  .${cls}`);
    });
  } else {
    console.log('\n✓ 所有 CSS class 都在使用！');
  }
  
  return unusedClasses;
}

let allUnusedClasses = [];
files.forEach(file => {
  const unused = analyzeFile(file);
  if (unused) {
    allUnusedClasses = allUnusedClasses.concat(unused.map(cls => ({ file: path.basename(file), class: cls })));
  }
});

console.log(`\n${'='.repeat(80)}`);
console.log('汇总报告');
console.log(`${'='.repeat(80)}`);
console.log(`总共发现 ${allUnusedClasses.length} 个未使用的 CSS class`);

if (allUnusedClasses.length > 0) {
  console.log('\n按文件分类:');
  const byFile = {};
  allUnusedClasses.forEach(({ file, class: cls }) => {
    if (!byFile[file]) byFile[file] = [];
    byFile[file].push(cls);
  });
  
  Object.keys(byFile).forEach(file => {
    console.log(`\n${file}: ${byFile[file].length} 个未使用的 class`);
  });
}

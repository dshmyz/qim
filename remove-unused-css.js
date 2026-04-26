#!/usr/bin/env node

const fs = require('fs');

// 读取文件
function readFile(path) {
  return fs.readFileSync(path, 'utf-8');
}

// 提取模板中使用的 class
function extractTemplateClasses(templateContent) {
  const classes = new Set();
  
  // class="..." 或 class='...'
  const classAttrRegex = /class=["']([^"']*)["']/g;
  let match;
  while ((match = classAttrRegex.exec(templateContent)) !== null) {
    match[1].split(/\s+/).forEach(cls => cls.trim() && classes.add(cls.trim()));
  }
  
  // :class="['...']"
  const arrayClassRegex = /:class=\["([^"]*)"\]/g;
  while ((match = arrayClassRegex.exec(templateContent)) !== null) {
    match[1].split(/['",\s]+/).forEach(cls => cls.trim() && classes.add(cls.trim()));
  }
  
  // :class="{ 'className': condition }"
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
  
  // v-bind:class
  const bindClassRegex = /v-bind:class=["']([^"']*)["']/g;
  while ((match = bindClassRegex.exec(templateContent)) !== null) {
    match[1].split(/\s+/).forEach(cls => cls.trim() && classes.add(cls.trim()));
  }
  
  return classes;
}

// 提取脚本中动态使用的 class
function extractScriptClasses(scriptContent) {
  const classes = new Set();
  
  // classList.add('...')
  const classListPattern = /classList\.add\(['"]([^'"]+)['"]\)/g;
  let match;
  while ((match = classListPattern.exec(scriptContent)) !== null) {
    classes.add(match[1]);
  }
  
  // className = '...'
  const classNamePattern = /\.className\s*=\s*['"]([^'"]+)['"]/g;
  while ((match = classNamePattern.exec(scriptContent)) !== null) {
    classes.add(match[1]);
  }
  
  // querySelector 中的 class
  const queryPattern = /querySelector\(['"][^"']*\.([^'" [\]]+)['"]\)/g;
  while ((match = queryPattern.exec(scriptContent)) !== null) {
    classes.add(match[1]);
  }
  
  const queryAllPattern = /querySelectorAll\(['"][^"']*\.([^'" [\]]+)['"]\)/g;
  while ((match = queryAllPattern.exec(scriptContent)) !== null) {
    classes.add(match[1]);
  }
  
  return classes;
}

// 提取 keyframes 名称
function extractKeyframesNames(styleContent) {
  const names = new Set();
  const keyframesRegex = /@keyframes\s+([a-zA-Z_][a-zA-Z0-9_-]*)/g;
  let match;
  while ((match = keyframesRegex.exec(styleContent)) !== null) {
    names.add(match[1]);
  }
  return names;
}

// 检查 class 是否在模板或脚本中使用
function isClassUsed(className, usedClasses, scriptClasses, keyframesNames, styleContent) {
  // 直接使用
  if (usedClasses.has(className)) return true;
  if (scriptClasses.has(className)) return true;
  if (keyframesNames.has(className)) return true;
  
  // 检查是否在 JS 代码中通过字符串引用
  const jsPatterns = [
    new RegExp(`['"]${className}['"]`),
    new RegExp(`\\.className.*['"]${className}['"]`),
    new RegExp(`classList\\..*['"]${className}['"]`),
  ];
  
  for (const pattern of jsPatterns) {
    if (pattern.test(styleContent)) return true;
  }
  
  // 检查是否是伪类/伪元素选择器的一部分
  const pseudoSelectors = [
    `${className}:hover`,
    `${className}:active`,
    `${className}:focus`,
    `${className}:visited`,
    `${className}:before`,
    `${className}:after`,
  ];
  
  for (const pseudo of pseudoSelectors) {
    if (styleContent.includes(pseudo)) return true;
  }
  
  // 检查是否作为子选择器使用（如 .parent .className）
  const childSelectorPattern = new RegExp(`\\.\\w+\\s+\\.${className}\\b`);
  if (childSelectorPattern.test(styleContent)) return true;
  
  // 检查是否是主题预览相关的class（这些是动态添加的）
  if (className.endsWith('-theme') && className !== 'dark-theme' && className !== 'light-theme') {
    // 主题类是动态添加的，需要保留
    return true;
  }
  
  // 检查是否是动画相关的class
  if (className.includes('fade-') || className.includes('slide-') || className.includes('scale-')) {
    // Vue transition 相关的 class
    return true;
  }
  
  return false;
}

// 从样式内容中移除未使用的 class 定义
function removeUnusedClasses(styleContent, usedClasses, scriptClasses, keyframesNames) {
  // 提取所有样式定义
  const classPattern = /\.([a-zA-Z_][a-zA-Z0-9_-]*)\s*\{/g;
  const allClasses = new Set();
  let match;
  
  while ((match = classPattern.exec(styleContent)) !== null) {
    allClasses.add(match[1]);
  }
  
  // 找出未使用的 class
  const unusedClasses = [];
  allClasses.forEach(cls => {
    if (!isClassUsed(cls, usedClasses, scriptClasses, keyframesNames, styleContent)) {
      unusedClasses.push(cls);
    }
  });
  
  console.log(`发现 ${unusedClasses.length} 个未使用的 class`);
  
  // 移除未使用的 class 定义
  let cleanedContent = styleContent;
  
  // 按 class 名称长度排序，先移除长的（避免误删嵌套的）
  unusedClasses.sort((a, b) => b.length - a.length);
  
  let removedCount = 0;
  unusedClasses.forEach(className => {
    // 匹配完整的 class 定义（包括所有嵌套的 {}）
    const escapedClassName = className.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    
    // 匹配 .className { ... } 包括嵌套的花括号
    const classRegex = new RegExp(`\\s*\\.${escapedClassName}\\s*(?:::[a-z-]+)?\\s*(?::[a-z-]+)?\\s*\\{`, 'g');
    
    let classMatch;
    while ((classMatch = classRegex.exec(cleanedContent)) !== null) {
      const startIndex = classMatch.index;
      // 找到匹配的结束花括号
      let braceCount = 0;
      let endIndex = startIndex;
      let found = false;
      
      for (let i = startIndex; i < cleanedContent.length; i++) {
        if (cleanedContent[i] === '{') braceCount++;
        else if (cleanedContent[i] === '}') {
          braceCount--;
          if (braceCount === 0) {
            endIndex = i + 1;
            found = true;
            break;
          }
        }
      }
      
      if (found) {
        // 检查是否包含子选择器（保留父选择器）
        const classDefinition = cleanedContent.substring(startIndex, endIndex);
        const hasChildSelectors = /\.([a-zA-Z_][a-zA-Z0-9_-]*)\s*\{/.test(
          classDefinition.replace(new RegExp(`\\.${escapedClassName}`), '')
        );
        
        if (!hasChildSelectors) {
          // 移除整个定义
          cleanedContent = cleanedContent.substring(0, startIndex) + cleanedContent.substring(endIndex);
          // 调整索引
          classRegex.lastIndex = startIndex;
          removedCount++;
        }
      }
    }
  });
  
  console.log(`移除了 ${removedCount} 个 class 定义`);
  
  // 清理多余的空行（超过2个连续空行替换为2个）
  cleanedContent = cleanedContent.replace(/\n{3,}/g, '\n\n');
  
  return cleanedContent;
}

// 主函数
function processFile(filePath) {
  console.log(`\n处理文件: ${filePath}`);
  const content = readFile(filePath);
  
  // 提取 template
  const templateMatch = content.match(/<template[^>]*>([\s\S]*?)<\/template>/);
  const templateContent = templateMatch ? templateMatch[1] : '';
  
  // 提取 script
  const scriptMatch = content.match(/<script[^>]*>([\s\S]*?)<\/script>/);
  const scriptContent = scriptMatch ? scriptMatch[1] : '';
  
  // 提取所有 style 块
  const styleRegex = /(<style[^>]*>)([\s\S]*?)(<\/style>)/g;
  const styles = [];
  let styleMatch;
  
  while ((styleMatch = styleRegex.exec(content)) !== null) {
    styles.push({
      full: styleMatch[0],
      open: styleMatch[1],
      content: styleMatch[2],
      close: styleMatch[3],
      index: styleMatch.index
    });
  }
  
  // 提取使用的 class
  const usedClasses = extractTemplateClasses(templateContent);
  const scriptClasses = extractScriptClasses(scriptContent);
  
  console.log(`模板中使用的 class: ${usedClasses.size}`);
  console.log(`脚本中使用的 class: ${scriptClasses.size}`);
  
  // 处理每个 style 块
  let newContent = content;
  styles.reverse().forEach((style, i) => {
    console.log(`\n处理 style 块 ${i + 1}...`);
    
    const keyframesNames = extractKeyframesNames(style.content);
    const cleanedStyle = removeUnusedClasses(style.content, usedClasses, scriptClasses, keyframesNames);
    
    // 替换原内容
    const newStyle = style.open + cleanedStyle + style.close;
    newContent = newContent.substring(0, style.index) + newStyle + newContent.substring(style.index + style.full.length);
  });
  
  // 写回文件
  fs.writeFileSync(filePath, newContent, 'utf-8');
  console.log(`\n文件已更新: ${filePath}`);
  
  // 计算减少的行数
  const originalLines = content.split('\n').length;
  const newLines = newContent.split('\n').length;
  console.log(`原始行数: ${originalLines}`);
  console.log(`新行数: ${newLines}`);
  console.log(`减少了 ${originalLines - newLines} 行`);
}

// 处理文件
const files = [
  '/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue',
  '/Users/gracegaoya/work/project/qim/qim-client/src/components/chat/ChatWindow.vue'
];

files.forEach(processFile);

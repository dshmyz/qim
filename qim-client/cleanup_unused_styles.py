#!/usr/bin/env python3
"""
清理 Main.vue 中未使用的样式
- 读取模板中实际使用的类名
- 删除未使用的样式块
- 备份到文件中
"""
import re
import os

# 读取文件
with open('src/views/Main.vue', 'r') as f:
    content = f.read()

# 提取模板中使用的类名
template_match = re.search(r'<template>(.*?)</template>', content, re.DOTALL)
template_classes = set()

if template_match:
    template_content = template_match.group(1)
    # 处理动态绑定 :class="..."
    for match in re.finditer(r':class="([^"]+)"', template_content):
        val = match.group(1)
        for str_match in re.findall(r"'([^']+)'", val):
            template_classes.add(str_match.strip().rstrip(':'))
    
    # 处理静态 class="..."
    for match in re.finditer(r'class="([^"]+)"', template_content):
        for c in match.group(1).split():
            c = c.strip()
            if c:
                template_classes.add(c)

# 保留的类名（即使在模板中未使用也需要保留）
keep_classes = {
    # 主题类（用于 data-theme 选择器）
    'modern-light', 'elegant-dark', 'ocean-blue', 'elegant-purple',
    'warm-amber', 'crimson-red', 'emerald-green', 'urban-jungle',
    'mediterranean-dream', 'monochrome-elegance', 'spring-blossom',
    # 过渡动画类
    'fade', 'slide', 'scale', 'pulse', 'gradient-bg',
    # 动态类名
    'active', 'selected', 'new-version',
}
template_classes.update(keep_classes)

print(f"模板使用了 {len(template_classes)} 个类")

# 提取 scoped style 块
style_scoped_start = content.index('<style scoped>')
style_scoped_end = content.index('</style>', style_scoped_start) + 8
scoped_content = content[style_scoped_start:style_scoped_end]

# 提取非 scoped style 块
style_start = content.index('<style>')
style_end = content.index('</style>', style_start) + 8
global_style_content = content[style_start:style_end]

# 未使用的样式模式（已迁移到子组件）
unused_style_patterns = [
    # Sidebar 相关 (已迁移到 Sidebar.vue)
    r'/\*\s*侧边栏样式\s*\*/\s*\.sidebar\s*\{[^}]+\}',
    r'\.sidebar\.collapsed\s*\{[^}]+\}',
    r'\.sidebar-header\s*\{[^}]+\}',
    r'\.sidebar-content\s*\{[^}]+\}',
    
    # Conversation 相关 (已迁移到 ConversationList.vue)
    r'\.conversation-list\s*\{[^}]+\}',
    r'\.conversation-item\s*\{[^}]+\}',
    r'\.conversation-item:hover\s*\{[^}]+\}',
    r'\.conversation-item\.active\s*\{[^}]+\}',
    r'\.conversation-avatar\s*\{[^}]+\}',
    r'\.conversation-info\s*\{[^}]+\}',
    r'\.conversation-name\s*\{[^}]+\}',
    r'\.conversation-preview\s*\{[^}]+\}',
    r'\.conversation-meta\s*\{[^}]+\}',
    r'\.conversation-time\s*\{[^}]+\}',
    
    # Group 相关 (已迁移到 GroupList.vue)
    r'\.groups-section\s*\{[^}]+\}',
    r'\.groups-header\s*\{[^}]+\}',
    r'\.groups-list\s*\{[^}]+\}',
    r'\.group-item\s*\{[^}]+\}',
    r'\.group-item:hover\s*\{[^}]+\}',
    r'\.group-badge\s*\{[^}]+\}',
    r'\.group-preview\s*\{[^}]+\}',
    r'\.group-meta\s*\{[^}]+\}',
    r'\.group-time\s*\{[^}]+\}',
    
    # Tree/Org 相关 (已迁移到 OrgTree.vue)
    r'\.tree-container\s*\{[^}]+\}',
    r'\.tree-node\s*\{[^}]+\}',
    r'\.tree-node-content\s*\{[^}]+\}',
    r'\.tree-node-content:hover\s*\{[^}]+\}',
    r'\.tree-children\s*\{[^}]+\}',
    r'\.department-node\s*\{[^}]+\}',
    r'\.employee-node\s*\{[^}]+\}',
    
    # Search popup 相关 (已迁移到 SearchResult.vue)
    r'\.search-popup\s*\{[^}]+\}',
    r'\.search-popup-content\s*\{[^}]+\}',
    r'\.search-popup-header\s*\{[^}]+\}',
    r'\.search-popup-count\s*\{[^}]+\}',
    r'\.search-popup-list\s*\{[^}]+\}',
    r'\.search-popup-item\s*\{[^}]+\}',
    r'\.search-popup-item:hover\s*\{[^}]+\}',
    r'\.search-popup-avatar\s*\{[^}]+\}',
    r'\.search-popup-info\s*\{[^}]+\}',
    r'\.search-popup-name\s*\{[^}]+\}',
    r'\.search-popup-meta\s*\{[^}]+\}',
    r'\.search-popup-btn\s*\{[^}]+\}',
    r'\.search-popup-btn:hover\s*\{[^}]+\}',
    
    # App/Category 相关 (已迁移到 AppPanel.vue)
    r'\.app-categories\s*\{[^}]+\}',
    r'\.app-category-item\s*\{[^}]+\}',
    r'\.category-header\s*\{[^}]+\}',
    r'\.category-icon\s*\{[^}]+\}',
    r'\.category-name\s*\{[^}]+\}',
    r'\.category-toggle\s*\{[^}]+\}',
    r'\.category-apps\s*\{[^}]+\}',
    r'\.category-app-item\s*\{[^}]+\}',
    r'\.category-app-icon\s*\{[^}]+\}',
    r'\.category-app-name\s*\{[^}]+\}',
    r'\.app-tab-content\s*\{[^}]+\}',
    r'\.app-tabs\s*\{[^}]+\}',
    r'\.app-tab-item\s*\{[^}]+\}',
    r'\.tab-icon\s*\{[^}]+\}',
    r'\.tab-name\s*\{[^}]+\}',
    r'\.apps-container\s*\{[^}]+\}',
    
    # 其他未使用的
    r'\.recent-apps-section\s*\{[^}]+\}',
    r'\.recent-app-grid-item\s*\{[^}]+\}',
    r'\.recent-app-grid-icon\s*\{[^}]+\}',
    r'\.recent-app-grid-name\s*\{[^}]+\}',
]

# 清理 scoped style
removed_count = 0
for pattern in unused_style_patterns:
    matches = re.findall(pattern, scoped_content, re.DOTALL)
    if matches:
        scoped_content = re.sub(pattern, '', scoped_content, flags=re.DOTALL)
        removed_count += len(matches)
        print(f"删除: {pattern[:50]}... ({len(matches)} 个)")

# 清理非 scoped style 块中的废弃主题子样式
unused_theme_patterns = [
    # 已迁移组件的主题样式
    r'\[data-theme="[^"]+"\]\s*\.sidebar-content\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.conversation-list\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.conversation-item\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.groups-list\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.group-item\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.tree-container\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.sidebar-header\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.org-content\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.employee-node\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.recent-app-grid-icon\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-app-icon\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.tab-icon\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-icon\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.recent-app-icon\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.search-popup\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.group-badge\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.markdown-code\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.markdown-link\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.note-content-input\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.file-item:hover\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.tree-node-content:hover\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.conversation-item:hover\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-header:hover\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-apps\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-app-item\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.recent-app-grid-item\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.app-category-item\s*\{[^}]+\}',
    r'\[data-theme="[^"]+"\]\s*\.category-header\s*\{[^}]+\}',
]

for pattern in unused_theme_patterns:
    matches = re.findall(pattern, global_style_content, re.DOTALL)
    if matches:
        global_style_content = re.sub(pattern, '', global_style_content, flags=re.DOTALL)
        removed_count += len(matches)
        print(f"删除主题样式: {pattern[:50]}... ({len(matches)} 个)")

# 清理连续空行
scoped_content = re.sub(r'\n\s*\n\s*\n', '\n\n', scoped_content)
global_style_content = re.sub(r'\n\s*\n\s*\n', '\n\n', global_style_content)

# 写回文件
new_content = content[:style_start] + global_style_content + content[style_end:style_scoped_start] + scoped_content + content[style_scoped_end:]

# 验证
original_lines = content.count('\n')
new_lines = new_content.count('\n')
removed_lines = original_lines - new_lines

print(f"\n清理完成!")
print(f"原始行数: {original_lines}")
print(f"清理后行数: {new_lines}")
print(f"删除行数: {removed_lines}")
print(f"共删除 {removed_count} 个样式规则")

# 写入新文件
with open('src/views/Main.vue', 'w') as f:
    f.write(new_content)

print("\n文件已更新")

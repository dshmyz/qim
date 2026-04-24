import re

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
            template_classes.add(str_match.strip())
    
    # 处理静态 class="..."
    for match in re.finditer(r'class="([^"]+)"', template_content):
        for c in match.group(1).split():
            c = c.strip()
            if c:
                template_classes.add(c)

# 这些是主题选择器中的类名，需要保留
theme_classes = {
    'modern-light', 'elegant-dark', 'ocean-blue', 'elegant-purple',
    'warm-amber', 'crimson-red', 'emerald-green', 'urban-jungle',
    'mediterranean-dream', 'monochrome-elegance', 'spring-blossom'
}
template_classes.update(theme_classes)

print(f"Template uses {len(template_classes)} unique classes")

# 找到非 scoped style 块 (6637-8102)
style1_start = content.index('<style>')
style1_end = content.index('</style>', style1_start)
style1_content = content[style1_start:style1_end + 8]

# 找到 scoped style 块 (8104-结尾)
style2_start = content.index('<style scoped>', style1_end)
style2_end = content.index('</style>', style2_start)
style2_content = content[style2_start:style2_end + 8]

print(f"Style 1 (non-scoped): {style1_content.count(chr(10))} lines")
print(f"Style 2 (scoped): {style2_content.count(chr(10))} lines")

# 清理非 scoped style 块
def clean_style_block(style_text, used_classes):
    lines = style_text.split('\n')
    result_lines = []
    skip_block = False
    block_depth = 0
    current_selectors = []
    
    i = 0
    while i < len(lines):
        line = lines[i]
        
        # 跳过 @import 和注释
        if line.strip().startswith('@import') or line.strip().startswith('/*'):
            result_lines.append(line)
            i += 1
            continue
        
        # 检查是否包含 :root 或 [data-theme
        if ':root' in line or '[data-theme' in line:
            # 保留整个主题定义块
            brace_count = 0
            block_lines = [line]
            for ch in line:
                if ch == '{':
                    brace_count += 1
                elif ch == '}':
                    brace_count -= 1
            
            if brace_count > 0:
                i += 1
                while i < len(lines) and brace_count > 0:
                    block_lines.append(lines[i])
                    for ch in lines[i]:
                        if ch == '{':
                            brace_count += 1
                        elif ch == '}':
                            brace_count -= 1
                    i += 1
                result_lines.extend(block_lines)
            else:
                result_lines.append(line)
                i += 1
            continue
        
        # 检查是否是新的 CSS 规则
        if '{' in line and '}' not in line:
            selector = line.split('{')[0].strip()
            # 提取类名
            selector_classes = set()
            for part in selector.split():
                if part.startswith('.'):
                    cls = part.split('.')[1].split(':')[0].split(',')[0].strip()
                    selector_classes.add(cls)
            
            # 检查是否有使用的类
            has_used = any(cls in used_classes for cls in selector_classes)
            
            if has_used:
                result_lines.append(line)
                i += 1
            else:
                # 跳过整个块
                skip_block = True
                block_depth = 1
                result_lines.append(f'/* REMOVED: {selector} */')
                i += 1
        elif skip_block:
            for ch in line:
                if ch == '{':
                    block_depth += 1
                elif ch == '}':
                    block_depth -= 1
                    if block_depth <= 0:
                        skip_block = False
                        break
            i += 1
        else:
            result_lines.append(line)
            i += 1
    
    return '\n'.join(result_lines)

# 由于清理逻辑复杂，改为精确删除已知未使用的样式块
# 这些是已确认在模板中未使用且属于已迁移组件的样式
unused_patterns = [
    # Sidebar 相关
    (r'\.sidebar\s*\{[^}]*\}', 'sidebar'),
    (r'\.sidebar\.collapsed\s*\{[^}]*\}', 'sidebar.collapsed'),
    (r'\.sidebar-header\s*\{[^}]*\}', 'sidebar-header'),
    (r'\.sidebar-content\s*\{[^}]*\}', 'sidebar-content'),
    # Conversation 相关
    (r'\.conversation-list\s*\{[^}]*\}', 'conversation-list'),
    (r'\.conversation-item\s*\{[^}]*\}', 'conversation-item'),
    (r'\.conversation-item:hover\s*\{[^}]*\}', 'conversation-item:hover'),
    (r'\.conversation-item\.active\s*\{[^}]*\}', 'conversation-item.active'),
    # Group 相关
    (r'\.groups-section\s*\{[^}]*\}', 'groups-section'),
    (r'\.groups-header\s*\{[^}]*\}', 'groups-header'),
    (r'\.groups-list\s*\{[^}]*\}', 'groups-list'),
    (r'\.group-item\s*\{[^}]*\}', 'group-item'),
    (r'\.group-item:hover\s*\{[^}]*\}', 'group-item:hover'),
    (r'\.group-badge\s*\{[^}]*\}', 'group-badge'),
    # Tree 相关
    (r'\.tree-container\s*\{[^}]*\}', 'tree-container'),
    (r'\.tree-node\s*\{[^}]*\}', 'tree-node'),
    (r'\.tree-node-content\s*\{[^}]*\}', 'tree-node-content'),
    (r'\.tree-node-content:hover\s*\{[^}]*\}', 'tree-node-content:hover'),
    # Search popup 相关
    (r'\.search-popup\s*\{[^}]*\}', 'search-popup'),
    (r'\.search-popup-content\s*\{[^}]*\}', 'search-popup-content'),
    (r'\.search-popup-header\s*\{[^}]*\}', 'search-popup-header'),
    (r'\.search-popup-count\s*\{[^}]*\}', 'search-popup-count'),
    (r'\.search-popup-list\s*\{[^}]*\}', 'search-popup-list'),
    (r'\.search-popup-item\s*\{[^}]*\}', 'search-popup-item'),
    (r'\.search-popup-item:hover\s*\{[^}]*\}', 'search-popup-item:hover'),
    (r'\.search-popup-avatar\s*\{[^}]*\}', 'search-popup-avatar'),
    (r'\.search-popup-info\s*\{[^}]*\}', 'search-popup-info'),
    (r'\.search-popup-name\s*\{[^}]*\}', 'search-popup-name'),
    (r'\.search-popup-meta\s*\{[^}]*\}', 'search-popup-meta'),
    (r'\.search-popup-btn\s*\{[^}]*\}', 'search-popup-btn'),
    (r'\.search-popup-btn:hover\s*\{[^}]*\}', 'search-popup-btn:hover'),
]

# 统计将要删除的样式数量
for pattern, name in unused_patterns:
    matches = re.findall(pattern, style2_content, re.DOTALL)
    if matches:
        print(f"Found {len(matches)} match(es) for {name}")

print("\nScript complete. Ready to clean up.")

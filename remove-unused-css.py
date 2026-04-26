#!/usr/bin/env python3
import re
import sys

def extract_template_classes(template_content):
    """提取模板中使用的所有class"""
    classes = set()
    
    # class="..." 或 class='...'
    pattern1 = r'class=["\']([^"\']*)["\']'
    for match in re.finditer(pattern1, template_content):
        for cls in match.group(1).split():
            if cls.strip():
                classes.add(cls.strip())
    
    # :class="{ 'className': condition }"
    pattern2 = r':class=\{([^}]*)\}'
    for match in re.finditer(pattern2, template_content):
        inner = match.group(1)
        for cls_match in re.finditer(r'["\']([^"\']+)["\']\s*:', inner):
            classes.add(cls_match.group(1))
    
    return classes

def extract_style_classes(style_content):
    """提取样式中定义的所有class"""
    classes = set()
    pattern = r'\.([a-zA-Z_][a-zA-Z0-9_-]*)\s*(?::(?:before|after|hover|active|focus|visited|first-child|last-child|nth-child\([^)]*\)))?\s*\{'
    for match in re.finditer(pattern, style_content):
        classes.add(match.group(1))
    return classes

def is_class_used(class_name, template_content, script_content, style_content):
    """检查class是否在模板或脚本中使用"""
    # 直接在模板中使用
    if f'class="{class_name}' in template_content or \
       f"class='{class_name}" in template_content or \
       f'class=" {class_name}' in template_content:
        return True
    
    # 在数组或对象语法中
    if f"'{class_name}'" in script_content or \
       f'"{class_name}"' in script_content:
        return True
    
    # 动态添加
    if f'classList.add("{class_name}")' in script_content or \
       f"classList.add('{class_name}')" in script_content:
        return True
    
    # 作为子选择器使用
    child_pattern = rf'\.\w+\s+{re.escape("." + class_name)}'
    if re.search(child_pattern, style_content):
        return True
    
    return False

def remove_unused_styles(file_path):
    """移除未使用的样式"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 提取模板
    template_match = re.search(r'<template>(.*?)</template>', content, re.DOTALL)
    template_content = template_match.group(1) if template_match else ''
    
    # 提取脚本
    script_match = re.search(r'<script[^>]*>(.*?)</script>', content, re.DOTALL)
    script_content = script_match.group(1) if script_match else ''
    
    # 提取样式块
    style_pattern = r'<style[^>]*>(.*?)</style>'
    style_matches = list(re.finditer(style_pattern, content, re.DOTALL))
    
    total_removed = 0
    
    # 从后向前处理，避免索引变化
    for style_match in reversed(style_matches):
        style_content = style_match.group(1)
        style_start = style_match.start(1)
        style_end = style_match.end(1)
        
        # 提取样式中定义的class
        defined_classes = extract_style_classes(style_content)
        
        unused_classes = []
        for cls in defined_classes:
            if not is_class_used(cls, template_content, script_content, style_content):
                unused_classes.append(cls)
        
        print(f"发现 {len(unused_classes)} 个未使用的class")
        
        # 删除未使用的class定义
        for cls in sorted(unused_classes, key=len, reverse=True):
            # 匹配完整的class定义（包括花括号内容）
            escaped_cls = re.escape(cls)
            # 匹配 .class-name { ... } 包括嵌套的花括号
            pattern = rf'\s*\.{escaped_cls}(?::[a-z-]+)?\s*\{{'
            
            for match in re.finditer(pattern, style_content):
                start = match.start()
                # 找到匹配的结束花括号
                brace_count = 0
                end = -1
                for i in range(match.start(), len(style_content)):
                    if style_content[i] == '{':
                        brace_count += 1
                    elif style_content[i] == '}':
                        brace_count -= 1
                        if brace_count == 0:
                            end = i + 1
                            break
                
                if end > 0:
                    # 检查是否包含子选择器
                    definition = style_content[start:end]
                    if not re.search(rf'\.\w+\s*\{{', definition.replace(f'.{cls}', '')):
                        style_content = style_content[:start] + style_content[end:]
                        total_removed += 1
                        break
        
        # 清理多余空行
        style_content = re.sub(r'\n{3,}', '\n\n', style_content)
        
        # 更新内容
        content = content[:style_start] + style_content + content[style_end:]
    
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"共移除了 {total_removed} 个样式定义")

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print('Usage: python remove-unused-css.py <file_path>')
        sys.exit(1)
    
    remove_unused_styles(sys.argv[1])

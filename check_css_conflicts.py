#!/usr/bin/env python3
"""检查 main.css 和 Main.vue 之间的 CSS 冲突"""

import re
import sys
from collections import defaultdict

def extract_css_rules(file_path, is_vue=False):
    """从 CSS 文件或 Vue 文件中提取样式规则"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    rules = {}
    
    # 匹配 CSS 规则: selector { property: value; }
    # 使用更强大的正则来匹配嵌套的括号
    pattern = r'([.#\w][^{]*?)\s*\{([^}]*)\}'
    
    for match in re.finditer(pattern, content, re.MULTILINE | re.DOTALL):
        selector = match.group(1).strip()
        properties_str = match.group(2).strip()
        
        # 跳过空规则
        if not selector or not properties_str:
            continue
        
        # 解析属性
        properties = {}
        for prop in properties_str.split(';'):
            prop = prop.strip()
            if ':' in prop:
                prop_name, prop_value = prop.split(':', 1)
                properties[prop_name.strip()] = prop_value.strip()
        
        # 如果是 Vue 文件，检查是否在 style 标签内
        if is_vue:
            # 简化处理，假设都在 style 内
            rules[selector] = properties
        else:
            rules[selector] = properties
    
    return rules

def find_conflicts(main_css_rules, main_vue_rules):
    """找出两个样式表之间的冲突"""
    conflicts = []
    
    for selector in main_css_rules:
        if selector in main_vue_rules:
            css_props = main_css_rules[selector]
            vue_props = main_vue_rules[selector]
            
            # 找出共同的属性
            for prop in css_props:
                if prop in vue_props:
                    css_value = css_props[prop]
                    vue_value = vue_props[prop]
                    
                    # 如果值不同，则记录冲突
                    if css_value != vue_value:
                        conflicts.append({
                            'selector': selector,
                            'property': prop,
                            'main_css': css_value,
                            'main_vue': vue_value
                        })
    
    return conflicts

def main():
    print("=== CSS 冲突检测 ===\n")
    
    # 提取样式规则
    print("正在解析 main.css...")
    main_css_rules = extract_css_rules('/Users/gracegaoya/work/project/qim/qim-client/src/assets/styles/main.css')
    print(f"找到 {len(main_css_rules)} 个规则")
    
    print("正在解析 Main.vue...")
    main_vue_rules = extract_css_rules('/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue', is_vue=True)
    print(f"找到 {len(main_vue_rules)} 个规则")
    
    print("\n=== 重复定义的选择器 ===")
    common_selectors = set(main_css_rules.keys()) & set(main_vue_rules.keys())
    if common_selectors:
        for selector in sorted(common_selectors):
            print(f"  - {selector}")
    else:
        print("  无重复定义")
    
    print("\n=== CSS 冲突详情 ===")
    conflicts = find_conflicts(main_css_rules, main_vue_rules)
    if conflicts:
        for i, conflict in enumerate(conflicts, 1):
            print(f"\n{i}. 选择器: {conflict['selector']}")
            print(f"   属性: {conflict['property']}")
            print(f"   main.css: {conflict['main_css']}")
            print(f"   Main.vue: {conflict['main_vue']}")
    else:
        print("  无冲突")
    
    # 检查硬编码颜色
    print("\n=== 硬编码颜色检查 (Main.vue) ===")
    hardcoded_colors = []
    with open('/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue', 'r', encoding='utf-8') as f:
        content = f.read()
        # 匹配十六进制颜色
        hex_colors = re.findall(r'#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b', content)
        if hex_colors:
            print(f"  找到 {len(hex_colors)} 个硬编码颜色:")
            for color in hex_colors[:20]:  # 只显示前20个
                print(f"    - #{color}")
            if len(hex_colors) > 20:
                print(f"    ... 还有 {len(hex_colors) - 20} 个")
        else:
            print("  无硬编码颜色")

if __name__ == '__main__':
    main()

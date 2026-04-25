#!/usr/bin/env python3
import re
import os
from pathlib import Path
from collections import defaultdict

def extract_css_classes(css_content):
    """从 CSS 内容中提取所有类选择器"""
    classes = set()
    # 匹配 .class-name { 格式的选择器
    pattern = r'\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{'
    matches = re.findall(pattern, css_content)
    classes.update(matches)
    return classes

def extract_vue_scoped_classes(vue_content):
    """从 Vue 文件中提取 scoped style 中的类选择器"""
    classes = set()
    # 匹配 <style scoped> 和 </style> 之间的内容
    style_pattern = r'<style[^>]*>(.*?)</style>'
    styles = re.findall(style_pattern, vue_content, re.DOTALL)

    for style in styles:
        # 在 scoped style 中，类选择器格式通常是 .class-name {
        pattern = r'\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{'
        matches = re.findall(pattern, style)
        classes.update(matches)
    return classes

def extract_all_classes_from_vue(vue_path):
    """从 Vue 文件中提取所有类（包含非 scoped 的）"""
    with open(vue_path, 'r', encoding='utf-8') as f:
        content = f.read()

    classes = set()
    # 匹配所有 <style 部分
    style_pattern = r'<style[^>]*>(.*?)</style>'
    styles = re.findall(style_pattern, content, re.DOTALL)

    for style in styles:
        pattern = r'\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{'
        matches = re.findall(pattern, style)
        classes.update(matches)
    return classes

def main():
    base_path = Path('/Users/gracegaoya/work/project/qim/qim-client/src')

    # CSS 文件路径
    css_files = {
        'main.css': base_path / 'assets/styles/main.css',
        'components.css': base_path / 'assets/styles/components.css',
        'markdown.css': base_path / 'assets/styles/markdown.css',
        'layout.css': base_path / 'assets/styles/layout.css',
        'dialogs.css': base_path / 'assets/styles/dialogs.css',
        'menus.css': base_path / 'assets/styles/menus.css',
    }

    # Vue 文件中的样式
    vue_files = {
        'Main.vue': base_path / 'views/Main.vue',
    }

    # 收集所有 CSS 文件中的类
    css_classes = defaultdict(set)
    for name, path in css_files.items():
        if path.exists():
            with open(path, 'r', encoding='utf-8') as f:
                css_classes[name] = extract_css_classes(f.read())
            print(f"{name}: {len(css_classes[name])} 个类")
        else:
            print(f"{name}: 文件不存在")

    # 收集 Main.vue 中的类
    main_vue_classes = set()
    if vue_files['Main.vue'].exists():
        with open(vue_files['Main.vue'], 'r', encoding='utf-8') as f:
            main_vue_classes = extract_all_classes_from_vue(vue_files['Main.vue'])
        print(f"Main.vue: {len(main_vue_classes)} 个类")

    print("\n" + "="*60)
    print("分析 Main.vue 与各 CSS 文件的重复类")
    print("="*60)

    conflicts = []
    for css_name, classes in css_classes.items():
        intersection = main_vue_classes & classes
        if intersection:
            conflicts.append((css_name, intersection))
            print(f"\n{css_name} 与 Main.vue 重复的类 ({len(intersection)}个):")
            for cls in sorted(intersection):
                print(f"  - .{cls}")

    # 分析组件级别的冲突（同一个类在多个 CSS 文件中定义）
    print("\n" + "="*60)
    print("分析同一类在多个 CSS 文件中的冲突")
    print("="*60)

    all_css_classes = defaultdict(list)
    for css_name, classes in css_classes.items():
        for cls in classes:
            all_css_classes[cls].append(css_name)

    multi_defined = {cls: files for cls, files in all_css_classes.items() if len(files) > 1}
    print(f"\n在多个 CSS 文件中定义的类: {len(multi_defined)}个")
    for cls, files in sorted(multi_defined.items()):
        print(f"  .{cls}: {', '.join(files)}")

if __name__ == '__main__':
    main()

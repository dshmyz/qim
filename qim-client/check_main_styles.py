#!/usr/bin/env python3
import re
import os

CSS_FILES = [
    'src/assets/styles/main.css',
    'src/assets/styles/components.css',
    'src/assets/styles/markdown.css',
    'src/assets/styles/layout.css',
    'src/assets/styles/dialogs.css',
    'src/assets/styles/menus.css',
]

def extract_classes_from_css(css_path):
    classes = {}
    full_path = os.path.join(os.path.dirname(__file__), css_path)
    if not os.path.exists(full_path):
        return classes

    with open(full_path, 'r', encoding='utf-8') as f:
        content = f.read()

    pattern = r'\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{'
    for match in re.finditer(pattern, content):
        class_name = match.group(1)
        classes[class_name] = css_path

    return classes

def extract_classes_from_vue_style(content):
    classes = {}
    style_match = re.search(r'<style>(.*?)</style>', content, re.DOTALL)
    if not style_match:
        return classes

    style_content = style_match.group(1)
    pattern = r'\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{'
    for match in re.finditer(pattern, style_content):
        class_name = match.group(1)
        classes[class_name] = 'Main.vue <style>'

    return classes

def main():
    vue_path = 'src/views/Main.vue'
    with open(vue_path, 'r', encoding='utf-8') as f:
        vue_content = f.read()

    all_css_classes = {}
    for css_file in CSS_FILES:
        classes = extract_classes_from_css(css_file)
        all_css_classes.update(classes)

    vue_classes = extract_classes_from_vue_style(vue_content)

    print("=" * 70)
    print("Main.vue 中定义的样式分析")
    print("=" * 70)

    defined_only_in_vue = []
    defined_in_both = []

    for class_name, location in sorted(vue_classes.items()):
        if class_name in all_css_classes:
            defined_in_both.append((class_name, all_css_classes[class_name]))
        else:
            defined_only_in_vue.append(class_name)

    print(f"\n已在 CSS 文件中定义的样式 (可以安全移除 Main.vue 中的定义): {len(defined_in_both)}个")
    print("-" * 70)
    for class_name, css_file in sorted(defined_in_both):
        print(f"  .{class_name:40} -> {css_file}")

    print(f"\n只在 Main.vue 中定义的样式 (需要保留或迁移到 CSS): {len(defined_only_in_vue)}个")
    print("-" * 70)
    for class_name in sorted(defined_only_in_vue):
        print(f"  .{class_name}")

    print("\n" + "=" * 70)
    print("建议操作:")
    print("=" * 70)
    print(f"1. 移除 {len(defined_in_both)} 个已在 CSS 文件中定义的样式")
    print(f"2. 保留或迁移 {len(defined_only_in_vue)} 个只在 Main.vue 中定义的样式")

    with open('styles_to_remove.txt', 'w', encoding='utf-8') as f:
        f.write("# 可以在 Main.vue 中移除的样式类 (已在 CSS 文件中定义)\n")
        for class_name, _ in sorted(defined_in_both):
            f.write(f"{class_name}\n")

    with open('styles_to_keep.txt', 'w', encoding='utf-8') as f:
        f.write("# 需要保留在 Main.vue 中的样式类 (只在 Main.vue 中定义)\n")
        for class_name in sorted(defined_only_in_vue):
            f.write(f"{class_name}\n")

    print("\n已将分析结果保存到:")
    print("  - styles_to_remove.txt (可在 Main.vue 中移除的样式)")
    print("  - styles_to_keep.txt (需要保留的样式)")

if __name__ == '__main__':
    main()
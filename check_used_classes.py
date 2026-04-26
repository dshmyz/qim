import re

# 读取 Main.vue 文件
with open('/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue', 'r', encoding='utf-8') as f:
    content = f.read()

# 提取 template 部分
template_match = re.search(r'<template>(.*?)</template>', content, re.DOTALL)
template_content = template_match.group(1) if template_match else ''

# 提取 script 部分  
script_match = re.search(r'<script[^>]*>(.*?)</script>', content, re.DOTALL)
script_content = script_match.group(1) if script_match else ''

# 提取模板中使用的 class
used_classes = set()

# class="..."
for match in re.finditer(r'class=["\']([^"\']*)["\']', template_content):
    for cls in match.group(1).split():
        if cls.strip():
            used_classes.add(cls.strip())

# :class="{ 'className': condition }"
for match in re.finditer(r':class=\{([^}]*)\}', template_content):
    inner = match.group(1)
    for cls_match in re.finditer(r'["\']([^"\']+)["\']\s*:', inner):
        used_classes.add(cls_match.group(1))

print(f"模板中使用的 class 总数: {len(used_classes)}")
print("\n使用的 class 列表:")
for cls in sorted(used_classes):
    print(f"  - {cls}")

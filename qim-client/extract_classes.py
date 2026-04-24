import re

with open('src/views/Main.vue', 'r') as f:
    content = f.read()

# 提取模板中使用的类名
template_match = re.search(r'<template>(.*?)</template>', content, re.DOTALL)
if template_match:
    template_content = template_match.group(1)
    template_classes = set()
    
    # 处理动态绑定 :class="..."
    for match in re.finditer(r':class="([^"]+)"', template_content):
        val = match.group(1)
        for str_match in re.findall(r"'([^']+)'", val):
            template_classes.add(str_match)
    
    # 处理静态 class="..."
    for match in re.finditer(r'class="([^"]+)"', template_content):
        for c in match.group(1).split():
            c = c.strip()
            if c:
                template_classes.add(c)

print(f"Template uses {len(template_classes)} unique classes:")
for c in sorted(template_classes):
    print(c)

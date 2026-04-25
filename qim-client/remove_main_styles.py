#!/usr/bin/env python3
import re
import os

REMOVE_CLASSES = [
    'call-duration', 'call-status', 'call-status-text', 'conversation-item',
    'delete-btn', 'end-call-btn', 'error-actions', 'error-icon', 'error-message',
    'file-action-btn', 'file-item', 'login-btn', 'main-app-item', 'markdown-bold',
    'markdown-code', 'markdown-heading', 'markdown-image', 'markdown-italic',
    'markdown-link', 'markdown-list', 'markdown-list-item', 'markdown-quote',
    'markdown-table', 'network-error', 'network-error-content', 'offline', 'online',
    'retry-btn', 'search-popup', 'search-popup-avatar', 'search-popup-btn',
    'search-popup-content', 'search-popup-count', 'search-popup-header',
    'search-popup-info', 'search-popup-item', 'search-popup-list', 'search-popup-meta',
    'search-popup-name', 'search-popup-status', 'search-popup-type', 'search-popup-username',
    'tree-node-content', 'voice-call-body', 'voice-call-content', 'voice-call-footer',
    'voice-call-header', 'voice-call-modal'
]

KEEP_CLASSES = ['main-content', 'modal-content', 'right-content', 'sidebar']

def remove_css_class(content, class_name):
    pattern = rf'\/\*\s*[^*]*\*\+\/\s*\.{class_name}\s*\{{[^}}]*\}}'
    content = re.sub(pattern, '', content, flags=re.DOTALL)

    pattern = rf'\/\*[^*]*\*\/\s*\.{class_name}\s*\{{[^}}]*\}}'
    content = re.sub(pattern, '', content, flags=re.DOTALL)

    pattern = rf'^\s*\.{class_name}\s*\{{[^}}]*\}}'
    content = re.sub(pattern, '', content, flags=re.MULTILINE | re.DOTALL)

    return content

def main():
    vue_path = 'src/views/Main.vue'

    with open(vue_path, 'r', encoding='utf-8') as f:
        content = f.read()

    print(f"处理前文件大小: {len(content)} 字符")

    original_content = content

    for class_name in REMOVE_CLASSES:
        new_content = remove_css_class(content, class_name)
        if new_content != content:
            print(f"移除样式: .{class_name}")
            content = new_content

    for class_name in KEEP_CLASSES:
        print(f"保留样式: .{class_name}")

    print(f"\n处理后文件大小: {len(content)} 字符")
    print(f"移除字符数: {len(original_content) - len(content)}")

    with open(vue_path, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"\n已更新 {vue_path}")

if __name__ == '__main__':
    main()
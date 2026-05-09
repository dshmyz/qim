#!/usr/bin/env python3
"""
Windows ICO 图标生成器
生成包含多种尺寸的 .ico 文件，用于 Windows 应用程序图标
"""

from PIL import Image
import os

# 配置
ICONS_DIR = os.path.dirname(os.path.abspath(__file__))
SOURCE_IMAGE = os.path.join(ICONS_DIR, 'icon_512x512.png')
OUTPUT_ICO = os.path.join(ICONS_DIR, 'icon.ico')

# Windows ICO 标准尺寸
ICO_SIZES = [16, 32, 48, 64, 128, 256]


def generate_ico():
    """生成 Windows .ico 文件"""
    if not os.path.exists(SOURCE_IMAGE):
        print(f'❌ 错误: 找不到源图片 {SOURCE_IMAGE}')
        print('提示: 请确保 electron/icons/icon_512x512.png 存在')
        return False

    print('🚀 开始生成 Windows ICO 文件...')
    print()

    # 打开源图片
    source = Image.open(SOURCE_IMAGE)
    print(f'📐 源图片尺寸: {source.size}')

    # 调整所有尺寸到列表
    icons = []
    for size in ICO_SIZES:
        # 缩放并保持宽高比
        icon = source.resize((size, size), Image.LANCZOS)
        icons.append(icon)
        print(f'✅ {size}x{size}')

    # 保存为 .ico 文件（包含所有尺寸）
    icons[0].save(
        OUTPUT_ICO,
        format='ICO',
        sizes=[(icon.size) for icon in icons],
        append_images=icons[1:]
    )

    print()
    print(f'✅ 成功生成: {OUTPUT_ICO}')
    print(f'📦 包含尺寸: {", ".join([f"{s}x{s}" for s in ICO_SIZES])}')
    return True


if __name__ == '__main__':
    try:
        success = generate_ico()
        if not success:
            exit(1)
    except Exception as e:
        print(f'❌ 生成失败: {e}')
        print()
        print('需要安装 Pillow 库:')
        print('  pip install Pillow')
        exit(1)

#!/usr/bin/env python3
"""
Icon 图标生成器
将图片裁剪/缩放到不同像素尺寸的图标文件
"""

from PIL import Image
import os

import sys

# 图标配置：基础尺寸和对应的@2x尺寸
ICON_CONFIGS = [
    {'base': 16, 'x2': 32},
    {'base': 22, 'x2': 44},
    {'base': 32, 'x2': 64},
    {'base': 48, 'x2': 96},
    {'base': 64, 'x2': 128},
    {'base': 128, 'x2': 256},
    {'base': 256, 'x2': 512},
    {'base': 512, 'x2': 1024},
    {'base': 1024, 'x2': 2048},
]

# 输出目录
OUTPUT_DIR = '/Users/gracegaoya/work/project/qim/qim-client/electron/icons'

# 默认输入图片路径
DEFAULT_INPUT_IMAGE = '/Users/gracegaoya/work/project/qim/qim-client/electron/icons/source-icon.png'

# Logo 配置列表：(输入文件, 输出前缀)
LOGO_CONFIGS = [
    ('source-icon.png', 'icon'),
    ('logo-v1.png', 'icon-v1'),
    ('logo-v2.png', 'icon-v2'),
]


def crop_transparent_border(img):
    """裁剪掉图片外围的透明区域"""
    if img.mode != 'RGBA':
        return img
    
    bbox = img.getbbox()
    if bbox:
        return img.crop(bbox)
    return img


def generate_icons(input_path, output_dir, sizes, prefix):
    """生成不同尺寸的图标"""
    if not os.path.exists(input_path):
        print(f'错误: 找不到输入图片 {input_path}')
        return

    print(f'正在处理: {input_path}')
    print(f'输出前缀: {prefix}')
    print(f'输出目录: {output_dir}')
    print()

    # 创建输出目录（如果不存在）
    os.makedirs(output_dir, exist_ok=True)

    # 打开原始图片
    original = Image.open(input_path)
    print(f'原始图片尺寸: {original.size}')
    print()

    generated_files = []

    PADDING_RATIO = 0.85

    for config in sizes:
        base = config['base']
        x2 = config['x2']
        
        icon_base = original.resize((base, base), Image.LANCZOS)
        content_size_base = int(base * PADDING_RATIO)
        icon_base_resized = icon_base.resize((content_size_base, content_size_base), Image.LANCZOS)
        canvas_base = Image.new('RGBA', (base, base), (0, 0, 0, 0))
        offset_base = (base - content_size_base) // 2
        canvas_base.paste(icon_base_resized, (offset_base, offset_base), icon_base_resized)
        filename_base = f'{prefix}_{base}x{base}.png'
        filepath_base = os.path.join(output_dir, filename_base)
        canvas_base.save(filepath_base, 'PNG')
        generated_files.append(filepath_base)
        print(f'✓ {base}x{base} -> {filename_base}')
        
        icon_x2 = original.resize((x2, x2), Image.LANCZOS)
        content_size_x2 = int(x2 * PADDING_RATIO)
        icon_x2_resized = icon_x2.resize((content_size_x2, content_size_x2), Image.LANCZOS)
        canvas_x2 = Image.new('RGBA', (x2, x2), (0, 0, 0, 0))
        offset_x2 = (x2 - content_size_x2) // 2
        canvas_x2.paste(icon_x2_resized, (offset_x2, offset_x2), icon_x2_resized)
        filename_x2 = f'{prefix}_{base}x{base}@2x.png'
        filepath_x2 = os.path.join(output_dir, filename_x2)
        canvas_x2.save(filepath_x2, 'PNG')
        generated_files.append(filepath_x2)
        print(f'✓ {x2}x{x2} -> {filename_x2}')

    print(f'\n成功生成 {len(generated_files)} 个图标文件!')
    print(f'所有文件已保存到: {output_dir}')
    return generated_files


if __name__ == '__main__':
    for input_file, prefix in LOGO_CONFIGS:
        input_path = os.path.join(os.path.dirname(__file__), input_file)
        generate_icons(input_path, OUTPUT_DIR, ICON_CONFIGS, prefix)
        print('\n' + '=' * 50 + '\n')

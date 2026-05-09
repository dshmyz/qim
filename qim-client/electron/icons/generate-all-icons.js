/**
 * 全平台图标生成脚本
 * 从统一源图标生成所有平台所需的图标格式
 * 确保 Windows / macOS / Linux 图标完全一致
 */

import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';
import pngToIco from 'png-to-ico';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// 统一源图标（绿色 QIM 图标）
const SOURCE_IMAGE = path.join(__dirname, 'icon_512x512.png');

// 输出目录
const ICONS_DIR = __dirname;

// Windows ICO 需要的尺寸
const ICO_SIZES = [16, 32, 48, 64, 128, 256];

// macOS iconset 需要的尺寸
const MAC_ICONSET_SIZES = [
  { name: 'icon_16x16', size: 16 },
  { name: 'icon_16x16@2x', size: 32 },
  { name: 'icon_32x32', size: 32 },
  { name: 'icon_32x32@2x', size: 64 },
  { name: 'icon_22x22', size: 22 },
  { name: 'icon_22x22@2x', size: 44 },
  { name: 'icon_128x128', size: 128 },
  { name: 'icon_128x128@2x', size: 256 },
  { name: 'icon_256x256', size: 256 },
  { name: 'icon_256x256@2x', size: 512 },
  { name: 'icon_512x512', size: 512 },
  { name: 'icon_512x512@2x', size: 1024 },
];

async function resizeImage(size) {
  return sharp(SOURCE_IMAGE)
    .resize(size, size, {
      fit: 'fill',
      background: { r: 0, g: 0, b: 0, alpha: 0 }
    })
    .png()
    .toBuffer();
}

async function generateWindowsIco() {
  console.log('🪟 生成 Windows ICO 图标...');
  
  const pngBuffers = [];
  for (const size of ICO_SIZES) {
    const buffer = await resizeImage(size);
    pngBuffers.push(buffer);
    console.log(`   ✓ ${size}x${size}`);
  }
  
  const icoBuffer = await pngToIco(pngBuffers);
  fs.writeFileSync(path.join(ICONS_DIR, 'icon.ico'), icoBuffer);
  console.log('   ✅ icon.ico 生成完成\n');
}

async function generateMacIconset() {
  console.log('🍎 生成 macOS Iconset...');
  
  const iconsetDir = path.join(ICONS_DIR, 'QIM.iconset');
  if (fs.existsSync(iconsetDir)) {
    fs.rmSync(iconsetDir, { recursive: true });
  }
  fs.mkdirSync(iconsetDir, { recursive: true });
  
  for (const { name, size } of MAC_ICONSET_SIZES) {
    const buffer = await resizeImage(size);
    fs.writeFileSync(path.join(iconsetDir, `${name}.png`), buffer);
    console.log(`   ✓ ${name}.png (${size}x${size})`);
  }
  
  // 使用 iconutil 生成 .icns（仅 macOS）
  if (process.platform === 'darwin') {
    try {
      execSync(`iconutil -c icns -o "${path.join(ICONS_DIR, 'QIM.icns')}" "${iconsetDir}"`);
      console.log('   ✅ QIM.icns 生成完成\n');
    } catch (error) {
      console.log('   ⚠️  iconutil 失败，请手动运行: iconutil -c icns -o QIM.icns QIM.iconset\n');
    }
  } else {
    console.log('   ️  非 macOS 系统，跳过 .icns 生成（请在 macOS 上运行 iconutil）\n');
  }
}

async function generateLinuxIcons() {
  console.log('🐧 生成 Linux 图标...');
  
  // Linux 需要 512x512 PNG（已有）
  // 同时生成 256x256 作为备用
  const buffer256 = await resizeImage(256);
  fs.writeFileSync(path.join(ICONS_DIR, 'icon_256x256_linux.png'), buffer256);
  console.log('   ✓ icon_256x256_linux.png');
  console.log('   ✓ icon_512x512.png (已存在)');
  console.log('   ✅ Linux 图标完成\n');
}

async function main() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error(`❌ 错误: 找不到源图标 ${SOURCE_IMAGE}`);
    console.error('请确保 icon_512x512.png 存在');
    process.exit(1);
  }
  
  console.log('🎨 全平台图标生成工具');
  console.log('====================');
  console.log(`源图标: ${SOURCE_IMAGE}\n`);
  
  await generateWindowsIco();
  await generateMacIconset();
  await generateLinuxIcons();
  
  console.log('====================');
  console.log('✅ 所有平台图标生成完成！');
  console.log('\n生成的文件:');
  console.log('   electron/icons/icon.ico          - Windows 安装包图标');
  console.log('   electron/icons/QIM.icns          - macOS 应用图标');
  console.log('  📁 electron/icons/QIM.iconset/      - macOS 图标集');
  console.log('   electron/icons/icon_512x512.png  - Linux 应用图标');
  console.log('\n所有图标均使用统一的绿色 QIM 图标源，确保跨平台视觉一致性。');
}

main().catch(err => {
  console.error(' 生成失败:', err);
  process.exit(1);
});

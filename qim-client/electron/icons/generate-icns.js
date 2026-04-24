import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SOURCE_IMAGE = path.join(__dirname, 'source-icon.png');

const iconSetName = 'QIM.iconset';
const iconSetDir = path.join(__dirname, iconSetName);

// Mac iconset 需要的完整尺寸配置
const macIconSizes = [
  { filename: 'icon_16x16.png', size: 16 },
  { filename: 'icon_16x16@2x.png', size: 32 },
  { filename: 'icon_32x32.png', size: 32 },
  { filename: 'icon_32x32@2x.png', size: 64 },
  { filename: 'icon_22x22.png', size: 22 },
  { filename: 'icon_22x22@2x.png', size: 44 },
  { filename: 'icon_128x128.png', size: 128 },
  { filename: 'icon_128x128@2x.png', size: 256 },
  { filename: 'icon_256x256.png', size: 256 },
  { filename: 'icon_256x256@2x.png', size: 512 },
  { filename: 'icon_512x512.png', size: 512 },
  { filename: 'icon_512x512@2x.png', size: 1024 },
];

async function generateIcns() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error(`错误: 找不到源图片 ${SOURCE_IMAGE}`);
    return;
  }

  console.log('开始生成 ICNS 文件...');
  console.log();

  // 清理并创建 iconset 目录
  if (fs.existsSync(iconSetDir)) {
    fs.rmSync(iconSetDir, { recursive: true });
  }
  fs.mkdirSync(iconSetDir, { recursive: true });

  // 生成所有 iconset 需要的图标
  for (const { filename, size } of macIconSizes) {
    const outputPath = path.join(iconSetDir, filename);
    
    await sharp(SOURCE_IMAGE)
      .resize(size, size, {
        fit: 'cover',
        position: 'centre'
      })
      .png()
      .toFile(outputPath);
    
    console.log(`✓ ${filename}`);
  }

  console.log();

  // 使用 iconutil 转换为 icns
  const icnsPath = path.join(__dirname, 'QIM.icns');
  
  try {
    execSync(`iconutil -c icns -o "${icnsPath}" "${iconSetDir}"`);
    console.log(`✓ 成功生成: QIM.icns`);
    console.log();
    console.log('ICNS 文件已生成!');
  } catch (error) {
    console.error('生成 ICNS 失败:', error.message);
  }
}

generateIcns().catch(console.error);

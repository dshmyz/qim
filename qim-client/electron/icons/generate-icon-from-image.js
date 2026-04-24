import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// 图标配置：基础尺寸和对应的@2x尺寸
const iconConfigs = [
  { base: 16, x2: 32 },
  { base: 32, x2: 64 },
  { base: 48, x2: 96 },
  { base: 64, x2: 128 },
  { base: 128, x2: 256 },
  { base: 256, x2: 512 },
  { base: 512, x2: 1024 },
  { base: 1024, x2: 2048 },
];

const SOURCE_IMAGE = path.join(__dirname, 'source-icon.png');

async function generateIcons() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error(`错误: 找不到源图片 ${SOURCE_IMAGE}`);
    console.log('请先将图片保存为 source-icon.png');
    return;
  }

  console.log('开始生成图标...');
  console.log(`源图片: ${SOURCE_IMAGE}`);
  console.log();

  const metadata = await sharp(SOURCE_IMAGE).metadata();
  console.log(`原始图片尺寸: ${metadata.width}x${metadata.height}`);
  console.log();

  for (const { base, x2 } of iconConfigs) {
    // 生成 1x 图标
    const filename1x = `icon_${base}x${base}.png`;
    const outputPath1x = path.join(__dirname, filename1x);
    
    await sharp(SOURCE_IMAGE)
      .resize(base, base, {
        fit: 'cover',
        position: 'centre'
      })
      .png()
      .toFile(outputPath1x);
    
    console.log(`✓ ${filename1x}`);

    // 生成 @2x 图标
    const filename2x = `icon_${base}x${base}@2x.png`;
    const outputPath2x = path.join(__dirname, filename2x);
    
    await sharp(SOURCE_IMAGE)
      .resize(x2, x2, {
        fit: 'cover',
        position: 'centre'
      })
      .png()
      .toFile(outputPath2x);
    
    console.log(`✓ ${filename2x}`);
  }

  console.log();
  console.log(`成功生成 ${iconConfigs.length * 2} 个图标!`);
}

generateIcons().catch(console.error);

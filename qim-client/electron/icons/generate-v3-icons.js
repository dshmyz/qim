import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SOURCE_IMAGE = path.join(__dirname, 'logo-v3.png');

const sizes = [
  { name: 'icon-v3_16x16', size: 16 },
  { name: 'icon-v3_16x16@2x', size: 32 },
  { name: 'icon-v3_22x22', size: 22 },
  { name: 'icon-v3_22x22@2x', size: 44 },
  { name: 'icon-v3_32x32', size: 32 },
  { name: 'icon-v3_32x32@2x', size: 64 },
  { name: 'icon-v3_48x48', size: 48 },
  { name: 'icon-v3_48x48@2x', size: 96 },
  { name: 'icon-v3_64x64', size: 64 },
  { name: 'icon-v3_64x64@2x', size: 128 },
  { name: 'icon-v3_128x128', size: 128 },
  { name: 'icon-v3_128x128@2x', size: 256 },
  { name: 'icon-v3_256x256', size: 256 },
  { name: 'icon-v3_256x256@2x', size: 512 },
  { name: 'icon-v3_512x512', size: 512 },
  { name: 'icon-v3_512x512@2x', size: 1024 },
];

async function generate() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error(`错误: 找不到源图片 ${SOURCE_IMAGE}`);
    process.exit(1);
  }

  const metadata = await sharp(SOURCE_IMAGE).metadata();
  console.log(`源图片: ${SOURCE_IMAGE} (${metadata.width}x${metadata.height})`);
  console.log('开始生成图标...\n');

  for (const { name, size } of sizes) {
    const outputPath = path.join(__dirname, `${name}.png`);
    await sharp(SOURCE_IMAGE)
      .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(outputPath);
    console.log(`  ✓ ${name}.png (${size}x${size})`);
  }

  console.log(`\n✅ 成功生成 ${sizes.length} 个图标`);
}

generate().catch(err => {
  console.error('生成失败:', err);
  process.exit(1);
});
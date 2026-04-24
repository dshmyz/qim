import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SOURCE_IMAGE = path.join(__dirname, 'logo.png');

const iconSizes = [
  16, 22, 32, 48, 64, 128, 256, 512, 1024
];

async function generateIcons() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error(`错误: 找不到源图片 ${SOURCE_IMAGE}`);
    console.log('请将 logo.png 放在 icons 目录下');
    return;
  }

  console.log('开始生成高质量图标...');
  console.log(`源图片: ${SOURCE_IMAGE}`);
  console.log();

  const metadata = await sharp(SOURCE_IMAGE).metadata();
  console.log(`原始图片尺寸: ${metadata.width}x${metadata.height}`);
  console.log();

  for (const size of iconSizes) {
    const filename = `logo-${size}x${size}.png`;
    const outputPath = path.join(__dirname, filename);
    
    await sharp(SOURCE_IMAGE)
      .resize(size, size, {
        fit: 'inside',
        kernel: 'lanczos3'
      })
      .png({ quality: 100 })
      .toFile(outputPath);
    
    console.log(`✓ ${size}x${size} -> ${filename}`);
  }

  console.log();
  console.log(`成功生成 ${iconSizes.length} 个高质量图标!`);
  console.log('所有文件已保存到:', __dirname);
}

generateIcons().catch(console.error);

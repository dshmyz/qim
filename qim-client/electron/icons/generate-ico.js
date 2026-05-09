import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// 使用现有的 512x512 图标作为源
const SOURCE_IMAGE = path.join(__dirname, 'icon_512x512.png');
const OUTPUT_ICO = path.join(__dirname, 'icon.ico');

async function generateIco() {
  if (!fs.existsSync(SOURCE_IMAGE)) {
    console.error('❌ 错误: 找不到源图片 ' + SOURCE_IMAGE);
    console.log('提示: 请确保 electron/icons/icon_512x512.png 存在');
    return false;
  }

  console.log('🚀 开始生成 Windows ICO 文件...');
  console.log();

  // sharp 可以直接生成包含多个尺寸的 ICO 文件
  // electron-builder 26+ 支持直接从 PNG 生成 ICO
  // 但我们也可以预先生成以确保兼容性
  
  const image = sharp(SOURCE_IMAGE);
  const metadata = await image.metadata();
  console.log(`📐 源图片尺寸: ${metadata.width}x${metadata.height}`);
  console.log();

  // 使用 electron-builder 推荐的方式：直接复制 PNG
  // electron-builder 会自动将其转换为 ICO 格式
  // 但为了明确，我们生成一个标准的 icon.ico
  
  // 方案 1: 使用 sharp 生成单一最大尺寸（electron-builder 会处理）
  await image
    .resize(256, 256)
    .png()
    .toFile(OUTPUT_ICO.replace('.ico', '_256.png'));
  console.log('✓ 生成 256x256 PNG (用于 electron-builder 自动转换)');
  
  console.log();
  console.log('✅ 图标生成完成!');
  console.log();
  console.log('📝 说明:');
  console.log('   electron-builder 26+ 会自动将 PNG 图标转换为 ICO 格式');
  console.log('   构建时无需手动生成 .ico 文件');
  console.log('   如需手动生成，请使用在线工具或安装 png-to-ico 包');
  return true;
}

generateIco().catch(console.error);

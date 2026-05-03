import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const sourceImagePath = path.join(__dirname, '..', '未命名图片_2026.05.03.png');

const iconSetDir = path.join(__dirname, 'IM.iconset');

const standardSizes = [16, 32, 64, 128, 256, 512, 1024];

const iconSetSizes = [
  { name: 'icon_16x16', size: 16 },
  { name: 'icon_16x16@2x', size: 32 },
  { name: 'icon_22x22', size: 22 },
  { name: 'icon_22x22@2x', size: 44 },
  { name: 'icon_32x32', size: 32 },
  { name: 'icon_32x32@2x', size: 64 },
  { name: 'icon_48x48', size: 48 },
  { name: 'icon_48x48@2x', size: 96 },
  { name: 'icon_64x64', size: 64 },
  { name: 'icon_64x64@2x', size: 128 },
  { name: 'icon_128x128', size: 128 },
  { name: 'icon_128x128@2x', size: 256 },
  { name: 'icon_256x256', size: 256 },
  { name: 'icon_256x256@2x', size: 512 },
  { name: 'icon_512x512', size: 512 },
  { name: 'icon_512x512@2x', size: 1024 },
];

async function generateIcons() {
  if (!fs.existsSync(sourceImagePath)) {
    console.error(`Source image not found: ${sourceImagePath}`);
    return;
  }

  const image = sharp(sourceImagePath);
  const metadata = await image.metadata();
  console.log(`Source image: ${metadata.width}x${metadata.height}`);

  for (const size of standardSizes) {
    const outputPath = path.join(__dirname, `icon_${size}x${size}.png`);
    await image
      .clone()
      .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(outputPath);
    console.log(`Generated icon_${size}x${size}.png`);
  }

  if (fs.existsSync(iconSetDir)) {
    fs.rmSync(iconSetDir, { recursive: true });
  }
  fs.mkdirSync(iconSetDir, { recursive: true });

  for (const { name, size } of iconSetSizes) {
    const iconsetOutputPath = path.join(iconSetDir, `${name}.png`);
    await image
      .clone()
      .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(iconsetOutputPath);
    console.log(`Generated iconset/${name}.png (${size}x${size})`);

    const dirOutputPath = path.join(__dirname, `${name}.png`);
    await image
      .clone()
      .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(dirOutputPath);
    console.log(`Generated ${name}.png (${size}x${size})`);
  }

  console.log('\nIcon generation complete!');
  console.log('\nFiles generated:');
  console.log(`- ${standardSizes.map(s => `icon_${s}x${s}.png`).join(', ')}`);
  console.log('- IM.iconset/ (for ICNS creation)');
  console.log('\nTo create ICNS file, run:');
  console.log(`  cd ${__dirname}`);
  console.log('  iconutil -c icns IM.iconset');
}

generateIcons().catch(console.error);
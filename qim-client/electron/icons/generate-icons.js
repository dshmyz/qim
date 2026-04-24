import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const svgContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <defs>
    <linearGradient id="bgGradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0891b2"/>
      <stop offset="50%" style="stop-color:#06b6d4"/>
      <stop offset="100%" style="stop-color:#22d3ee"/>
    </linearGradient>
    <linearGradient id="whiteGradient" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#ffffff"/>
      <stop offset="100%" style="stop-color:#e0f2fe"/>
    </linearGradient>
  </defs>

  <rect x="50" y="50" width="412" height="412" rx="80" fill="url(#bgGradient)"/>

  <g transform="translate(256, 256)">
    <rect x="-130" y="-85" width="260" height="170" rx="85" fill="url(#whiteGradient)"/>

    <path d="M -70 -40 L -70 40 L -28 0 L -28 40" fill="none" stroke="#0891b2" stroke-width="18" stroke-linecap="round"/>
    <path d="M 0 -40 L 0 40" fill="none" stroke="#0891b2" stroke-width="18" stroke-linecap="round"/>
    <path d="M 28 -40 L 70 0 L 70 40" fill="none" stroke="#0891b2" stroke-width="18" stroke-linecap="round"/>
  </g>
</svg>`;

const iconsDir = __dirname;
const iconSetDir = path.join(iconsDir, 'QIM.iconset');

async function generateIcons() {
  if (!fs.existsSync(iconsDir)) {
    fs.mkdirSync(iconsDir, { recursive: true });
  }

  if (fs.existsSync(iconSetDir)) {
    fs.rmSync(iconSetDir, { recursive: true });
  }
  fs.mkdirSync(iconSetDir, { recursive: true });

  const svgBuffer = Buffer.from(svgContent);

  const sizes = [16, 32, 64, 128, 256, 512, 1024];
  for (const size of sizes) {
    const outputPath = path.join(iconsDir, `icon-${size}.png`);
    await sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(outputPath);
    console.log(`Generated icon-${size}.png`);
  }

  const iconSetSizes = [
    { name: 'icon_16x16', size: 16 },
    { name: 'icon_16x16@2x', size: 32 },
    { name: 'icon_32x32', size: 32 },
    { name: 'icon_32x32@2x', size: 64 },
    { name: 'icon_128x128', size: 128 },
    { name: 'icon_128x128@2x', size: 256 },
    { name: 'icon_256x256', size: 256 },
    { name: 'icon_256x256@2x', size: 512 },
    { name: 'icon_512x512', size: 512 },
    { name: 'icon_512x512@2x', size: 1024 },
  ];

  for (const { name, size } of iconSetSizes) {
    const outputPath = path.join(iconSetDir, `${name}.png`);
    await sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(outputPath);
    console.log(`Generated iconset/${name}.png (${size}x${size})`);
  }

  console.log('\nIcon generation complete!');
  console.log('\nFiles generated in:', iconsDir);
  console.log('- icon.svg (source)');
  console.log('- icon-16.png, icon-32.png, icon-64.png, icon-128.png, icon-256.png, icon-512.png, icon-1024.png');
  console.log('- QIM.iconset/ (for ICNS creation)');
  console.log('\nTo create ICNS file, run:');
  console.log(`  cd ${iconsDir}`);
  console.log('  iconutil -c icns QIM.iconset');
  console.log('\nThis will create QIM.icns in the icons directory.');
}

generateIcons().catch(console.error);

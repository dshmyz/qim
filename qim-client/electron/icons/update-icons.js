import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const svgContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <defs>
    <linearGradient id="bgGradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#6366f1"/>
      <stop offset="100%" style="stop-color:#8b5cf6"/>
    </linearGradient>
    <linearGradient id="whiteGradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#ffffff"/>
      <stop offset="100%" style="stop-color:#f8fafc"/>
    </linearGradient>
  </defs>

  <circle cx="256" cy="256" r="240" fill="url(#bgGradient)"/>

  <g transform="translate(256, 256)">
    <circle cx="0" cy="0" r="130" fill="url(#whiteGradient)"/>
    
    <path d="M -60 -40 Q -60 -90 0 -90 Q 60 -90 60 -40 Q 60 20 40 35 L 20 65 Q 15 72 5 70 Q -5 68 -10 60 L -5 35 Q -25 20 -25 -10 Q -25 -25 -15 -35 Q -5 -45 10 -45 Q 25 -45 30 -35" fill="#6366f1"/>
    
    <circle cx="-15" cy="-5" r="14" fill="url(#bgGradient)"/>
    <circle cx="30" cy="-5" r="14" fill="url(#bgGradient)"/>
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
}

async function generateIconDataURL() {
  const svgBuffer = Buffer.from(svgContent);
  const pngBuffer = await sharp(svgBuffer)
    .resize(512, 512)
    .png()
    .toBuffer();
  
  const base64 = pngBuffer.toString('base64');
  return base64;
}

async function updateMainJS(base64Data) {
  const mainJSPath = path.join(__dirname, '../main.js');
  let content = fs.readFileSync(mainJSPath, 'utf8');
  
  const iconDataRegex = /const iconData = ['"][^'"]+['"];/g;
  const replacement = `const iconData = '${base64Data}';`;
  content = content.replace(iconDataRegex, replacement);
  
  fs.writeFileSync(mainJSPath, content, 'utf8');
  console.log('Updated main.js with new icon data');
}

async function main() {
  await generateIcons();
  const base64Data = await generateIconDataURL();
  await updateMainJS(base64Data);
  console.log('\nAll done!');
}

main().catch(console.error);

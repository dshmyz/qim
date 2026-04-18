const fs = require('fs');
const path = require('path');

// Check if localStorage file exists (for Electron apps)
const appDataPath = process.platform === 'win32' 
  ? path.join(process.env.APPDATA, 'qim-client')
  : process.platform === 'darwin'
  ? path.join(process.env.HOME, 'Library', 'Application Support', 'qim-client')
  : path.join(process.env.HOME, '.config', 'qim-client');

console.log('Checking app data path:', appDataPath);

// Also check if there's a token in the browser's localStorage by examining any storage files
if (fs.existsSync(appDataPath)) {
  const files = fs.readdirSync(appDataPath);
  console.log('Files in app data:', files);
  
  // Look for storage files
  files.forEach(file => {
    if (file.includes('storage') || file.includes('localStorage')) {
      const filePath = path.join(appDataPath, file);
      console.log('Found storage file:', filePath);
      try {
        const content = fs.readFileSync(filePath, 'utf8');
        console.log('File content:', content);
      } catch (error) {
        console.log('Error reading file:', error.message);
      }
    }
  });
} else {
  console.log('App data directory does not exist');
}

console.log('Token check complete');

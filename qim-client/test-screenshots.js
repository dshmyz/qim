const path = require('path');
process.chdir(path.join(__dirname, 'electron/screenshots-main'));
const Screenshots = require('./index.cjs.js');
console.log('✅ Screenshots class loaded', Screenshots.name);

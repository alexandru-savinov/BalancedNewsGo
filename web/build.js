// Simple build script for development and production
const fs = require('fs');
const path = require('path');

const isDev = process.argv.includes('--dev');

console.log(`Building NewsBalancer Frontend (${isDev ? 'development' : 'production'} mode)...`);

// Basic build tasks:
// 1. Copy normalize.css from node_modules
// 2. Copy Chart.js and DOMPurify from node_modules
// 3. Create index files that include all CSS/JS

function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

function copyFile(src, dest) {
  ensureDir(path.dirname(dest));
  fs.copyFileSync(src, dest);
  console.log(`Copied: ${src} -> ${dest}`);
}

// Copy vendor dependencies
const vendorDir = path.join(__dirname, 'src', 'vendor');
ensureDir(vendorDir);

if (fs.existsSync('node_modules')) {
  // Copy normalize.css
  copyFile(
    'node_modules/normalize.css/normalize.css',
    path.join(vendorDir, 'normalize.css')
  );

  // Copy Chart.js
  copyFile(
    'node_modules/chart.js/dist/chart.umd.js',
    path.join(vendorDir, 'chart.js')
  );

  // Copy DOMPurify
  copyFile(
    'node_modules/dompurify/dist/purify.min.js',
    path.join(vendorDir, 'dompurify.js')
  );

  console.log('✓ Vendor dependencies copied');
} else {
  console.log('⚠ Node modules not found. Run "npm install" first.');
}

console.log('✓ Build complete');

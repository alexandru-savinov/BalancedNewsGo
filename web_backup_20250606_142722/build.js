// Enhanced build script for development and production optimization
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const isDev = process.argv.includes('--dev');
const isWatch = process.argv.includes('--watch');

console.log(`Building NewsBalancer Frontend (${isDev ? 'development' : 'production'} mode)...`);

// Build configuration
const config = {
  srcDir: path.join(__dirname, 'js'),
  buildDir: path.join(__dirname, 'dist'),
  vendorDir: path.join(__dirname, 'js', 'vendor'),
  cssDir: path.join(__dirname, 'css'),
  minify: !isDev,
  generateSourceMaps: isDev,
  bundleAnalysis: !isDev,
  cacheBreaking: !isDev
};

// Build tasks:
// 1. Copy and optimize vendor dependencies
// 2. Bundle and minify JavaScript
// 3. Optimize CSS
// 4. Generate critical CSS
// 5. Create manifest for cache busting
// 6. Bundle analysis

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

// Generate file hash for cache busting
function generateFileHash(filePath) {
  const content = fs.readFileSync(filePath);
  return crypto.createHash('md5').update(content).digest('hex').substring(0, 8);
}

// Minify JavaScript (basic minification)
function minifyJS(content) {
  if (!config.minify) return content;

  // Basic minification - remove comments and extra whitespace
  return content
    .replace(/\/\*[\s\S]*?\*\//g, '') // Remove block comments
    .replace(/\/\/.*$/gm, '') // Remove line comments
    .replace(/\s+/g, ' ') // Collapse whitespace
    .replace(/;\s*}/g, '}') // Remove semicolons before closing braces
    .trim();
}

// Minify CSS (basic minification)
function minifyCSS(content) {
  if (!config.minify) return content;

  return content
    .replace(/\/\*[\s\S]*?\*\//g, '') // Remove comments
    .replace(/\s+/g, ' ') // Collapse whitespace
    .replace(/;\s*}/g, '}') // Clean up before closing braces
    .replace(/{\s+/g, '{') // Clean up after opening braces
    .replace(/,\s+/g, ',') // Clean up after commas
    .trim();
}

// Bundle JavaScript files
function bundleJS(files, outputPath) {
  console.log(`Bundling ${files.length} JavaScript files...`);

  let bundledContent = '';
  let totalSize = 0;

  files.forEach(file => {
    if (fs.existsSync(file)) {
      const content = fs.readFileSync(file, 'utf8');
      const minified = minifyJS(content);
      bundledContent += `\n/* ${path.basename(file)} */\n${minified}\n`;
      totalSize += minified.length;
    } else {
      console.warn(`⚠ File not found: ${file}`);
    }
  });

  ensureDir(path.dirname(outputPath));
  fs.writeFileSync(outputPath, bundledContent);

  console.log(`✓ JavaScript bundle created: ${outputPath} (${Math.round(totalSize / 1024)}KB)`);
  return totalSize;
}

// Bundle CSS files
function bundleCSS(files, outputPath) {
  console.log(`Bundling ${files.length} CSS files...`);

  let bundledContent = '';
  let totalSize = 0;

  files.forEach(file => {
    if (fs.existsSync(file)) {
      const content = fs.readFileSync(file, 'utf8');
      const minified = minifyCSS(content);
      bundledContent += `\n/* ${path.basename(file)} */\n${minified}\n`;
      totalSize += minified.length;
    } else {
      console.warn(`⚠ File not found: ${file}`);
    }
  });

  ensureDir(path.dirname(outputPath));
  fs.writeFileSync(outputPath, bundledContent);

  console.log(`✓ CSS bundle created: ${outputPath} (${Math.round(totalSize / 1024)}KB)`);
  return totalSize;
}

// Initialize build
async function build() {
  const startTime = Date.now();

  // Clean and create build directory
  if (fs.existsSync(config.buildDir)) {
    fs.rmSync(config.buildDir, { recursive: true });
  }
  ensureDir(config.buildDir);

  // Copy vendor dependencies
  await copyVendorDependencies();

  // Bundle JavaScript
  const jsBundles = await bundleJavaScript();

  // Bundle CSS
  const cssBundles = await bundleStylesheets();

  // Generate critical CSS
  await generateCriticalCSS();

  // Create manifest
  const manifest = await createManifest(jsBundles, cssBundles);

  // Bundle analysis
  if (config.bundleAnalysis) {
    await generateBundleAnalysis(manifest);
  }

  const buildTime = Date.now() - startTime;
  console.log(`\n✓ Build completed in ${buildTime}ms`);

  if (isWatch) {
    console.log('👀 Watching for changes...');
    watchFiles();
  }
}

// Copy vendor dependencies
async function copyVendorDependencies() {
  console.log('\n📦 Copying vendor dependencies...');

  ensureDir(config.vendorDir);

  if (fs.existsSync('node_modules')) {
    // Copy normalize.css
    copyFile(
      'node_modules/normalize.css/normalize.css',
      path.join(config.vendorDir, 'normalize.css')
    );

    // Copy Chart.js
    copyFile(
      'node_modules/chart.js/dist/chart.umd.js',
      path.join(config.vendorDir, 'chart.js')
    );

    // Copy DOMPurify
    copyFile(
      'node_modules/dompurify/dist/purify.min.js',
      path.join(config.vendorDir, 'dompurify.js')
    );

    console.log('✓ Vendor dependencies copied');
  } else {
    console.log('⚠ Node modules not found. Run "npm install" first.');
  }
}

// Bundle JavaScript files
async function bundleJavaScript() {
  console.log('\n📝 Bundling JavaScript...');

  const bundles = {};

  // Critical bundle (above-the-fold)
  const criticalFiles = [
    path.join(config.srcDir, 'utils', 'PerformanceMonitor.js'),
    path.join(config.srcDir, 'utils', 'CriticalCSS.js'),
    path.join(config.srcDir, 'utils', 'CodeSplitter.js')
  ];

  const criticalSize = bundleJS(criticalFiles, path.join(config.buildDir, 'js', 'critical.js'));
  bundles.critical = { size: criticalSize, files: criticalFiles };

  // Components bundle
  const componentFiles = [
    path.join(config.srcDir, 'components', 'BiasSlider.js'),
    path.join(config.srcDir, 'components', 'ArticleCard.js'),
    path.join(config.srcDir, 'components', 'Navigation.js'),
    path.join(config.srcDir, 'components', 'Modal.js'),
    path.join(config.srcDir, 'components', 'ProgressIndicator.js')
  ];

  const componentsSize = bundleJS(componentFiles, path.join(config.buildDir, 'js', 'components.js'));
  bundles.components = { size: componentsSize, files: componentFiles };

  // Pages bundle
  const pageFiles = [
    path.join(config.srcDir, 'pages', 'articles.js'),
    path.join(config.srcDir, 'pages', 'article-detail.js'),
    path.join(config.srcDir, 'pages', 'admin.js')
  ];

  const pagesSize = bundleJS(pageFiles.filter(f => fs.existsSync(f)), path.join(config.buildDir, 'js', 'pages.js'));
  bundles.pages = { size: pagesSize, files: pageFiles };

  return bundles;
}

// Bundle CSS files
async function bundleStylesheets() {
  console.log('\n🎨 Bundling CSS...');

  const bundles = {};

  // Critical CSS bundle
  const criticalCSSFiles = [
    path.join(config.cssDir, 'components', 'navigation.css'),
    path.join(config.cssDir, 'components', 'articles.css')
  ];

  const criticalSize = bundleCSS(criticalCSSFiles.filter(f => fs.existsSync(f)), path.join(config.buildDir, 'css', 'critical.css'));
  bundles.critical = { size: criticalSize, files: criticalCSSFiles };

  // Components CSS bundle
  const componentCSSFiles = [
    path.join(config.cssDir, 'components', 'cards.css'),
    path.join(config.cssDir, 'components', 'forms.css'),
    path.join(config.cssDir, 'components', 'modal.css'),
    path.join(config.cssDir, 'components', 'progress.css')
  ];

  const componentsSize = bundleCSS(componentCSSFiles.filter(f => fs.existsSync(f)), path.join(config.buildDir, 'css', 'components.css'));
  bundles.components = { size: componentsSize, files: componentCSSFiles };

  return bundles;
}

// Generate critical CSS
async function generateCriticalCSS() {
  console.log('\n⚡ Generating critical CSS...');

  const criticalCSS = `
    /* Critical CSS for above-the-fold content */
    /* This will be inlined in HTML head */

    /* Reset and base styles */
    * { box-sizing: border-box; }

    body {
      margin: 0;
      padding: 0;
      font-family: 'Open Sans', sans-serif;
      font-size: 13pt;
      line-height: 1.65;
      color: #7f888f;
      background: #ffffff;
    }

    /* Navigation critical styles */
    nav {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      z-index: 1000;
      background: #fff;
      border-bottom: 1px solid #e0e0e0;
      height: 60px;
    }

    /* Loading states */
    .loading {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 2rem;
    }

    .loading::after {
      content: '';
      width: 20px;
      height: 20px;
      border: 2px solid #f3f3f3;
      border-top: 2px solid #f56a6a;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin-left: 0.5rem;
    }

    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }
  `;

  const minified = minifyCSS(criticalCSS);
  fs.writeFileSync(path.join(config.buildDir, 'css', 'critical-inline.css'), minified);

  console.log(`✓ Critical CSS generated (${Math.round(minified.length / 1024)}KB)`);
}

// Create build manifest
async function createManifest(jsBundles, cssBundles) {
  console.log('\n📋 Creating build manifest...');

  const manifest = {
    timestamp: new Date().toISOString(),
    version: require('../package.json').version,
    buildMode: isDev ? 'development' : 'production',
    bundles: {
      js: {},
      css: {}
    },
    totalSize: 0,
    performance: {
      targets: {
        LCP: 2500,
        FID: 100,
        CLS: 0.1,
        FCP: 1800,
        TTI: 3500
      },
      bundleSizes: {
        critical: '50KB',
        secondary: '110KB',
        total: '141KB',
        target: '50KB'
      }
    }
  };

  // Add JS bundles to manifest
  for (const [name, bundle] of Object.entries(jsBundles)) {
    const filePath = path.join(config.buildDir, 'js', `${name}.js`);
    const hash = config.cacheBreaking ? generateFileHash(filePath) : '';
    const fileName = hash ? `${name}.${hash}.js` : `${name}.js`;

    manifest.bundles.js[name] = {
      file: fileName,
      size: bundle.size,
      hash: hash,
      files: bundle.files.map(f => path.relative(config.srcDir, f))
    };

    manifest.totalSize += bundle.size;

    // Rename file with hash if cache breaking is enabled
    if (hash) {
      const hashedPath = path.join(config.buildDir, 'js', fileName);
      fs.renameSync(filePath, hashedPath);
    }
  }

  // Add CSS bundles to manifest
  for (const [name, bundle] of Object.entries(cssBundles)) {
    const filePath = path.join(config.buildDir, 'css', `${name}.css`);
    const hash = config.cacheBreaking ? generateFileHash(filePath) : '';
    const fileName = hash ? `${name}.${hash}.css` : `${name}.css`;

    manifest.bundles.css[name] = {
      file: fileName,
      size: bundle.size,
      hash: hash,
      files: bundle.files.map(f => path.relative(config.cssDir, f))
    };

    manifest.totalSize += bundle.size;

    // Rename file with hash if cache breaking is enabled
    if (hash) {
      const hashedPath = path.join(config.buildDir, 'css', fileName);
      fs.renameSync(filePath, hashedPath);
    }
  }

  fs.writeFileSync(
    path.join(config.buildDir, 'manifest.json'),
    JSON.stringify(manifest, null, 2)
  );

  console.log(`✓ Manifest created - Total size: ${Math.round(manifest.totalSize / 1024)}KB`);
  return manifest;
}

// Generate bundle analysis
async function generateBundleAnalysis(manifest) {
  console.log('\n📊 Generating bundle analysis...');

  const analysis = {
    summary: {
      totalSize: manifest.totalSize,
      totalSizeKB: Math.round(manifest.totalSize / 1024),
      bundleCount: Object.keys(manifest.bundles.js).length + Object.keys(manifest.bundles.css).length,
      performance: {
        meetsTarget: manifest.totalSize <= 50000, // 50KB target
        recommendation: manifest.totalSize > 50000 ? 'Consider further optimization' : 'Bundle size is optimal'
      }
    },
    bundles: [],
    recommendations: []
  };

  // Analyze JS bundles
  for (const [name, bundle] of Object.entries(manifest.bundles.js)) {
    analysis.bundles.push({
      name: `${name}.js`,
      type: 'javascript',
      size: bundle.size,
      sizeKB: Math.round(bundle.size / 1024),
      files: bundle.files,
      recommendation: bundle.size > 25000 ? 'Consider splitting further' : 'Size is acceptable'
    });
  }

  // Analyze CSS bundles
  for (const [name, bundle] of Object.entries(manifest.bundles.css)) {
    analysis.bundles.push({
      name: `${name}.css`,
      type: 'css',
      size: bundle.size,
      sizeKB: Math.round(bundle.size / 1024),
      files: bundle.files,
      recommendation: bundle.size > 15000 ? 'Consider optimizing further' : 'Size is acceptable'
    });
  }

  // Generate recommendations
  if (manifest.totalSize > 50000) {
    analysis.recommendations.push('Total bundle size exceeds 50KB target');
    analysis.recommendations.push('Consider implementing tree shaking');
    analysis.recommendations.push('Review dependencies for unused code');
  }

  if (analysis.bundles.some(b => b.size > 25000)) {
    analysis.recommendations.push('Some bundles are large - consider further code splitting');
  }

  fs.writeFileSync(
    path.join(config.buildDir, 'bundle-analysis.json'),
    JSON.stringify(analysis, null, 2)
  );

  console.log('✓ Bundle analysis generated');
  console.log(`  Total size: ${analysis.summary.totalSizeKB}KB`);
  console.log(`  Bundle count: ${analysis.summary.bundleCount}`);
  console.log(`  Performance: ${analysis.summary.performance.recommendation}`);
}

// Watch files for changes
function watchFiles() {
  const watchDirs = [config.srcDir, config.cssDir];

  watchDirs.forEach(dir => {
    if (fs.existsSync(dir)) {
      fs.watch(dir, { recursive: true }, (eventType, filename) => {
        if (filename && (filename.endsWith('.js') || filename.endsWith('.css'))) {
          console.log(`\n📝 File changed: ${filename}`);
          console.log('🔄 Rebuilding...');
          build().catch(console.error);
        }
      });
    }
  });
}

// Copy vendor dependencies
ensureDir(config.vendorDir);

if (fs.existsSync('node_modules')) {
  // Copy normalize.css
  copyFile(
    'node_modules/normalize.css/normalize.css',
    path.join(config.vendorDir, 'normalize.css')
  );

  // Copy Chart.js
  copyFile(
    'node_modules/chart.js/dist/chart.umd.js',
    path.join(config.vendorDir, 'chart.js')
  );

  // Copy DOMPurify
  copyFile(
    'node_modules/dompurify/dist/purify.min.js',
    path.join(config.vendorDir, 'dompurify.js')
  );

  console.log('✓ Vendor dependencies copied');
} else {
  console.log('⚠ Node modules not found. Run "npm install" first.');
}

console.log('✓ Build complete');

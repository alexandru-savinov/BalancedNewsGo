const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

/**
 * Comprehensive admin screenshot capture for CSS migration analysis
 */
async function captureAdminScreenshots() {
  // Create admin analysis directory
  const analysisDir = path.join(__dirname, '..', 'docs', 'admin-visual-analysis');
  if (!fs.existsSync(analysisDir)) {
    fs.mkdirSync(analysisDir, { recursive: true });
  }

  const browser = await chromium.launch();
  
  // Define viewport configurations for responsive analysis
  const viewports = [
    { name: 'desktop', width: 1200, height: 800 },
    { name: 'tablet', width: 768, height: 1024 },
    { name: 'mobile', width: 375, height: 667 },
    { name: 'large-desktop', width: 1920, height: 1080 }
  ];

  // Define admin pages to capture
  const adminPages = [
    { 
      name: 'admin-dashboard', 
      url: 'http://localhost:8080/admin', 
      description: 'Main admin dashboard',
      waitForSelector: '.admin-controls' // Wait for admin content to load
    }
  ];

  try {
    console.log('🔍 Starting comprehensive admin screenshot capture...');
    console.log(`📁 Screenshots will be saved to: ${analysisDir}`);

    for (const viewport of viewports) {
      console.log(`\n📱 Capturing ${viewport.name} viewport (${viewport.width}x${viewport.height})`);
      
      const context = await browser.newContext({
        viewport: { width: viewport.width, height: viewport.height }
      });
      const page = await context.newPage();

      for (const adminPage of adminPages) {
        console.log(`  📸 Capturing ${adminPage.name}...`);
        
        try {
          // Navigate to admin page
          await page.goto(adminPage.url, { waitUntil: 'networkidle', timeout: 15000 });
          
          // Wait for admin-specific content to load
          if (adminPage.waitForSelector) {
            await page.waitForSelector(adminPage.waitForSelector, { timeout: 10000 });
          }
          
          // Wait for any dynamic content
          await page.waitForTimeout(2000);
          
          const screenshotPath = path.join(analysisDir, `${adminPage.name}-${viewport.name}.png`);
          
          // Capture full page screenshot
          await page.screenshot({ 
            path: screenshotPath,
            fullPage: true 
          });

          console.log(`    ✅ Captured: ${adminPage.name}-${viewport.name}.png`);
          
          // Capture specific admin components if on desktop view
          if (viewport.name === 'desktop') {
            await captureAdminComponents(page, analysisDir, adminPage.name);
          }
          
        } catch (error) {
          console.log(`    ❌ Failed to capture ${adminPage.name} at ${viewport.name}: ${error.message}`);
        }
      }

      await context.close();
    }

    // Generate a comparison view
    await generateComparisonScreenshot(browser, analysisDir);
    
    console.log('\n✅ Admin screenshot capture complete!');
    console.log(`📁 All screenshots saved to: ${analysisDir}`);

  } catch (error) {
    console.error('❌ Error during admin screenshot capture:', error);
  } finally {
    await browser.close();
  }
}

/**
 * Capture specific admin components for detailed analysis
 */
async function captureAdminComponents(page, analysisDir, pageName) {
  console.log(`    🔍 Capturing admin components...`);
  
  try {
    // Capture admin controls section
    const adminControls = page.locator('.admin-controls');
    if (await adminControls.count() > 0) {
      await adminControls.screenshot({ 
        path: path.join(analysisDir, `${pageName}-controls-component.png`) 
      });
      console.log(`    ✅ Captured: ${pageName}-controls-component.png`);
    }

    // Capture individual control sections
    const controlSections = page.locator('.control-section');
    const sectionCount = await controlSections.count();
    
    if (sectionCount > 0) {
      for (let i = 0; i < Math.min(sectionCount, 4); i++) {
        await controlSections.nth(i).screenshot({ 
          path: path.join(analysisDir, `${pageName}-control-section-${i + 1}.png`) 
        });
        console.log(`    ✅ Captured: ${pageName}-control-section-${i + 1}.png`);
      }
    }

    // Capture button groups
    const btnGroups = page.locator('.btn-group');
    const btnGroupCount = await btnGroups.count();
    
    if (btnGroupCount > 0) {
      await btnGroups.first().screenshot({ 
        path: path.join(analysisDir, `${pageName}-button-group-example.png`) 
      });
      console.log(`    ✅ Captured: ${pageName}-button-group-example.png`);
    }

    // Capture stats section if present
    const statsSection = page.locator('.stats');
    if (await statsSection.count() > 0) {
      await statsSection.screenshot({ 
        path: path.join(analysisDir, `${pageName}-stats-component.png`) 
      });
      console.log(`    ✅ Captured: ${pageName}-stats-component.png`);
    }

    // Capture recent activity if present
    const recentActivity = page.locator('.recent-activity');
    if (await recentActivity.count() > 0) {
      await recentActivity.screenshot({ 
        path: path.join(analysisDir, `${pageName}-recent-activity.png`) 
      });
      console.log(`    ✅ Captured: ${pageName}-recent-activity.png`);
    }

  } catch (error) {
    console.log(`    ⚠️  Component capture warning: ${error.message}`);
  }
}

/**
 * Generate a side-by-side comparison screenshot
 */
async function generateComparisonScreenshot(browser, analysisDir) {
  console.log(`\n🔄 Generating responsive comparison view...`);
  
  try {
    const context = await browser.newContext({
      viewport: { width: 1400, height: 1000 }
    });
    const page = await context.newPage();

    // Create a simple HTML page showing responsive comparison
    const comparisonHTML = `
      <!DOCTYPE html>
      <html>
      <head>
        <title>Admin Responsive Comparison</title>
        <style>
          body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
          .comparison-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; }
          .viewport-section { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
          .viewport-section h3 { margin-top: 0; color: #333; text-align: center; }
          .iframe-container { border: 2px solid #ddd; border-radius: 4px; overflow: hidden; }
          iframe { width: 100%; height: 400px; border: none; transform-origin: top left; }
          .desktop iframe { transform: scale(0.3); width: 333%; height: 1333px; }
          .tablet iframe { transform: scale(0.5); width: 200%; height: 800px; }
          .mobile iframe { transform: scale(0.8); width: 125%; height: 500px; }
        </style>
      </head>
      <body>
        <h1>Admin Dashboard - Responsive Comparison</h1>
        <div class="comparison-grid">
          <div class="viewport-section desktop">
            <h3>Desktop (1200px)</h3>
            <div class="iframe-container">
              <iframe src="http://localhost:8080/admin"></iframe>
            </div>
          </div>
          <div class="viewport-section tablet">
            <h3>Tablet (768px)</h3>
            <div class="iframe-container">
              <iframe src="http://localhost:8080/admin"></iframe>
            </div>
          </div>
          <div class="viewport-section mobile">
            <h3>Mobile (375px)</h3>
            <div class="iframe-container">
              <iframe src="http://localhost:8080/admin"></iframe>
            </div>
          </div>
        </div>
      </body>
      </html>
    `;

    await page.setContent(comparisonHTML);
    await page.waitForTimeout(3000); // Wait for iframes to load

    await page.screenshot({ 
      path: path.join(analysisDir, 'admin-responsive-comparison.png'),
      fullPage: true 
    });

    console.log(`    ✅ Captured: admin-responsive-comparison.png`);
    await context.close();

  } catch (error) {
    console.log(`    ⚠️  Comparison screenshot warning: ${error.message}`);
  }
}

// Run if called directly
if (require.main === module) {
  captureAdminScreenshots().catch(console.error);
}

module.exports = { captureAdminScreenshots };

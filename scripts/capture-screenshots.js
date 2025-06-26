const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

async function captureScreenshots() {
  // Create screenshots directory
  const screenshotsDir = path.join(__dirname, '..', 'docs', 'screenshots');
  if (!fs.existsSync(screenshotsDir)) {
    fs.mkdirSync(screenshotsDir, { recursive: true });
  }

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1200, height: 800 }
  });
  const page = await context.newPage();

  try {
    console.log('Capturing screenshots for style guide...');

    // Articles page
    console.log('📸 Capturing articles page...');
    await page.goto('http://localhost:8080/articles');
    await page.waitForLoadState('networkidle');
    await page.screenshot({ 
      path: path.join(screenshotsDir, 'articles-page.png'),
      fullPage: true 
    });

    // Article detail page
    console.log('📸 Capturing article detail page...');
    await page.goto('http://localhost:8080/article/580');
    await page.waitForLoadState('networkidle');
    await page.screenshot({ 
      path: path.join(screenshotsDir, 'article-detail.png'),
      fullPage: true 
    });

    // Admin page
    console.log('📸 Capturing admin page...');
    await page.goto('http://localhost:8080/admin');
    await page.waitForLoadState('networkidle');
    await page.screenshot({ 
      path: path.join(screenshotsDir, 'admin-page.png'),
      fullPage: true 
    });

    // Component examples - Articles grid
    console.log('📸 Capturing articles grid component...');
    await page.goto('http://localhost:8080/articles');
    await page.waitForLoadState('networkidle');
    const articlesGrid = page.locator('.articles-grid');
    await articlesGrid.screenshot({ 
      path: path.join(screenshotsDir, 'articles-grid-component.png') 
    });

    // Component examples - Article card
    console.log('📸 Capturing article card component...');
    const articleCard = page.locator('.article-item').first();
    await articleCard.screenshot({ 
      path: path.join(screenshotsDir, 'article-card-component.png') 
    });

    // Mobile view
    console.log('📸 Capturing mobile view...');
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('http://localhost:8080/articles');
    await page.waitForLoadState('networkidle');
    await page.screenshot({ 
      path: path.join(screenshotsDir, 'articles-mobile.png'),
      fullPage: true 
    });

    console.log('✅ Screenshots captured successfully!');
    console.log(`📁 Screenshots saved to: ${screenshotsDir}`);

  } catch (error) {
    console.error('❌ Error capturing screenshots:', error);
  } finally {
    await browser.close();
  }
}

// Run if called directly
if (require.main === module) {
  captureScreenshots().catch(console.error);
}

module.exports = { captureScreenshots };

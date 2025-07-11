import { test, expect, devices } from '@playwright/test';

// Helper function to create mobile test with specific device
async function createMobileTest(browser: any, deviceName: string) {
  const device = devices[deviceName];
  if (!device) {
    throw new Error(`Device "${deviceName}" not found in Playwright devices. Available devices: ${Object.keys(devices).join(', ')}`);
  }
  const context = await browser.newContext(device);
  const page = await context.newPage();
  return { page, context };
}

// iPhone 12 Mobile Tests
test.describe('Mobile Tests - iPhone 12', () => {

  test('should display properly on iPhone 12', async ({ browser }) => {
    const { page, context } = await createMobileTest(browser, 'iPhone 12');
    try {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      const mainContent = page.locator('main, .main-content, .content, body');
      await expect(mainContent.first()).toBeVisible();
    } finally {
      await context.close();
    }
  });

  test('should handle touch interactions on iPhone 12', async ({ browser }) => {
    const { page, context } = await createMobileTest(browser, 'iPhone 12');
    try {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Find touchable elements
      const touchableElements = page.locator('button, a, [role="button"], [onclick]');

      if (await touchableElements.count() > 0) {
        const element = touchableElements.first();

        // Perform touch interaction
        await element.tap();
        await page.waitForTimeout(500);

        // Verify page is still functional
        await expect(page.locator('body')).toBeVisible();
      }
    } finally {
      await context.close();
    }
  });
});

// Configure mobile devices at top level
const mobileTestPixel = test.extend({});
mobileTestPixel.use(devices['Pixel 5']);

// Pixel 5 Mobile Tests
test.describe('Mobile Tests - Pixel 5', () => {

  mobileTestPixel('should display properly on Pixel 5', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const mainContent = page.locator('main, .main-content, .content, body');
    await expect(mainContent.first()).toBeVisible();
  });

  mobileTestPixel('should handle touch interactions on Pixel 5', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Find touchable elements
    const touchableElements = page.locator('button, a, [role="button"], [onclick]');
    
    if (await touchableElements.count() > 0) {
      const element = touchableElements.first();
      
      // Perform touch interaction
      await element.tap();
      await page.waitForTimeout(500);
      
      // Verify page is still functional
      await expect(page.locator('body')).toBeVisible();
    }
  });
});

// Configure iPad device at top level
const mobileTestIPad = test.extend({});
mobileTestIPad.use(devices['iPad (gen 5)']);

// iPad Tests
test.describe('Mobile Tests - iPad', () => {

  mobileTestIPad('should display properly on iPad', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const mainContent = page.locator('main, .main-content, .content, body');
    await expect(mainContent.first()).toBeVisible();
  });
});

// Configure general mobile testing
const mobileTestGeneral = test.extend({});
mobileTestGeneral.use(devices['iPhone 12']);

// General Mobile Tests
test.describe('Mobile Functionality', () => {

  mobileTestGeneral('should scroll properly on mobile', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Test vertical scroll
    await page.evaluate(() => {
      window.scrollTo(0, 500);
    });
    
    const scrollY = await page.evaluate(() => window.scrollY);
    expect(scrollY).toBeGreaterThan(0);
  });

  mobileTestGeneral('should handle orientation changes', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Test portrait orientation
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(500);
    
    const content = page.locator('main, .main-content, .content, body');
    await expect(content.first()).toBeVisible();
    
    // Test landscape orientation
    await page.setViewportSize({ width: 812, height: 375 });
    await page.waitForTimeout(500);
    
    await expect(content.first()).toBeVisible();
  });

  mobileTestGeneral('should handle mobile navigation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for mobile menu toggle
    const menuToggle = page.locator('.menu-toggle, .hamburger, .mobile-menu-toggle, [aria-label*="menu"]');
    
    if (await menuToggle.count() > 0) {
      await menuToggle.first().tap();
      await page.waitForTimeout(500);
      
      // Check if mobile menu is visible
      const mobileMenu = page.locator('.mobile-menu, .menu-mobile, nav .menu');
      if (await mobileMenu.count() > 0) {
        await expect(mobileMenu.first()).toBeVisible();
      }
    }
  });

  mobileTestGeneral('should handle form inputs on mobile', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const inputs = page.locator('input[type="text"], input[type="email"], textarea');
    
    if (await inputs.count() > 0) {
      const input = inputs.first();
      
      // Focus and type
      await input.tap();
      await input.fill('Test input on mobile');
      
      // Verify input has focus and content
      await expect(input).toBeFocused();
      await expect(input).toHaveValue('Test input on mobile');
    }
  });

  mobileTestGeneral('should maintain readability on mobile', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check text is readable (not too small)
    const textElements = page.locator('p, span, div, h1, h2, h3, h4, h5, h6');
    
    if (await textElements.count() > 0) {
      const fontSize = await textElements.first().evaluate((el) => {
        return window.getComputedStyle(el).fontSize;
      });
      
      const fontSizeNumber = parseInt(fontSize.replace('px', ''));
      expect(fontSizeNumber).toBeGreaterThanOrEqual(14); // Minimum readable font size
    }
  });

  mobileTestGeneral('should handle pinch-to-zoom', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check viewport meta tag allows zooming
    const viewportMeta = page.locator('meta[name="viewport"]');
    if (await viewportMeta.count() > 0) {
      const content = await viewportMeta.getAttribute('content');
      
      // Should not prevent zooming
      expect(content).not.toContain('user-scalable=no');
      expect(content).not.toContain('maximum-scale=1');
    }
  });
});

// Configure mobile performance testing
const mobileTestPerformance = test.extend({});
mobileTestPerformance.use(devices['iPhone 12']);

// Mobile Performance Tests
test.describe('Mobile Performance', () => {

  mobileTestPerformance('should load efficiently on mobile', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const loadTime = Date.now() - startTime;
    
    // Mobile should load within 5 seconds
    expect(loadTime).toBeLessThan(5000);
  });

  mobileTestPerformance('should handle slow network conditions', async ({ page, context }) => {
    // Simulate slow network
    await context.route('**/*', async route => {
      await new Promise(resolve => setTimeout(resolve, 100)); // Add 100ms delay
      await route.continue();
    });
    
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Verify page still loads and is functional
    await expect(page.locator('body')).toBeVisible();
  });
});

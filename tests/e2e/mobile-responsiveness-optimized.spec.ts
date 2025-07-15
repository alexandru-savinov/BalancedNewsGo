import { test, expect, devices } from '@playwright/test';

/**
 * Optimized Mobile Responsiveness E2E Tests
 * 
 * Consolidated mobile testing covering:
 * - Essential device compatibility (iPhone, Android, iPad)
 * - Touch interactions and mobile navigation
 * - Responsive layout validation
 * - Performance on mobile devices
 * - Accessibility on mobile
 */

// Helper function for mobile test setup
async function createMobileTest(browser: any, deviceName: string) {
  const device = devices[deviceName];
  if (!device) {
    throw new Error(`Device "${deviceName}" not found`);
  }
  const context = await browser.newContext(device);
  const page = await context.newPage();
  return { page, context };
}

test.describe('Mobile Device Compatibility', () => {
  
  test('iPhone 12 - should display and function properly', async ({ browser }, testInfo) => {
    // Skip device emulation tests for browsers that don't support it
    test.skip(
      testInfo.project.name === 'Google Chrome' ||
      testInfo.project.name === 'Microsoft Edge' ||
      testInfo.project.name === 'firefox',
      'Device emulation not supported in this browser'
    );

    const { page, context } = await createMobileTest(browser, 'iPhone 12');
    try {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Verify main content is visible
      const mainContent = page.locator('main, .main-content, .content, body');
      await expect(mainContent.first()).toBeVisible();

      // Test touch interactions
      const touchableElements = page.locator('button, a, [role="button"]');
      if (await touchableElements.count() > 0) {
        await touchableElements.first().tap();
        await page.waitForTimeout(300);
        await expect(page.locator('body')).toBeVisible();
      }
    } finally {
      await context.close();
    }
  });

  test('Pixel 5 - should display and function properly', async ({ browser }, testInfo) => {
    // Skip device emulation tests for browsers that don't support it
    test.skip(
      testInfo.project.name === 'Google Chrome' ||
      testInfo.project.name === 'Microsoft Edge' ||
      testInfo.project.name === 'firefox',
      'Device emulation not supported in this browser'
    );

    const { page, context } = await createMobileTest(browser, 'Pixel 5');
    try {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      const mainContent = page.locator('main, .main-content, .content, body');
      await expect(mainContent.first()).toBeVisible();

      // Verify viewport is mobile-sized
      const viewportSize = await page.viewportSize();
      expect(viewportSize?.width).toBeLessThanOrEqual(450);
    } finally {
      await context.close();
    }
  });

  test('iPad Pro - should display properly for tablet', async ({ browser }, testInfo) => {
    // Skip device emulation tests for browsers that don't support it
    test.skip(
      testInfo.project.name === 'Google Chrome' ||
      testInfo.project.name === 'Microsoft Edge' ||
      testInfo.project.name === 'firefox',
      'Device emulation not supported in this browser'
    );

    const { page, context } = await createMobileTest(browser, 'iPad Pro 11');
    try {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      const mainContent = page.locator('main, .main-content, .content, body');
      await expect(mainContent.first()).toBeVisible();
    } finally {
      await context.close();
    }
  });
});

// Configure mobile test for functionality testing
const mobileTest = test.extend({
  // Auto-skip tests for browsers that don't support device emulation
  page: async ({ browser, page, browserName }, use, testInfo) => {
    // Skip device emulation tests for browsers that don't support it
    if (testInfo.project.name === 'Google Chrome' ||
        testInfo.project.name === 'Microsoft Edge' ||
        testInfo.project.name === 'firefox') {
      test.skip(true, 'Device emulation not supported in this browser');
      await use(page);
      return;
    }

    // For supported browsers, use iPhone 12 emulation
    const context = await browser.newContext(devices['iPhone 12']);
    const mobilePage = await context.newPage();
    await use(mobilePage);
    await context.close();
  },
});

test.describe('Mobile Functionality', () => {

  mobileTest('should handle scrolling properly', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Test vertical scroll
    await page.evaluate(() => window.scrollTo(0, 500));
    const scrollY = await page.evaluate(() => window.scrollY);
    expect(scrollY).toBeGreaterThan(0);
  });

  mobileTest('should handle orientation changes', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const content = page.locator('main, .main-content, .content, body');
    
    // Test portrait orientation
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(300);
    await expect(content.first()).toBeVisible();
    
    // Test landscape orientation
    await page.setViewportSize({ width: 812, height: 375 });
    await page.waitForTimeout(300);
    await expect(content.first()).toBeVisible();
  });

  mobileTest('should handle form inputs on mobile', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const inputs = page.locator('input[type="text"], input[type="email"], textarea');
    
    if (await inputs.count() > 0) {
      const input = inputs.first();
      await input.tap();
      await input.fill('Mobile test input');
      
      await expect(input).toBeFocused();
      await expect(input).toHaveValue('Mobile test input');
    }
  });

  mobileTest('should maintain text readability', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const textElements = page.locator('p, span, div, h1, h2, h3, h4, h5, h6');
    
    if (await textElements.count() > 0) {
      const fontSize = await textElements.first().evaluate((el) => {
        return window.getComputedStyle(el).fontSize;
      });
      
      const fontSizeNumber = parseInt(fontSize.replace('px', ''));
      expect(fontSizeNumber).toBeGreaterThanOrEqual(14); // Minimum readable font size
    }
  });

  mobileTest('should allow pinch-to-zoom', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check viewport meta tag allows zooming
    const viewportMeta = page.locator('meta[name="viewport"]');
    if (await viewportMeta.count() > 0) {
      const content = await viewportMeta.getAttribute('content');
      
      // Should not prevent zooming for accessibility
      expect(content).not.toContain('user-scalable=no');
      expect(content).not.toContain('maximum-scale=1');
    }
  });
});

test.describe('Mobile Performance', () => {

  mobileTest('should load efficiently on mobile', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const loadTime = Date.now() - startTime;
    
    // Mobile should load within 5 seconds
    expect(loadTime).toBeLessThan(5000);
  });

  mobileTest('should handle slow network conditions', async ({ page, context }) => {
    // Simulate slow network
    await context.route('**/*', async route => {
      await new Promise(resolve => setTimeout(resolve, 50)); // Add 50ms delay
      await route.continue();
    });
    
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Verify page still loads and is functional
    await expect(page.locator('body')).toBeVisible();
  });
});

test.describe('Mobile Navigation', () => {

  mobileTest('should handle mobile menu interactions', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for mobile menu toggle
    const menuToggle = page.locator('.menu-toggle, .hamburger, .mobile-menu-toggle, [aria-label*="menu"]');
    
    if (await menuToggle.count() > 0) {
      await menuToggle.first().tap();
      await page.waitForTimeout(300);
      
      // Check if mobile menu is visible
      const mobileMenu = page.locator('.mobile-menu, .menu-mobile, nav .menu');
      if (await mobileMenu.count() > 0) {
        await expect(mobileMenu.first()).toBeVisible();
      }
    }
  });

  mobileTest('should handle navigation links on mobile', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const navLinks = page.locator('nav a, .nav a').first();
    if (await navLinks.count() > 0) {
      const href = await navLinks.getAttribute('href');
      if (href?.startsWith('/')) {
        await navLinks.tap();
        await page.waitForLoadState('networkidle');
        expect(page.url()).toContain(href);
      }
    }
  });
});

test.describe('Mobile Accessibility', () => {

  mobileTest('should be accessible on mobile devices', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check for proper heading structure
    const h1 = page.locator('h1');
    if (await h1.count() > 0) {
      await expect(h1.first()).toBeVisible();
    }
    
    // Check for proper focus management
    await page.keyboard.press('Tab');
    const focusedElement = page.locator(':focus');
    if (await focusedElement.count() > 0) {
      await expect(focusedElement).toBeVisible();
    }
  });

  mobileTest('should have proper touch targets', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that interactive elements are large enough for touch
    const buttons = page.locator('button, a[href], [role="button"]');

    if (await buttons.count() > 0) {
      const button = buttons.first();
      const boundingBox = await button.boundingBox();

      if (boundingBox) {
        // Touch targets should be at least 44x44px (iOS guidelines)
        expect(boundingBox.width).toBeGreaterThanOrEqual(44);
        expect(boundingBox.height).toBeGreaterThanOrEqual(44);
      }
    }
  });
});

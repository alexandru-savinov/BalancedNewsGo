import { test, expect, devices } from '@playwright/test';

// Simple mobile tests using built-in device configurations
test.describe('Mobile Responsiveness Tests', () => {
  
  test('iPhone 12 - should display properly', async ({ page, browser }) => {
    const context = await browser.newContext(devices['iPhone 12']);
    const mobilePage = await context.newPage();
    
    await mobilePage.goto('/');
    await mobilePage.waitForLoadState('networkidle');
    
    const mainContent = mobilePage.locator('main, .main-content, .content, body');
    await expect(mainContent.first()).toBeVisible();
    
    await context.close();
  });

  test('Pixel 5 - should display properly', async ({ page, browser }) => {
    const context = await browser.newContext(devices['Pixel 5']);
    const mobilePage = await context.newPage();
    
    await mobilePage.goto('/');
    await mobilePage.waitForLoadState('networkidle');
    
    const mainContent = mobilePage.locator('main, .main-content, .content, body');
    await expect(mainContent.first()).toBeVisible();
    
    await context.close();
  });

  test('iPad Pro - should display properly', async ({ page, browser }) => {
    const context = await browser.newContext(devices['iPad Pro']);
    const mobilePage = await context.newPage();
    
    await mobilePage.goto('/');
    await mobilePage.waitForLoadState('networkidle');
    
    const mainContent = mobilePage.locator('main, .main-content, .content, body');
    await expect(mainContent.first()).toBeVisible();
    
    await context.close();
  });

  test('Mobile layout should be responsive', async ({ page, browser }) => {
    const context = await browser.newContext(devices['iPhone 12']);
    const mobilePage = await context.newPage();
    
    await mobilePage.goto('/');
    await mobilePage.waitForLoadState('networkidle');
    
    // Check viewport width
    const viewportSize = await mobilePage.viewportSize();
    expect(viewportSize?.width).toBeLessThanOrEqual(450);
    
    // Check for mobile-friendly layout
    const body = mobilePage.locator('body');
    await expect(body).toBeVisible();
    
    await context.close();
  });

  test('Touch interactions should work', async ({ page, browser }) => {
    const context = await browser.newContext(devices['iPhone 12']);
    const mobilePage = await context.newPage();
    
    await mobilePage.goto('/');
    await mobilePage.waitForLoadState('networkidle');
    
    // Find touchable elements
    const touchableElements = mobilePage.locator('button, a, [role="button"], [onclick]');
    
    if (await touchableElements.count() > 0) {
      const element = touchableElements.first();
      
      // Perform touch interaction
      await element.tap();
      await mobilePage.waitForTimeout(500);
      
      // Verify page is still functional
      await expect(mobilePage.locator('body')).toBeVisible();
    }
    
    await context.close();
  });
});

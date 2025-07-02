import { test, expect } from '@playwright/test';

test.describe('Simple Debug Test', () => {
  test('should load article page without networkidle', async ({ page }) => {
    console.log('🌐 Navigating to article page: /article/5');
    await page.goto('/article/5');
    
    // Wait for DOM content loaded instead of networkidle
    await page.waitForLoadState('domcontentloaded');
    console.log('✅ DOM content loaded');
    
    // Check if basic elements exist
    const title = await page.title();
    console.log('📄 Page title:', title);
    
    const reanalyzeBtn = page.locator('#reanalyze-btn');
    await expect(reanalyzeBtn).toBeVisible({ timeout: 5000 });
    console.log('✅ Reanalyze button found');
    
    const btnText = page.locator('#btn-text');
    await expect(btnText).toBeVisible({ timeout: 5000 });
    await expect(btnText).toHaveText('Request Reanalysis');
    console.log('✅ Button text correct');
  });

  test('should load article page with load state', async ({ page }) => {
    console.log('🌐 Navigating to article page: /article/5');
    await page.goto('/article/5');
    
    // Wait for load instead of networkidle
    await page.waitForLoadState('load');
    console.log('✅ Page loaded');
    
    const reanalyzeBtn = page.locator('#reanalyze-btn');
    await expect(reanalyzeBtn).toBeVisible({ timeout: 5000 });
    console.log('✅ Test passed with load state');
  });

  test('should identify what prevents networkidle', async ({ page }) => {
    console.log('🌐 Navigating to article page: /article/5');
    
    // Monitor network requests
    const requests: string[] = [];
    page.on('request', request => {
      requests.push(`${request.method()} ${request.url()}`);
    });
    
    await page.goto('/article/5');
    await page.waitForLoadState('domcontentloaded');
    
    console.log('📊 Network requests made:');
    requests.forEach(req => console.log('  -', req));
    
    // Try to wait for networkidle with shorter timeout
    try {
      await page.waitForLoadState('networkidle', { timeout: 10000 });
      console.log('✅ Network idle achieved');
    } catch (error) {
      console.log('❌ Network idle timeout - ongoing requests likely');
      
      // Check for any ongoing requests or timers
      const activeRequests = await page.evaluate(() => {
        // Check for any active fetch requests or timers
        return {
          activeTimers: window.setTimeout.toString(),
          activeIntervals: window.setInterval.toString()
        };
      });
      console.log('🔍 Active timers/intervals:', activeRequests);
    }
  });
});

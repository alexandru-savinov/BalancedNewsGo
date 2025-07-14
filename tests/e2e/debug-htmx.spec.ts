import { test, expect } from '@playwright/test';

test.describe('Debug HTMX Loading', () => {
  test('should debug HTMX source loading', async ({ page }) => {
    // Listen for console messages
    page.on('console', msg => console.log('CONSOLE:', msg.text()));
    
    // Listen for network requests
    page.on('request', request => {
      console.log('REQUEST:', request.method(), request.url());
    });
    
    page.on('response', response => {
      console.log('RESPONSE:', response.status(), response.url());
    });

    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
    
    // Check if HTMX is loaded
    const htmxLoaded = await page.evaluate(() => {
      return typeof (window as any).htmx !== 'undefined';
    });
    console.log('HTMX loaded:', htmxLoaded);
    
    // Check if the container exists
    const containerExists = await page.locator('[data-testid="source-management-container"]').count();
    console.log('Container exists:', containerExists > 0);
    
    // Get the container content
    const containerContent = await page.locator('[data-testid="source-management-container"]').textContent();
    console.log('Container content:', containerContent);
    
    // Wait a bit longer and check again
    await page.waitForTimeout(5000);
    
    const containerContentAfter = await page.locator('[data-testid="source-management-container"]').textContent();
    console.log('Container content after 5s:', containerContentAfter);
    
    // Try to manually trigger the HTMX request
    await page.evaluate(() => {
      const container = document.querySelector('[data-testid="source-management-container"]');
      if (container && (window as any).htmx) {
        console.log('Manually triggering HTMX request');
        (window as any).htmx.ajax('GET', '/htmx/sources', container);
      }
    });
    
    // Wait for the manual request
    await page.waitForTimeout(3000);
    
    const finalContent = await page.locator('[data-testid="source-management-container"]').textContent();
    console.log('Final content:', finalContent);
    
    // Check if we can find the source management header
    const hasSourceManagement = finalContent?.includes('Source Management');
    console.log('Has Source Management:', hasSourceManagement);
    
    expect(hasSourceManagement).toBe(true);
  });
});

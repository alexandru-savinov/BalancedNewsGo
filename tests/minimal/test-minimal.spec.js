// Minimal test to isolate the Playwright issue
const { test, expect } = require('@playwright/test');

console.log('Starting minimal test...');

test('minimal test', async ({ page }) => {
  console.log('Test running');
  // Just try to load a basic page
  await page.goto('https://example.com');
  console.log('Page loaded');
});

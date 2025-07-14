import { test, expect } from '@playwright/test';

test.describe('Debug Form Submission', () => {
  test('should debug form submission process', async ({ page }) => {
    // Listen for network requests
    page.on('request', request => {
      if (request.url().includes('/sources')) {
        console.log('REQUEST:', request.method(), request.url());
        console.log('Headers:', request.headers());
        console.log('Post data:', request.postData());
      }
    });

    page.on('response', response => {
      if (response.url().includes('/sources')) {
        console.log('RESPONSE:', response.status(), response.url());
      }
    });

    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load with longer timeout
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Get initial source count
    const initialCount = await page.locator('[data-testid^="source-card-"]').count();
    console.log('Initial source count:', initialCount);
    
    // Open add new source form
    await page.click('button:has-text("Add New Source")');
    await page.waitForSelector('#source-form-container form');
    
    // Fill out the form with unique name
    const uniqueName = `Debug Test Source ${Date.now()}`;
    await page.fill('input[name="name"]', uniqueName);
    await page.fill('input[name="feed_url"]', `https://example.com/debug-test-${Date.now()}.xml`);
    await page.selectOption('select[name="category"]', 'center');
    await page.fill('input[name="default_weight"]', '1.5');
    
    // Submit the form
    console.log('Submitting form...');
    await page.click('button:has-text("Add Source")');
    
    // Wait for potential response
    await page.waitForTimeout(5000);
    
    // Check if source count changed
    const newCount = await page.locator('[data-testid^="source-card-"]').count();
    console.log('New source count:', newCount);
    
    // Check if form was cleared
    const formVisible = await page.locator('#source-form-container form').count();
    console.log('Form still visible:', formVisible > 0);
    
    // Look for any error messages
    const errorMessages = await page.locator('.error, .alert-danger, [role="alert"]').count();
    console.log('Error messages found:', errorMessages);
    
    expect(newCount).toBeGreaterThan(initialCount);
  });
});

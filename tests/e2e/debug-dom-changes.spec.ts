import { test, expect } from '@playwright/test';

test.describe('Debug DOM Changes', () => {
  test('should debug what happens to DOM after disable click', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Get BBC News card (source ID 2)
    const bbcCard = page.locator('[data-testid="source-card-2"]');
    await expect(bbcCard).toBeVisible();
    
    // Check initial state
    console.log('=== INITIAL STATE ===');
    const initialDisableButton = bbcCard.locator('[data-testid="disable-source-2"]');
    const initialEnableButton = bbcCard.locator('[data-testid="enable-source-2"]');
    const initialDisabledBadge = bbcCard.locator('.badge-disabled');
    
    console.log(`Disable button exists: ${await initialDisableButton.count() > 0}`);
    console.log(`Enable button exists: ${await initialEnableButton.count() > 0}`);
    console.log(`Disabled badge exists: ${await initialDisabledBadge.count() > 0}`);
    
    // Get the full HTML of the source card before clicking
    const beforeHTML = await bbcCard.innerHTML();
    console.log(`\nBefore HTML (first 200 chars): ${beforeHTML.substring(0, 200)}...`);
    
    // Click disable button
    console.log('\n=== CLICKING DISABLE BUTTON ===');
    await initialDisableButton.click();
    
    // Wait a bit for HTMX to process
    await page.waitForTimeout(3000);
    
    // Check state after click
    console.log('\n=== AFTER DISABLE CLICK ===');
    const afterDisableButton = bbcCard.locator('[data-testid="disable-source-2"]');
    const afterEnableButton = bbcCard.locator('[data-testid="enable-source-2"]');
    const afterDisabledBadge = bbcCard.locator('.badge-disabled');
    
    console.log(`Disable button exists: ${await afterDisableButton.count() > 0}`);
    console.log(`Enable button exists: ${await afterEnableButton.count() > 0}`);
    console.log(`Disabled badge exists: ${await afterDisabledBadge.count() > 0}`);
    
    // Get the full HTML of the source card after clicking
    const afterHTML = await bbcCard.innerHTML();
    console.log(`\nAfter HTML (first 200 chars): ${afterHTML.substring(0, 200)}...`);
    
    // Check if HTML actually changed
    const htmlChanged = beforeHTML !== afterHTML;
    console.log(`\nHTML changed: ${htmlChanged}`);
    
    if (!htmlChanged) {
      console.log('❌ HTML did not change - HTMX request may have failed');
      
      // Check if the source card still exists
      const cardStillExists = await bbcCard.count() > 0;
      console.log(`Source card still exists: ${cardStillExists}`);
      
      // Check if the container was replaced
      const containerExists = await page.locator('[data-testid="source-management-container"]').count() > 0;
      console.log(`Source management container exists: ${containerExists}`);
      
    } else {
      console.log('✅ HTML changed - HTMX request worked');
    }
  });
});

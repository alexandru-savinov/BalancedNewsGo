import { test, expect } from '@playwright/test';

test.describe('Debug Button States', () => {
  test('should debug enable/disable button states', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Get all source cards
    const sourceCards = page.locator('[data-testid^="source-card-"]');
    const count = await sourceCards.count();
    console.log(`Found ${count} source cards`);
    
    // Check each source card for button states
    for (let i = 0; i < count; i++) {
      const card = sourceCards.nth(i);
      const cardId = await card.getAttribute('data-source-id');
      
      // Check for enable button
      const enableButton = card.locator(`[data-testid="enable-source-${cardId}"]`);
      const enableExists = await enableButton.count() > 0;
      
      // Check for disable button  
      const disableButton = card.locator(`[data-testid="disable-source-${cardId}"]`);
      const disableExists = await disableButton.count() > 0;
      
      // Check for disabled badge
      const disabledBadge = card.locator('.badge-disabled');
      const hasDisabledBadge = await disabledBadge.count() > 0;
      
      // Get source name
      const sourceName = await card.locator('strong').first().textContent();
      
      console.log(`Source ${cardId} (${sourceName}): Enable=${enableExists}, Disable=${disableExists}, DisabledBadge=${hasDisabledBadge}`);
      
      // Verify only one button exists
      if (enableExists && disableExists) {
        console.log(`❌ ERROR: Source ${cardId} has BOTH enable and disable buttons!`);
      } else if (!enableExists && !disableExists) {
        console.log(`❌ ERROR: Source ${cardId} has NO enable or disable buttons!`);
      } else {
        console.log(`✅ Source ${cardId} has correct button state`);
      }
    }
  });
});

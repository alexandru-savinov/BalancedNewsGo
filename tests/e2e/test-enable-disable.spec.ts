import { test, expect } from '@playwright/test';

test.describe('Test Enable/Disable Functionality', () => {
  test('should toggle source enable/disable state', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Test with BBC News (source ID 2)
    const bbcCard = page.locator('[data-testid="source-card-2"]');
    await expect(bbcCard).toBeVisible();
    
    // Initially should be enabled (showing disable button)
    const initialDisableButton = bbcCard.locator('[data-testid="disable-source-2"]');
    await expect(initialDisableButton).toBeVisible();
    console.log('✓ BBC News initially enabled (disable button visible)');
    
    // Click disable button
    await initialDisableButton.click();
    
    // Wait for HTMX to update the UI
    await page.waitForTimeout(2000);
    
    // Now should be disabled (showing enable button and disabled badge)
    const enableButton = bbcCard.locator('[data-testid="enable-source-2"]');
    await expect(enableButton).toBeVisible();
    
    const disabledBadge = bbcCard.locator('.badge-disabled');
    await expect(disabledBadge).toBeVisible();
    console.log('✓ BBC News successfully disabled (enable button and disabled badge visible)');
    
    // Click enable button to re-enable
    await enableButton.click();
    
    // Wait for HTMX to update the UI
    await page.waitForTimeout(2000);
    
    // Should be back to enabled state (disable button visible, no disabled badge)
    const finalDisableButton = bbcCard.locator('[data-testid="disable-source-2"]');
    await expect(finalDisableButton).toBeVisible();
    
    const noBadge = bbcCard.locator('.badge-disabled');
    await expect(noBadge).not.toBeVisible();
    console.log('✓ BBC News successfully re-enabled (disable button visible, no disabled badge)');
    
    console.log('✅ Enable/disable functionality working correctly!');
  });
});

import { test, expect } from '@playwright/test';

test.describe('Test Enable/Disable Functionality', () => {
  test('should toggle source enable/disable state', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');

    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });

    // Find BBC News source dynamically (it might not always be ID 2)
    const bbcCard = page.locator('[data-testid^="source-card-"]').filter({ hasText: 'BBC News' }).first();
    await expect(bbcCard).toBeVisible();

    // Get the source ID from the card's data-testid attribute
    const sourceId = await bbcCard.getAttribute('data-testid');
    const idMatch = sourceId?.match(/source-card-(\d+)/);
    if (!idMatch) {
      throw new Error('Could not extract source ID from BBC News card');
    }
    const bbcSourceId = idMatch[1];
    console.log(`Found BBC News with source ID: ${bbcSourceId}`);

    // Set up dialog handler for confirmation
    page.on('dialog', async dialog => {
      console.log(`Dialog message: ${dialog.message()}`);
      await dialog.accept();
    });

    // Check current state and toggle accordingly
    const disableButton = page.locator(`[data-testid="disable-source-${bbcSourceId}"]`);
    const enableButton = page.locator(`[data-testid="enable-source-${bbcSourceId}"]`);

    if (await disableButton.isVisible()) {
      // Source is currently enabled, test disable -> enable
      console.log('✓ BBC News initially enabled (disable button visible)');

      // Click disable button
      await disableButton.click();

      // Wait for HTMX to update the UI
      await page.waitForTimeout(5000);

      // Check if the source list was refreshed
      await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 10000 });

      // Get the updated BBC card after HTMX refresh
      const updatedBbcCard = page.locator(`[data-testid="source-card-${bbcSourceId}"]`);
      await expect(updatedBbcCard).toBeVisible();

      // Check for disabled badge first (more reliable indicator)
      const disabledBadge = updatedBbcCard.locator('.badge-disabled');
      await expect(disabledBadge).toBeVisible();
      console.log('✓ Disabled badge is visible');

      // Now look for enable button using data-testid
      const newEnableButton = page.locator(`[data-testid="enable-source-${bbcSourceId}"]`);
      await expect(newEnableButton).toBeVisible();
      console.log('✓ BBC News successfully disabled (enable button and disabled badge visible)');

      // Click enable button to re-enable
      await newEnableButton.click();

      // Wait for HTMX to update the UI
      await page.waitForTimeout(5000);

      // Get the updated BBC card again after re-enabling
      const finalBbcCard = page.locator(`[data-testid="source-card-${bbcSourceId}"]`);
      await expect(finalBbcCard).toBeVisible();

      // Should be back to enabled state (disable button visible, no disabled badge)
      const finalDisableButton = page.locator(`[data-testid="disable-source-${bbcSourceId}"]`);
      await expect(finalDisableButton).toBeVisible();

      // Check that disabled badge is gone
      const noBadge = finalBbcCard.locator('.badge-disabled');
      await expect(noBadge).not.toBeVisible();
      console.log('✓ BBC News successfully re-enabled (disable button visible, no disabled badge)');

    } else if (await enableButton.isVisible()) {
      // Source is currently disabled, test enable -> disable
      console.log('✓ BBC News initially disabled (enable button visible)');

      // Click enable button
      await enableButton.click();

      // Wait for HTMX to update the UI
      await page.waitForTimeout(5000);

      // Check if the source list was refreshed
      await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 10000 });

      // Get the updated BBC card after HTMX refresh
      const updatedBbcCard = page.locator(`[data-testid="source-card-${bbcSourceId}"]`);
      await expect(updatedBbcCard).toBeVisible();

      // Should now be enabled (disable button visible, no disabled badge)
      const newDisableButton = page.locator(`[data-testid="disable-source-${bbcSourceId}"]`);
      await expect(newDisableButton).toBeVisible();

      // Check that disabled badge is gone
      const noBadge = updatedBbcCard.locator('.badge-disabled');
      await expect(noBadge).not.toBeVisible();
      console.log('✓ BBC News successfully enabled (disable button visible, no disabled badge)');

      // Click disable button to disable again
      await newDisableButton.click();

      // Wait for HTMX to update the UI
      await page.waitForTimeout(5000);

      // Get the updated BBC card again after disabling
      const finalBbcCard = page.locator(`[data-testid="source-card-${bbcSourceId}"]`);
      await expect(finalBbcCard).toBeVisible();

      // Check for disabled badge
      const disabledBadge = finalBbcCard.locator('.badge-disabled');
      await expect(disabledBadge).toBeVisible();

      // Now look for enable button using data-testid
      const finalEnableButton = page.locator(`[data-testid="enable-source-${bbcSourceId}"]`);
      await expect(finalEnableButton).toBeVisible();
      console.log('✓ BBC News successfully disabled (enable button and disabled badge visible)');

    } else {
      throw new Error('Neither enable nor disable button found - source state is unclear');
    }

    console.log('✅ Enable/disable functionality working correctly!');
  });
});

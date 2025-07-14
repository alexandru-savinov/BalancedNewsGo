import { test, expect } from '@playwright/test';

test.describe('Debug Modal', () => {
  test('should debug modal structure and close button', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Click stats button for BBC News
    console.log('=== CLICKING STATS BUTTON ===');
    await page.click('[data-testid="stats-source-2"]');
    
    // Wait for modal to appear
    await page.waitForSelector('#source-stats-modal', { 
      state: 'visible',
      timeout: 10000 
    });
    
    console.log('✓ Modal appeared');
    
    // Debug modal structure
    const modal = page.locator('#source-stats-modal');
    const modalHTML = await modal.innerHTML();
    console.log(`\nModal HTML (first 300 chars):\n${modalHTML.substring(0, 300)}...`);
    
    // Check for different close button selectors
    console.log('\n=== CHECKING CLOSE BUTTON SELECTORS ===');
    
    const selectors = [
      '.close',
      'button.close', 
      'button:has-text("Close")',
      '[aria-label="Close modal"]',
      'button[onclick*="hideSourceStatsModal"]',
      'button:has-text("×")',
      'button:has-text("&times;")'
    ];
    
    for (const selector of selectors) {
      const element = modal.locator(selector);
      const count = await element.count();
      const visible = count > 0 ? await element.first().isVisible() : false;
      console.log(`${selector}: count=${count}, visible=${visible}`);
    }
    
    // Try to find any button in the modal
    console.log('\n=== ALL BUTTONS IN MODAL ===');
    const allButtons = modal.locator('button');
    const buttonCount = await allButtons.count();
    console.log(`Total buttons in modal: ${buttonCount}`);
    
    for (let i = 0; i < buttonCount; i++) {
      const button = allButtons.nth(i);
      const text = await button.textContent();
      const className = await button.getAttribute('class');
      const onclick = await button.getAttribute('onclick');
      console.log(`Button ${i}: text="${text}", class="${className}", onclick="${onclick}"`);
    }
    
    // Try to close modal using the most likely selector
    console.log('\n=== ATTEMPTING TO CLOSE MODAL ===');
    const closeButton = modal.locator('.close').first();
    if (await closeButton.count() > 0) {
      console.log('Found .close button, attempting to click...');
      await closeButton.click();
      
      // Wait a bit and check if modal is hidden
      await page.waitForTimeout(1000);
      const isHidden = await modal.isHidden();
      console.log(`Modal hidden after click: ${isHidden}`);
    } else {
      console.log('No .close button found');
    }
  });
});

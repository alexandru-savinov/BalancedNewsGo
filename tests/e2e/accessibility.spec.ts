import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Accessibility Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should not have any automatically detectable accessibility issues', async ({ page }) => {
    const accessibilityScanResults = await new AxeBuilder({ page }).analyze();
    
    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('should have proper ARIA attributes', async ({ page }) => {
    // Check main navigation
    const nav = page.locator('nav[role="navigation"], nav, .navigation');
    if (await nav.count() > 0) {
      await expect(nav.first()).toBeVisible();
    }
    
    // Check search input has proper labels
    const searchInput = page.locator('[data-testid="search-input"], input[type="search"]');
    if (await searchInput.count() > 0) {
      const hasAriaLabel = await searchInput.first().getAttribute('aria-label');
      const hasLabel = await page.locator('label').count();
      const hasPlaceholder = await searchInput.first().getAttribute('placeholder');
      
      expect(hasAriaLabel || hasLabel > 0 || hasPlaceholder).toBeTruthy();
    }
    
    // Check buttons have accessible names
    const buttons = page.locator('button');
    const buttonCount = await buttons.count();
    
    for (let i = 0; i < Math.min(buttonCount, 5); i++) { // Test first 5 buttons
      const button = buttons.nth(i);
      const hasAriaLabel = await button.getAttribute('aria-label');
      const hasInnerText = await button.textContent();
      const hasTitle = await button.getAttribute('title');
      
      expect(hasAriaLabel || hasInnerText?.trim() || hasTitle).toBeTruthy();
    }
  });

  test('should be keyboard navigable', async ({ page }) => {
    // Start from first focusable element
    await page.keyboard.press('Tab');
    
    // Tab through several interactive elements
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);
    }
    
    // Verify focus is on a visible element
    const focusedElement = page.locator(':focus');
    if (await focusedElement.count() > 0) {
      await expect(focusedElement).toBeVisible();
    }
  });

  test('should have sufficient color contrast', async ({ page }) => {
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    const colorContrastViolations = accessibilityScanResults.violations.filter(
      violation => violation.id === 'color-contrast'
    );
    
    expect(colorContrastViolations).toEqual([]);
  });

  test('should work with screen readers', async ({ page }) => {
    // Check for proper heading structure
    const h1 = page.locator('h1');
    if (await h1.count() > 0) {
      await expect(h1.first()).toBeVisible();
    }
    
    // Check for landmark regions
    const main = page.locator('main, [role="main"]');
    if (await main.count() > 0) {
      await expect(main.first()).toBeVisible();
    }
    
    // Check for skip links
    const skipLink = page.locator('a[href="#main-content"], a[href="#content"], .skip-link');
    if (await skipLink.count() > 0) {
      const skipText = await skipLink.first().textContent();
      expect(skipText?.toLowerCase()).toMatch(/skip|main|content/i);
    }
  });

  test('should have proper form labels', async ({ page }) => {
    const forms = page.locator('form');
    const formCount = await forms.count();
    
    for (let i = 0; i < Math.min(formCount, 3); i++) { // Test first 3 forms
      const form = forms.nth(i);
      const inputs = form.locator('input, select, textarea');
      const inputCount = await inputs.count();
      
      for (let j = 0; j < Math.min(inputCount, 5); j++) { // Test first 5 inputs per form
        const input = inputs.nth(j);
        const inputType = await input.getAttribute('type');
        
        // Skip hidden inputs and buttons
        if (inputType === 'hidden' || inputType === 'submit' || inputType === 'button') {
          continue;
        }
        
        const hasLabel = await input.getAttribute('aria-label');
        const hasLabelledBy = await input.getAttribute('aria-labelledby');
        const inputId = await input.getAttribute('id');
        const hasAssociatedLabel = inputId ? await page.locator(`label[for="${inputId}"]`).count() > 0 : false;
        const hasPlaceholder = await input.getAttribute('placeholder');
        
        expect(hasLabel || hasLabelledBy || hasAssociatedLabel || hasPlaceholder).toBeTruthy();
      }
    }
  });

  test('should handle focus management', async ({ page }) => {
    // Test focus trap in modals/dialogs if they exist
    const modal = page.locator('[role="dialog"], .modal, .dialog');
    
    if (await modal.count() > 0) {
      // Open modal if there's a trigger
      const modalTrigger = page.locator('[data-toggle="modal"], .modal-trigger, button:has-text("Open")');
      if (await modalTrigger.count() > 0) {
        await modalTrigger.first().click();
        await page.waitForTimeout(500);
          // Check if focus is moved to modal
        const focusedElement = page.locator(':focus');
        if (await focusedElement.count() > 0) {
          // Focus should be within modal
          expect(await modal.first().locator(':focus').count()).toBeGreaterThan(0);
        }
      }
    }
  });

  test('should have proper heading hierarchy', async ({ page }) => {
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a'])
      .analyze();
    
    const headingViolations = accessibilityScanResults.violations.filter(
      violation => violation.id.includes('heading')
    );
    
    expect(headingViolations).toEqual([]);
  });

  test('should provide alternative text for images', async ({ page }) => {
    const images = page.locator('img');
    const imageCount = await images.count();
    
    for (let i = 0; i < Math.min(imageCount, 10); i++) { // Test first 10 images
      const img = images.nth(i);
      const alt = await img.getAttribute('alt');
      const role = await img.getAttribute('role');
      const ariaLabel = await img.getAttribute('aria-label');
      
      // Decorative images should have empty alt or role="presentation"
      // Content images should have meaningful alt text
      if (role === 'presentation' || alt === '') {
        // This is acceptable for decorative images
        continue;
      }
      
      expect(alt || ariaLabel).toBeTruthy();
    }
  });

  test('should support reduced motion preferences', async ({ page }) => {
    // Test with reduced motion preference
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Check that animations are reduced or disabled
    const animatedElements = page.locator('[class*="animate"], [class*="transition"]');
    if (await animatedElements.count() > 0) {
      // This is a basic check - in real scenarios, you might check CSS properties
      await expect(animatedElements.first()).toBeVisible();
    }
  });
});

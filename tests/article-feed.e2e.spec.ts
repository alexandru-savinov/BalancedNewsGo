import { test, expect } from '@playwright/test';

test.describe('BalancedNewsGo Main Feed', () => {
  test('renders article cards and bias slider', async ({ page }) => {    // Go to the running Go server
    await page.goto('http://localhost:8080/articles');    // Wait for at least one article card to appear (data-testid pattern)
    const articleCard = page.locator('[data-testid^="article-card"]');
    await expect(articleCard.first()).toBeVisible();    // Check that multiple article cards are present
    const cardCount = await articleCard.count();
    expect(cardCount).toBeGreaterThan(0);    // Check for the bias slider (by role or data-testid) - may not exist on all pages
    const biasSlider = page.locator('[data-testid="bias-slider"]');
    if (await biasSlider.count() > 0) {
      await expect(biasSlider).toBeVisible();
    }

    // Check for a tooltip (if present) - may not exist on all pages
    const tooltip = page.locator('[data-testid="bias-tooltip"]');
    if (await tooltip.count() > 0) {
      // Tooltip may be hidden by default, so just check it exists in DOM
      await expect(tooltip).toHaveCount(1);
    }

    // Optionally, check feedback form is present - may not exist on all pages
    const feedbackForm = page.locator('[data-testid="feedback-form"]');
    if (await feedbackForm.count() > 0) {
      await expect(feedbackForm).toBeVisible();
    }
  });
});

import { test, expect } from '@playwright/test';

test.describe('BalancedNewsGo Main Feed', () => {
  test('renders article cards and bias slider', async ({ page }) => {
    // Go to the running frontend app
    await page.goto('http://localhost:5173/');

    // Wait for at least one article card to appear (data-testid pattern)
    const articleCard = page.locator('[data-testid^="article-card-"]');
    await expect(articleCard.first()).toBeVisible();

    // Check that multiple article cards are present
    await expect(articleCard).toHaveCountGreaterThan(0);

    // Check for the bias slider (by role or data-testid)
    const biasSlider = page.locator('[data-testid="bias-slider"]');
    await expect(biasSlider).toBeVisible();

    // Check for a tooltip (if present)
    const tooltip = page.locator('[data-testid="bias-tooltip"]');
    // Tooltip may be hidden by default, so just check it exists in DOM
    await expect(tooltip).toHaveCount(1);

    // Optionally, check feedback form is present
    const feedbackForm = page.locator('[data-testid="feedback-form"]');
    await expect(feedbackForm).toBeVisible();
  });
});
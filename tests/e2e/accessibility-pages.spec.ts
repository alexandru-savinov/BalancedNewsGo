import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Page-Specific Accessibility Tests', () => {
  
  test('Articles listing page (/articles) should be accessible', async ({ page }) => {
    await page.goto('/articles');
    await page.waitForLoadState('networkidle');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    // Fail on critical accessibility violations
    const criticalViolations = accessibilityScanResults.violations.filter(
      violation => violation.impact === 'critical' || violation.impact === 'serious'
    );
    
    if (criticalViolations.length > 0) {
      console.log('Critical accessibility violations found:', criticalViolations);
    }
    
    expect(criticalViolations).toEqual([]);
    
    // Check specific elements for articles page
    const articlesGrid = page.locator('.articles-grid');
    if (await articlesGrid.count() > 0) {
      await expect(articlesGrid).toBeVisible();
    }
    
    // Check article cards have proper structure
    const articleItems = page.locator('.article-item');
    if (await articleItems.count() > 0) {
      const firstArticle = articleItems.first();
      const articleTitle = firstArticle.locator('.article-title a, h2 a, h3 a');
      if (await articleTitle.count() > 0) {
        await expect(articleTitle).toBeVisible();
        const titleText = await articleTitle.textContent();
        expect(titleText?.trim()).toBeTruthy();
      }
    }
  });

  test('Article detail page (/article/:id) should be accessible', async ({ page }) => {
    await page.goto('/article/1');
    await page.waitForLoadState('networkidle');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    // Fail on critical accessibility violations
    const criticalViolations = accessibilityScanResults.violations.filter(
      violation => violation.impact === 'critical' || violation.impact === 'serious'
    );
    
    if (criticalViolations.length > 0) {
      console.log('Critical accessibility violations found:', criticalViolations);
    }
    
    expect(criticalViolations).toEqual([]);
    
    // Check article content structure
    const articleTitle = page.locator('.article-title, h1');
    if (await articleTitle.count() > 0) {
      await expect(articleTitle.first()).toBeVisible();
      const titleText = await articleTitle.first().textContent();
      expect(titleText?.trim()).toBeTruthy();
    }
    
    // Check bias analysis section if present
    const biasAnalysis = page.locator('.bias-analysis');
    if (await biasAnalysis.count() > 0) {
      await expect(biasAnalysis).toBeVisible();
      
      const biasIndicator = biasAnalysis.locator('.bias-indicator');
      if (await biasIndicator.count() > 0) {
        const biasText = await biasIndicator.textContent();
        expect(biasText?.trim()).toBeTruthy();
      }
    }
  });

  test('Admin dashboard (/admin) should be accessible', async ({ page }) => {
    await page.goto('/admin');
    await page.waitForLoadState('networkidle');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    // Fail on critical accessibility violations
    const criticalViolations = accessibilityScanResults.violations.filter(
      violation => violation.impact === 'critical' || violation.impact === 'serious'
    );
    
    if (criticalViolations.length > 0) {
      console.log('Critical accessibility violations found:', criticalViolations);
    }
    
    expect(criticalViolations).toEqual([]);
    
    // Check dashboard structure
    const dashboardCards = page.locator('.dashboard-card, .admin-card');
    if (await dashboardCards.count() > 0) {
      await expect(dashboardCards.first()).toBeVisible();
    }
    
    // Check admin controls if present
    const adminControls = page.locator('.admin-controls');
    if (await adminControls.count() > 0) {
      await expect(adminControls).toBeVisible();
    }
  });

  test('All pages should have proper document structure', async ({ page }) => {
    const pages = ['/articles', '/article/1', '/admin'];
    
    for (const pagePath of pages) {
      await page.goto(pagePath);
      await page.waitForLoadState('networkidle');
      
      // Check for proper document structure
      const title = await page.title();
      expect(title).toBeTruthy();
      expect(title.length).toBeGreaterThan(0);
      
      // Check for main content area
      const main = page.locator('main, [role="main"], .main-content');
      if (await main.count() > 0) {
        await expect(main.first()).toBeVisible();
      }
      
      // Check for proper heading hierarchy
      const h1 = page.locator('h1');
      if (await h1.count() > 0) {
        await expect(h1.first()).toBeVisible();
      }
    }
  });

  test('All pages should be keyboard navigable', async ({ page }) => {
    const pages = ['/articles', '/article/1', '/admin'];
    
    for (const pagePath of pages) {
      await page.goto(pagePath);
      await page.waitForLoadState('networkidle');
      
      // Test keyboard navigation
      await page.keyboard.press('Tab');
      
      // Tab through several elements
      for (let i = 0; i < 3; i++) {
        await page.keyboard.press('Tab');
        await page.waitForTimeout(100);
      }
      
      // Verify focus is on a visible element
      const focusedElement = page.locator(':focus');
      if (await focusedElement.count() > 0) {
        await expect(focusedElement).toBeVisible();
      }
    }
  });
});

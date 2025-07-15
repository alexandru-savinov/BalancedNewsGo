import { test, expect } from '@playwright/test';

test.describe('Create Baseline Sources', () => {
  test('should create baseline sources for E2E tests', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');

    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });

    // Get initial count (may not be 0 if database has existing sources)
    const initialCount = await page.locator('[data-testid^="source-card-"]').count();
    console.log(`Initial source count: ${initialCount}`);

    // Check if the required baseline sources already exist
    const bbcNewsExists = await page.locator('[data-testid="source-card-2"]').count() > 0;
    const huffPostExists = await page.locator('text=HuffPost').count() > 0;
    const msnbcExists = await page.locator('text=MSNBC').count() > 0;

    if (bbcNewsExists && huffPostExists && msnbcExists) {
      console.log('✓ Baseline sources already exist - test passed');
      return;
    }

    // Create Source 1: HuffPost (if it doesn't exist)
    if (!huffPostExists) {
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');

      await page.fill('[data-testid="source-name-input"]', 'HuffPost');
      await page.selectOption('select[name="channel_type"]', 'rss');
      await page.fill('[data-testid="source-feed-url-input"]', 'https://www.huffpost.com/section/front-page/feed');
      await page.selectOption('select[name="category"]', 'left');
      await page.fill('[data-testid="source-weight-input"]', '1.0');

      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000); // Wait for form submission

      console.log('✓ HuffPost created');
    } else {
      console.log('✓ HuffPost already exists');
    }

    // Create Source 2: BBC News (THIS IS WHAT TESTS EXPECT) - if it doesn't exist
    if (!bbcNewsExists) {
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');

      await page.fill('[data-testid="source-name-input"]', 'BBC News');
      await page.selectOption('select[name="channel_type"]', 'rss');
      await page.fill('[data-testid="source-feed-url-input"]', 'https://feeds.bbci.co.uk/news/rss.xml');
      await page.selectOption('select[name="category"]', 'center');
      await page.fill('[data-testid="source-weight-input"]', '1.0');

      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000); // Wait for form submission

      console.log('✓ BBC News created');
    } else {
      console.log('✓ BBC News already exists');
    }

    // Create Source 3: MSNBC (if it doesn't exist)
    if (!msnbcExists) {
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');

      await page.fill('[data-testid="source-name-input"]', 'MSNBC');
      await page.selectOption('select[name="channel_type"]', 'rss');
      await page.fill('[data-testid="source-feed-url-input"]', 'http://www.msnbc.com/feeds/latest');
      await page.selectOption('select[name="category"]', 'right');
      await page.fill('[data-testid="source-weight-input"]', '1.0');

      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000); // Wait for form submission

      console.log('✓ MSNBC created');
    } else {
      console.log('✓ MSNBC already exists');
    }

    // Reload the page to ensure we have the latest state
    await page.reload();
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });

    // Verify BBC News is accessible as source-card-2 or by name
    const bbcNewsCard = page.locator('text=BBC News').first();
    await expect(bbcNewsCard).toBeVisible();

    // Verify HuffPost exists
    const huffPostCard = page.locator('text=HuffPost').first();
    await expect(huffPostCard).toBeVisible();

    // Verify MSNBC exists
    const msnbcCard = page.locator('text=MSNBC').first();
    await expect(msnbcCard).toBeVisible();

    console.log('✓ All baseline sources verified - test passed!');
  });
});

import { test, expect } from '@playwright/test';

test.describe('Create Baseline Sources', () => {
  test('should create baseline sources for E2E tests', async ({ page }) => {
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for source management to load
    await page.waitForSelector('[data-testid="source-management-container"]', { timeout: 30000 });
    
    // Verify we start with 0 sources
    const initialCount = await page.locator('[data-testid^="source-card-"]').count();
    console.log(`Initial source count: ${initialCount}`);
    expect(initialCount).toBe(0);
    
    // Create Source 1: HuffPost
    await page.click('button:has-text("Add New Source")');
    await page.waitForSelector('#source-form-container form');
    
    await page.fill('[data-testid="source-name-input"]', 'HuffPost');
    await page.selectOption('select[name="channel_type"]', 'rss');
    await page.fill('[data-testid="source-feed-url-input"]', 'https://www.huffpost.com/section/front-page/feed');
    await page.selectOption('select[name="category"]', 'left');
    await page.fill('[data-testid="source-weight-input"]', '1.0');
    
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000); // Wait for form submission
    
    // Verify HuffPost was created
    const countAfter1 = await page.locator('[data-testid^="source-card-"]').count();
    expect(countAfter1).toBe(1);
    console.log('✓ HuffPost created (should be ID 1)');
    
    // Create Source 2: BBC News (THIS IS WHAT TESTS EXPECT)
    await page.click('button:has-text("Add New Source")');
    await page.waitForSelector('#source-form-container form');
    
    await page.fill('[data-testid="source-name-input"]', 'BBC News');
    await page.selectOption('select[name="channel_type"]', 'rss');
    await page.fill('[data-testid="source-feed-url-input"]', 'https://feeds.bbci.co.uk/news/rss.xml');
    await page.selectOption('select[name="category"]', 'center');
    await page.fill('[data-testid="source-weight-input"]', '1.0');
    
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000); // Wait for form submission
    
    // Verify BBC News was created
    const countAfter2 = await page.locator('[data-testid^="source-card-"]').count();
    expect(countAfter2).toBe(2);
    console.log('✓ BBC News created (should be ID 2)');
    
    // Create Source 3: MSNBC
    await page.click('button:has-text("Add New Source")');
    await page.waitForSelector('#source-form-container form');
    
    await page.fill('[data-testid="source-name-input"]', 'MSNBC');
    await page.selectOption('select[name="channel_type"]', 'rss');
    await page.fill('[data-testid="source-feed-url-input"]', 'http://www.msnbc.com/feeds/latest');
    await page.selectOption('select[name="category"]', 'right');
    await page.fill('[data-testid="source-weight-input"]', '1.0');
    
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000); // Wait for form submission
    
    // Verify MSNBC was created
    const finalCount = await page.locator('[data-testid^="source-card-"]').count();
    expect(finalCount).toBe(3);
    console.log('✓ MSNBC created (should be ID 3)');
    
    // Verify BBC News is accessible as source-card-2
    const bbcNewsCard = page.locator('[data-testid="source-card-2"]');
    await expect(bbcNewsCard).toBeVisible();
    await expect(bbcNewsCard.locator('text=BBC News')).toBeVisible();
    await expect(bbcNewsCard.locator('.badge-rss')).toBeVisible();
    await expect(bbcNewsCard.locator('.badge-center')).toBeVisible();
    
    console.log('✓ BBC News correctly has ID 2 - baseline sources created successfully!');
  });
});

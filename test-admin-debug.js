#!/usr/bin/env node

const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('Testing admin dashboard...');
    
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
    
    // Check if admin dashboard loads
    const title = await page.title();
    console.log('Page title:', title);
    
    // Check if main heading exists
    const heading = await page.locator('h1').textContent();
    console.log('Main heading:', heading);
    
    // Check for admin sections
    const sections = ['Feed Management', 'Analysis Control', 'Database Management', 'Monitoring'];
    for (const section of sections) {
      const element = page.locator(`h4:has-text("${section}")`);
      const isVisible = await element.isVisible();
      console.log(`${section}: ${isVisible ? 'FOUND' : 'NOT FOUND'}`);
    }
    
    // Check for status elements
    const statusElements = ['Database', 'LLM Service', 'RSS Feeds', 'Server'];
    for (const status of statusElements) {
      const element = page.locator(`h4:has-text("${status}")`);
      const isVisible = await element.isVisible();
      console.log(`${status} status: ${isVisible ? 'FOUND' : 'NOT FOUND'}`);
    }
    
    console.log('Test completed successfully!');
    
  } catch (error) {
    console.error('Test failed:', error);
    process.exit(1);
  } finally {
    await browser.close();
  }
})();

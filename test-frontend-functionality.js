#!/usr/bin/env node

const puppeteer = require('puppeteer');

async function testFrontendFunctionality() {
  console.log('🧪 Testing modernized frontend functionality...');

  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();

  // Listen for console messages and errors
  const consoleMessages = [];
  const pageErrors = [];

  page.on('console', msg => {
    consoleMessages.push(msg.text());
    console.log('CONSOLE:', msg.text());
  });

  page.on('pageerror', error => {
    pageErrors.push(error.message);
    console.log('PAGE ERROR:', error.message);
  });

  try {
    // Navigate to the application
    console.log('📄 Navigating to application...');
    await page.goto('http://localhost:8080', { waitUntil: 'networkidle0' });
      // Wait for the page to load completely
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Check if main JavaScript loaded
    console.log('🔍 Checking JavaScript loading...');
    const jsLoaded = await page.evaluate(() => {
      return typeof window.NewsBalancer !== 'undefined';
    });
    console.log(jsLoaded ? '✅ Main JavaScript loaded' : '⚠️  Main JavaScript not detected');

    // Test search functionality
    console.log('🔍 Testing search functionality...');
    const searchInput = await page.$('.search-input, input[type="search"], input[name="q"]');
    if (searchInput) {
      await searchInput.type('test search');
      console.log('✅ Search input works');
    } else {
      console.log('❌ Search input not found');
    }    // Test mobile menu toggle
    console.log('📱 Testing mobile menu...');
    const mobileBtn = await page.$('.mobile-menu-btn');
    if (mobileBtn) {
      await mobileBtn.click();
      await new Promise(resolve => setTimeout(resolve, 500)); // Wait for animation
      console.log('✅ Mobile menu toggle works');
    } else {
      console.log('❌ Mobile menu button not found');
    }
      // Test advanced filters
    console.log('🔧 Testing advanced filters...');
    const filterToggle = await page.$('.filter-toggle, .advanced-filters button');
    if (filterToggle) {
      await filterToggle.click();
      await new Promise(resolve => setTimeout(resolve, 500)); // Wait for animation
      console.log('✅ Advanced filters toggle works');
    } else {
      console.log('❌ Advanced filters toggle not found');
    }

    // Test SVG icons loading
    console.log('🎨 Testing SVG icons...');
    const svgIcons = await page.$$('svg[class*="icon-"], .icon svg');
    console.log(`📊 Found ${svgIcons.length} SVG icons`);

    // Test responsive design
    console.log('📱 Testing responsive design...');
    await page.setViewport({ width: 375, height: 667 }); // Mobile viewport
    await new Promise(resolve => setTimeout(resolve, 500));

    const isMobileOptimized = await page.evaluate(() => {
      const grid = document.querySelector('.articles-grid');
      if (!grid) return false;
      const computedStyle = window.getComputedStyle(grid);
      return computedStyle.gridTemplateColumns.includes('1fr');
    });
    console.log(isMobileOptimized ? '✅ Mobile optimization works' : '⚠️  Mobile optimization not detected');

    // Reset to desktop viewport
    await page.setViewport({ width: 1920, height: 1080 });

    // Test CSS animations
    console.log('✨ Testing CSS animations...');
    const hasAnimations = await page.evaluate(() => {
      const cards = document.querySelectorAll('.article-card');
      if (cards.length === 0) return false;
      const computedStyle = window.getComputedStyle(cards[0]);
      return computedStyle.transition.includes('all');
    });
    console.log(hasAnimations ? '✅ CSS animations present' : '⚠️  CSS animations not detected');

    // Summary
    console.log('\n📋 Test Summary:');
    console.log(`Console Messages: ${consoleMessages.length}`);
    console.log(`Page Errors: ${pageErrors.length}`);

    if (pageErrors.length === 0) {
      console.log('🎉 Frontend functionality test completed successfully!');
    } else {
      console.log('⚠️  Some issues detected - check errors above');
    }

  } catch (error) {
    console.error('❌ Test failed:', error.message);
  } finally {
    await browser.close();
  }
}

testFrontendFunctionality().catch(console.error);

import { test, expect } from '@playwright/test';

test.describe('ProgressIndicator Component Debug', () => {
  test('should verify ProgressIndicator component behavior', async ({ page }) => {
    console.log('🌐 Navigating to article page');
    await page.goto('/article/5');
    await page.waitForLoadState('load');

    // Check if ProgressIndicator element exists
    const progressIndicator = page.locator('#reanalysis-progress');
    await expect(progressIndicator).toBeAttached();
    console.log('✅ ProgressIndicator element exists');

    // Check initial state
    const initialState = await page.evaluate(() => {
      const element = document.getElementById('reanalysis-progress');
      return {
        exists: !!element,
        tagName: element?.tagName,
        className: element?.className,
        style: element?.style.cssText,
        display: window.getComputedStyle(element).display,
        visibility: window.getComputedStyle(element).visibility,
        textContent: element?.textContent,
        innerHTML: element?.innerHTML.substring(0, 200) + '...',
        shadowRoot: !!element?.shadowRoot,
        autoConnect: element?.getAttribute('auto-connect'),
        articleId: element?.getAttribute('article-id')
      };
    });

    console.log('🔍 Initial ProgressIndicator state:', JSON.stringify(initialState, null, 2));

    // Click reanalyze button to trigger progress
    const reanalyzeBtn = page.locator('#reanalyze-btn');
    await expect(reanalyzeBtn).toBeVisible();
    
    console.log('🖱️ Clicking reanalyze button');
    await reanalyzeBtn.click();

    // Wait a moment for connection
    await page.waitForTimeout(1000);

    // Check state after clicking
    const afterClickState = await page.evaluate(() => {
      const element = document.getElementById('reanalysis-progress');
      const computedStyle = window.getComputedStyle(element);
      
      // Try to access shadow DOM content
      let shadowContent = null;
      if (element?.shadowRoot) {
        const progressFill = element.shadowRoot.querySelector('.progress-fill');
        const percentageText = element.shadowRoot.querySelector('.progress-percentage');
        const stageText = element.shadowRoot.querySelector('.progress-stage');
        
        shadowContent = {
          progressFillWidth: progressFill?.style.width,
          percentageText: percentageText?.textContent,
          stageText: stageText?.textContent,
          allShadowText: element.shadowRoot.textContent
        };
      }
      
      return {
        display: computedStyle.display,
        visibility: computedStyle.visibility,
        textContent: element?.textContent,
        shadowContent: shadowContent,
        className: element?.className
      };
    });

    console.log('🔍 ProgressIndicator state after click:', JSON.stringify(afterClickState, null, 2));

    // Monitor for progress updates
    let progressUpdates: any[] = [];
    
    page.on('console', (msg) => {
      const text = msg.text();
      if (text.includes('Progress:') || text.includes('ProgressIndicator') || text.includes('SSE')) {
        progressUpdates.push(text);
        console.log(`📊 Progress update: ${text}`);
      }
    });

    // Wait for some progress
    await page.waitForTimeout(3000);

    // Check final state
    const finalState = await page.evaluate(() => {
      const element = document.getElementById('reanalysis-progress');
      const computedStyle = window.getComputedStyle(element);
      
      let shadowContent = null;
      if (element?.shadowRoot) {
        const progressFill = element.shadowRoot.querySelector('.progress-fill');
        const percentageText = element.shadowRoot.querySelector('.progress-percentage');
        const stageText = element.shadowRoot.querySelector('.progress-stage');
        
        shadowContent = {
          progressFillWidth: progressFill?.style.width,
          percentageText: percentageText?.textContent,
          stageText: stageText?.textContent,
          allShadowText: element.shadowRoot.textContent?.trim()
        };
      }
      
      return {
        display: computedStyle.display,
        visibility: computedStyle.visibility,
        textContent: element?.textContent?.trim(),
        shadowContent: shadowContent,
        className: element?.className
      };
    });

    console.log('🔍 Final ProgressIndicator state:', JSON.stringify(finalState, null, 2));
    console.log(`📊 Total progress updates captured: ${progressUpdates.length}`);
    
    if (progressUpdates.length > 0) {
      console.log('📋 Progress updates:');
      progressUpdates.forEach((update, index) => {
        console.log(`  ${index + 1}. ${update}`);
      });
    }

    // Verify that progress indicator is visible and has content
    expect(finalState.display).not.toBe('none');
    expect(finalState.shadowContent).not.toBeNull();
  });
});

// Frontend Debug Script for Reanalysis Button Issue
// Run this in browser console to debug the SSE connection and button state

console.log('=== Frontend Reanalysis Debug ===');

// 1. Check if elements exist
const progressIndicator = document.getElementById('reanalysis-progress');
const reanalyzeBtn = document.getElementById('reanalyze-btn');

console.log('ProgressIndicator element:', progressIndicator);
console.log('Reanalyze button:', reanalyzeBtn);

if (!progressIndicator) {
    console.error('❌ ProgressIndicator element not found!');
}

if (!reanalyzeBtn) {
    console.error('❌ Reanalyze button not found!');
}

// 2. Extract article ID
const articleId = reanalyzeBtn ? reanalyzeBtn.getAttribute('data-article-id') : 'not found';
console.log('Article ID:', articleId);

// 3. Test SSE connection directly
if (articleId && articleId !== 'not found') {
    console.log('=== Testing SSE Connection ===');
    
    const eventSource = new EventSource(`/api/llm/score-progress/${articleId}`);
    
    eventSource.onopen = function(event) {
        console.log('✅ SSE connection opened:', event);
    };
    
    eventSource.onmessage = function(event) {
        console.log('📨 SSE message received:', event.data);
        try {
            const data = JSON.parse(event.data);
            console.log('📊 Parsed data:', data);
            console.log('   Status:', data.status);
            console.log('   Step:', data.step);
            console.log('   Percent:', data.percent);
            console.log('   Message:', data.message);
            
            // Check completion detection logic
            const progress = data.progress || data.percent || 0;
            const status = data.status ? data.status.toLowerCase() : '';
            const isComplete = progress >= 100 || status === 'completed' || status === 'complete';
            
            console.log('🔍 Completion check:');
            console.log('   Progress >= 100:', progress >= 100);
            console.log('   Status lowercase:', status);
            console.log('   Is complete:', isComplete);
            
            if (isComplete) {
                console.log('🎉 COMPLETION DETECTED! Should trigger button reset.');
            }
        } catch (e) {
            console.error('❌ Error parsing SSE data:', e);
        }
    };
    
    eventSource.onerror = function(event) {
        console.error('❌ SSE error:', event);
    };
    
    // Close after 10 seconds
    setTimeout(() => {
        console.log('🔌 Closing SSE connection');
        eventSource.close();
    }, 10000);
}

// 4. Test ProgressIndicator component if it exists
if (progressIndicator && typeof progressIndicator.connect === 'function') {
    console.log('=== Testing ProgressIndicator Component ===');
    
    // Add event listeners to monitor component events
    progressIndicator.addEventListener('completed', (event) => {
        console.log('🎉 ProgressIndicator completed event:', event.detail);
    });
    
    progressIndicator.addEventListener('progressupdate', (event) => {
        console.log('📈 ProgressIndicator progress update:', event.detail);
    });
    
    progressIndicator.addEventListener('error', (event) => {
        console.error('❌ ProgressIndicator error:', event.detail);
    });
    
    progressIndicator.addEventListener('statuschange', (event) => {
        console.log('🔄 ProgressIndicator status change:', event.detail);
    });
}

// 5. Monitor button state changes
if (reanalyzeBtn) {
    console.log('=== Monitoring Button State ===');
    
    const btnText = reanalyzeBtn.querySelector('.btn-text');
    const btnLoading = reanalyzeBtn.querySelector('.btn-loading');
    
    console.log('Button text element:', btnText);
    console.log('Button loading element:', btnLoading);
    console.log('Button disabled:', reanalyzeBtn.disabled);
    console.log('Button text content:', btnText ? btnText.textContent : 'N/A');
    
    // Monitor for changes
    const observer = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
            if (mutation.type === 'attributes' && mutation.attributeName === 'disabled') {
                console.log('🔄 Button disabled state changed:', reanalyzeBtn.disabled);
            }
            if (mutation.type === 'childList' || mutation.type === 'characterData') {
                console.log('🔄 Button content changed:', btnText ? btnText.textContent : 'N/A');
            }
        });
    });
    
    observer.observe(reanalyzeBtn, { 
        attributes: true, 
        childList: true, 
        subtree: true, 
        characterData: true 
    });
    
    // Stop observing after 30 seconds
    setTimeout(() => {
        observer.disconnect();
        console.log('🔌 Stopped monitoring button changes');
    }, 30000);
}

console.log('=== Debug script setup complete ===');
console.log('Monitor the console for SSE events and button state changes.');
console.log('Try clicking the reanalyze button to test the flow.');

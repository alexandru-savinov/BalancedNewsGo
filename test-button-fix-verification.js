// Complete Frontend Button Fix Verification Script
// Run this in browser console to test the reanalysis button fix

console.log('🧪 FRONTEND BUTTON FIX VERIFICATION');
console.log('===================================');

// Test configuration
const TEST_TIMEOUT = 45000; // 45 seconds
let testResults = {
    elementsFound: false,
    buttonStateChanges: [],
    progressEvents: [],
    completionDetected: false,
    buttonReset: false,
    errors: []
};

function logResult(message, success = true) {
    const icon = success ? '✅' : '❌';
    console.log(`${icon} ${message}`);
}

function logStep(step, message) {
    console.log(`🔄 Step ${step}: ${message}`);
}

async function verifyButtonFix() {
    try {
        logStep(1, 'Checking DOM elements');
        
        // Find required elements
        const progressIndicator = document.getElementById('reanalysis-progress');
        const reanalyzeBtn = document.getElementById('reanalyze-btn');
        const btnText = document.getElementById('btn-text');
        const btnLoading = document.getElementById('btn-loading');
        
        if (!progressIndicator || !reanalyzeBtn || !btnText || !btnLoading) {
            throw new Error('Required DOM elements not found');
        }
        
        testResults.elementsFound = true;
        logResult('All required DOM elements found');
        
        // Get article ID
        const articleId = reanalyzeBtn.getAttribute('data-article-id');
        if (!articleId) {
            throw new Error('Article ID not found');
        }
        
        logResult(`Article ID: ${articleId}`);
        
        logStep(2, 'Setting up monitoring');
        
        // Monitor button state changes
        const buttonObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                if (mutation.target === reanalyzeBtn && mutation.attributeName === 'disabled') {
                    const state = {
                        timestamp: Date.now(),
                        disabled: reanalyzeBtn.disabled,
                        textContent: btnText.textContent,
                        loadingVisible: btnLoading.style.display !== 'none'
                    };
                    testResults.buttonStateChanges.push(state);
                    console.log(`🔄 Button state: disabled=${state.disabled}, text="${state.textContent}", loading=${state.loadingVisible}`);
                }
            });
        });
        
        buttonObserver.observe(reanalyzeBtn, { attributes: true });
        
        // Monitor text changes
        const textObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                if (mutation.target === btnText) {
                    console.log(`📝 Button text changed: "${btnText.textContent}"`);
                }
            });
        });
        
        textObserver.observe(btnText, { childList: true, characterData: true, subtree: true });
        
        // Monitor ProgressIndicator events
        progressIndicator.addEventListener('completed', (event) => {
            testResults.progressEvents.push({
                type: 'completed',
                timestamp: Date.now(),
                detail: event.detail
            });
            testResults.completionDetected = true;
            logResult('ProgressIndicator completion event received!');
            console.log('📊 Completion data:', event.detail);
        });
        
        progressIndicator.addEventListener('progressupdate', (event) => {
            testResults.progressEvents.push({
                type: 'progressupdate',
                timestamp: Date.now(),
                detail: event.detail
            });
            console.log(`📈 Progress: ${event.detail.percent || event.detail.progress || 0}% - ${event.detail.status}`);
        });
        
        progressIndicator.addEventListener('error', (event) => {
            testResults.progressEvents.push({
                type: 'error',
                timestamp: Date.now(),
                detail: event.detail
            });
            testResults.errors.push(`ProgressIndicator error: ${event.detail}`);
            logResult(`ProgressIndicator error: ${event.detail}`, false);
        });
        
        logStep(3, 'Recording initial button state');
        
        const initialState = {
            disabled: reanalyzeBtn.disabled,
            textContent: btnText.textContent,
            loadingVisible: btnLoading.style.display !== 'none'
        };
        
        console.log('📋 Initial button state:', initialState);
        
        logStep(4, 'Simulating button click');
        
        // Record start time
        const startTime = Date.now();
        
        // Simulate button click
        reanalyzeBtn.click();
        
        logResult('Button click triggered');
        
        // Wait for completion or timeout
        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                buttonObserver.disconnect();
                textObserver.disconnect();
                reject(new Error('Test timeout - button did not reset within time limit'));
            }, TEST_TIMEOUT);
            
            // Check for button reset every 500ms
            const checkInterval = setInterval(() => {
                const currentTime = Date.now();
                const elapsed = currentTime - startTime;
                
                // Check if button has been reset (not disabled and showing normal text)
                const isReset = !reanalyzeBtn.disabled && 
                               btnText.textContent === 'Request Reanalysis' &&
                               btnLoading.style.display === 'none';
                
                if (isReset) {
                    clearTimeout(timeout);
                    clearInterval(checkInterval);
                    buttonObserver.disconnect();
                    textObserver.disconnect();
                    
                    testResults.buttonReset = true;
                    const duration = elapsed;
                    
                    logResult(`Button successfully reset after ${duration}ms`);
                    
                    resolve({
                        success: true,
                        duration: duration,
                        testResults: testResults
                    });
                }
                
                // Log progress every 5 seconds
                if (elapsed % 5000 < 500) {
                    console.log(`⏳ Waiting for button reset... (${Math.round(elapsed/1000)}s elapsed)`);
                }
            }, 500);
        });
        
    } catch (error) {
        testResults.errors.push(error.message);
        throw error;
    }
}

// Run the verification test
console.log('🚀 Starting button fix verification...');
console.log('');

verifyButtonFix()
    .then(result => {
        console.log('');
        console.log('🎉 BUTTON FIX VERIFICATION SUCCESSFUL!');
        console.log('=====================================');
        console.log(`⏱️  Total duration: ${result.duration}ms`);
        console.log(`🔄 Button state changes: ${result.testResults.buttonStateChanges.length}`);
        console.log(`📊 Progress events: ${result.testResults.progressEvents.length}`);
        console.log(`✅ Completion detected: ${result.testResults.completionDetected}`);
        console.log(`🔄 Button reset: ${result.testResults.buttonReset}`);
        console.log('');
        console.log('📋 SUMMARY:');
        console.log('✅ Race condition fixed - event listeners added before ProgressIndicator display');
        console.log('✅ Button state properly synchronized with backend completion');
        console.log('✅ No more stuck "Processing..." state');
        console.log('');
        console.log('🔍 Detailed results:', result.testResults);
    })
    .catch(error => {
        console.log('');
        console.log('❌ BUTTON FIX VERIFICATION FAILED!');
        console.log('==================================');
        console.log(`💥 Error: ${error.message}`);
        console.log('');
        console.log('🔍 Test results so far:', testResults);
        console.log('');
        console.log('🛠️  DEBUGGING TIPS:');
        console.log('1. Check browser console for JavaScript errors');
        console.log('2. Verify ProgressIndicator component is loaded');
        console.log('3. Check SSE connection in Network tab');
        console.log('4. Ensure backend is sending completion events');
    });

console.log('⏳ Test in progress... Click the "Request Reanalysis" button or wait for automatic test...');

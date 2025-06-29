// Test script to verify button responsiveness and DOM elements
// Run this in browser console to debug the unresponsive button issue

console.log('🔍 BUTTON RESPONSIVENESS DEBUG');
console.log('==============================');

function checkDOMElements() {
    console.log('📋 Step 1: Checking DOM Elements');
    
    const elements = {
        progressIndicator: document.getElementById('reanalysis-progress'),
        reanalyzeBtn: document.getElementById('reanalyze-btn'),
        btnText: document.getElementById('btn-text'),
        btnLoading: document.getElementById('btn-loading')
    };
    
    console.log('🔍 Element check results:');
    Object.entries(elements).forEach(([name, element]) => {
        const exists = element !== null;
        const icon = exists ? '✅' : '❌';
        console.log(`   ${icon} ${name}: ${exists ? 'Found' : 'NOT FOUND'}`);
        
        if (exists && name === 'reanalyzeBtn') {
            console.log(`      - Article ID: ${element.getAttribute('data-article-id')}`);
            console.log(`      - Disabled: ${element.disabled}`);
            console.log(`      - Display: ${element.style.display || 'default'}`);
        }
        
        if (exists && name === 'btnText') {
            console.log(`      - Text: "${element.textContent}"`);
            console.log(`      - Display: ${element.style.display || 'default'}`);
        }
        
        if (exists && name === 'btnLoading') {
            console.log(`      - Text: "${element.textContent}"`);
            console.log(`      - Display: ${element.style.display || 'default'}`);
        }
    });
    
    return elements;
}

function checkEventListeners(button) {
    console.log('📋 Step 2: Checking Event Listeners');
    
    if (!button) {
        console.log('❌ Button not found, cannot check event listeners');
        return false;
    }
    
    // Check if button has click event listeners
    const listeners = getEventListeners ? getEventListeners(button) : null;
    
    if (listeners && listeners.click) {
        console.log(`✅ Button has ${listeners.click.length} click event listener(s)`);
        return true;
    } else {
        console.log('❌ No click event listeners found (or getEventListeners not available)');
        console.log('💡 This might indicate the event listener was not properly attached');
        return false;
    }
}

function testButtonClick(button) {
    console.log('📋 Step 3: Testing Button Click');
    
    if (!button) {
        console.log('❌ Button not found, cannot test click');
        return;
    }
    
    console.log('🖱️  Simulating button click...');
    
    // Add a temporary click listener to see if clicks are being detected
    let clickDetected = false;
    const testListener = () => {
        clickDetected = true;
        console.log('✅ Click event detected!');
    };
    
    button.addEventListener('click', testListener);
    
    // Simulate click
    button.click();
    
    // Remove test listener
    button.removeEventListener('click', testListener);
    
    if (clickDetected) {
        console.log('✅ Button is responsive to clicks');
    } else {
        console.log('❌ Button click not detected');
    }
    
    return clickDetected;
}

function checkJavaScriptErrors() {
    console.log('📋 Step 4: Checking for JavaScript Errors');
    
    // Override console.error temporarily to catch errors
    const originalError = console.error;
    const errors = [];
    
    console.error = function(...args) {
        errors.push(args.join(' '));
        originalError.apply(console, args);
    };
    
    // Restore after a short delay
    setTimeout(() => {
        console.error = originalError;
        
        if (errors.length > 0) {
            console.log('❌ JavaScript errors detected:');
            errors.forEach((error, index) => {
                console.log(`   ${index + 1}. ${error}`);
            });
        } else {
            console.log('✅ No JavaScript errors detected');
        }
    }, 1000);
}

function testProgressIndicator(progressIndicator) {
    console.log('📋 Step 5: Testing ProgressIndicator');
    
    if (!progressIndicator) {
        console.log('❌ ProgressIndicator not found');
        return;
    }
    
    console.log('🔍 ProgressIndicator properties:');
    console.log(`   - Article ID: ${progressIndicator.getAttribute('article-id')}`);
    console.log(`   - Auto-connect: ${progressIndicator.getAttribute('auto-connect')}`);
    console.log(`   - Show details: ${progressIndicator.getAttribute('show-details')}`);
    console.log(`   - Display: ${progressIndicator.style.display || 'default'}`);
    
    // Test if ProgressIndicator has required methods
    const methods = ['connect', 'disconnect', 'reset', 'updateProgress'];
    methods.forEach(method => {
        const hasMethod = typeof progressIndicator[method] === 'function';
        const icon = hasMethod ? '✅' : '❌';
        console.log(`   ${icon} Method ${method}: ${hasMethod ? 'Available' : 'Missing'}`);
    });
}

async function testBackendAPI() {
    console.log('📋 Step 6: Testing Backend API');
    
    const elements = checkDOMElements();
    const articleId = elements.reanalyzeBtn ? elements.reanalyzeBtn.getAttribute('data-article-id') : '584';
    
    try {
        console.log(`🌐 Testing POST /api/llm/reanalyze/${articleId}`);
        
        const response = await fetch(`/api/llm/reanalyze/${articleId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (response.ok) {
            const data = await response.json();
            console.log('✅ Backend API working correctly');
            console.log('📊 Response:', data);
        } else {
            console.log(`❌ Backend API error: ${response.status} ${response.statusText}`);
        }
    } catch (error) {
        console.log(`❌ Backend API request failed: ${error.message}`);
    }
}

// Run all tests
async function runAllTests() {
    console.log('🚀 Starting comprehensive button debug...');
    console.log('');
    
    const elements = checkDOMElements();
    console.log('');
    
    checkEventListeners(elements.reanalyzeBtn);
    console.log('');
    
    testButtonClick(elements.reanalyzeBtn);
    console.log('');
    
    checkJavaScriptErrors();
    console.log('');
    
    testProgressIndicator(elements.progressIndicator);
    console.log('');
    
    await testBackendAPI();
    console.log('');
    
    console.log('🏁 Debug complete! Check results above for issues.');
}

// Start the tests
runAllTests();

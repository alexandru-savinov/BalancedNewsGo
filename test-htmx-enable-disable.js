// Test the HTMX endpoint directly to see if enable/disable works
const http = require('http');
const querystring = require('querystring');

function makeHTMXRequest(method, path, data, contentType = 'application/x-www-form-urlencoded') {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: path,
            method: method,
            headers: {
                'Content-Type': contentType,
                'HX-Request': 'true'
            }
        };

        const req = http.request(options, (res) => {
            let body = '';
            res.on('data', (chunk) => {
                body += chunk;
            });
            res.on('end', () => {
                resolve({
                    statusCode: res.statusCode,
                    headers: res.headers,
                    body: body
                });
            });
        });

        req.on('error', (err) => {
            reject(err);
        });

        if (data) {
            if (contentType === 'application/x-www-form-urlencoded') {
                req.write(querystring.stringify(data));
            } else {
                req.write(JSON.stringify(data));
            }
        }
        req.end();
    });
}

async function testHTMXEnableDisable() {
    console.log('Testing HTMX enable/disable functionality...');
    
    try {
        // Test 1: Try to disable BBC News using form data
        console.log('\n1. Disabling BBC News using form data...');
        const disableResponse1 = await makeHTMXRequest('PUT', '/htmx/sources/2', {
            enabled: 'false'
        });
        console.log(`Status: ${disableResponse1.statusCode}`);
        console.log(`Content-Type: ${disableResponse1.headers['content-type']}`);
        console.log(`Body length: ${disableResponse1.body.length}`);
        if (disableResponse1.statusCode !== 200) {
            console.log(`Error body: ${disableResponse1.body.substring(0, 200)}...`);
        }
        
        // Test 2: Try to enable BBC News using form data
        console.log('\n2. Enabling BBC News using form data...');
        const enableResponse1 = await makeHTMXRequest('PUT', '/htmx/sources/2', {
            enabled: 'true'
        });
        console.log(`Status: ${enableResponse1.statusCode}`);
        console.log(`Content-Type: ${enableResponse1.headers['content-type']}`);
        console.log(`Body length: ${enableResponse1.body.length}`);
        if (enableResponse1.statusCode !== 200) {
            console.log(`Error body: ${enableResponse1.body.substring(0, 200)}...`);
        }
        
        // Test 3: Try to disable BBC News using JSON data (like hx-vals)
        console.log('\n3. Disabling BBC News using JSON data...');
        const disableResponse2 = await makeHTMXRequest('PUT', '/htmx/sources/2', {
            enabled: false
        }, 'application/json');
        console.log(`Status: ${disableResponse2.statusCode}`);
        console.log(`Content-Type: ${disableResponse2.headers['content-type']}`);
        console.log(`Body length: ${disableResponse2.body.length}`);
        if (disableResponse2.statusCode !== 200) {
            console.log(`Error body: ${disableResponse2.body.substring(0, 200)}...`);
        }
        
    } catch (error) {
        console.error('Error:', error.message);
    }
}

testHTMXEnableDisable();

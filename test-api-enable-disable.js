// Test the API endpoint directly to see if enable/disable works
const http = require('http');

function makeRequest(method, path, data) {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: path,
            method: method,
            headers: {
                'Content-Type': 'application/json',
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
                    body: body
                });
            });
        });

        req.on('error', (err) => {
            reject(err);
        });

        if (data) {
            req.write(JSON.stringify(data));
        }
        req.end();
    });
}

async function testEnableDisable() {
    console.log('Testing enable/disable functionality...');
    
    try {
        // First, get current state of BBC News (ID 2)
        console.log('\n1. Getting current state of BBC News...');
        const getResponse = await makeRequest('GET', '/api/sources/2');
        console.log(`Status: ${getResponse.statusCode}`);
        const currentSource = JSON.parse(getResponse.body);
        console.log(`Current enabled state: ${currentSource.data.enabled}`);
        
        // Disable the source
        console.log('\n2. Disabling BBC News...');
        const disableResponse = await makeRequest('PUT', '/api/sources/2', {
            enabled: false
        });
        console.log(`Status: ${disableResponse.statusCode}`);
        if (disableResponse.statusCode === 200) {
            const disabledSource = JSON.parse(disableResponse.body);
            console.log(`New enabled state: ${disabledSource.data.enabled}`);
        } else {
            console.log(`Error: ${disableResponse.body}`);
        }
        
        // Enable the source
        console.log('\n3. Enabling BBC News...');
        const enableResponse = await makeRequest('PUT', '/api/sources/2', {
            enabled: true
        });
        console.log(`Status: ${enableResponse.statusCode}`);
        if (enableResponse.statusCode === 200) {
            const enabledSource = JSON.parse(enableResponse.body);
            console.log(`Final enabled state: ${enabledSource.data.enabled}`);
        } else {
            console.log(`Error: ${enableResponse.body}`);
        }
        
    } catch (error) {
        console.error('Error:', error.message);
    }
}

testEnableDisable();

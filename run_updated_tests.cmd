@echo off
echo Running Updated NewsBalancer Integration Tests...

rem Set the base URL for the API server
set BASE_URL=http://localhost:8080

rem Check if server is running
curl -s %BASE_URL%/api/feeds/healthz > nul
if %errorlevel% neq 0 (
    echo Error: API server does not appear to be running at %BASE_URL%
    echo Please start the server before running tests
    exit /b 1
)

echo API server is running at %BASE_URL%

rem Install newman if not already installed
where newman > nul 2>&1
if %errorlevel% neq 0 (
    echo Newman not found. Installing Newman (Postman CLI)...
    npm install -g newman
    if %errorlevel% neq 0 (
        echo Failed to install Newman. Please install manually: npm install -g newman
        exit /b 1
    )
)

echo Running tests with newman...
newman run postman/updated_backend_tests.json --env-var "baseUrl=%BASE_URL%" --reporters cli,json --reporter-json-export test-results/updated-backend-results.json

if %errorlevel% neq 0 (
    echo One or more tests failed. Check the test results for details.
    exit /b %errorlevel%
) else (
    echo All tests completed successfully!
)
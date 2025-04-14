@echo off
REM Create test-results directory if it doesn't exist
if not exist test-results mkdir test-results

REM Start the server in the background
start /B go run cmd/server/main.go

REM Wait for the server to start
echo Waiting for server to start...
timeout /t 5

REM Run the tests
echo Running tests...
npx newman run memory-bank/essential_rescoring_tests.json --reporters cli,json --reporter-json-export test-results/essential_rescoring_tests.json

REM Capture the exit code
set TEST_EXIT_CODE=%ERRORLEVEL%

REM Display the results
echo Test execution completed with exit code %TEST_EXIT_CODE%. Results saved to test-results/essential_rescoring_tests.json

REM Exit with the test exit code
exit /b %TEST_EXIT_CODE%
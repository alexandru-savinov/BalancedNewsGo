@echo off
if not exist test-results mkdir test-results

echo Starting the server...
start /b cmd /c "go run cmd/server/main.go"

echo Waiting for the server to start...
timeout /t 5 /nobreak > nul

echo Running Newman tests...
npx newman run postman/backend_fixes_tests_updated.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/backend_fixes_tests.json

echo Stopping the server...
taskkill /f /im go.exe > nul 2>&1

echo Tests completed.
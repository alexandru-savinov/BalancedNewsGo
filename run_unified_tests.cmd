@echo off
if not exist test-results mkdir test-results

echo Starting the server...
start /b cmd /c "go run cmd/server/main.go"

echo Waiting for the server to start...
timeout /t 5 /nobreak > nul

echo Running Newman tests...
npx newman run postman/unified_backend_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/unified_backend_tests.json

echo Stopping the server...
taskkill /f /im go.exe > nul 2>&1

echo Tests completed.
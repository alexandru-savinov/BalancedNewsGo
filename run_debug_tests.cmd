@echo off
if not exist test-results mkdir test-results

echo Starting the server...
start /b cmd /c "go run cmd/server/main.go"

echo Waiting for the server to start...
timeout /t 5 /nobreak > nul

echo Running Debug Tests...
npx newman run postman/debug_collection.json -e postman/debug_environment.json --reporters cli,json --reporter-json-export test-results/debug_tests.json

echo Stopping the server...
taskkill /f /im go.exe > nul 2>&1

echo Debug tests completed.
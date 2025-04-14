@echo off
REM Create test-results directory if it doesn't exist
if not exist test-results mkdir test-results

echo Running all Postman tests...

echo 1. Running Backend Fixes Tests...
call run_backend_tests.cmd

echo 2. Running Rescoring Tests...
start /b cmd /c "go run cmd/server/main.go"
timeout /t 5 /nobreak > nul
npx newman run memory-bank/postman_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/postman_rescoring_collection.json
taskkill /f /im go.exe > nul 2>&1

echo 3. Running Essential Rescoring Tests...
start /b cmd /c "go run cmd/server/main.go"
timeout /t 5 /nobreak > nul
npx newman run memory-bank/essential_rescoring_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/essential_rescoring_tests.json
taskkill /f /im go.exe > nul 2>&1

echo 4. Running Extended Rescoring Tests...
start /b cmd /c "go run cmd/server/main.go"
timeout /t 5 /nobreak > nul
npx newman run memory-bank/extended_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/extended_rescoring_collection.json
taskkill /f /im go.exe > nul 2>&1

echo All tests completed.
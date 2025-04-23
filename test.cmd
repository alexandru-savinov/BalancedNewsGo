@echo off
setlocal enabledelayedexpansion

REM Create test-results directory if it doesn't exist
if not exist test-results mkdir test-results

REM Parse command line arguments
set "command=%~1"
if "%command%"=="" set "command=help"

if "%command%"=="help" (
    echo.
    echo NewBalancer Go Testing CLI
    echo ========================
    echo.
    echo Available commands:
    echo.
    echo   test backend        - Run backend fixes tests
    echo   test all            - Run all tests
    echo   test debug          - Run debug tests
    echo   test report         - Generate HTML test report
    echo   test analyze        - Analyze test results
    echo   test list           - List test result files
    echo   test clean          - Clean test results
    echo   test help           - Show this help message
    echo.
    goto :end
)

if "%command%"=="backend" (
    echo Running backend fixes tests...
    call :run_backend_tests
    goto :end
)

if "%command%"=="all" (
    echo Running all tests...
    call :run_all_tests
    goto :end
)

if "%command%"=="debug" (
    echo Running debug tests...
    call :run_debug_tests
    goto :end
)

if "%command%"=="report" (
    echo Generating test report...
    call :generate_report
    goto :end
)

if "%command%"=="clean" (
    echo Cleaning test results...
    call :clean_results
    goto :end
)

if "%command%"=="analyze" (
    echo Analyzing test results...
    node analyze_test_results.js analyze-all
    goto :end
)

if "%command%"=="list" (
    echo Listing test result files...
    node analyze_test_results.js list
    goto :end
)

echo Unknown command: %command%
echo Run 'test help' for usage information.
goto :end

:run_backend_tests
    echo Starting the server...
    start /b cmd /c "go run cmd/server/main.go" > test-results/server.log 2>&1
    
    echo Waiting for the server to start...
    timeout /t 5 /nobreak > nul
    
    echo Running backend fixes tests...
    npx newman run postman/backend_fixes_tests_updated.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/backend_fixes_tests.json
    
    REM === Run Node.js SSE backend progress validation ===
    REM Automatically extract articleId from test results
    node test_sse_progress.js
    REM Optionally, parse output or fail the script if SSE status is not Success
    
    echo Stopping the server...
    taskkill /f /im go.exe > nul 2>&1
    
    echo Backend tests completed.
    exit /b 0

:run_all_tests
    echo Running backend fixes tests...
    call :run_backend_tests
    
    echo Running rescoring tests...
    start /b cmd /c "go run cmd/server/main.go" > test-results/server_rescoring.log 2>&1
    timeout /t 5 /nobreak > nul
    npx newman run memory-bank/postman_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/postman_rescoring_collection.json
    taskkill /f /im go.exe > nul 2>&1
    
    echo Running essential rescoring tests...
    start /b cmd /c "go run cmd/server/main.go" > test-results/server_essential.log 2>&1
    timeout /t 5 /nobreak > nul
    npx newman run memory-bank/essential_rescoring_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/essential_rescoring_tests.json
    taskkill /f /im go.exe > nul 2>&1
    
    echo Running extended rescoring tests...
    start /b cmd /c "go run cmd/server/main.go" > test-results/server_extended.log 2>&1
    timeout /t 5 /nobreak > nul
    npx newman run memory-bank/extended_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/extended_rescoring_collection.json
    taskkill /f /im go.exe > nul 2>&1
    
    echo All tests completed.
    exit /b 0

:run_debug_tests
    echo Starting the server...
    start /b cmd /c "go run cmd/server/main.go" > test-results/server_debug.log 2>&1
    
    echo Waiting for the server to start...
    timeout /t 5 /nobreak > nul
    
    echo Running debug tests...
    npx newman run postman/debug_collection.json -e postman/debug_environment.json --reporters cli,json --reporter-json-export test-results/debug_tests.json
    
    echo Stopping the server...
    taskkill /f /im go.exe > nul 2>&1
    
    echo Debug tests completed.
    exit /b 0

:generate_report
    echo Generating HTML test report...
    node generate_test_report.js
    echo Report generated at test-results/test_report.html
    exit /b 0

:clean_results
    echo Cleaning test results...
    del /q test-results\*.json 2>nul
    del /q test-results\*.html 2>nul
    del /q test-results\*.log 2>nul
    echo Test results cleaned.
    exit /b 0

:end
endlocal
@echo off
setlocal enabledelayedexpansion

REM Create test-results directory if it doesn't exist
set RESULTS_DIR=test-results
if not exist %RESULTS_DIR% mkdir %RESULTS_DIR%

REM Parse command line arguments
set "command=%~1"
if "%command%"=="" set "command=help"

REM --- Command Dispatch ---
if /I "%command%" == "help" ( call :run_help & goto :eof )
if /I "%command%" == "backend" ( call :run_backend & goto :eof )
if /I "%command%" == "api" ( call :run_api & goto :eof )
if /I "%command%" == "essential" ( call :run_essential & goto :eof )
if /I "%command%" == "debug" ( call :run_debug & goto :eof )
if /I "%command%" == "all" ( call :run_all & goto :eof )
if /I "%command%" == "confidence" ( call :run_confidence & goto :eof )
if /I "%command%" == "report" ( call :generate_report & goto :eof )
if /I "%command%" == "clean" ( call :clean_results & goto :eof )
if /I "%command%" == "analyze" ( call :analyze_results & goto :eof )
if /I "%command%" == "list" ( call :list_results & goto :eof )

REM --- Unknown Command Handling ---
echo Unknown command: %command%
echo Run 'test help' for usage information.
goto :end_fail


REM --- Subroutines ---

:run_help
    echo.
    echo NewBalancer Go Testing CLI
    echo ========================
    echo.
    echo Usage: test.cmd [command]
    echo.
    echo Available commands:
    echo.
    echo   backend        - Run backend fixes/integration tests (Newman: backend_fixes_tests_updated.json)
    echo   api            - Run basic API tests (Newman: newsbalancer_api_tests.json)
    echo   essential      - Run essential rescoring tests (Newman: essential_rescoring_tests.json)
    echo   debug          - Run debug tests (Newman: debug_collection.json)
    echo   all            - Run essential, extended, and confidence tests (Multiple Newman collections)
    echo   confidence     - Run confidence validation tests (Newman: confidence_validation_tests.json) [If available]
    echo.
    echo   report         - Generate HTML test report from results
    echo   analyze        - Analyze test results via CLI
    echo   list           - List existing test result files
    echo   clean          - Clean (delete) test result files (*.json, *.html, *.log)
    echo   help           - Show this help message
    echo.
    echo Notes:
    echo   - Test commands (backend, api, essential, debug, all, confidence) typically start/stop the Go server.
    echo   - Ensure 'go run cmd/server/main.go' works before running tests.
    echo   - Requires Node.js and Newman ('npm install -g newman').
    echo.
goto :eof

:run_backend
    echo Running backend fixes tests...
    call :run_newman_test "backend_fixes" "postman/unified_backend_tests.json" "postman/local_environment.json"
    REM === Add Node.js SSE backend progress validation if still needed ===
    REM node scripts/test_sse_progress.js
goto :eof

:run_api
    echo Running basic API tests...
    call :run_newman_test "api_tests" "postman/unified_backend_tests.json" "postman/local_environment.json"
goto :eof

:run_essential
    echo Running essential rescoring tests (using unified_backend_tests.json)...
    call :run_newman_test "essential_tests" "postman/unified_backend_tests.json" ""
goto :eof

:run_debug
    echo Running debug tests...
    call :run_newman_test "debug_tests" "postman/debug_collection.json" "postman/debug_environment.json"
goto :eof

:run_all
    echo Running all test suites...
    for /f "tokens=2-4 delims=/ " %%a in ('date /t') do (set DATE=%%c%%a%%b)
    for /f "tokens=1-2 delims=: " %%a in ('time /t') do (set TIME=%%a%%b)
    set TIMESTAMP=%DATE%%TIME%
    set ALL_LOG_FILE=%RESULTS_DIR%\all_tests_run_%TIMESTAMP%.log

    echo ====== NewsBalancer Full API Test Run - %date% %time% ====== > %ALL_LOG_FILE%
    echo. >> %ALL_LOG_FILE%

    echo ===== Running Essential Tests (using unified_backend_tests.json) ===== >> %ALL_LOG_FILE%
    echo Running Newman: unified_backend_tests.json (as Essential)
    call npx newman run postman/unified_backend_tests.json ^
      --reporters cli,json ^
      --reporter-json-export=%RESULTS_DIR%/essential_results_%TIMESTAMP%.json >> %ALL_LOG_FILE%
    if errorlevel 1 ( echo Essential tests FAILED. Check %ALL_LOG_FILE% & goto :end_fail )

    echo. >> %ALL_LOG_FILE%
    echo ===== Running Extended Tests ===== >> %ALL_LOG_FILE%
    echo Running Newman: extended_rescoring_collection.json (path updated)
    call npx newman run postman/extended_rescoring_collection.json ^
      --reporters cli,json ^
      --reporter-json-export=%RESULTS_DIR%/extended_results_%TIMESTAMP%.json >> %ALL_LOG_FILE%
     if errorlevel 1 ( echo Extended tests FAILED. Check %ALL_LOG_FILE% & goto :end_fail )

    if exist postman\confidence_validation_tests.json (
      echo. >> %ALL_LOG_FILE%
      echo ===== Running Confidence Validation Tests ===== >> %ALL_LOG_FILE%
      echo Running Newman: confidence_validation_tests.json (path updated)
      call npx newman run postman/confidence_validation_tests.json ^
        --reporters cli,json ^
        --reporter-json-export=%RESULTS_DIR%/confidence_results_%TIMESTAMP%.json >> %ALL_LOG_FILE%
      if errorlevel 1 ( echo Confidence tests FAILED. Check %ALL_LOG_FILE% & goto :end_fail )
    )

    echo. >> %ALL_LOG_FILE%
    echo ===== Generating Test Report ===== >> %ALL_LOG_FILE%
    call :generate_report >> %ALL_LOG_FILE%
    if errorlevel 1 ( echo Report generation FAILED. Check %ALL_LOG_FILE% & goto :end_fail )

    echo. >> %ALL_LOG_FILE%
    echo ===== All tests completed SUCCESSFULLY ===== >> %ALL_LOG_FILE%
    echo Test log saved to: %ALL_LOG_FILE%
    type %ALL_LOG_FILE%
goto :eof

:run_confidence
    if exist postman\confidence_validation_tests.json (
        echo Running confidence validation tests...
        call :run_newman_test "confidence_tests" "postman/confidence_validation_tests.json" ""
    ) else (
        echo Confidence test collection not found: postman\confidence_validation_tests.json
    )
goto :eof

:analyze_results
    echo Analyzing test results...
    node scripts/analyze_test_results.js analyze-all
goto :eof

:list_results
    echo Listing test result files...
    node scripts/analyze_test_results.js list
goto :eof


REM --- Helper Function: Run Newman Test ---
REM %1: Test name (for logging/filenames)
REM %2: Newman collection file path
REM %3: Newman environment file path (optional, use "" if none)
:run_newman_test
    set TEST_NAME=%~1
    set COLLECTION_FILE=%~2
    set ENV_FILE=%~3
    set SERVER_LOG=%RESULTS_DIR%\server_%TEST_NAME%.log
    set RESULT_FILE=%RESULTS_DIR%\%TEST_NAME%_results.json
    set GO_SERVER_PID=

    echo Setting NO_AUTO_ANALYZE=true for test run
    set NO_AUTO_ANALYZE=true

    echo Starting the server for %TEST_NAME% tests (logging to %SERVER_LOG%)...
    REM Start go run in a visible window
    start "GoServer_%TEST_NAME%" cmd /c "go run cmd/server/main.go > %SERVER_LOG% 2>&1"
    REM Give it a moment to potentially fail or for the process to register
    echo Waiting 5 seconds for server to initialize and process to be discoverable...
    timeout /t 5 /nobreak > nul
    REM Rudimentary check if go.exe is running (might catch other go processes)
    tasklist /FI "IMAGENAME eq go.exe" | find /I "go.exe" > nul
    if errorlevel 1 (
        echo ERROR: Failed to verify go.exe running after attempt to start server. Check %SERVER_LOG%.
        goto :stop_server_and_fail
    )
    echo Server process likely started.

    echo Waiting for the server to fully initialize...
    timeout /t 5 /nobreak > nul

    echo Running Newman test: %COLLECTION_FILE%
    set NEWMAN_CMD=npx newman run "%COLLECTION_FILE%"
    if not "%ENV_FILE%"=="" ( set NEWMAN_CMD=!NEWMAN_CMD! -e "%ENV_FILE%" )
    set NEWMAN_CMD=!NEWMAN_CMD! --reporters cli,json --reporter-json-export "%RESULT_FILE%"

    echo Executing: !NEWMAN_CMD!
    call !NEWMAN_CMD!
    set NEWMAN_EXIT_CODE=%errorlevel%

    echo Clearing NO_AUTO_ANALYZE
    set NO_AUTO_ANALYZE=

:stop_server_and_continue
    echo Stopping the Go server process(es)...
    taskkill /F /FI "WINDOWTITLE eq GoServer_%TEST_NAME%*" /T > nul 2>&1
    taskkill /F /IM go.exe /T > nul 2>&1
    REM Optional: Kill compiled binary if used
    taskkill /f /im newbalancer_server.exe > nul 2>&1
    echo Server stop commands issued.

    if %NEWMAN_EXIT_CODE% neq 0 (
        echo %TEST_NAME% tests FAILED (Newman Exit Code: %NEWMAN_EXIT_CODE%). Check output above and %RESULT_FILE%.
        exit /b %NEWMAN_EXIT_CODE%
    )

    echo %TEST_NAME% tests completed successfully. Results: %RESULT_FILE%
    exit /b 0

:stop_server_and_fail
    echo Stopping potentially running Go server process(es)...
    taskkill /F /FI "WINDOWTITLE eq GoServer_%TEST_NAME%*" /T > nul 2>&1
    taskkill /F /IM go.exe /T > nul 2>&1
    taskkill /f /im newbalancer_server.exe > nul 2>&1
    echo Server stop commands issued after failure.
    echo Clearing NO_AUTO_ANALYZE (on failure path)
    set NO_AUTO_ANALYZE=
    exit /b 1


REM --- Helper Function: Generate Report ---
:generate_report
    echo Generating HTML test report...
    node scripts/generate_test_report.js
    if errorlevel 1 (
        echo Failed to generate report.
        exit /b 1
    )
    echo Report generated at %RESULTS_DIR%/test_report.html
goto :eof

REM --- Helper Function: Clean Results ---
:clean_results
    echo Cleaning test results...
    del /q "%RESULTS_DIR%\*.json" 2>nul
    del /q "%RESULTS_DIR%\*.html" 2>nul
    del /q "%RESULTS_DIR%\*.log" 2>nul
    echo Test results cleaned.
goto :eof

:end_fail
echo Command FAILED.
endlocal
exit /b 1

:end
endlocal
exit /b 0
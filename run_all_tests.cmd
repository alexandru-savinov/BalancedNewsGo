@echo off
setlocal enabledelayedexpansion

REM Set up output directory
set RESULTS_DIR=.\test-results
if not exist %RESULTS_DIR% mkdir %RESULTS_DIR%
for /f "tokens=2-4 delims=/ " %%a in ('date /t') do (set DATE=%%c%%a%%b)
for /f "tokens=1-2 delims=: " %%a in ('time /t') do (set TIME=%%a%%b)
set TIMESTAMP=%DATE%%TIME%
set LOG_FILE=%RESULTS_DIR%\test_run_%TIMESTAMP%.log

echo ====== NewsBalancer API Test Run - %date% %time% ====== > %LOG_FILE%
echo Running with NODE_ENV=%NODE_ENV% >> %LOG_FILE%

REM Check for .env file
if exist .env (
  echo Using .env configuration file >> %LOG_FILE%
) else (
  echo Warning: No .env file found - using default environment variables >> %LOG_FILE%
)

REM Run essential tests first
echo. >> %LOG_FILE%
echo ===== Running Essential Tests ===== >> %LOG_FILE%
call npx newman run postman/backup/essential_rescoring_tests.json ^
  --reporters cli,json ^
  --reporter-json-export=%RESULTS_DIR%/essential_results.json >> %LOG_FILE%

REM Run extended tests 
echo. >> %LOG_FILE%
echo ===== Running Extended Tests ===== >> %LOG_FILE%
call npx newman run postman/backup/extended_rescoring_collection.json ^
  --reporters cli,json ^
  --reporter-json-export=%RESULTS_DIR%/extended_results.json >> %LOG_FILE%

REM Run confidence validation tests if they exist
if exist postman\backup\confidence_validation_tests.json (
  echo. >> %LOG_FILE%
  echo ===== Running Confidence Validation Tests ===== >> %LOG_FILE%
  call npx newman run postman/backup/confidence_validation_tests.json ^
    --reporters cli,json ^
    --reporter-json-export=%RESULTS_DIR%/confidence_results.json >> %LOG_FILE%
)

REM Generate test summary report
echo. >> %LOG_FILE%
echo ===== Generating Test Report ===== >> %LOG_FILE%
call node analyze_test_results.js >> %LOG_FILE%

echo. >> %LOG_FILE%
echo ===== All tests completed ===== >> %LOG_FILE%
echo Test log saved to: %LOG_FILE%

type %LOG_FILE%
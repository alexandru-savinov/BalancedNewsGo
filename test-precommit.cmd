@echo off
REM Test script to run pre-commit checks without actually committing

echo Testing pre-commit hook...
echo.

REM Run the Windows pre-commit script
call .git\hooks\pre-commit.cmd

if %ERRORLEVEL% equ 0 (
    echo.
    echo ✅ Pre-commit checks passed! Ready to commit.
) else (
    echo.
    echo ❌ Pre-commit checks failed! Fix issues before committing.
)

echo.
echo You can also run: make precommit-check
echo.

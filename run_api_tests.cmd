@echo off
REM Navigate to the project root
cd %~dp0

REM Set the environment to test mode
set GIN_MODE=test

REM Run the tests for critical API handlers
go test -v -coverprofile=api_coverage_critical.out ./internal/api -run "^Test(CreateArticleHandler|GetArticlesHandler|GetArticleByID|BiasHandler|SummaryHandler)"

REM Generate a coverage report if tests passed
if %ERRORLEVEL% EQU 0 (
  echo Tests passed! Generating coverage report...
  go tool cover -html=api_coverage_critical.out -o api_coverage_critical.html
  echo Coverage report saved to api_coverage_critical.html
) else (
  echo Tests failed!
)
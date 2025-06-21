# CI/CD Pipeline Setup and Configuration

## Overview

This document outlines the GitHub Actions CI/CD pipeline configuration, required secrets, environments, and troubleshooting steps.

## Pipeline Structure

The CI/CD pipeline consists of the following jobs:

1. **Unit Tests** - Runs Go unit tests with coverage reporting
2. **Code Quality & Linting** - Runs golangci-lint for code quality checks
3. **Security Scan** - Runs gosec for security vulnerability scanning
4. **API Integration Tests** - Tests API endpoints with PostgreSQL database
5. **Performance Benchmarks** - Runs performance benchmarks
6. **Build & Deploy (Staging)** - Builds and deploys to staging environment
7. **Build & Deploy (Production)** - Builds and deploys to production environment

## Required GitHub Repository Settings

### Environments

The following environments must be configured in GitHub repository settings:

1. **staging** - For staging deployments
2. **production** - For production deployments

To configure environments:
1. Go to repository Settings → Environments
2. Create "staging" and "production" environments
3. Configure environment protection rules as needed

### Secrets

The following secrets must be configured in GitHub repository settings:

#### Container Registry Secrets
- `CONTAINER_REGISTRY` - Container registry URL (e.g., ghcr.io/username)
- `REGISTRY_USERNAME` - Container registry username
- `REGISTRY_PASSWORD` - Container registry password/token

#### Database Secrets (for integration tests)
- `POSTGRES_PASSWORD` - PostgreSQL password for test database

#### API Keys (for LLM integration tests)
- `OPENROUTER_API_KEY` - OpenRouter API key for LLM testing
- `OPENAI_API_KEY` - OpenAI API key for LLM testing

To configure secrets:
1. Go to repository Settings → Secrets and variables → Actions
2. Add each required secret with appropriate values

## Fixed Issues

### 1. PostgreSQL Service Configuration
**Issue**: Environment variable syntax errors in PostgreSQL service configuration
**Fix**: Replaced `postgres:${{ env.POSTGRES_VERSION }}` with hardcoded `postgres:15`

### 2. Deprecated GitHub Actions
**Issue**: Using deprecated action versions
**Fixes**:
- Updated `actions/setup-go@v4` to `actions/setup-go@v5`
- Updated `actions/cache@v3` to `actions/cache@v4`
- Updated `actions/setup-node@v3` to `actions/setup-node@v4`
- Updated `actions/upload-artifact@v3` to `actions/upload-artifact@v4`
- Updated `codecov/codecov-action@v3` to `codecov/codecov-action@v4`
- Updated `github/codeql-action/upload-sarif@v2` to `github/codeql-action/upload-sarif@v3`
- Replaced `actions/create-release@v1` with `softprops/action-gh-release@v2`

### 3. Security Scanner Configuration
**Issue**: Incorrect gosec repository reference
**Fix**: Updated from `securecodewarrior/gosec` to `securego/gosec`

### 4. Database Migration Syntax
**Issue**: SQLite syntax in PostgreSQL migrations
**Fixes**:
- Replaced `INTEGER PRIMARY KEY AUTOINCREMENT` with `SERIAL PRIMARY KEY`
- Replaced `DATETIME` with `TIMESTAMP`
- Replaced `BOOLEAN DEFAULT 0` with `BOOLEAN DEFAULT FALSE`

### 5. Security Vulnerabilities
**Issues and Fixes**:
- **G404**: Replaced `math/rand` with `crypto/rand` for secure random number generation
- **G302**: Changed file permissions from `0666` to `0600` for log files
- **G104**: Added proper error handling for `w.Write()` calls

### 6. Import Path Issues
**Issue**: Incorrect module import paths in benchmark package
**Fix**: Updated import from `newsbalancer/internal/benchmark` to `github.com/alexandru-savinov/BalancedNewsGo/internal/benchmark`

## Workflow Validation

The workflow has been validated using:
- **actionlint** - GitHub Actions workflow linter
- **golangci-lint** - Go code quality and linting
- **gosec** - Go security scanner

All validation tools report no issues with the current configuration.

## Troubleshooting

### Common Issues

1. **Environment not found**: Ensure staging/production environments are created in repository settings
2. **Missing secrets**: Verify all required secrets are configured with correct values
3. **Database connection failures**: Check PostgreSQL service configuration and credentials
4. **Container registry authentication**: Verify registry credentials and permissions

### Monitoring Workflow Runs

1. Navigate to repository → Actions tab
2. Select the workflow run to view details
3. Click on failed jobs to view logs and error messages
4. Use GitHub CLI: `gh run list` and `gh run view <run-id>`

## Next Steps

1. Monitor the current workflow run for any remaining issues
2. Configure required secrets in repository settings
3. Set up staging and production environments
4. Test deployment to staging environment
5. Configure production deployment approval process

## Tools Installed

The following tools have been installed and configured:
- **actionlint** (v1.7.4) - GitHub Actions workflow validator
- **golangci-lint** (latest) - Go linting and code quality
- **gosec** (latest) - Go security scanner

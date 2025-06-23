# Pre-commit Setup - Identical to CI/CD Pipeline

This document describes the pre-commit setup that provides a tight feedback loop by running the same checks as the CI/CD pipeline locally before commits.

## Overview

The pre-commit hooks are designed to catch the same issues that would fail in CI/CD, preventing failed builds and providing immediate feedback to developers.

## What's Included

### Pre-commit Hooks
- **Location**: `.git/hooks/pre-commit` (bash) and `.git/hooks/pre-commit.cmd` (Windows)
- **Runs automatically**: Before every `git commit`

### Manual Testing
- **Make target**: `make precommit-check`
- **Test script**: `test-precommit.cmd` (Windows)

## Checks Performed (Matching CI/CD)

### 1. Code Formatting & Tidying
- `go mod tidy` - Ensures dependencies are clean
- `go fmt ./...` - Formats Go code

### 2. Linting & Static Analysis
- `golangci-lint` - Comprehensive linting with project configuration
- `go vet -composites=false ./...` - Go static analysis
- `staticcheck ./...` - Advanced static analysis (if installed)

### 3. Build Verification
- `go build ./cmd/server` - Ensures code compiles

### 4. Unit Tests
- `make unit ENABLE_RACE_DETECTION=false` - Runs all unit tests
- Uses same environment variables as CI: `NO_AUTO_ANALYZE=true`, `NO_DOCKER=true`

## Usage

### Automatic (Recommended)
Pre-commit hooks run automatically when you commit:
```bash
git add .
git commit -m "Your commit message"
# Pre-commit checks run automatically
```

### Manual Testing
Run checks manually before committing:
```bash
# Using make target
make precommit-check

# Using test script (Windows)
test-precommit.cmd

# Direct hook execution
.git/hooks/pre-commit.cmd
```

## Benefits

1. **Tight Feedback Loop**: Catch issues immediately, not after pushing
2. **CI/CD Alignment**: Same checks as CI pipeline
3. **Time Saving**: Fix issues locally instead of waiting for CI failure
4. **Quality Assurance**: Prevents broken commits from entering the repository

## Troubleshooting

### Missing staticcheck
If you see warnings about staticcheck not being found:
```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Hook Not Running
Ensure the hook is executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Bypassing Hooks (Emergency Only)
To skip pre-commit checks (not recommended):
```bash
git commit --no-verify -m "Emergency commit"
```

## Configuration

The pre-commit hooks use the same configuration as CI:
- **golangci-lint**: `.golangci.yml`
- **Go version**: Defined in `go.mod`
- **Test packages**: Defined in `Makefile`

## Integration with CI/CD

This setup ensures that:
- If pre-commit passes, CI should also pass
- Same linting rules and test coverage
- Consistent code quality standards
- Reduced CI/CD pipeline failures

## Next Steps

1. Run `make precommit-check` to verify setup
2. Make a test commit to ensure hooks work
3. Share this setup with team members
4. Consider adding integration tests to pre-commit for even tighter feedback

# Repository Contribution Guide

This file provides guidance for contributors and automated agents working in this repository.

## Commit Messages
- Follow the **Conventional Commits** format enforced by `commitlint`.
- Structure commits as `type(scope): subject` or `type: subject`.
- Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`.
- Use lower-case subject lines and keep the summary under 72 characters.
- Include a brief body explaining *why* when the change is not obvious.

## Code Style
- Format Go files with `gofmt -w` before committing.
- Run `go vet ./...` and `golangci-lint run ./...` (or `make lint`).
- For JavaScript/TypeScript files, run `npm run lint` if available.

## Testing
- Run the Go unit tests before submitting changes:
  ```bash
  NO_AUTO_ANALYZE=true go test ./...
  # or
  make unit
  ```
- Integration tests can be run via the helper script:
  ```bash
  NO_AUTO_ANALYZE=true bash scripts/test.sh all
  ```
- Node based tests can be executed via npm:
  ```bash
  npm test
  ```
  (this invokes `scripts/test.cmd` or `scripts/test.sh` depending on the platform).

## Pull Requests
- Summarize key changes and reference any relevant documentation updates.
- Include test results (or note if they were skipped due to environment limits).
- Keep PR titles concise and in the same style as commits.
- See `docs/local_ci_workflow.md` for more details on local testing.

# Repository Custom Instructions for GitHub Copilot

# This file is used to provide custom instructions to GitHub Copilot.
# It is not intended to be read by humans, but rather to guide the AI in generating code and comments.

- **Project context** – This is a *monorepo* whose primary deliverable is a **Go 1.24** backend API (Gin, sqlx, gofeed, robfig/cron) that aggregates news, runs multi‑perspective LLM analysis and serves JSON.  Supplementary tooling (tests, scripts) is written in **TypeScript (ES2020, CommonJS, strict mode)**.

- **Lint & style gates (Go)** – All new Go code **must pass** the rules in `.golangci.yml`:
  - cyclomatic‑complexity < 20 (`gocyclo`)
  - function length < 100 lines or 50 statements (`funlen`)
  - max line length ≤ 140 (`lll`)
  Run `go fmt` and `goimports`; prefer small helper functions over deep nesting.

- **Public‑facing HTTP handlers** live in `internal/api`.  When you generate or modify one:
  1. Accept `*gin.Context` as `c`.
  2. Validate inputs immediately; on failure call `api.RespondError(c, apperrors.New(...))`.
  3. On success call `api.RespondSuccess(c, data)`.
  4. Log timing with `api.LogPerformance()`.

- **Error handling** – Never return a raw `error` from exported functions.  Use the helpers in `internal/apperrors` (`New`, `Wrap`, `HandleError`, `Join`) so that callers can pattern‑match and API responses stay consistent.

- **Database access (sqlx)** – Queries belong in `internal/db`.  Use **named parameters** (`db.NamedExec` / `sqlx.NamedExecContext`) or positional `?` placeholders – *never* build SQL by string concatenation.  Wrap DB errors with `handleError`.  Prefer context‑aware helpers and honour cancellation.

- **Background jobs & concurrency** – Follow the patterns in `internal/rss` and `internal/llm`: spawn goroutines via `go func(ctx context.Context)`, respect `ctx.Done()`, and surface failures up the call stack.  Always log `[PERF]` blocks for tasks exceeding 100 ms.

- **Testing strategy** –
  - Unit tests live next to source (`*_test.go`), use table‑driven style, and must run with `go test ./...`.
  - End‑to‑end coverage is provided by **Playwright** (`playwright/e2e`, `playwright-report`).  New flows should extend that suite rather than introducing a new framework.

- **TypeScript utilities** (CLI helpers, test harnesses): honour `tsconfig.json` (`strict: true`, ES2020 target) and import style (`import fs from "fs/promises"`).  Avoid `any` and enable `skipLibCheck` where required.

- **Naming conventions** – Follow repository style:
  - Go identifiers: `camelCase` for local/private, `PascalCase` for exported; struct JSON tags use `snake_case`.
  - TypeScript variables and functions: `camelCase`; file names: `kebab-case`.
  - SQL tables and columns: `snake_case`.

- **Logging convention** – Use the standard library `log` package with the structured prefixes already in use:
  - `[INFO]` for normal flow
  - `[WARN]` for recoverable issues
  - `[ERROR]` for failures
  - `[PERF]` for timing entries

- **Commit messages** – Write imperative, present‑tense messages ("Add bias handler pagination") following the template in `.git_commit_msg.txt`.  Each change set must include tests when possible.

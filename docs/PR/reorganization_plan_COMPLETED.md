# Workspace Reorganization Plan

**Goal:** To clean and reorganize the `newbalancer_go` workspace for improved clarity, maintainability, and to ensure all path dependencies are satisfied.

---

## A. Required Code Change

1.  **File:** `cmd/test_parse/main.go`
    *   **Action:** Modify the file opening for `sample_feed.xml` to reflect its new location in `testdata/`.
    *   **Replace:**
        ```go
        file, err := os.Open("sample_feed.xml")
        ```
    *   **With:**
        ```go
        // Add "path/filepath" to your imports if not already present
        file, err := os.Open(filepath.Join("..", "..", "testdata", "sample_feed.xml")) // Assumes testdata is at root
        ```
        *Self-correction: The original `main.go` is in `cmd/test_parse/`, so `../..` is needed to get to the root if `testdata` is there.*

---

## B. `.gitignore` Work

1.  **Rename:** `\.gitignore___` to `\.gitignore`.
2.  **Append Patterns:** Add or ensure the following patterns are present in `\.gitignore`:
    ```gitignore
    # Build Artifacts
    *.exe
    /bin/
    /build/

    # Runtime Database Files
    news.db
    news.db-shm
    news.db-wal

    # Coverage & Test Reports
    *.out
    *.html
    /test-results/
    /playwright-report/

    # IDE / OS / Project Junk
    /junk/
    .DS_Store
    *.log
    # Add any other generated files or temporary directories specific to your tools/OS
    ```

---

## C. Folder Creation and File Moves

1.  **Create New Directories (and commit their creation, e.g., with a `.gitkeep` file if empty initially):**
    *   `tools/` (at project root)
    *   `testdata/` (at project root)
    *   `docs/archive/` (if it doesn't already exist)

2.  **Move Files:**
    *   `mock_llm_service.go` ➜ `tools/mock_llm_service.go`
    *   `mock_llm_service.py` ➜ `tools/mock_llm_service.py`
    *   `# Tools Configuration.md` ➜ `tools/tools_configuration_snapshot.md` (or archive it if truly unneeded after review)
    *   `sample_feed.xml` ➜ `testdata/sample_feed.xml`
    *   `app_requirements.md` ➜ `docs/archive/app_requirements.md`
    *   `newman_environment.json` ➜ `postman/newman_environment.json`

---

## D. Mock Service Path Audit

1.  **Action:** Thoroughly search (`grep` or IDE search) across `Makefile`, all files in `scripts/`, and any other potential execution points (e.g., CI scripts).
2.  **Search For:**
    *   `go run mock_llm_service.go` (or similar invocations)
    *   `python mock_llm_service.py` (or similar invocations)
3.  **Update:** If found, change paths to reflect the new location in `tools/`. *Initial searches were inconclusive, a deeper check is good practice.*

---

## E. Newman Environment File Path Audit

1.  **Context:** `newman_environment.json` has been moved to `postman/`.
2.  **Action:** Search in `scripts/` (especially `test.sh`, `test.cmd`, and individual `run_tc*.sh/.cmd` files) for direct references to `newman_environment.json` (the one formerly in the root).
3.  **Update:** If any script specifically referenced the root-level `newman_environment.json`, update the path to `postman/newman_environment.json`. *Note: `test.sh` and `test.cmd` already use other environment files from `postman/`.*

---

## F. JS/TS Configuration Files (`tsconfig.json`, `test.config.js`, `global.d.ts`)

1.  **Decision:** For maximum safety and to avoid breaking unverified TypeScript tooling or test runner (e.g., Playwright, Vitest, Jest) configurations, these files will **remain in the project root** for now.
2.  **Future Consideration:** If confirmed that these are solely for frontend components within `web/`, they can be moved into `web/`. This would require:
    *   Updating any `tsc` commands or `package.json` scripts that invoke the TypeScript compiler.
    *   Ensuring test runner configurations (e.g., Playwright's `playwright.config.ts`) correctly locate these files or are updated.
    *   Verifying IDE settings for TypeScript support.

---

## G. Build Artifacts Output Directory

1.  **Goal:** Ensure compiled binaries are not placed in the project root.
2.  **Action:** Modify Go build commands.
    *   **Example (Makefile or build scripts):**
        ```makefile
        # In Makefile or build script
        GOBIN := $(CURDIR)/bin
        # or
        OUTPUT_DIR := $(CURDIR)/../build_output # Example for outside workspace

        build_server:
        	go build -o $(GOBIN)/newbalancer_server ./cmd/server/main.go
        	# or for outside workspace (adjust path as needed)
        	# go build -o $(OUTPUT_DIR)/newbalancer_server ./cmd/server/main.go
        ```
    *   Ensure the chosen output directory (e.g., `bin/` at project root) is added to `\.gitignore`.

---

## H. Documentation Link Updates

1.  **Action:** Manually review and update all internal links and textual references in documentation files (`.md` files in `docs/`, `README.md`, etc.) that point to moved files.
2.  **Files to check links for:**
    *   `mock_llm_service.go` and `.py` (now in `tools/`)
    *   `sample_feed.xml` (now in `testdata/`)
    *   `app_requirements.md` (now in `docs/archive/`)
    *   `newman_environment.json` (now in `postman/`)
    *   `# Tools Configuration.md` (if kept and renamed in `tools/`)

---

## I. CI/CD and Docker Path Verification

1.  **Action:** If the project uses CI/CD pipelines (e.g., GitHub Actions workflows in `.github/workflows/`) or Docker (`Dockerfile`, `docker-compose.yml`):
    *   Review these files for any hardcoded paths to the files or directories that were moved (especially mock services, test data, Newman environments, or build output locations).
    *   Update paths as necessary to reflect the new structure.

---

## J. Communication and Rollback Notes

1.  **File:** Create `docs/PR/reorganization_impact_and_rollback.md` (or similar).
2.  **Content:**
    *   Briefly summarize the main structural changes (files moved, new directories).
    *   Provide a mapping of critical old paths to new paths.
    *   Outline simple rollback steps (e.g., "If tests fail due to `mock_llm_service.go` not found, verify scripts are calling `tools/mock_llm_service.go`. To temporarily revert, `git mv tools/mock_llm_service.go ./mock_llm_service.go` and revert path changes in calling scripts.").
    *   Inform team members of the changes before merging.

---
By following these steps, the workspace will be significantly cleaner, easier to navigate, and less prone to issues caused by generated files or misplaced configurations. 
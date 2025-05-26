### 1  Bootstrap the exact env that CI uses

| What                                                         | Local command                                                                                                     | Why it matches CI                                                                     |
| ------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| **Go + Node toolchain**                                      | Use the official Docker image or `asdf`:<br>`asdf install golang 1.22.0`<br>`asdf install nodejs 20.12.2`         | Both versions are hard-coded in the workflow matrix (see `.github/workflows/ci.yml`). |
| **Env vars**                                                 | ```bash
cp .env.example .env            # once
export NO_AUTO_ANALYZE=true      # prevents SQLite lockups
export OPENAI_API_KEY=…
``` | The README notes that every Newman-driven suite sets `NO_AUTO_ANALYZE=true` to avoid DB locks on CI ([GitHub][1]) |

> **Tip:** Add the export lines to a `./env.local` file and source it in your shell profile so `act` (below) also picks them up.

---

### 2  Run the *fast* gates first (Git hook safe)

1. **Tidy modules**

   ```bash
   make tidy
   ```
2. **Static analysis**

   ```bash
   make lint           # golangci-lint, same config as CI
   ```

   The linter is driven by `.golangci.yml` which enables `govet`, `staticcheck`, `funlen`, `gocyclo`, etc.
3. **Unit tests with race detector**

   ```bash
   make unit           # ≈ go test ./... -race
   ```

   Race detection is on by default; disable if you lack a C compiler:
   `make unit ENABLE_RACE_DETECTION=false`

Hook them with **pre-commit**:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.56.2
    hooks:
      - id: golangci-lint
        args: ["--config=.golangci.yml"]
  - repo: local
    hooks:
      - id: go-tests
        name: go-unit
        entry: make unit
        language: system
```

```bash
pip install pre-commit && pre-commit install
```

Now every `git commit` is blocked unless code **builds, lints, and unit-tests** cleanly.

---

### 3  Run the same integration suites that Actions runs

CI drives Newman through the Bash helper at `scripts/test.sh`; it exposes the same aliases locally :

```bash
# Integration tests (backend API surface)
NO_AUTO_ANALYZE=true scripts/test.sh backend

# Essential happy-path smoke tests
NO_AUTO_ANALYZE=true scripts/test.sh essential

# Full matrix (backend + API + confidence) -- same as PR workflow
NO_AUTO_ANALYZE=true scripts/test.sh all
```

These commands:

* spin up the Go server on port 8080,
* hit `/health` until it's ready,
* drive the Postman collections,
* tear down automatically (trap EXIT).

If they fail locally they'll fail on CI for exactly the same reason.

---

### 4  Verify coverage & API contract (fail-fast like CI)

```bash
make coverage-core        # enforces 90 % threshold 
make contract             # spectral lint + oasdiff diff
```

Both targets run in every Pull-Request job, so catching them now saves a round-trip.

---

### 5  Dry-run the whole workflow with **`act`**

```bash
brew install act    # or choco / npm / scoop
act -j test \
    -P ubuntu-latest=catthehacker/ubuntu:act-22.04
```

`act` mounts your repo, pulls the same Ubuntu image GitHub uses, and executes `.github/workflows/ci.yml` end-to-end, including the multi-Go-version matrix, Newman steps, and coverage upload. It's the closest thing to "CI in a box".

---

### 6  (Optional) pre-push safety net

Create `.git/hooks/pre-push`:

```bash
#!/usr/bin/env bash
set -e
echo "▶ Lint + unit"
make tidy lint unit
echo "▶ Essential integration"
NO_AUTO_ANALYZE=true scripts/test.sh essential
```

`chmod +x .git/hooks/pre-push`

This runs in < 1 minute and stops you from pushing broken code while still letting heavy suites run only on branch pushes.

---

## TL;DR printable checklist

```bash
# one-time
asdf install golang 1.22.0 && asdf install nodejs 20.12.2
cp .env.example .env && echo 'NO_AUTO_ANALYZE=true' >> .env
pre-commit install

# each work session
git pull --rebase
make tidy lint unit
NO_AUTO_ANALYZE=true scripts/test.sh backend
NO_AUTO_ANALYZE=true scripts/test.sh essential
make coverage-core contract
act                     # optional full CI dress rehearsal
git push
```

Follow this flow and what's green locally will stay green in GitHub Actions—no more surprise CI reds.

[1]: https://github.com/alexandru-savinov/BalancedNewsGo "GitHub - alexandru-savinov/BalancedNewsGo" 
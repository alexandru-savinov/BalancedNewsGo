# BalancedNewsGo

BalancedNewsGo is a Go service that collects news from RSS feeds, scores political bias using large language models, and exposes the results through a small web UI and a REST API.

## Features
- RSS ingestion
- LLM based bias scoring
- SQLite storage
- HTTP API (`/api`)
- Server rendered HTML templates and static assets

## Requirements
- Go 1.23+
- Node.js (for the optional test helpers)
- `.env` file containing `LLM_API_KEY` and `LLM_API_KEY_SECONDARY`

## Running
```bash
# build and run
make run
# or
go run cmd/server/main.go
```
Visit `http://localhost:8080` for the UI or `/api` for JSON endpoints.

## Tests
```bash
# unit tests
go test ./...
# backend integration tests
scripts/test.sh backend   # use .cmd on Windows
```

## Project layout
- `cmd/`          – application entry point
- `internal/`     – core packages
- `configs/`      – configuration files
- `static/`       – assets for the web UI
- `templates/`    – HTML templates
- `scripts/`      – helper scripts
- `tests/`        – integration tests

## License
[MIT](LICENSE)

## Documentation
- [Codebase overview](docs/codebase_documentation.md)
- [Configuration reference](docs/configuration_reference.md)
- [Testing guide](docs/testing.md)
- [Deployment guide](docs/deployment.md)
- [Contributing](CONTRIBUTING.md)

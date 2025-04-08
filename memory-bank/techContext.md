# Technology Context

## Technology Stack
- **Backend:** Go (Golang)
- **Database:** SQLite (local, lightweight)
- **Language Models:** OpenAI GPT APIs or local LLMs
- **Frontend:** HTML, JavaScript, htmx for dynamic UI
- **API:** RESTful, JSON-based

## Development Setup
- Go modules for dependency management
- SQLite database file stored locally
- LLM API keys configured via environment variables or config files
- Frontend served via Go HTTP server

## Technical Constraints
- Lightweight and local-first design
- Minimize LLM API costs and latency
- Modular to support multiple LLM providers
- Extendable for new sources and features

## Dependencies
- Go standard library
- SQLite driver for Go
- LLM API clients (OpenAI, etc.)
- htmx JavaScript library
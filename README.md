# NewsBalancer Go Backend

A Go-based backend service that provides politically balanced news aggregation using LLM-based analysis.

## Overview

NewsBalancer analyzes news articles from diverse sources using multiple LLM perspectives (left, center, right) to provide balanced viewpoints and identify potential biases. 

## Features

- RSS feed aggregation from multiple sources
- Multi-perspective LLM analysis (left/center/right)
- Composite score calculation with confidence metrics
- RESTful API for article retrieval and analysis
- Caching and database persistence
- Real-time progress tracking via SSE

## Running Tests

### Prerequisites

- Go 1.24+ installed
- Node.js for running newman tests
- SQLite database

### Unit Tests

Run all unit tests:
```
go test ./...
```

### Backend Integration Tests

Run the backend test suite:
```
./run_backend_tests.cmd
```

### All Tests (Including E2E)

Run the complete test suite:
```
./run_all_tests.cmd
```

### Debug Tests

Run tests with verbose output:
```
./run_debug_tests.cmd
```

## Development

### Environment Setup

1. Copy `.env.example` to `.env`
2. Configure RSS feed sources in `configs/feed_sources.json`
3. Set up LLM API keys in `.env`

### Running Locally

Start the server:
```
go run cmd/server/main.go
```

The server will be available at http://localhost:8080

## Contributing

1. Ensure tests pass locally
2. Add tests for new functionality
3. Update documentation as needed
4. Submit a pull request

## License

MIT License - see LICENSE file for details
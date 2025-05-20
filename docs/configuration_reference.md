# Configuration Reference

This document provides a comprehensive reference for all configuration options in the NewsBalancer system.

## Environment Variables (.env)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LLM_API_KEY` | Yes | - | Primary API key for LLM service (e.g., OpenRouter) |
| `LLM_API_KEY_SECONDARY` | No | - | Backup API key for fallback during rate limiting |
| `LLM_BASE_URL` | No | OpenRouter URL | Base URL for the LLM service |
| `LLM_HTTP_TIMEOUT` | No | 60s | Timeout for HTTP requests to LLM service |
| `NO_AUTO_ANALYZE` | No | false | When set to "true", disables automatic background LLM analysis |
| `DATABASE_URL` | No | news.db | While supported in code, most components default to `news.db` in the execution directory |
| `GIN_MODE` | No | debug | Set to "release" for production environment |
| `LEGACY_HTML` | No | false | Enable legacy server-side rendering mode |

## Configuration Files

### `configs/feed_sources.json`

This file defines the RSS feeds that the system will collect articles from. Example:

```json
[
  "https://www.cnn.com/rss/cnn_topstories.rss",
  "https://feeds.foxnews.com/foxnews/latest",
  "https://www.npr.org/rss/rss.php?id=1001",
  "https://feeds.bbci.co.uk/news/rss.xml"
]
```

### `configs/composite_score_config.json`

This file defines the LLM ensemble strategy and composite score calculation configuration. Example:

```json
{
  "models": [
    {
      "name": "claude-3-haiku-20240307",
      "perspective": "left",
      "role": "Left perspective analyzer"
    },
    {
      "name": "gpt-3.5-turbo",
      "perspective": "center",
      "role": "Centrist perspective analyzer"
    },
    {
      "name": "gemini-1.0-pro",
      "perspective": "right",
      "role": "Right perspective analyzer"
    }
  ],
  "min_score": -1.0,
  "max_score": 1.0,
  "default_missing": 0.0,
  "handle_invalid": "default",
  "formula": "weighted",
  "weights": {
    "left": 0.33,
    "center": 0.34,
    "right": 0.33
  },
  "confidence_method": "spread_based",
  "confidence_params": {
    "min_count": 2
  }
}
```

#### Configuration Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `models` | Array | List of LLM models to use for ensemble analysis |
| `models[].name` | String | Model identifier (must match what LLM service expects) |
| `models[].perspective` | String | Political perspective of this model ("left", "center", "right", "other") |
| `models[].role` | String | Optional descriptive role for the model |
| `models[].url` | String | Optional model-specific endpoint (overrides base URL) |
| `min_score` | Float | Minimum bound for normalized scores (typically -1.0) |
| `max_score` | Float | Maximum bound for normalized scores (typically 1.0) |
| `default_missing` | Float | Default value to use when perspective is missing (used if handle_invalid is "default") |
| `handle_invalid` | String | How to handle NaN/Infinity values: "ignore" or "default" |
| `formula` | String | Method to combine perspective scores: "average" or "weighted" |
| `weights` | Object | Used when formula is "weighted"; perspective-to-weight mapping |
| `confidence_method` | String | How to calculate confidence: "average", "min", "max", or "spread_based" |
| `confidence_params` | Object | Additional parameters for confidence calculation |
| `confidence_params.min_count` | Integer | For "spread_based" mode: min perspectives needed for valid confidence |

## Prompt Templates

LLM prompt templates are located in `internal/llm/configs/*.txt` and are used by the ensemble analysis system:

- `DefaultPromptVariant.txt` - Standard prompt for political bias analysis
- `concise_json_prompt.txt` - Prompt optimized for JSON response format

## Web Server Configuration

The web server (Gin) runs on port 8080 by default. This can be modified by editing the `cmd/server/main.go` file.

## Database Schema

Database schema is defined in `internal/db/db.go` and includes the following tables:

- `articles` - Stores article data and metadata
- `llm_scores` - Stores individual model scores (has UNIQUE constraint on article_id+model)
- `feedback` - Stores user feedback on article bias
- `labels` - Stores ground truth labels for validation

## Deployment Configurations

For production deployment, consider the following configuration updates:

1. Set `GIN_MODE=release` environment variable
2. Use a proper database path with backup strategy
3. Set up monitoring endpoints for health checks
4. Configure rate limiting for API endpoints
5. Set appropriate timeouts for LLM services 
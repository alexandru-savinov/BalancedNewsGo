# Source Management System - Technical Specification

## Overview
Transform NewsBalancer from static JSON-based source configuration to dynamic database-backed source management with full CRUD operations via admin UI and REST API.

## Architecture

### Database Schema

#### Sources Table
```sql
CREATE TABLE IF NOT EXISTS sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,                    -- Display name (e.g., "CNN Politics")
    channel_type TEXT NOT NULL DEFAULT 'rss',    -- 'rss', 'telegram', 'twitter', etc.
    feed_url TEXT NOT NULL,                       -- RSS URL or channel identifier
    category TEXT NOT NULL,                       -- 'left', 'center', 'right'
    enabled BOOLEAN NOT NULL DEFAULT 1,           -- Active/inactive toggle
    default_weight REAL NOT NULL DEFAULT 1.0,    -- Scoring weight multiplier
    last_fetched_at TIMESTAMP,                    -- Last successful fetch
    error_streak INTEGER NOT NULL DEFAULT 0,     -- Consecutive error count
    metadata TEXT,                                -- JSON for channel-specific config
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sources_enabled ON sources(enabled);
CREATE INDEX idx_sources_channel_type ON sources(channel_type);
CREATE INDEX idx_sources_category ON sources(category);
```

#### Source Stats Table (Analytics)
```sql
CREATE TABLE IF NOT EXISTS source_stats (
    source_id INTEGER PRIMARY KEY,
    article_count INTEGER NOT NULL DEFAULT 0,
    avg_score REAL,                               -- Average composite score
    last_article_at TIMESTAMP,                    -- Most recent article
    computed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_id) REFERENCES sources (id)
);
```

### Data Models

#### Core Source Model
```go
type Source struct {
    ID            int64      `db:"id" json:"id"`
    Name          string     `db:"name" json:"name" binding:"required"`
    ChannelType   string     `db:"channel_type" json:"channel_type" binding:"required"`
    FeedURL       string     `db:"feed_url" json:"feed_url" binding:"required"`
    Category      string     `db:"category" json:"category" binding:"required"`
    Enabled       bool       `db:"enabled" json:"enabled"`
    DefaultWeight float64    `db:"default_weight" json:"default_weight"`
    LastFetchedAt *time.Time `db:"last_fetched_at" json:"last_fetched_at,omitempty"`
    ErrorStreak   int        `db:"error_streak" json:"error_streak"`
    Metadata      *string    `db:"metadata" json:"metadata,omitempty"`
    CreatedAt     time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type SourceStats struct {
    SourceID      int64      `db:"source_id" json:"source_id"`
    ArticleCount  int64      `db:"article_count" json:"article_count"`
    AvgScore      *float64   `db:"avg_score" json:"avg_score,omitempty"`
    LastArticleAt *time.Time `db:"last_article_at" json:"last_article_at,omitempty"`
    ComputedAt    time.Time  `db:"computed_at" json:"computed_at"`
}
```

#### API DTOs
```go
type CreateSourceRequest struct {
    Name          string  `json:"name" binding:"required"`
    ChannelType   string  `json:"channel_type" binding:"required"`
    FeedURL       string  `json:"feed_url" binding:"required,url"`
    Category      string  `json:"category" binding:"required,oneof=left center right"`
    DefaultWeight float64 `json:"default_weight,omitempty"`
    Metadata      *string `json:"metadata,omitempty"`
}

type UpdateSourceRequest struct {
    Name          *string  `json:"name,omitempty"`
    FeedURL       *string  `json:"feed_url,omitempty,url"`
    Category      *string  `json:"category,omitempty,oneof=left center right"`
    Enabled       *bool    `json:"enabled,omitempty"`
    DefaultWeight *float64 `json:"default_weight,omitempty"`
    Metadata      *string  `json:"metadata,omitempty"`
}

type SourceWithStats struct {
    Source
    Stats *SourceStats `json:"stats,omitempty"`
}
```

## API Endpoints

### REST API Specification

#### GET /api/sources
List all sources with optional filtering and pagination.

**Query Parameters:**
- `enabled` (bool): Filter by enabled status
- `channel_type` (string): Filter by channel type
- `category` (string): Filter by category
- `limit` (int): Page size (default: 50, max: 100)
- `offset` (int): Pagination offset
- `include_stats` (bool): Include source statistics

**Response:**
```json
{
  "success": true,
  "data": {
    "sources": [
      {
        "id": 1,
        "name": "CNN Politics",
        "channel_type": "rss",
        "feed_url": "https://rss.cnn.com/rss/edition.rss",
        "category": "center",
        "enabled": true,
        "default_weight": 1.0,
        "error_streak": 0,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "stats": {
          "source_id": 1,
          "article_count": 150,
          "avg_score": 0.05,
          "last_article_at": "2024-01-01T12:00:00Z"
        }
      }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### POST /api/sources
Create a new source.

**Request Body:**
```json
{
  "name": "New Source",
  "channel_type": "rss",
  "feed_url": "https://example.com/feed.xml",
  "category": "center",
  "default_weight": 1.0
}
```

#### PUT /api/sources/:id
Update an existing source.

#### DELETE /api/sources/:id
Soft delete (disable) a source.

#### GET /api/sources/:id/stats
Get detailed statistics for a specific source.

#### POST /api/sources/:id/test
Test source connectivity and validate feed format.

### Admin UI Endpoints

#### GET /admin/sources
Admin interface for source management.

#### GET /htmx/sources
HTMX fragment for source list.

#### GET /htmx/sources/:id/form
HTMX fragment for source edit form.

## Migration Strategy

### Phase 1: Database Setup
1. Create sources and source_stats tables
2. Import existing sources from `configs/feed_sources.json`
3. Validate migration success

### Phase 2: API Implementation
1. Implement source CRUD operations
2. Add API endpoints with validation
3. Test API functionality

### Phase 3: RSS Integration
1. Modify RSS collector to load from database
2. Maintain backward compatibility
3. Test RSS functionality

### Phase 4: Admin UI
1. Enhance admin interface
2. Add HTMX-powered source management
3. Implement source statistics display

## Validation Requirements

### Data Validation
- Source names must be unique
- Feed URLs must be valid URLs
- Categories must be one of: left, center, right
- Channel types must be supported
- Default weights must be positive numbers

### Business Rules
- Cannot delete sources with recent articles (< 7 days)
- Disabled sources should not be fetched
- Error streak resets on successful fetch
- Source stats updated nightly

### Security Considerations
- Admin-only access for source management
- Input validation and sanitization
- SQL injection prevention
- Rate limiting for API endpoints

## Performance Requirements
- Source list API: < 200ms response time
- Source creation: < 500ms response time
- Admin UI: < 1s page load time
- Migration: < 30s for existing sources

## Monitoring & Observability
- Source health checks every 30 minutes
- Error tracking and alerting
- Performance metrics collection
- Audit log for source changes

CREATE TABLE articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    pub_date TIMESTAMP NOT NULL,
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending',
    fail_count INTEGER DEFAULT 0,
    last_attempt TIMESTAMP,
    escalated BOOLEAN DEFAULT FALSE,
    composite_score REAL,
    confidence REAL,
    score_source TEXT
);

CREATE TABLE llm_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    model TEXT NOT NULL,
    score REAL NOT NULL,
    metadata TEXT,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (article_id) REFERENCES articles (id),
    UNIQUE(article_id, model)
);

CREATE TABLE feedback (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    user_id TEXT,
    feedback_text TEXT,
    category TEXT,
    ensemble_output_id INTEGER,
    source TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (article_id) REFERENCES articles (id)
);

CREATE TABLE labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    data TEXT NOT NULL,
    label TEXT NOT NULL,
    source TEXT NOT NULL,
    date_labeled TIMESTAMP NOT NULL,
    labeler TEXT NOT NULL,
    confidence REAL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_llm_scores_article_version ON llm_scores(article_id, version);

CREATE TABLE sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    channel_type TEXT NOT NULL DEFAULT 'rss',
    feed_url TEXT NOT NULL,
    category TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    default_weight REAL NOT NULL DEFAULT 1.0,
    last_fetched_at TIMESTAMP,
    error_streak INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sources_enabled ON sources(enabled);
CREATE INDEX idx_sources_channel_type ON sources(channel_type);
CREATE INDEX idx_sources_category ON sources(category);

CREATE TABLE source_stats (
    source_id INTEGER PRIMARY KEY,
    article_count INTEGER NOT NULL DEFAULT 0,
    avg_score REAL,
    last_article_at TIMESTAMP,
    computed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_id) REFERENCES sources (id)
);

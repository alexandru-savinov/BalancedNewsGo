-- Seed data for sources table
-- This data is required for E2E tests that expect specific sources with specific IDs

INSERT INTO sources (id, name, feed_url, channel_type, category, default_weight, enabled, created_at, updated_at, last_fetched_at, error_streak) VALUES
(1, 'HuffPost', 'https://www.huffpost.com/section/front-page/feed', 'rss', 'left', 1.0, true, datetime('now'), datetime('now'), datetime('now'), 0),
(2, 'BBC News', 'https://feeds.bbci.co.uk/news/rss.xml', 'rss', 'center', 1.0, true, datetime('now'), datetime('now'), datetime('now'), 0),
(3, 'MSNBC', 'http://www.msnbc.com/feeds/latest', 'rss', 'right', 1.0, true, datetime('now'), datetime('now'), datetime('now'), 0);

-- Insert some test articles to ensure the database has content
INSERT INTO articles (id, title, content, url, source_id, published_at, created_at, updated_at) VALUES
(1, 'Test Article 1', 'This is a test article for testing purposes', 'https://example.com/test1', 1, datetime('now'), datetime('now'), datetime('now')),
(2, 'Test Article 2', 'This is another test article', 'https://example.com/test2', 2, datetime('now'), datetime('now'), datetime('now')),
(3, 'Test Article 3', 'Third test article for comprehensive testing', 'https://example.com/test3', 3, datetime('now'), datetime('now'), datetime('now'));

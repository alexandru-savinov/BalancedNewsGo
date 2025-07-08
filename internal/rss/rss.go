package rss

import (
	"log"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3"
)

// Define the CollectorInterface
// CollectorInterface defines the methods that an RSS collector must implement.
type CollectorInterface interface {
	ManualRefresh()
	CheckFeedHealth() map[string]bool
}



type Collector struct {
	DB        *sqlx.DB
	FeedURLs  []string
	Cron      *cron.Cron
	LLMClient *llm.LLMClient
}

// NewCollector creates a new RSS Collector with DB and feed URLs.
func NewCollector(dbConn *sqlx.DB, urls []string, llmClient *llm.LLMClient) *Collector {
	return &Collector{
		DB:        dbConn,
		FeedURLs:  urls,
		Cron:      cron.New(),
		LLMClient: llmClient,
	}
}

// StartScheduler starts the cron job to fetch feeds every 30 minutes.
func (c *Collector) StartScheduler() {
	_, err := c.Cron.AddFunc("@every 30m", func() {
		log.Println("[RSS] Scheduled fetch started")
		c.FetchAndStore()
	})
	if err != nil {
		log.Printf("[RSS] Failed to schedule fetch: %v", err)

		return
	}

	c.Cron.Start()
	log.Println("[RSS] Scheduler started, fetching every 30 minutes")
}

// ManualRefresh triggers an immediate fetch.
func (c *Collector) ManualRefresh() {
	log.Println("[RSS] Manual refresh triggered")
	c.FetchAndStore()
}

// FetchAndStore fetches all feeds, parses, validates, deduplicates, inserts.
func (c *Collector) FetchAndStore() {
	parser := gofeed.NewParser()

	for _, feedURL := range c.FeedURLs {
		feed := c.fetchFeed(parser, feedURL)
		if feed == nil {
			continue
		}

		for _, item := range feed.Items {
			c.processFeedItem(feed, item)
		}
	}
}

func (c *Collector) processFeedItem(feed *gofeed.Feed, item *gofeed.Item) {
	if c.shouldSkipItem(item) {
		return
	}

	dup, err := c.isDuplicate(item)
	if err != nil {
		log.Printf("[RSS] Error checking duplicates: %v", err)
		return
	}
	if dup {
		return
	}

	// Check for duplicates using title similarity
	isDuplicate, err := db.ArticleExistsBySimilarTitle(c.DB, item.Title)
	if err != nil {
		log.Printf("[RSS] Error checking for duplicate article: %v", err)
		return
	}
	if isDuplicate {
		log.Printf("[RSS] Skipping duplicate article: %s", item.Title)
		return
	}

	article := c.createArticle(feed, item)

	if err := c.storeArticle(article); err != nil {
		log.Printf("[RSS] Failed to store article: %v", err)
		return
	}
}

func (c *Collector) fetchFeed(parser *gofeed.Parser, feedURL string) *gofeed.Feed {
	log.Printf("[RSS] Fetching feed: %s", feedURL)
	feed, err := parser.ParseURL(feedURL)
	if err != nil {
		log.Printf("[RSS] Failed to parse feed %s: %v", feedURL, err)
		return nil
	}
	return feed
}

func (c *Collector) shouldSkipItem(item *gofeed.Item) bool {
	if !isValidItem(item) {
		log.Printf("[RSS] Invalid item skipped: %+v", item)
		return true
	}
	return false
}

func (c *Collector) isDuplicate(item *gofeed.Item) (bool, error) {
	exists, err := db.ArticleExistsByURL(c.DB, item.Link)
	if err != nil {
		return false, err
	}
	if exists {
		log.Printf("[RSS] Duplicate article by URL skipped: %s", item.Link)
		return true, nil
	}

	return false, nil
}

func (c *Collector) extractContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	if ext, ok := item.Extensions["content"]; ok {
		if encoded, ok := ext["encoded"]; ok && len(encoded) > 0 {
			if encoded[0].Value != "" {
				return encoded[0].Value
			}
		}
	}
	if item.Description != "" {
		return item.Description
	}
	return ""
}

func (c *Collector) createArticle(feed *gofeed.Feed, item *gofeed.Item) *db.Article {
	pubTime := time.Now()
	if item.PublishedParsed != nil {
		pubTime = *item.PublishedParsed
	}

	return &db.Article{
		Source:  feed.Title,
		PubDate: pubTime,
		URL:     item.Link,
		Title:   item.Title,
		Content: c.extractContent(item),
	}
}

func (c *Collector) storeArticle(article *db.Article) error {
	_, err := db.InsertArticle(c.DB, article)
	if err != nil {
		return err
	}

	log.Printf("[RSS] Inserted new article: %s", article.URL)
	return nil
}

// isValidItem performs basic validation on a feed item.
func isValidItem(item *gofeed.Item) bool {
	if item == nil {
		return false
	}

	if item.Link == "" || item.Title == "" {
		return false
	}

	// Check for non-empty content using the same logic as extractContent
	content := ""
	if item.Content != "" {
		content = item.Content
	} else if ext, ok := item.Extensions["content"]; ok {
		if encoded, ok := ext["encoded"]; ok && len(encoded) > 0 {
			if encoded[0].Value != "" {
				content = encoded[0].Value
			}
		}
	} else if item.Description != "" {
		content = item.Description
	}
	if strings.TrimSpace(content) == "" {
		return false
	}

	return true
}

// CheckFeedHealth checks connectivity and format for each feed source.
func (c *Collector) CheckFeedHealth() map[string]bool {
	results := make(map[string]bool)
	parser := gofeed.NewParser()

	for _, feedURL := range c.FeedURLs {
		_, err := parser.ParseURL(feedURL)
		if err != nil {
			results[feedURL] = false
			log.Printf("[RSS][Health] %s - Error: %v", feedURL, err)
			continue
		}
		results[feedURL] = true
		log.Printf("[RSS][Health] %s - OK", feedURL)
	}
	return results
}

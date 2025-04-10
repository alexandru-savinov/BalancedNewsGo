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

func detectPartisanCues(text string) []string {
	cues := []string{
		"radical left", "far-left", "far right", "fake news", "patriotic", "woke", "social justice",
		"deep state", "mainstream media", "liberal agenda", "conservative values", "culture war",
		"globalist", "elitist", "freedom-loving", "authoritarian", "biased media",
	}
	var found = make([]string, 0, len(cues))
	lower := strings.ToLower(text)
	for _, cue := range cues {
		if strings.Contains(lower, cue) {
			found = append(found, cue)
		}
	}
	return found
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

	titleExists, err := db.ArticleExistsBySimilarTitle(c.DB, item.Title, 7)
	if err != nil {
		return false, err
	}
	if titleExists {
		log.Printf("[RSS] Duplicate article by similar title skipped: %s", item.Title)
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
	cues := detectPartisanCues(article.Title + " " + article.Content)
	if len(cues) > 0 {
		log.Printf("[RSS] Partisan cues detected in article: %s | Cues: %v", article.Title, cues)
	}

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

	return true
}

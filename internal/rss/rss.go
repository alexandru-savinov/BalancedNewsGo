package rss

import (
	"log"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3"
)

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
		log.Printf("[RSS] Fetching feed: %s", feedURL)

		feed, err := parser.ParseURL(feedURL)
		if err != nil {
			log.Printf("[RSS] Failed to parse feed %s: %v", feedURL, err)

			continue
		}

		for _, item := range feed.Items {
			if !isValidItem(item) {
				log.Printf("[RSS] Invalid item skipped: %+v", item)

				continue
			}

			exists, err := db.ArticleExistsByURL(c.DB, item.Link)
			if err != nil {
				log.Printf("[RSS] DB error checking duplicate: %v", err)

				continue
			}

			if exists {
				log.Printf("[RSS] Duplicate article skipped: %s", item.Link)

				continue
			}

			pubTime := time.Now()
			if item.PublishedParsed != nil {
				pubTime = *item.PublishedParsed
			}

			article := &db.Article{
				Source:  feed.Title,
				PubDate: pubTime,
				URL:     item.Link,
				Title:   item.Title,
				Content: item.Content,
			}

			_, err = db.InsertArticle(c.DB, article)
			if err != nil {
				log.Printf("[RSS] Failed to insert article: %v", err)

				continue
			}

			log.Printf("[RSS] Inserted new article: %s", item.Link)

			// Trigger political analysis
			err = c.LLMClient.AnalyzeAndStore(article)
			if err != nil {
				log.Printf("[RSS] Failed to analyze article: %v", err)
			} else {
				log.Printf("[RSS] Political analysis completed for article: %s", item.Link)
			}
		}
	}
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

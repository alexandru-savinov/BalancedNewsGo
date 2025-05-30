package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mmcdole/gofeed"
)

func main() {
	file, err := os.Open(filepath.Join("..", "..", "testdata", "sample_feed.xml"))
	if err != nil {
		log.Fatalf("Failed to open sample_feed.xml: %v", err)
	}
	parser := gofeed.NewParser()

	feed, err := parser.Parse(file)
	if err != nil {
		log.Fatalf("Failed to parse feed: %v", err)
	}

	if err := file.Close(); err != nil {
		log.Printf("Warning: failed to close file: %v", err)
	}

	fmt.Printf("Feed Title: %s\n", feed.Title)

	for _, item := range feed.Items {
		fmt.Println("----")
		fmt.Printf("Title: %s\n", item.Title)
		fmt.Printf("Link: %s\n", item.Link)
		fmt.Printf("Published: %s\n", item.Published)

		if item.Content != "" {
			fmt.Printf("Content: %.100s...\n", item.Content)
		} else if item.Description != "" {
			fmt.Printf("Description: %.100s...\n", item.Description)
		}
	}
}

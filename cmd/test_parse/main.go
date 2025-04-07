package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mmcdole/gofeed"
)

func main() {
	file, err := os.Open("sample_feed.xml")
	if err != nil {
		log.Fatalf("Failed to open sample_feed.xml: %v", err)
	}
	defer file.Close()

	parser := gofeed.NewParser()
	feed, err := parser.Parse(file)
	if err != nil {
		log.Fatalf("Failed to parse feed: %v", err)
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

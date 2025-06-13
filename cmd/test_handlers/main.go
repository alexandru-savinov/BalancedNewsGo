package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("Testing API-based template handlers...")

	// Create handlers
	handlers := NewAPITemplateHandlers("http://localhost:8080/api")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test basic stats functionality
	fmt.Println("\n=== Test 1: Basic Stats ===")
	stats, err := handlers.getBasicStats(ctx)
	if err != nil {
		log.Printf("Error getting basic stats: %v", err)
	} else {
		fmt.Printf("✓ Successfully got basic stats: %d total articles\n", stats["TotalArticles"])
	}

	// Test system status
	fmt.Println("\n=== Test 2: System Status ===")
	status, err := handlers.getSystemStatus(ctx)
	if err != nil {
		log.Printf("Error getting system status: %v", err)
	} else {
		fmt.Printf("✓ Successfully got system status - DB OK: %v\n", status["DatabaseOK"])
	}

	// Test recent activity
	fmt.Println("\n=== Test 3: Recent Activity ===")
	activity, err := handlers.getRecentActivity(ctx)
	if err != nil {
		log.Printf("Error getting recent activity: %v", err)
	} else {
		fmt.Printf("✓ Successfully got recent activity: %d items\n", len(activity))
	}

	fmt.Println("\n=== Template Handlers Test Complete ===")
}

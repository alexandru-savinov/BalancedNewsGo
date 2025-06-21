package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Set test environment
	os.Setenv("TEST_MODE", "true")
	os.Setenv("DB_CONNECTION", ":memory:")
	os.Setenv("GIN_MODE", "test")
	os.Setenv("PORT", "8080")

	// Test health check endpoint
	go func() {
		time.Sleep(5 * time.Second)
		resp, err := http.Get("http://localhost:8080/healthz")
		if err != nil {
			log.Printf("Health check failed: %v", err)
			return
		}
		defer resp.Body.Close()
		log.Printf("Health check status: %d", resp.StatusCode)
	}()

	// Import and run the server
	fmt.Println("Starting server in test mode...")
	
	// This would normally import the server main, but for debugging let's just wait
	time.Sleep(10 * time.Second)
	fmt.Println("Debug test complete")
}

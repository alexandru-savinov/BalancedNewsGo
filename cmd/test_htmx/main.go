package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("Testing HTMX endpoints...")

	// Set environment variable for API handlers
	if err := os.Setenv("USE_API_HANDLERS", "true"); err != nil {
		fmt.Printf("Warning: Failed to set environment variable: %v\n", err)
	}

	// Wait for server to start
	fmt.Println("Waiting for server to start...")
	time.Sleep(3 * time.Second)

	// Test endpoints
	endpoints := []struct {
		name string
		url  string
	}{
		{"Main Articles Page", "http://localhost:8080/articles"},
		{"Articles Fragment", "http://localhost:8080/api/fragments/articles"},
		{"Articles Fragment with Filter", "http://localhost:8080/api/fragments/articles?bias=left&page=1"},
		{"Health Check", "http://localhost:8080/healthz"},
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, endpoint := range endpoints {
		fmt.Printf("\n=== Testing %s ===\n", endpoint.name)
		fmt.Printf("URL: %s\n", endpoint.url)

		resp, err := client.Get(endpoint.url)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("Status: %s\n", resp.Status)

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("✅ Success\n")

			// Read and show response size
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				fmt.Printf("Response size: %d bytes\n", len(body))

				// Show first 200 characters for HTML responses
				if resp.Header.Get("Content-Type") == "text/html; charset=utf-8" {
					preview := string(body)
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					fmt.Printf("Preview: %s\n", preview)
				}
			}
		} else {
			fmt.Printf("❌ Failed with status: %s\n", resp.Status)
		}
	}

	fmt.Println("\n=== HTMX Test Complete ===")
	fmt.Println("If successful, you can now:")
	fmt.Println("1. Open http://localhost:8080/articles in your browser")
	fmt.Println("2. Test the dynamic filtering and pagination")
	fmt.Println("3. Click on article links to see HTMX navigation")
	fmt.Println("4. Use browser dev tools to see HTMX requests")
}

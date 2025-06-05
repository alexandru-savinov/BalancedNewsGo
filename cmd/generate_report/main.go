package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var baseURL = getBaseURL()

func getBaseURL() string {
	if env := os.Getenv("REPORT_BASE_URL"); env != "" {
		return env
	}
	return "http://localhost:8080"
}

func fetchAndSave(endpoint, filename string) error {
	resp, err := http.Get(baseURL + endpoint)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close response body: %v\n", err)
		}
	}()

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Failed to close file: %v\n", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	var header = make([]string, 0, len(data[0]))
	for k := range data[0] {
		header = append(header, k)
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, row := range data {
		var record = make([]string, 0, len(header))
		for _, k := range header {
			val := fmt.Sprintf("%v", row[k])
			record = append(record, val)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	timestamp := time.Now().Format("20060102_150405")

	endpoints := map[string]string{
		"/metrics/validation":    fmt.Sprintf("validation_metrics_%s.csv", timestamp),
		"/metrics/feedback":      fmt.Sprintf("feedback_summary_%s.csv", timestamp),
		"/metrics/uncertainty":   fmt.Sprintf("uncertainty_rates_%s.csv", timestamp),
		"/metrics/disagreements": fmt.Sprintf("disagreements_%s.csv", timestamp),
		"/metrics/outliers":      fmt.Sprintf("outliers_%s.csv", timestamp),
	}

	for endpoint, filename := range endpoints {
		fmt.Printf("Fetching %s...\n", endpoint)
		if err := fetchAndSave(endpoint, filename); err != nil {
			fmt.Printf("Error fetching %s: %v\n", endpoint, err)
		} else {
			fmt.Printf("Saved report to %s\n", filename)
		}
	}

	// Basic alerting example: check uncertainty rates
	resp, err := http.Get(baseURL + "/metrics/uncertainty")
	if err != nil {
		fmt.Printf("Error fetching uncertainty rates: %v\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close response body: %v\n", err)
		}
	}()

	var rates []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		fmt.Printf("Error decoding uncertainty rates: %v\n", err)
		return
	}

	for _, rate := range rates {
		day := rate["day"]
		val, _ := rate["low_confidence_ratio"].(float64)
		if val > 0.3 { // example threshold
			fmt.Printf("ALERT: High uncertainty (%.2f) on %s\n", val, day)
		}
	}
}

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

const LabelUnknown = "unknown"

func main() {
	dbPath := flag.String("db", "news.db", "Path to SQLite database")
	filePath := flag.String("file", "", "Path to labeled dataset file (CSV or JSON)")
	format := flag.String("format", "csv", "File format: csv or json")
	source := flag.String("source", LabelUnknown, "Data source name")
	labeler := flag.String("labeler", LabelUnknown, "Labeler name")
	confidence := flag.Float64("confidence", 1.0, "Default confidence score")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("Please provide --file path to labeled dataset")
	}

	database, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	var count int
	switch *format {
	case "csv":
		count, err = importCSV(database, f, *source, *labeler, *confidence)
	case "json":
		count, err = importJSON(database, f, *source, *labeler, *confidence)
	default:
		log.Fatalf("Unsupported format: %s", *format)
	}
	if err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	fmt.Printf("Imported %d labels from %s\n", count, *filePath)
}

func importCSV(database *sqlx.DB, f *os.File, source, labeler string, confidence float64) (int, error) {
	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}
	if len(records) < 1 {
		return 0, fmt.Errorf("empty CSV file")
	}

	header := records[0]
	dataIdx, labelIdx := -1, -1
	for i, h := range header {
		switch h {
		case "data":
			dataIdx = i
		case "label":
			labelIdx = i
		}
	}
	if dataIdx == -1 || labelIdx == -1 {
		return 0, fmt.Errorf("CSV must have 'data' and 'label' columns")
	}

	count := 0
	for _, rec := range records[1:] {
		label := db.Label{
			Data:        rec[dataIdx],
			Label:       rec[labelIdx],
			Source:      source,
			DateLabeled: time.Now(),
			Labeler:     labeler,
			Confidence:  confidence,
			CreatedAt:   time.Now(),
		}
		if err := db.InsertLabel(database, &label); err != nil {
			log.Printf("Failed to insert label: %v", err)
			continue
		}
		count++
	}
	return count, nil
}

func importJSON(database *sqlx.DB, f *os.File, source, labeler string, confidence float64) (int, error) {
	var items []map[string]interface{}
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&items); err != nil {
		return 0, err
	}

	count := 0
	for _, item := range items {
		dataVal, ok1 := item["data"].(string)
		labelVal, ok2 := item["label"].(string)
		if !ok1 || !ok2 {
			log.Printf("Skipping invalid item: %v", item)
			continue
		}
		label := db.Label{
			Data:        dataVal,
			Label:       labelVal,
			Source:      source,
			DateLabeled: time.Now(),
			Labeler:     labeler,
			Confidence:  confidence,
			CreatedAt:   time.Now(),
		}
		if err := db.InsertLabel(database, &label); err != nil {
			log.Printf("Failed to insert label: %v", err)
			continue
		}
		count++
	}
	return count, nil
}

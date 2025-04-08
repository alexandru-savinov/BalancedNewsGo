package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type AnalysisResult struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

func analyzeHandler(label string, score float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := AnalysisResult{
			Score: score,
			Label: label,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: mock_llm_service <label> <port>")
	}
	label := os.Args[1]
	port := os.Args[2]

	var score float64
	switch label {
	case "left":
		score = -1.0
	case "center":
		score = 0.0
	case "right":
		score = 1.0
	default:
		score = 0.0
	}

	http.HandleFunc("/analyze", analyzeHandler(label, score))

	log.Printf("Starting mock LLM service for %s on port %s...", label, port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	LabelLeft  = "left"
	LabelRight = "right"
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

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
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
	case LabelLeft:
		score = -1.0
	case "center":
		score = 0.0
	case LabelRight:
		score = 1.0
	default:
		score = 0.0
	}

	http.HandleFunc("/analyze", analyzeHandler(label, score))

	log.Printf("Starting mock LLM service for %s on port %s...", label, port)

	// Create HTTP server with security timeouts
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           nil,               // Use default mux
		ReadHeaderTimeout: 30 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,  // Maximum time to read request
		WriteTimeout:      60 * time.Second,  // Maximum time to write response
		IdleTimeout:       120 * time.Second, // Maximum time for idle connections
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

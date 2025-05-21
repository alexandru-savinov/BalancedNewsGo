package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

// chatCompletionsHandler mimics the OpenRouter /chat/completions endpoint.
// It returns a minimal JSON structure that wraps an AnalysisResult in the
// expected format so the real server can parse it during tests.
func chatCompletionsHandler(label string, score float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inner := AnalysisResult{
			Score: score,
			Label: label,
		}

		// Construct an OpenAI-compatible response
		wrapper := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": string(mustJSONMarshal(map[string]interface{}{
							"score":       inner.Score,
							"explanation": "mock explanation",
							"confidence":  0.99,
						})),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(wrapper); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func mustJSONMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
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
	// Also support the OpenRouter-compatible path used by the Go code
	http.HandleFunc("/chat/completions", chatCompletionsHandler(label, score))

	log.Printf("Starting mock LLM service for %s on port %s...", label, port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

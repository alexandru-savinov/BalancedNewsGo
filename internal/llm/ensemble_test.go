package llm

import (
	"log"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Warning: .env file not loaded, relying on existing environment variables")
	}
}

func TestCallLLM(t *testing.T) {
	// pv := PromptVariant{
	// 	ID:       "test",
	// 	Template: "Test prompt",
	// 	Examples: []string{},
	// }
	// score, explanation, confidence, rawResp, err := callLLM("gpt-3.5", pv, "Test article content")
	// assert.NoError(t, err)
	// assert.InDelta(t, 0.0, score, 1.0)
	// assert.NotEmpty(t, explanation)
	// assert.InDelta(t, 1.0, confidence, 1.0)
	// assert.NotEmpty(t, rawResp)
}

func TestEnsembleAnalyze(t *testing.T) {
	// client := &LLMClient{}
	// score, err := client.EnsembleAnalyze(0, "Test article content")
	// assert.NoError(t, err)
	// assert.NotNil(t, score)
	// assert.InDelta(t, 0.0, score.Score, 1.0)
	// assert.NotEmpty(t, score.Metadata)
}

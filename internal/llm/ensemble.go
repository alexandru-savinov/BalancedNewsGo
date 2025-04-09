package llm

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

const stubLLMResponseJSON = "{\"score\":0.0,\"explanation\":\"stub explanation\",\"confidence\":1.0}"

// PromptVariant defines a prompt template with few-shot examples
type PromptVariant struct {
	ID       string
	Template string
	Examples []string
}

func (pv *PromptVariant) GeneratePrompt(content string) string {
	examplesText := strings.Join(pv.Examples, "\n")
	return fmt.Sprintf("%s\n%s\nArticle:\n%s", pv.Template, examplesText, content)
}

// callLLM queries a specific LLM with a prompt variant
func callLLM(modelName string, promptVariant PromptVariant, content string) (float64, string, float64, string, error) {
	prompt := promptVariant.GeneratePrompt(content)

	var rawResp string
	var err error

	switch modelName {
	case "gpt-3.5", "gpt-4":
		rawResp, err = callOpenAIAPI(modelName, prompt)
	case "claude":
		rawResp, err = callClaudeAPI(prompt)
	case "finetuned":
		rawResp, err = callFineTunedModelAPI(prompt)
	default:
		return 0, "", 0, "", fmt.Errorf("unsupported model: %s", modelName)
	}
	if err != nil {
		return 0, "", 0, rawResp, err
	}

	score, explanation, confidence, parseErr := parseLLMResponse(rawResp)
	if parseErr != nil {
		return 0, "", 0, rawResp, parseErr
	}

	return score, explanation, confidence, rawResp, nil
}

// Stub LLM API calls
func callOpenAIAPI(model, prompt string) (string, error) {
	return stubLLMResponseJSON, nil
}

func callClaudeAPI(prompt string) (string, error) {
	return stubLLMResponseJSON, nil
}

func callFineTunedModelAPI(prompt string) (string, error) {
	return stubLLMResponseJSON, nil
}

// parseLLMResponse extracts score, explanation, confidence from raw response
func parseLLMResponse(rawResp string) (float64, string, float64, error) {
	var resp struct {
		Score       float64 `json:"score"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}
	err := json.Unmarshal([]byte(rawResp), &resp)
	if err != nil {
		return 0, "", 0, err
	}
	return resp.Score, resp.Explanation, resp.Confidence, nil
}

// EnsembleAnalyze performs multi-model, multi-prompt ensemble analysis
func (c *LLMClient) EnsembleAnalyze(content string) (*db.LLMScore, error) {
	models := []string{"gpt-3.5", "gpt-4", "claude", "finetuned"}
	promptVariants := loadPromptVariants()

	type SubResult struct {
		Model         string  `json:"model"`
		PromptVariant string  `json:"prompt_variant"`
		Score         float64 `json:"score"`
		Explanation   string  `json:"explanation"`
		Confidence    float64 `json:"confidence"`
		RawResponse   string  `json:"raw_response"`
	}

	var subResults []SubResult

	for _, model := range models {
		for _, pv := range promptVariants {
			score, explanation, confidence, rawResp, err := callLLM(model, pv, content)
			if err != nil {
				continue
			}
			subResults = append(subResults, SubResult{
				Model: model, PromptVariant: pv.ID,
				Score: score, Explanation: explanation,
				Confidence: confidence, RawResponse: rawResp,
			})
		}
	}

	if len(subResults) == 0 {
		return nil, fmt.Errorf("no valid LLM responses")
	}

	// Aggregate
	var sum, weightedSum, sumWeights float64
	scores := make([]float64, len(subResults))
	confidences := make([]float64, len(subResults))

	for i, r := range subResults {
		scores[i] = r.Score
		confidences[i] = r.Confidence
		sum += r.Score
		weightedSum += r.Score * r.Confidence
		sumWeights += r.Confidence
	}

	mean := sum / float64(len(scores))
	weightedMean := weightedSum / math.Max(sumWeights, 1e-6)

	var varianceSum float64
	for _, s := range scores {
		diff := s - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(scores))

	uncertaintyFlag := variance > 0.1 || (sumWeights/float64(len(scores)) < 0.5)

	meta := map[string]interface{}{
		"sub_results": subResults,
		"aggregation": map[string]interface{}{
			"mean":             mean,
			"weighted_mean":    weightedMean,
			"variance":         variance,
			"uncertainty_flag": uncertaintyFlag,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
	metaBytes, _ := json.Marshal(meta)

	return &db.LLMScore{
		Model:     "ensemble",
		Score:     weightedMean,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}, nil
}

// loadPromptVariants returns hardcoded prompt variants (replace with config later)
func loadPromptVariants() []PromptVariant {
	return []PromptVariant{
		{
			ID:       "default",
			Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) to 1.0 (strongly right). Respond with a JSON object containing 'score', 'explanation', and 'confidence'.",
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
			},
		},
	}
}

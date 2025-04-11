package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

const (
	LabelUnknown = "unknown"
	LabelLeft    = "left"
	LabelRight   = "right"
	LabelNeutral = "neutral"
)

type ModelConfig struct {
	Perspective string `json:"perspective"`
	ModelName   string `json:"modelName"`
	URL         string `json:"url"`
}

type CompositeScoreConfig struct {
	Formula          string             `json:"formula"`
	Weights          map[string]float64 `json:"weights"`
	MinScore         float64            `json:"min_score"`
	MaxScore         float64            `json:"max_score"`
	DefaultMissing   float64            `json:"default_missing"`
	HandleInvalid    string             `json:"handle_invalid"`
	ConfidenceMethod string             `json:"confidence_method"`
	MinConfidence    float64            `json:"min_confidence"`
	MaxConfidence    float64            `json:"max_confidence"`
	Models           []ModelConfig      `json:"models"`
}

var (
	compositeScoreConfig     *CompositeScoreConfig
	compositeScoreConfigOnce sync.Once
)

func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	var err error
	compositeScoreConfigOnce.Do(func() {
		f, e := os.Open("configs/composite_score_config.json")
		if e != nil {
			err = e
			return
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		var cfg CompositeScoreConfig
		if e := decoder.Decode(&cfg); e != nil {
			err = e
			return
		}
		compositeScoreConfig = &cfg
	})
	return compositeScoreConfig, err
}

// Returns (compositeScore, confidence, error)
func ComputeCompositeScoreWithConfidence(scores []db.LLMScore) (float64, float64, error) {
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return 0, 0, err
	}

	// Map for left/center/right
	scoreMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}
	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)
	for _, s := range scores {
		model := strings.ToLower(s.Model)
		if model == LabelLeft || model == "left" {
			model = "left"
		} else if model == LabelRight || model == "right" {
			model = "right"
		} else if model == "center" {
			model = "center"
		} else {
			continue
		}
		val := s.Score
		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
			continue
		}
		if isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore {
			val = cfg.DefaultMissing
		}
		scoreMap[model] = &val
		validCount++
		validModels[model] = true
	}

	// Use weights if formula is weighted
	for k, v := range scoreMap {
		score := cfg.DefaultMissing
		if v != nil {
			score = *v
		}
		w := 1.0
		if cfg.Formula == "weighted" {
			if weight, ok := cfg.Weights[k]; ok {
				w = weight
			}
		}
		weightedSum += score * w
		weightTotal += w
		sum += score
	}

	var composite float64
	switch cfg.Formula {
	case "average":
		composite = sum / 3.0
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			composite = 0.0
		}
	case "min":
		composite = minNonNil(scoreMap, cfg.DefaultMissing)
	case "max":
		composite = maxNonNil(scoreMap, cfg.DefaultMissing)
	default:
		composite = sum / 3.0
	}

	// Composite favors balance (closer to center = higher score)
	composite = 1.0 - abs(composite)

	// Confidence metric
	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		confidence = float64(len(validModels)) / 3.0
	case "spread":
		confidence = 1.0 - scoreSpread(scoreMap)
	default:
		confidence = float64(len(validModels)) / 3.0
	}
	if confidence < cfg.MinConfidence {
		confidence = cfg.MinConfidence
	}
	if confidence > cfg.MaxConfidence {
		confidence = cfg.MaxConfidence
	}

	return composite, confidence, nil
}

func ComputeCompositeScore(scores []db.LLMScore) float64 {
	score, _, _ := ComputeCompositeScoreWithConfidence(scores)
	return score
}

// Helper functions
func isInvalid(f float64) bool {
	return (f != f) || (f > 1e10) || (f < -1e10) // NaN or extreme values
}

func minNonNil(m map[string]*float64, def float64) float64 {
	min := def
	first := true
	for _, v := range m {
		if v != nil {
			if first || *v < min {
				min = *v
				first = false
			}
		}
	}
	return min
}

func maxNonNil(m map[string]*float64, def float64) float64 {
	max := def
	first := true
	for _, v := range m {
		if first || (v != nil && *v > max) {
			if v != nil {
				max = *v
			}
			first = false
		}
	}
	return max
}

func scoreSpread(m map[string]*float64) float64 {
	vals := []float64{}
	for _, v := range m {
		if v != nil {
			vals = append(vals, *v)
		}
	}
	if len(vals) < 2 {
		return 0.0
	}
	sort.Float64s(vals)
	return vals[len(vals)-1] - vals[0]
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

type BiasConfig struct {
	Categories          []string            `json:"categories"`
	ConfidenceThreshold float64             `json:"confidence_threshold"`
	KeywordHeuristics   map[string][]string `json:"keyword_heuristics"`
}

type BiasResult struct {
	Category    string  `json:"category"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

var (
	PromptTemplate string
	BiasCfg        BiasConfig
)

func LoadPromptTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func LoadBiasConfig(path string) (BiasConfig, error) {
	var cfg BiasConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)

	return cfg, err
}

func init() {
	var err error

	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	promptPath := os.Getenv("PROMPT_TEMPLATE_PATH")
	if promptPath != "" {
		log.Printf("Using prompt template path from environment: %s", promptPath)
		PromptTemplate, err = LoadPromptTemplate(promptPath)
		if err != nil {
			log.Fatalf("Failed to load prompt template from environment path: %v", err)
		}
	} else {
		PromptTemplate, err = LoadPromptTemplate("configs/prompt_template.txt")
		if err != nil {
			log.Printf("Error loading prompt template from configs/prompt_template.txt: %v", err)
			log.Fatalf("Failed to load prompt template: %v", err)
		}
	}

	BiasCfg, err = LoadBiasConfig("configs/bias_config.json")
	if err != nil {
		log.Printf("Error loading bias config from configs/bias_config.json: %v", err)
		log.Fatalf("Failed to load bias config: %v", err)
	}
}

// LLM API endpoints.
var (
	LeftModelURL   = "https://api.openai.com/v1/chat/completions"
	CenterModelURL = "https://api.openai.com/v1/chat/completions"
	RightModelURL  = "https://api.openai.com/v1/chat/completions"
)

// Cache key: hash(content) + model.
type cacheKey struct {
	ContentHash string
	Model       string
}

type Cache struct {
	mu    sync.RWMutex
	store map[cacheKey]*db.LLMScore
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[cacheKey]*db.LLMScore),
	}
}

func (c *Cache) Get(contentHash, model string) (*db.LLMScore, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	score, ok := c.store[cacheKey{contentHash, model}]

	return score, ok
}

func (c *Cache) Set(contentHash, model string, score *db.LLMScore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[cacheKey{contentHash, model}] = score
}

// Simple hash function (replace with a secure hash in production).
func hashContent(content string) string {
	// For simplicity, use content itself (not recommended for production)
	return content
}

type LLMService interface {
	Analyze(content string) (*db.LLMScore, error)
}

/*
type MockLLMService struct{}

func (m *MockLLMService) Analyze(content string) (*db.LLMScore, error) {
	score := 0.5 // fixed or random score
	metadata := `{"mock": true}`

	return &db.LLMScore{
		Model:    "mock",
		Score:    score,
		Metadata: metadata,
	}, nil
}
*/

type OpenAILLMService struct {
	client *resty.Client
	apiKey string
}

func NewOpenAILLMService(client *resty.Client, apiKey string) *OpenAILLMService {
	return &OpenAILLMService{
		client: client,
		apiKey: apiKey,
	}
}

func (o *OpenAILLMService) AnalyzeWithPrompt(model, prompt, content string) (*db.LLMScore, error) {
	prompt = strings.Replace(prompt, "{{ARTICLE_CONTENT}}", content, 1)

	var resp *resty.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = o.callOpenAI(model, prompt)
		if err == nil && resp != nil && resp.IsSuccess() {
			return o.processOpenAIResponse(resp, content, model)
		}

		log.Printf("OpenAI API call failed (attempt %d): %v", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, errors.New("OpenAI API call failed after retries")
}

func (o *OpenAILLMService) AnalyzeWithModel(model, content string) (*db.LLMScore, error) {
	return o.AnalyzeWithPrompt(model, PromptTemplate, content)
}

// Satisfy LLMService interface

// Satisfy LLMService interface: default to gpt-3.5-turbo
func (o *OpenAILLMService) Analyze(content string) (*db.LLMScore, error) {
	defaultModel := "gpt-3.5-turbo"
	return o.AnalyzeWithPrompt(defaultModel, PromptTemplate, content)
}

func (o *OpenAILLMService) callOpenAI(model string, prompt string) (*resty.Response, error) {
	url := "https://api.openai.com/v1/chat/completions"

	req := o.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+o.apiKey)

	body := map[string]interface{}{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  300,
		"temperature": 0.7,
	}
	req.SetBody(body)

	log.Printf("[OpenAI] Sending real API request to %s with model %s", url, model)

	resp, err := req.Post(url)

	if err != nil {
		log.Printf("[OpenAI] API request error: %v", err)
		return nil, err
	}

	log.Printf("[OpenAI] API response status: %d", resp.StatusCode())
	log.Printf("[OpenAI] API response body: %.500s", resp.String())

	if !resp.IsSuccess() {
		return resp, errors.New("API response not successful")
	}

	return resp, nil
}

func (o *OpenAILLMService) processOpenAIResponse(resp *resty.Response, content string, model string) (*db.LLMScore, error) {
	contentResp, err := parseOpenAIResponse(resp.Body())
	if err != nil {
		log.Printf("[Analyze] Failed to parse OpenAI response: %v\nRaw response:\n%s", err, string(resp.Body()))
		return nil, err
	}

	log.Printf("[Analyze] Successful raw completion from model %s:\n%s", model, contentResp)

	biasResult := parseBiasResult(contentResp)
	heuristicCat := heuristicCategory(content)

	uncertain := false
	if biasResult.Category == LabelUnknown || biasResult.Confidence < 0.3 {
		uncertain = true
	}

	metadataMap := map[string]interface{}{
		"raw_response":       contentResp,
		"parsed_bias":        biasResult,
		"heuristic_category": heuristicCat,
		"uncertain":          uncertain,
	}
	metadataBytes, err := json.Marshal(metadataMap)
	if err != nil {
		log.Printf("Failed to marshal metadata: %v", err)
	}

	return &db.LLMScore{
		Model:    model,
		Score:    biasResult.Confidence,
		Metadata: string(metadataBytes),
	}, nil
}

func (o *OpenAILLMService) RobustAnalyze(content string) (*db.LLMScore, error) {
	var (
		validScores           []*db.LLMScore
		attempts              int
		maxValidScores        = 5
		maxAttemptsPerVariant = 3
		confidenceThreshold   = 0.5
	)

	// Define prompt variants
	promptVariants := []string{
		PromptTemplate,
		`Respond ONLY in this strict JSON format: {"parsed_bias":{"Category":"...", "Confidence":...}}. Article: {{ARTICLE_CONTENT}}`,
		`Return JSON: {"parsed_bias":{"Category":"...", "Confidence":...}}. Text: {{ARTICLE_CONTENT}}`,
	}

	// Define models: primary and fallback
	models := []string{"gpt-4", "gpt-3.5-turbo"}

	for _, modelName := range models {

		for variantIdx, promptTemplate := range promptVariants {
			for attempt := 1; attempt <= maxAttemptsPerVariant; attempt++ {
				attempts++

				score, err := o.AnalyzeWithPrompt(modelName, promptTemplate, content)

				failType := ""
				confidence := 0.0

				if err != nil || score == nil {
					failType = "api_error"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Error: %v", attempts, modelName, variantIdx, failType, err)
					continue
				}

				var meta struct {
					ParsedBias struct {
						Category   string  `json:"Category"`
						Confidence float64 `json:"Confidence"`
					} `json:"parsed_bias"`
				}
				if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
					failType = "parse_error"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Error: %v", attempts, modelName, variantIdx, failType, err)
					continue
				}

				cat := strings.ToLower(strings.TrimSpace(meta.ParsedBias.Category))
				confidence = meta.ParsedBias.Confidence

				if cat == "" || cat == LabelUnknown {
					failType = "empty_category"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Category: '%s'", attempts, modelName, variantIdx, failType, cat)
					continue
				}

				if confidence < confidenceThreshold {
					failType = "low_confidence"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Confidence: %.2f", attempts, modelName, variantIdx, failType, confidence)
					continue
				}

				// Success
				log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Success | Category: '%s' | Confidence: %.2f", attempts, modelName, variantIdx, cat, confidence)
				validScores = append(validScores, score)

				if len(validScores) >= maxValidScores {
					break
				}
			}
			if len(validScores) >= maxValidScores {
				break
			}
		}
		if len(validScores) >= maxValidScores {
			break
		}
	}

	if len(validScores) < maxValidScores {
		return nil, fmt.Errorf("RobustAnalyze: only %d valid responses after %d attempts", len(validScores), attempts)
	}

	// Extract scores
	scores := make([]float64, len(validScores))
	for i, s := range validScores {
		scores[i] = s.Score
	}

	sort.Float64s(scores)

	// Average middle scores (trimmed mean)
	start := 1
	end := len(scores) - 2
	if end <= start {
		start = 0
		end = len(scores) - 1
	}
	sum := 0.0
	count := 0
	for i := start; i <= end; i++ {
		sum += scores[i]
		count++
	}
	avg := sum / float64(count)

	// Use metadata from median score
	medianIdx := len(validScores) / 2
	medianScore := validScores[medianIdx]

	return &db.LLMScore{
		Model:    "", // model info unavailable here
		Score:    avg,
		Metadata: medianScore.Metadata,
	}, nil
}

func parseOpenAIResponse(body []byte) (string, error) {
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &openaiResp); err == nil {
		if len(openaiResp.Choices) == 0 {
			return "", errors.New("no choices in OpenAI response")
		}
		return openaiResp.Choices[0].Message.Content, nil
	}

	// If parsing as chat completion failed, try parsing as error response
	var openaiErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Param   string `json:"param"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err2 := json.Unmarshal(body, &openaiErr); err2 == nil && openaiErr.Error.Message != "" {
		return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)", openaiErr.Error.Message, openaiErr.Error.Type, openaiErr.Error.Code)
	}

	// Log raw response for debugging
	log.Printf("Failed to parse OpenAI response as chat completion or error.\nRaw response:\n%s", string(body))
	return "", errors.New("invalid OpenAI API response format")
}

func parseBiasResult(contentResp string) BiasResult {
	var biasResult BiasResult

	// Extract JSON strictly between triple backticks
	re := regexp.MustCompile("(?s)```(.*?)```")
	matches := re.FindStringSubmatch(contentResp)
	if len(matches) < 2 {
		log.Printf("No JSON block found between triple backticks in LLM response")
		return BiasResult{}
	}
	jsonStr := matches[1]

	if err := json.Unmarshal([]byte(jsonStr), &biasResult); err != nil {
		log.Printf("Failed to parse JSON inside triple backticks: %v", err)
		biasResult = BiasResult{}
	}

	// Validate category
	validCategory := false
	for _, cat := range BiasCfg.Categories {
		if biasResult.Category == cat {
			validCategory = true
			break
		}
	}
	if !validCategory {
		biasResult.Category = LabelUnknown
	}

	// Validate confidence
	if biasResult.Confidence < 0 || biasResult.Confidence > 1 {
		biasResult.Confidence = 0
	}

	return biasResult
}

func heuristicCategory(content string) string {
	contentLower := strings.ToLower(content)

	for cat, keywords := range BiasCfg.KeywordHeuristics {
		for _, kw := range keywords {
			if strings.Contains(contentLower, strings.ToLower(kw)) {
				return cat
			}
		}
	}

	return LabelUnknown
}

type LLMClient struct {
	client     *resty.Client
	cache      *Cache
	db         *sqlx.DB
	llmService LLMService
}

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	client := resty.New()
	cache := NewCache()

	provider := os.Getenv("LLM_PROVIDER")

	var service LLMService

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("ERROR: OPENAI_API_KEY not set, cannot use OpenAI LLM provider")
		}

		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-3.5-turbo"
		}

		service = NewOpenAILLMService(client, apiKey)
	default:
		log.Fatal("ERROR: LLM_PROVIDER not set or unknown, cannot initialize LLM service")
	}

	return &LLMClient{
		client:     client,
		cache:      cache,
		db:         dbConn,
		llmService: service,
	}
}

func (c *LLMClient) analyzeContent(articleID int64, content string, model string, _ string) (*db.LLMScore, error) {
	contentHash := hashContent(content)

	// Check cache
	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	var score *db.LLMScore
	var err error

	// Use only the general/default prompt variant to limit to 1 API call per model
	generalPrompt := PromptVariant{
		ID: "default",
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
			"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
		Examples: []string{
			`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
			`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
			`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
		},
	}

	scoreVal, explanation, confidence, _, err := c.callLLM(articleID, model, generalPrompt, content)
	if err != nil {
		return nil, err
	}

	meta := fmt.Sprintf(`{"explanation": %q, "confidence": %.3f}`, explanation, confidence)

	score = &db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     scoreVal,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}

	score.ArticleID = articleID
	score.Model = model
	score.CreatedAt = time.Now()

	// Cache it
	c.cache.Set(contentHash, model, score)

	return score, nil
}

func (c *LLMClient) ProcessUnscoredArticles() error {
	query := `
	SELECT a.* FROM articles a
	WHERE NOT EXISTS (
		SELECT 1 FROM llm_scores s
		WHERE s.article_id = a.id
	)
	`
	var articles []db.Article
	if err := c.db.Select(&articles, query); err != nil {
		return err
	}

	for _, article := range articles {
		if err := c.AnalyzeAndStore(&article); err != nil {
			log.Printf("Failed to analyze article ID %d: %v", article.ID, err)
		}
	}

	return nil
}

func (c *LLMClient) AnalyzeAndStore(article *db.Article) error {
	models := []struct {
		perspective string
		modelName   string
		url         string
	}{
		{LabelLeft, "gpt-3.5-turbo", LeftModelURL},
		{"center", "gpt-4", CenterModelURL},
		{LabelRight, "gpt-3.5-turbo", RightModelURL},
	}

	for _, m := range models {
		log.Printf("[DEBUG][AnalyzeAndStore] Article %d | Perspective: %s | ModelName passed: %s | URL: %s", article.ID, m.perspective, m.modelName, m.url)
		score, err := c.analyzeContent(article.ID, article.Content, m.modelName, m.url)
		if err != nil {
			log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.modelName, err)

			continue
		}

		_, err = db.InsertLLMScore(c.db, score)
		if err != nil {
			log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.modelName, err)
		}
	}

	return nil
}

func (c *LLMClient) ReanalyzeArticle(articleID int64) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	// Delete existing scores
	_, err = tx.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	var article db.Article

	err = tx.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	models := []struct {
		name string
		url  string
	}{
		{LabelLeft, LeftModelURL},
		{"center", CenterModelURL},
		{LabelRight, RightModelURL},
	}

	for _, m := range models {
		score, err := c.analyzeContent(article.ID, article.Content, m.name, m.url)
		if err != nil {
			log.Printf("Error reanalyzing article %d with model %s: %v", article.ID, m.name, err)
			continue
		}

		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata)
			VALUES (:article_id, :model, :score, :metadata)`, score)
		if err != nil {
			log.Printf("Error inserting reanalysis score for article %d model %s: %v", article.ID, m.name, err)
		}
	}

	// Call ensemble aggregation and save result
	ensembleScore, err := c.EnsembleAnalyze(article.ID, article.Content)
	if err != nil {
		log.Printf("Error generating ensemble score for article %d: %v", article.ID, err)
	} else {
		ensembleScore.ArticleID = article.ID
		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
			VALUES (:article_id, :model, :score, :metadata, :created_at)`, ensembleScore)
		if err != nil {
			log.Printf("Error inserting ensemble score for article %d: %v", article.ID, err)
		}

		// Parse ensemble metadata to extract variance and compute confidence
		var metaMap map[string]interface{}
		if err := json.Unmarshal([]byte(ensembleScore.Metadata), &metaMap); err != nil {
			log.Printf("Error parsing ensemble metadata for article %d: %v", article.ID, err)
		} else {
			confidence := 0.0
			if finalAgg, ok := metaMap["final_aggregation"].(map[string]interface{}); ok {
				if varianceVal, ok := finalAgg["variance"].(float64); ok {
					confidence = 1.0 - varianceVal
					if confidence < 0 {
						confidence = 0
					}
					if confidence > 1 {
						confidence = 1
					}
				}
			}
			err = db.UpdateArticleScore(c.db, article.ID, ensembleScore.Score, confidence)
		}
	}
	return nil
}

// Exported wrapper for analyzeContent to allow external packages to call it
func (c *LLMClient) AnalyzeContent(articleID int64, content string, model string, url string) (*db.LLMScore, error) {
	return c.analyzeContent(articleID, content, model, url)
}

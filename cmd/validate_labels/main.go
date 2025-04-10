package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/jmoiron/sqlx"
)

const (
	LabelLeft    = "left"
	LabelRight   = "right"
	LabelNeutral = "neutral"
)

type Metrics struct {
	Total           int
	Correct         int
	Incorrect       int
	Uncertain       int
	Disagreements   int
	Accuracy        float64
	Precision       float64
	Recall          float64
	F1              float64
	ConfusionMatrix map[string]map[string]int
	Timestamp       string
}

// FlaggedCase holds info about uncertain/disagreement samples
type FlaggedCase struct {
	ID             int64   `json:"id"`
	Text           string  `json:"text"`
	TrueLabel      string  `json:"true_label"`
	PredictedLabel string  `json:"predicted_label"`
	Score          float64 `json:"score"`
	Uncertain      bool    `json:"uncertain"`
	Disagreement   bool    `json:"disagreement"`
	ErrorCategory  string  `json:"error_category"` // prompt_issue, model_failure, data_noise, or empty
}

func main() {
	dbPath := flag.String("db", "news.db", "Path to SQLite database")
	flag.Parse()

	database, client := initDBAndClient(*dbPath)
	labels := fetchLabels(database)

	log.Printf("Processing %d labeled samples...", len(labels))

	metrics, flaggedCases := processLabels(database, client, labels)

	computeMetrics(&metrics)

	saveAndPrintResults(metrics)

	saveAllFlaggedCases(flaggedCases)

	sampleAndSaveFlaggedCases(flaggedCases)
}

func initDBAndClient(dbPath string) (*sqlx.DB, *llm.LLMClient) {
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	client := &llm.LLMClient{}
	return database, client
}

func fetchLabels(database *sqlx.DB) []db.Label {
	var labels []db.Label
	err := database.Select(&labels, "SELECT * FROM labels")
	if err != nil {
		log.Fatalf("Failed to fetch labels: %v", err)
	}
	return labels
}

func processLabels(database *sqlx.DB, client *llm.LLMClient, labels []db.Label) (Metrics, []FlaggedCase) {
	metrics := Metrics{
		ConfusionMatrix: map[string]map[string]int{
			LabelLeft:    {LabelLeft: 0, LabelRight: 0, LabelNeutral: 0},
			LabelRight:   {LabelLeft: 0, LabelRight: 0, LabelNeutral: 0},
			LabelNeutral: {LabelLeft: 0, LabelRight: 0, LabelNeutral: 0},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	flaggedCases := []FlaggedCase{}

	for _, label := range labels {
		scoreObj, err := client.EnsembleAnalyze(label.Data)
		if err != nil {
			log.Printf("Error scoring label ID %d: %v", label.ID, err)
			continue
		}

		scoreObj.ArticleID = 0
		scoreObj.Model = "validation_ensemble"
		_, err = db.InsertLLMScore(database, scoreObj)
		if err != nil {
			log.Printf("Failed to insert ensemble score: %v", err)
		}

		var meta struct {
			Aggregation struct {
				UncertaintyFlag bool `json:"uncertainty_flag"`
			} `json:"aggregation"`
		}
		if err := json.Unmarshal([]byte(scoreObj.Metadata), &meta); err != nil {
			log.Printf("Failed to parse score metadata: %v", err)
			continue
		}
		isUncertain := meta.Aggregation.UncertaintyFlag
		if isUncertain {
			metrics.Uncertain++
		}

		predLabel := scoreToLabel(scoreObj.Score)
		trueLabel := normalizeLabel(label.Label)
		disagreement := false
		if predLabel == trueLabel {
			metrics.Correct++
		} else {
			metrics.Incorrect++
			metrics.Disagreements++
			disagreement = true
		}

		if isUncertain || disagreement {
			flaggedCases = append(flaggedCases, FlaggedCase{
				ID:             label.ID,
				Text:           label.Data,
				TrueLabel:      trueLabel,
				PredictedLabel: predLabel,
				Score:          scoreObj.Score,
				Uncertain:      isUncertain,
				Disagreement:   disagreement,
				ErrorCategory:  "",
			})
		}

		if _, ok := metrics.ConfusionMatrix[trueLabel]; ok {
			if _, ok2 := metrics.ConfusionMatrix[trueLabel][predLabel]; ok2 {
				metrics.ConfusionMatrix[trueLabel][predLabel]++
			}
		}

		metrics.Total++
	}

	return metrics, flaggedCases
}

func computeMetrics(metrics *Metrics) {
	metrics.Accuracy = float64(metrics.Correct) / math.Max(float64(metrics.Total), 1)

	var tp, fp, fn float64
	for trueLbl, preds := range metrics.ConfusionMatrix {
		for predLbl, count := range preds {
			if predLbl != LabelNeutral && trueLbl != LabelNeutral {
				if predLbl == trueLbl {
					tp += float64(count)
				} else {
					fp += float64(count)
					fn += float64(count)
				}
			} else if predLbl != LabelNeutral && trueLbl == LabelNeutral {
				fp += float64(count)
			} else if predLbl == LabelNeutral && trueLbl != LabelNeutral {
				fn += float64(count)
			}
		}
	}
	metrics.Precision = tp / math.Max(tp+fp, 1)
	metrics.Recall = tp / math.Max(tp+fn, 1)
	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1 = 2 * metrics.Precision * metrics.Recall / (metrics.Precision + metrics.Recall)
	}
}

func saveAndPrintResults(metrics Metrics) {
	saveMetrics(metrics)

	fmt.Printf("Validation completed on %d samples\n", metrics.Total)
	fmt.Printf("Accuracy: %.3f\n", metrics.Accuracy)
	fmt.Printf("Precision: %.3f\n", metrics.Precision)
	fmt.Printf("Recall: %.3f\n", metrics.Recall)
	fmt.Printf("F1 Score: %.3f\n", metrics.F1)
	fmt.Printf("Uncertain cases: %d\n", metrics.Uncertain)
	fmt.Printf("Disagreements: %d\n", metrics.Disagreements)
	fmt.Printf("Confusion Matrix: %+v\n", metrics.ConfusionMatrix)
}

func saveAllFlaggedCases(flaggedCases []FlaggedCase) {
	saveFlaggedCases(flaggedCases, "flagged_cases")
}

func sampleAndSaveFlaggedCases(flaggedCases []FlaggedCase) {
	sampleSize := int(math.Max(1, math.Round(0.1*float64(len(flaggedCases)))))
	if sampleSize > len(flaggedCases) {
		sampleSize = len(flaggedCases)
	}
	sampled := make([]FlaggedCase, 0, sampleSize)
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	perm := r.Perm(len(flaggedCases))
	for i := 0; i < sampleSize; i++ {
		sampled = append(sampled, flaggedCases[perm[i]])
	}
	saveFlaggedCases(sampled, "sampled_flagged_cases")
}

func saveFlaggedCases(cases []FlaggedCase, prefix string) {
	if len(cases) == 0 {
		return
	}
	fname := fmt.Sprintf("%s_%s.json", prefix, time.Now().Format("20060102_150405"))
	f, err := os.Create(fname)
	if err != nil {
		log.Printf("Failed to create %s file: %v", prefix, err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(cases)
	if err != nil {
		log.Printf("Failed to write %s: %v", prefix, err)
	}
}

func scoreToLabel(score float64) string {
	if score < -0.33 {
		return LabelLeft
	} else if score > 0.33 {
		return LabelRight
	}
	return LabelNeutral
}

func normalizeLabel(label string) string {
	switch label {
	case "Left", "left", "-1", "-1.0":
		return LabelLeft
	case "Right", "right", "1", "1.0":
		return LabelRight
	case "Neutral", "neutral", "0", "0.0":
		return LabelNeutral
	default:
		return "neutral"
	}
}

func saveMetrics(metrics Metrics) {
	fname := fmt.Sprintf("validation_log_%s.json", time.Now().Format("20060102_150405"))
	f, err := os.Create(fname)
	if err != nil {
		log.Printf("Failed to create metrics log file: %v", err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(metrics)
	if err != nil {
		log.Printf("Failed to write metrics log: %v", err)
	}
}

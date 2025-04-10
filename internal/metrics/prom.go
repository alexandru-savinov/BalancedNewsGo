package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	LLMRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests",
		},
		[]string{"model", "prompt_hash"},
	)

	LLMFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_failures_total",
			Help: "Total number of LLM failures by type",
		},
		[]string{"model", "prompt_hash", "failure_type"},
	)

	LLMFailureStreak = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llm_failure_streak",
			Help: "Consecutive LLM failure streaks",
		},
		[]string{"model", "prompt_hash"},
	)
)

func InitLLMMetrics() {
	prometheus.MustRegister(LLMRequestsTotal)
	prometheus.MustRegister(LLMFailuresTotal)
	prometheus.MustRegister(LLMFailureStreak)
}

func IncLLMRequest(model, promptHash string) {
	LLMRequestsTotal.WithLabelValues(model, promptHash).Inc()
}

func IncLLMFailure(model, promptHash, failureType string) {
	LLMFailuresTotal.WithLabelValues(model, promptHash, failureType).Inc()
}

func SetFailureStreak(model, promptHash string, count int) {
	LLMFailureStreak.WithLabelValues(model, promptHash).Set(float64(count))
}

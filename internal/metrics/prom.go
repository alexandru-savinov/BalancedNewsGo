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

	// New metrics for specific LLM error types
	LLMRateLimitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_rate_limit_total",
			Help: "Total number of rate limit errors from OpenRouter",
		},
	)

	LLMAuthFailureCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_auth_failure_total",
			Help: "Total number of authentication failures with OpenRouter",
		},
	)

	LLMCreditsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_credits_exhausted_total",
			Help: "Total number of credit exhaustion errors from OpenRouter",
		},
	)

	LLMStreamingErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_streaming_errors_total",
			Help: "Total number of streaming-related errors from OpenRouter",
		},
	)
)

func InitLLMMetrics() {
	prometheus.MustRegister(LLMRequestsTotal)
	prometheus.MustRegister(LLMFailuresTotal)
	prometheus.MustRegister(LLMFailureStreak)

	// Register new error-specific metrics
	prometheus.MustRegister(LLMRateLimitCounter)
	prometheus.MustRegister(LLMAuthFailureCounter)
	prometheus.MustRegister(LLMCreditsCounter)
	prometheus.MustRegister(LLMStreamingErrors)
}

func IncLLMRequest(model, promptHash string) {
	LLMRequestsTotal.WithLabelValues(model, promptHash).Inc()
}

func IncLLMFailure(model, promptHash, failureType string) {
	LLMFailuresTotal.WithLabelValues(model, promptHash, failureType).Inc()

	// Also increment specific counters based on failure type
	switch failureType {
	case "rate_limit":
		IncLLMRateLimit()
	case "authentication":
		IncLLMAuthFailure()
	case "credits":
		IncLLMCreditsExhausted()
	case "streaming":
		IncLLMStreamingError()
	}
}

func SetFailureStreak(model, promptHash string, count int) {
	LLMFailureStreak.WithLabelValues(model, promptHash).Set(float64(count))
}

// Helper functions for incrementing specific error metrics
func IncLLMRateLimit() {
	LLMRateLimitCounter.Inc()
}

func IncLLMAuthFailure() {
	LLMAuthFailureCounter.Inc()
}

func IncLLMCreditsExhausted() {
	LLMCreditsCounter.Inc()
}

func IncLLMStreamingError() {
	LLMStreamingErrors.Inc()
}

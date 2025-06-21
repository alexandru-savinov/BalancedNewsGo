package metrics

import (
	"fmt"
	"time"

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

	// Detailed LLM API error counter with provider, model, and error type dimensions
	LLMAPIErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_api_errors_total",
			Help: "Total number of LLM API errors by provider, model, and error type",
		},
		[]string{"provider", "model", "error_type", "status_code"},
	)

	// New metrics for caching and request monitoring
	CacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	RequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_errors_total",
			Help: "Total number of request errors by type",
		},
		[]string{"error_type"},
	)

	RequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_latency_seconds",
			Help:    "Request latency distribution in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)

	ResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "response_status_total",
			Help: "Total number of responses by status code",
		},
		[]string{"status_code"},
	)

	// Additional metrics for comprehensive monitoring
	ArticlesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "newsbalancer_articles_total",
			Help: "Total number of articles in the system",
		},
		[]string{"status"},
	)

	ArticlesByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "newsbalancer_articles_by_status",
			Help: "Number of articles by analysis status",
		},
		[]string{"status"},
	)

	LLMQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "newsbalancer_llm_queue_size",
			Help: "Current size of the LLM analysis queue",
		},
	)

	LLMQueueProcessing = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "newsbalancer_llm_queue_processing",
			Help: "Number of articles currently being processed by LLM",
		},
	)

	LLMAnalysesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newsbalancer_llm_analyses_total",
			Help: "Total number of LLM analyses completed",
		},
		[]string{"status", "model"},
	)

	LLMRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newsbalancer_llm_request_duration_seconds",
			Help:    "Duration of LLM requests in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"model", "provider"},
	)

	LLMConfidenceScore = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newsbalancer_llm_confidence_score",
			Help:    "Distribution of LLM confidence scores",
			Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"model"},
	)

	BiasScoreBucket = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newsbalancer_bias_score_bucket",
			Help:    "Distribution of bias scores",
			Buckets: []float64{-1.0, -0.8, -0.6, -0.4, -0.2, 0.0, 0.2, 0.4, 0.6, 0.8, 1.0},
		},
		[]string{"perspective"},
	)

	DatabaseConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "newsbalancer_db_connections",
			Help: "Number of database connections",
		},
		[]string{"state"}, // active, idle
	)

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint"},
	)

	LLMErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newsbalancer_llm_errors_total",
			Help: "Total number of LLM errors by type",
		},
		[]string{"type", "model"},
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

	// Register detailed LLM API errors metric
	prometheus.MustRegister(LLMAPIErrorsTotal)

	// Register new metrics
	prometheus.MustRegister(CacheHits)
	prometheus.MustRegister(CacheMisses)
	prometheus.MustRegister(RequestErrors)
	prometheus.MustRegister(RequestLatency)
	prometheus.MustRegister(ResponseStatus)

	// Register comprehensive monitoring metrics
	prometheus.MustRegister(ArticlesTotal)
	prometheus.MustRegister(ArticlesByStatus)
	prometheus.MustRegister(LLMQueueSize)
	prometheus.MustRegister(LLMQueueProcessing)
	prometheus.MustRegister(LLMAnalysesTotal)
	prometheus.MustRegister(LLMRequestDuration)
	prometheus.MustRegister(LLMConfidenceScore)
	prometheus.MustRegister(BiasScoreBucket)
	prometheus.MustRegister(DatabaseConnections)
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(LLMErrorsTotal)
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

// IncLLMAPIError increments the detailed LLM API error counter
func IncLLMAPIError(provider, model, errorType string, statusCode int) {
	statusCodeStr := fmt.Sprintf("%d", statusCode)
	LLMAPIErrorsTotal.WithLabelValues(provider, model, errorType, statusCodeStr).Inc()
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

// New helper functions for cache and request metrics
func RecordCacheHit() {
	CacheHits.Inc()
}

func RecordCacheMiss() {
	CacheMisses.Inc()
}

func RecordError(errorType string) {
	RequestErrors.WithLabelValues(errorType).Inc()
}

func RecordLatency(duration time.Duration) {
	RequestLatency.WithLabelValues("", "").Observe(duration.Seconds())
}

func RecordStatus(statusCode int) {
	ResponseStatus.WithLabelValues(fmt.Sprintf("%d", statusCode)).Inc()
}

// Helper functions for new comprehensive metrics
func SetArticleCount(status string, count int) {
	ArticlesTotal.WithLabelValues(status).Set(float64(count))
}

func SetArticlesByStatus(status string, count int) {
	ArticlesByStatus.WithLabelValues(status).Set(float64(count))
}

func SetLLMQueueSize(size int) {
	LLMQueueSize.Set(float64(size))
}

func SetLLMQueueProcessing(count int) {
	LLMQueueProcessing.Set(float64(count))
}

func IncLLMAnalysis(status, model string) {
	LLMAnalysesTotal.WithLabelValues(status, model).Inc()
}

func RecordLLMRequestDuration(duration time.Duration, model, provider string) {
	LLMRequestDuration.WithLabelValues(model, provider).Observe(duration.Seconds())
}

func RecordLLMConfidenceScore(score float64, model string) {
	LLMConfidenceScore.WithLabelValues(model).Observe(score)
}

func RecordBiasScore(score float64, perspective string) {
	BiasScoreBucket.WithLabelValues(perspective).Observe(score)
}

func SetDatabaseConnections(state string, count int) {
	DatabaseConnections.WithLabelValues(state).Set(float64(count))
}

func RecordHTTPRequest(method, endpoint, status string) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

func RecordHTTPDuration(duration time.Duration, method, endpoint string) {
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func IncLLMError(errorType, model string) {
	LLMErrorsTotal.WithLabelValues(errorType, model).Inc()
}

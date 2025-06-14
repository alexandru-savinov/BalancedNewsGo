package llm

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// SSEEvents represents a collection of SSE events
type SSEEvents []SSEEvent

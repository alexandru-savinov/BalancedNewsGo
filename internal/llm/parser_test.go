package llm

import (
	"testing"
)

func TestParseNestedLLMJSONResponse_Valid(t *testing.T) {
	raw := `{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"ok\",\"confidence\":0.8}"}}]}`
	score, exp, conf, err := parseNestedLLMJSONResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0.5 {
		t.Errorf("expected score 0.5, got %v", score)
	}
	if exp != "ok" {
		t.Errorf("expected explanation 'ok', got %v", exp)
	}
	if conf != 0.8 {
		t.Errorf("expected confidence 0.8, got %v", conf)
	}
}

func TestParseNestedLLMJSONResponse_Backticks(t *testing.T) {
	// Use double-quoted literal to include backticks
	raw := "{\"choices\":[{\"message\":{\"content\":\"```json {\\\"score\\\":1.0,\\\"explanation\\\":\\\"text\\\",\\\"confidence\\\":0.9} ```\"}}]}"
	score, exp, conf, err := parseNestedLLMJSONResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 1.0 {
		t.Errorf("expected score 1.0, got %v", score)
	}
	if exp != "text" {
		t.Errorf("expected explanation 'text', got %v", exp)
	}
	if conf != 0.9 {
		t.Errorf("expected confidence 0.9, got %v", conf)
	}
}

func TestParseNestedLLMJSONResponse_NoChoices(t *testing.T) {
	raw := `{"choices":[]}`
	_, _, _, err := parseNestedLLMJSONResponse(raw)
	if err == nil {
		t.Error("expected error for no choices, got nil")
	}
}

func TestParseNestedLLMJSONResponse_InvalidOuterJSON(t *testing.T) {
	_, _, _, err := parseNestedLLMJSONResponse(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

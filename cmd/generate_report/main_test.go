package main

import "testing"

func TestGetReportBaseURLDefault(t *testing.T) {
	t.Setenv("REPORT_BASE_URL", "")
	got := getReportBaseURL()
	if got != "http://localhost:8080" {
		t.Errorf("expected default URL, got %s", got)
	}
}

func TestGetReportBaseURLEnv(t *testing.T) {
	t.Setenv("REPORT_BASE_URL", "http://example.com")
	got := getReportBaseURL()
	if got != "http://example.com" {
		t.Errorf("expected env URL, got %s", got)
	}
}

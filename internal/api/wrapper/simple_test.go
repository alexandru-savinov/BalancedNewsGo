package client

import (
	"testing"
)

// TestSimple is a basic test to verify the test framework is working
func TestSimple(t *testing.T) {
	t.Log("Simple test is working")
}

// TestNewAPIClientSimple tests basic client creation
func TestNewAPIClientSimple(t *testing.T) {
	client := NewAPIClient("http://localhost:8080")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}

	if client.cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected BaseURL to be 'http://localhost:8080', got '%s'", client.cfg.BaseURL)
	}

	t.Log("NewAPIClient working correctly")
}

// TestBuildCacheKeySimple tests the cache key generation
func TestBuildCacheKeySimple(t *testing.T) {
	key1 := buildCacheKey("test", "value1", "value2")
	key2 := buildCacheKey("test", "value1", "value3")

	if key1 == key2 {
		t.Error("Different parameters should generate different cache keys")
	}

	if key1 == "" || key2 == "" {
		t.Error("Cache keys should not be empty")
	}

	t.Log("buildCacheKey working correctly")
}

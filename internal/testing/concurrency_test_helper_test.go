package testing

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrencyTestHelper_WithGoroutineLeakDetection(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	// Test that should pass - no goroutine leaks
	helper.WithGoroutineLeakDetection(func() {
		// Simple test that doesn't leak goroutines
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
		wg.Wait()
	})
}

func TestConcurrencyTestHelper_StressTest(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	var counter int64

	helper.StressTest(100, 10, func(iteration int) {
		// This should be safe with atomic operations
		atomic.AddInt64(&counter, 1)
	})

	if atomic.LoadInt64(&counter) != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
}

func TestConcurrencyTestHelper_ConcurrentExecution(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	var results []int
	var mu sync.Mutex

	helper.ConcurrentExecution(
		func() {
			mu.Lock()
			results = append(results, 1)
			mu.Unlock()
		},
		func() {
			mu.Lock()
			results = append(results, 2)
			mu.Unlock()
		},
		func() {
			mu.Lock()
			results = append(results, 3)
			mu.Unlock()
		},
	)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestConcurrencyTestHelper_TimeoutTest(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	// Test that completes within timeout
	helper.TimeoutTest(100*time.Millisecond, func() {
		time.Sleep(10 * time.Millisecond)
	})
}

func TestConcurrencyTestHelper_MemoryPressureTest(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	helper.MemoryPressureTest(func() {
		// Allocate and release memory
		data := make([]byte, 1024*1024) // 1MB
		_ = data
	})
}

func TestConcurrencyTestHelper_ConcurrentMapTest(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	testMap := make(map[string]interface{})
	testMap["test-key"] = 0

	// Run concurrent map operations for a short duration
	helper.ConcurrentMapTest(testMap, 5, 2, 50*time.Millisecond)
}

func TestConcurrencyTestHelper_ChannelTest(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	// Test channel operations with multiple senders and receivers
	helper.ChannelTest(10, 3, 2, 10)
}

func TestConcurrencyTestHelper_DetectDataRaces(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	var safeCounter int64

	helper.DetectDataRaces(50, func() {
		// Use atomic operations to avoid data races
		atomic.AddInt64(&safeCounter, 1)
	})

	if atomic.LoadInt64(&safeCounter) != 50 {
		t.Errorf("Expected counter to be 50, got %d", safeCounter)
	}
}

// Example of testing a potentially problematic concurrent function
func TestConcurrentCounter(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	// Test a counter that uses proper synchronization
	type SafeCounter struct {
		mu    sync.Mutex
		value int
	}

	counter := &SafeCounter{}

	increment := func() {
		counter.mu.Lock()
		counter.value++
		counter.mu.Unlock()
	}

	getValue := func() int {
		counter.mu.Lock()
		defer counter.mu.Unlock()
		return counter.value
	}

	helper.WithGoroutineLeakDetection(func() {
		helper.StressTest(100, 10, func(iteration int) {
			increment()
		})

		if getValue() != 100 {
			t.Errorf("Expected counter value to be 100, got %d", getValue())
		}
	})
}

// Example of testing HTTP client concurrency (without actual HTTP calls)
func TestConcurrentHTTPClientSimulation(t *testing.T) {
	helper := NewConcurrencyTestHelper(t)

	// Define mock HTTP client for this test
	type MockHTTPClient struct {
		inUse bool
		mu    sync.Mutex
	}

	// Simulate HTTP client pool
	type HTTPClientPool struct {
		clients chan *MockHTTPClient
	}

	pool := &HTTPClientPool{
		clients: make(chan *MockHTTPClient, 5),
	}

	// Initialize pool
	for i := 0; i < 5; i++ {
		pool.clients <- &MockHTTPClient{}
	}

	makeRequest := func() {
		client := <-pool.clients
		client.mu.Lock()
		client.inUse = true
		// Simulate work
		time.Sleep(1 * time.Millisecond)
		client.inUse = false
		client.mu.Unlock()
		pool.clients <- client
	}

	helper.WithGoroutineLeakDetection(func() {
		helper.StressTest(20, 10, func(iteration int) {
			makeRequest()
		})
	})
}

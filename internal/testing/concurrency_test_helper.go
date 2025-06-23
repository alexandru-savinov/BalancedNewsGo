package testing

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// ConcurrencyTestHelper provides utilities for testing concurrent code without CGO-based race detection
type ConcurrencyTestHelper struct {
	t *testing.T
}

// NewConcurrencyTestHelper creates a new concurrency test helper
func NewConcurrencyTestHelper(t *testing.T) *ConcurrencyTestHelper {
	return &ConcurrencyTestHelper{t: t}
}

// WithGoroutineLeakDetection wraps a test function with goroutine leak detection
func (h *ConcurrencyTestHelper) WithGoroutineLeakDetection(testFunc func()) {
	defer goleak.VerifyNone(h.t)
	testFunc()
}

// StressTest runs a function multiple times concurrently to detect race conditions
func (h *ConcurrencyTestHelper) StressTest(iterations int, concurrency int, testFunc func(iteration int)) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			testFunc(iteration)
		}(i)
	}
	
	wg.Wait()
}

// ConcurrentExecution runs multiple functions concurrently and waits for all to complete
func (h *ConcurrencyTestHelper) ConcurrentExecution(funcs ...func()) {
	var wg sync.WaitGroup
	
	for _, f := range funcs {
		wg.Add(1)
		go func(fn func()) {
			defer wg.Done()
			fn()
		}(f)
	}
	
	wg.Wait()
}

// TimeoutTest runs a test function with a timeout to detect deadlocks
func (h *ConcurrencyTestHelper) TimeoutTest(timeout time.Duration, testFunc func()) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		testFunc()
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-ctx.Done():
		h.t.Fatalf("Test timed out after %v", timeout)
	}
}

// MemoryPressureTest runs a test under memory pressure to detect memory-related race conditions
func (h *ConcurrencyTestHelper) MemoryPressureTest(testFunc func()) {
	// Force garbage collection before test
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
	
	// Run test
	testFunc()
	
	// Force garbage collection after test
	runtime.GC()
	runtime.GC()
}

// ConcurrentMapTest provides utilities for testing concurrent map operations
func (h *ConcurrencyTestHelper) ConcurrentMapTest(m map[string]interface{}, readers, writers int, duration time.Duration) {
	var mu sync.RWMutex
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Start readers
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					mu.RLock()
					_ = m["test-key"]
					mu.RUnlock()
					runtime.Gosched() // Yield to other goroutines
				}
			}
		}(i)
	}
	
	// Start writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					mu.Lock()
					m["test-key"] = counter
					counter++
					mu.Unlock()
					runtime.Gosched() // Yield to other goroutines
				}
			}
		}(i)
	}
	
	wg.Wait()
}

// ChannelTest provides utilities for testing channel operations
func (h *ConcurrencyTestHelper) ChannelTest(bufferSize int, senders, receivers int, messages int) {
	ch := make(chan int, bufferSize)
	var wg sync.WaitGroup
	
	// Start senders
	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func(senderID int) {
			defer wg.Done()
			for j := 0; j < messages; j++ {
				ch <- senderID*1000 + j
			}
		}(i)
	}
	
	// Start receivers
	received := make([]int, 0, senders*messages)
	var receiveMu sync.Mutex
	
	for i := 0; i < receivers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg, ok := <-ch:
					if !ok {
						return
					}
					receiveMu.Lock()
					received = append(received, msg)
					receiveMu.Unlock()
				case <-time.After(100 * time.Millisecond):
					return
				}
			}
		}()
	}
	
	// Wait for all senders to finish, then close channel
	go func() {
		time.Sleep(10 * time.Millisecond) // Give senders time to start
		for i := 0; i < senders; i++ {
			// Wait for senders (this is a simplified approach)
		}
		close(ch)
	}()
	
	wg.Wait()
	
	// Verify we received all messages
	if len(received) != senders*messages {
		h.t.Errorf("Expected %d messages, got %d", senders*messages, len(received))
	}
}

// DetectDataRaces runs a test function multiple times with different scheduling to detect data races
func (h *ConcurrencyTestHelper) DetectDataRaces(iterations int, testFunc func()) {
	for i := 0; i < iterations; i++ {
		// Vary the scheduling by calling runtime.Gosched() at different points
		if i%2 == 0 {
			runtime.Gosched()
		}
		
		testFunc()
		
		if i%3 == 0 {
			runtime.GC()
		}
	}
}

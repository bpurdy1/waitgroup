package waitgroup

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewLimitWaitGroup(t *testing.T) {
	tests := []struct {
		name          string
		maxConcurrent int
		wantErr       error
	}{
		{
			name:          "valid max concurrent",
			maxConcurrent: 5,
			wantErr:       nil,
		},
		{
			name:          "zero max concurrent",
			maxConcurrent: 0,
			wantErr:       ErrInvalidMaxConcurrent,
		},
		{
			name:          "negative max concurrent",
			maxConcurrent: -1,
			wantErr:       ErrInvalidMaxConcurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wg, err := NewLimitWaitGroup(tt.maxConcurrent)
			if err != tt.wantErr {
				t.Errorf("NewLimitWaitGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && wg == nil {
				t.Error("NewLimitWaitGroup() returned nil WaitGroup with no error")
			}
		})
	}
}

func TestLimitWaitGroup_BasicUsage(t *testing.T) {
	wg, err := NewLimitWaitGroup(3)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var counter int64
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("expected counter = 10, got %d", counter)
	}
}

func TestLimitWaitGroup_ConcurrencyLimit(t *testing.T) {
	maxConcurrent := 3
	wg, err := NewLimitWaitGroup(maxConcurrent)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var currentConcurrent int64
	var maxObserved int64
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			current := atomic.AddInt64(&currentConcurrent, 1)

			mu.Lock()
			if current > maxObserved {
				maxObserved = current
			}
			mu.Unlock()

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			atomic.AddInt64(&currentConcurrent, -1)
		}()
	}

	wg.Wait()

	if maxObserved > int64(maxConcurrent) {
		t.Errorf("max concurrent goroutines = %d, exceeded limit of %d", maxObserved, maxConcurrent)
	}
}

// TestLimitWaitGroup_RaceCondition tests for race conditions with concurrent Add/Done/Wait calls.
// Run with: go test -race
func TestLimitWaitGroup_RaceCondition(t *testing.T) {
	wg, err := NewLimitWaitGroup(5)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var counter int64

	// Launch multiple goroutines that all call Add/Done
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}

	wg.Wait()

	if counter != 100 {
		t.Errorf("expected counter = 100, got %d", counter)
	}
}

// TestLimitWaitGroup_RaceMultipleAdds tests race conditions when calling Add with delta > 1
func TestLimitWaitGroup_RaceMultipleAdds(t *testing.T) {
	wg, err := NewLimitWaitGroup(10)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var counter int64
	numBatches := 20
	batchSize := 5

	for i := 0; i < numBatches; i++ {
		wg.Add(batchSize)
		for j := 0; j < batchSize; j++ {
			go func() {
				defer wg.Done()
				atomic.AddInt64(&counter, 1)
			}()
		}
	}

	wg.Wait()

	expected := int64(numBatches * batchSize)
	if counter != expected {
		t.Errorf("expected counter = %d, got %d", expected, counter)
	}
}

// TestLimitWaitGroup_RapidAddDone tests rapid Add/Done cycling for race conditions
func TestLimitWaitGroup_RapidAddDone(t *testing.T) {
	wg, err := NewLimitWaitGroup(2)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	iterations := 1000
	var completed int64

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&completed, 1)
		}()
	}

	wg.Wait()

	if completed != int64(iterations) {
		t.Errorf("expected completed = %d, got %d", iterations, completed)
	}
}

// TestLimitWaitGroup_ConcurrentWaiters tests multiple goroutines calling Wait concurrently
func TestLimitWaitGroup_ConcurrentWaiters(t *testing.T) {
	wg, err := NewLimitWaitGroup(3)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var workDone int64
	var waitersFinished int64

	// Start some work
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt64(&workDone, 1)
		}()
	}

	// Have multiple goroutines wait
	var waiterWg sync.WaitGroup
	for i := 0; i < 5; i++ {
		waiterWg.Add(1)
		go func() {
			defer waiterWg.Done()
			wg.Wait()
			atomic.AddInt64(&waitersFinished, 1)
		}()
	}

	waiterWg.Wait()

	if workDone != 10 {
		t.Errorf("expected workDone = 10, got %d", workDone)
	}
	if waitersFinished != 5 {
		t.Errorf("expected waitersFinished = 5, got %d", waitersFinished)
	}
}

// TestLimitWaitGroup_StressTest performs a stress test with many concurrent operations
func TestLimitWaitGroup_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	wg, err := NewLimitWaitGroup(50)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var counter int64
	numGoroutines := 10000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}

	wg.Wait()

	if counter != int64(numGoroutines) {
		t.Errorf("expected counter = %d, got %d", numGoroutines, counter)
	}
}

// TestLimitWaitGroup_SingleConcurrency ensures limit of 1 works correctly (sequential execution)
func TestLimitWaitGroup_SingleConcurrency(t *testing.T) {
	wg, err := NewLimitWaitGroup(1)
	if err != nil {
		t.Fatalf("NewLimitWaitGroup() error = %v", err)
	}

	var currentConcurrent int64
	var maxObserved int64
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			current := atomic.AddInt64(&currentConcurrent, 1)

			mu.Lock()
			if current > maxObserved {
				maxObserved = current
			}
			mu.Unlock()

			time.Sleep(1 * time.Millisecond)

			atomic.AddInt64(&currentConcurrent, -1)
		}()
	}

	wg.Wait()

	if maxObserved > 1 {
		t.Errorf("max concurrent goroutines = %d, should never exceed 1", maxObserved)
	}
}

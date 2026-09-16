package persist

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkerSerializes(t *testing.T) {
	w := NewWorker()
	defer w.Stop()
	var sum atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Go(func() { sum.Add(1) })
		}()
	}
	wg.Wait()
	w.Flush()
	if got := sum.Load(); got != 50 {
		t.Fatalf("sum = %d", got)
	}
}

func TestWorkerFIFO(t *testing.T) {
	w := NewWorker()
	defer w.Stop()
	var order []int
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		i := i
		w.Go(func() {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
		})
	}
	w.Flush()
	for i, v := range order {
		if v != i {
			t.Fatalf("order = %v", order)
		}
	}
}

func TestWorkerStopIdempotent(t *testing.T) {
	w := NewWorker()
	w.Go(func() {})
	w.Stop()
	w.Stop()
	w.Go(func() {}) // dropped, must not hang or panic
}

package persist

import "sync"

// Worker serializes persistence steps through one goroutine — the Go
// equivalent of the writeChain promise chain in ws-server/sync.ts. Enqueueing
// never blocks the sync loop; Flush provides the finalize barrier ("await
// writeChain" before the last-leave snapshot).
type Worker struct {
	mu     sync.Mutex
	cond   *sync.Cond
	ops    []func()
	closed bool
	done   chan struct{}
}

// NewWorker starts the loop.
func NewWorker() *Worker {
	w := &Worker{done: make(chan struct{})}
	w.cond = sync.NewCond(&w.mu)
	go w.loop()
	return w
}

func (w *Worker) loop() {
	defer close(w.done)
	for {
		w.mu.Lock()
		for len(w.ops) == 0 && !w.closed {
			w.cond.Wait()
		}
		if len(w.ops) == 0 && w.closed {
			w.mu.Unlock()
			return
		}
		fn := w.ops[0]
		w.ops = w.ops[1:]
		w.mu.Unlock()
		fn()
	}
}

// Go enqueues a step (fire-and-forget, ordered). Enqueues after Stop are
// dropped — Finalize is the only post-Stop caller and it Flushes first.
func (w *Worker) Go(fn func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.ops = append(w.ops, fn)
	w.cond.Signal()
}

// Flush blocks until all steps enqueued so far complete.
// Must precede Stop (enqueues after Stop are dropped).
func (w *Worker) Flush() {
	done := make(chan struct{})
	w.Go(func() { close(done) })
	<-done
}

// Stop flushes, then stops the loop. Idempotent.
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		w.cond.Signal()
	}
	w.mu.Unlock()
	<-w.done
}

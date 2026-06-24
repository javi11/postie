package poster

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestEngine_ConcurrentStormStaysBounded hammers the process-wide upload gate
// with far more concurrent "uploads" than its worker/budget allows — mirroring
// many queue jobs storming the engine at once (the exact scenario that caused
// the original OOM/crash). It asserts the engine never admits more than its
// worker_count concurrent posts or more than its byte budget, never deadlocks,
// and drains cleanly. Run under -race to catch data races.
func TestEngine_ConcurrentStormStaysBounded(t *testing.T) {
	const articleSize = 768 * 1024
	e := NewEngine(articleSize, 0, 8)
	per := e.Budget().PerArticleBytes
	workerCap := e.Budget().WorkerCount
	budget := e.Budget().BudgetBytes

	var curWorkers, maxWorkers atomic.Int64
	var curBytes, maxBytes atomic.Int64
	bumpMax := func(cur int64, max *atomic.Int64) {
		for {
			m := max.Load()
			if cur <= m || max.CompareAndSwap(m, cur) {
				return
			}
		}
	}

	ctx := context.Background()
	const goroutines = 64
	const iters = 40
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iters {
				if err := e.ReserveBuffer(ctx, per); err != nil {
					t.Errorf("ReserveBuffer: %v", err)
					return
				}
				bumpMax(curBytes.Add(per), &maxBytes)

				if err := e.AcquireWorker(ctx); err != nil {
					t.Errorf("AcquireWorker: %v", err)
					curBytes.Add(-per)
					e.ReleaseBuffer(per)
					return
				}
				bumpMax(curWorkers.Add(1), &maxWorkers)

				// Simulated post work.
				curWorkers.Add(-1)
				e.ReleaseWorker()
				curBytes.Add(-per)
				e.ReleaseBuffer(per)
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("storm did not complete in time (possible deadlock)")
	}

	if maxWorkers.Load() > workerCap {
		t.Errorf("peak concurrent workers = %d, exceeds worker_count %d", maxWorkers.Load(), workerCap)
	}
	if maxBytes.Load() > budget {
		t.Errorf("peak reserved bytes = %d, exceeds budget %d", maxBytes.Load(), budget)
	}
	if m := e.Metrics(); m.ActiveWorkers != 0 || m.ReservedBytes != 0 {
		t.Errorf("engine not drained after storm: %+v", m)
	}
}

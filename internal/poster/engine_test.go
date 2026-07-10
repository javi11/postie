package poster

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestComputeEngineBudget_AutoSizing(t *testing.T) {
	const articleSize = 768 * 1024
	b := ComputeEngineBudget(articleSize, 0, 40)

	if b.PerArticleBytes <= articleSize {
		t.Errorf("PerArticleBytes = %d, want > raw article size %d", b.PerArticleBytes, articleSize)
	}
	if b.BudgetBytes < minAutoBudgetBytes || b.BudgetBytes > maxAutoBudgetBytes {
		t.Errorf("auto BudgetBytes = %d, want within [%d, %d]", b.BudgetBytes, minAutoBudgetBytes, maxAutoBudgetBytes)
	}
	// worker_count must never exceed connection capacity.
	if b.WorkerCount > 40 {
		t.Errorf("WorkerCount = %d, want <= 40", b.WorkerCount)
	}
	if b.WorkerCount < 1 {
		t.Errorf("WorkerCount = %d, want >= 1", b.WorkerCount)
	}
}

func TestComputeEngineBudget_ClampsHugeConnCapacity(t *testing.T) {
	// A provider advertising 1000 connections must still be clamped to the
	// 512 MiB ceiling under auto-sizing.
	b := ComputeEngineBudget(768*1024, 0, 1000)
	if b.BudgetBytes > maxAutoBudgetBytes {
		t.Errorf("BudgetBytes = %d, want <= %d", b.BudgetBytes, maxAutoBudgetBytes)
	}
}

func TestComputeEngineBudget_ExplicitLimitBoundsMemoryNotWorkers(t *testing.T) {
	// An explicit small limit bounds in-flight buffer memory; worker slots stay
	// at connection capacity (the memory semaphore throttles effective
	// concurrency on its own).
	per := ComputeEngineBudget(768*1024, 0, 1).PerArticleBytes
	b := ComputeEngineBudget(768*1024, per*3, 100)
	if b.WorkerCount != 100 {
		t.Errorf("WorkerCount = %d, want 100 (connection capacity)", b.WorkerCount)
	}
	if b.BudgetBytes != per*3 {
		t.Errorf("BudgetBytes = %d, want %d", b.BudgetBytes, per*3)
	}
}

func TestComputeEngineBudget_WorkersMatchConnCapacity(t *testing.T) {
	// Regression for issue #168 (slow uploads since v0.0.30): auto-sizing used
	// to cap effective concurrency at ~37 articles regardless of connection
	// capacity. Workers must equal connection capacity, and the auto budget
	// must be large enough to keep that many articles in flight.
	const articleSize = 750_000
	b := ComputeEngineBudget(articleSize, 0, 100)
	if b.WorkerCount != 100 {
		t.Errorf("WorkerCount = %d, want 100", b.WorkerCount)
	}
	if b.BudgetBytes < b.PerArticleBytes*100 {
		t.Errorf("BudgetBytes = %d, want >= %d so 100 articles fit in flight",
			b.BudgetBytes, b.PerArticleBytes*100)
	}
}

func TestComputeEngineBudget_TinyLimitStillAllowsOneArticle(t *testing.T) {
	b := ComputeEngineBudget(768*1024, 1024, 8)
	if b.BudgetBytes < b.PerArticleBytes {
		t.Errorf("BudgetBytes = %d, want >= PerArticleBytes %d", b.BudgetBytes, b.PerArticleBytes)
	}
	if b.WorkerCount < 1 {
		t.Errorf("WorkerCount = %d, want >= 1", b.WorkerCount)
	}
}

func TestEngine_BufferBudgetIsBounded(t *testing.T) {
	// Budget for exactly 2 articles; a 3rd reservation must block until one is
	// released.
	per := ComputeEngineBudget(768*1024, 0, 1).PerArticleBytes
	e := NewEngine(768*1024, per*2, 100)

	ctx := context.Background()
	if err := e.ReserveBuffer(ctx, per); err != nil {
		t.Fatalf("reserve 1: %v", err)
	}
	if err := e.ReserveBuffer(ctx, per); err != nil {
		t.Fatalf("reserve 2: %v", err)
	}
	if got := e.Metrics().ReservedBytes; got != per*2 {
		t.Errorf("ReservedBytes = %d, want %d", got, per*2)
	}

	// Third reservation should block.
	blocked := make(chan error, 1)
	go func() { blocked <- e.ReserveBuffer(ctx, per) }()
	select {
	case <-blocked:
		t.Fatal("third reservation did not block when budget exhausted")
	case <-time.After(50 * time.Millisecond):
	}

	// Release one; the blocked reservation should now succeed.
	e.ReleaseBuffer(per)
	select {
	case err := <-blocked:
		if err != nil {
			t.Errorf("unblocked reservation err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reservation did not unblock after release")
	}
}

func TestEngine_WorkerSlotsAreBounded(t *testing.T) {
	// Worker count equals connection capacity (2 here).
	e := NewEngine(768*1024, 0, 2)
	if e.Budget().WorkerCount != 2 {
		t.Fatalf("WorkerCount = %d, want 2", e.Budget().WorkerCount)
	}

	const total = 8
	var maxActive, cur atomic.Int64
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.AcquireWorker(context.Background()); err != nil {
				return
			}
			defer e.ReleaseWorker()
			n := cur.Add(1)
			for {
				m := maxActive.Load()
				if n <= m || maxActive.CompareAndSwap(m, n) {
					break
				}
			}
			<-release
			cur.Add(-1)
		}()
	}

	// Let goroutines saturate the worker slots.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if maxActive.Load() > 2 {
		t.Errorf("max concurrent workers = %d, want <= 2", maxActive.Load())
	}
}

func TestEngine_NilSafe(t *testing.T) {
	var e *Engine
	if err := e.ReserveBuffer(context.Background(), 1024); err != nil {
		t.Errorf("nil ReserveBuffer = %v, want nil", err)
	}
	e.ReleaseBuffer(1024)
	if err := e.AcquireWorker(context.Background()); err != nil {
		t.Errorf("nil AcquireWorker = %v, want nil", err)
	}
	e.ReleaseWorker()
	if e.PerArticleBytes() != 0 {
		t.Errorf("nil PerArticleBytes = %d, want 0", e.PerArticleBytes())
	}
	if (e.Metrics() != Metrics{}) {
		t.Errorf("nil Metrics = %+v, want zero", e.Metrics())
	}
}

func TestEngine_ReserveBufferCancellation(t *testing.T) {
	per := ComputeEngineBudget(768*1024, 0, 1).PerArticleBytes
	e := NewEngine(768*1024, per, 100) // budget for exactly one article

	if err := e.ReserveBuffer(context.Background(), per); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- e.ReserveBuffer(ctx, per) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("cancelled reservation returned nil error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled reservation did not return")
	}
	// Reserved bytes must not have grown from the cancelled attempt.
	if got := e.Metrics().ReservedBytes; got != per {
		t.Errorf("ReservedBytes = %d, want %d (cancelled reservation must not count)", got, per)
	}
}

package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

const (
	runningJobsTickInterval = 1 * time.Second
	nntpMetricsTickInterval = 5 * time.Second
)

// eventBroadcaster periodically snapshots app state and emits change events to
// connected UIs (Wails runtime in desktop mode, WebSocket hub in web mode).
// One shared ticker per metric replaces what was previously per-client polling.
type eventBroadcaster struct {
	app    *App
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newEventBroadcaster(app *App) *eventBroadcaster {
	return &eventBroadcaster{app: app}
}

// start launches the broadcast loops. Safe to call once; subsequent calls are
// no-ops while a previous start is still active.
func (b *eventBroadcaster) start() {
	if b.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	b.wg.Add(2)
	go b.runRunningJobsLoop(ctx)
	go b.runPoolMetricsLoop(ctx)
}

// stop cancels all broadcast loops and waits for them to exit.
func (b *eventBroadcaster) stop() {
	if b.cancel == nil {
		return
	}
	b.cancel()
	b.wg.Wait()
	b.cancel = nil
}

// runRunningJobsLoop emits running-jobs-updated when the snapshot changes, and
// piggybacks auto-pause state-change detection on the same tick.
func (b *eventBroadcaster) runRunningJobsLoop(ctx context.Context) {
	defer b.wg.Done()
	ticker := time.NewTicker(runningJobsTickInterval)
	defer ticker.Stop()

	var lastJobsJSON []byte
	var lastPauseJSON []byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.tickRunningJobs(&lastJobsJSON)
			b.tickAutoPause(&lastPauseJSON)
		}
	}
}

func (b *eventBroadcaster) tickRunningJobs(lastJSON *[]byte) {
	jobs, err := b.app.GetRunningJobsDetails()
	if err != nil {
		slog.Debug("broadcaster: failed to read running jobs", "error", err)
		return
	}
	payload, err := json.Marshal(jobs)
	if err != nil {
		slog.Debug("broadcaster: failed to marshal running jobs", "error", err)
		return
	}
	if bytes.Equal(payload, *lastJSON) {
		return
	}
	*lastJSON = payload
	b.app.emit("running-jobs-updated", jobs)
}

func (b *eventBroadcaster) tickAutoPause(lastJSON *[]byte) {
	state := b.app.pauseState()
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	if bytes.Equal(payload, *lastJSON) {
		return
	}
	*lastJSON = payload
	b.app.emit("processing:auto-paused", state)
}

// runPoolMetricsLoop emits NNTP pool metrics on a fixed cadence. We do not
// dedupe because timestamp/elapsed/avg-speed drift every tick, and a 5s
// cadence is cheap enough.
func (b *eventBroadcaster) runPoolMetricsLoop(ctx context.Context) {
	defer b.wg.Done()
	ticker := time.NewTicker(nntpMetricsTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := b.app.GetNntpPoolMetrics()
			if err != nil {
				slog.Debug("broadcaster: failed to read pool metrics", "error", err)
				continue
			}
			b.app.emit("nntp-pool-metrics-updated", metrics)
		}
	}
}

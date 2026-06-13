package postie

import (
	"context"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/poster"
)

// Runtime owns the process-wide transfer resources shared across all upload
// jobs. Today it holds the PAR2 scheduler; later phases extend it with the
// upload engine, durable verification service, shared throttle, and global
// upload-buffer budget.
//
// The Processor creates one Runtime during initialization and closes it during
// shutdown. Each per-job Postie borrows these shared resources (via
// NewWithRuntime) instead of creating its own, which is what makes the limits
// process-wide rather than per queue job.
type Runtime struct {
	par2Scheduler *par2.Scheduler
	uploadEngine  *poster.Engine
}

// NewRuntime builds the shared transfer runtime from cfg. poolManager may be
// nil; when it is (or when no upload connections are advertised) the upload
// engine is left unset and jobs fall back to standalone, unbounded behaviour.
// It is safe to call with a cfg whose PAR2 settings are unset; the PAR2
// scheduler then defaults to a single concurrent job.
func NewRuntime(ctx context.Context, cfg config.Config, poolManager *pool.Manager) (*Runtime, error) {
	maxJobs := 1
	var uploadEngine *poster.Engine

	if cfg != nil {
		par2Cfg, err := cfg.GetPar2Config(ctx)
		if err != nil {
			return nil, err
		}
		if par2Cfg != nil && par2Cfg.MaxConcurrentJobs > 0 {
			maxJobs = par2Cfg.MaxConcurrentJobs
		}

		// Build the process-wide upload engine sized from article size, the
		// configured buffer limit (0 = auto), and the total upload connection
		// capacity reported by the pool.
		if connCapacity := uploadConnectionCapacity(poolManager); connCapacity > 0 {
			postingCfg := cfg.GetPostingConfig()
			uploadEngine = poster.NewEngine(
				postingCfg.ArticleSizeInBytes,
				postingCfg.UploadBufferMemoryLimit,
				connCapacity,
			)
		}
	}

	return &Runtime{
		par2Scheduler: par2.NewScheduler(maxJobs),
		uploadEngine:  uploadEngine,
	}, nil
}

// uploadConnectionCapacity sums the configured connection slots across all
// upload providers, which bounds how many articles can be in flight at once.
func uploadConnectionCapacity(poolManager *pool.Manager) int {
	if poolManager == nil {
		return 0
	}
	uploadPool := poolManager.GetUploadPool()
	if uploadPool == nil {
		return 0
	}
	capacity := 0
	for _, pr := range uploadPool.Stats().Providers {
		capacity += pr.MaxConnections
	}
	return capacity
}

// Par2Scheduler returns the shared PAR2 scheduler, or nil if r is nil.
func (r *Runtime) Par2Scheduler() *par2.Scheduler {
	if r == nil {
		return nil
	}
	return r.par2Scheduler
}

// UploadEngine returns the shared upload engine, or nil if r is nil or no
// engine was created.
func (r *Runtime) UploadEngine() *poster.Engine {
	if r == nil {
		return nil
	}
	return r.uploadEngine
}

// RuntimeMetrics is a point-in-time snapshot of process-wide transfer resource
// usage, suitable for surfacing in the UI or logs so memory growth can be
// attributed to a subsystem.
type RuntimeMetrics struct {
	UploadActiveWorkers int64 `json:"uploadActiveWorkers"`
	UploadQueuedWorkers int64 `json:"uploadQueuedWorkers"`
	UploadWorkerCount   int64 `json:"uploadWorkerCount"`
	UploadReservedBytes int64 `json:"uploadReservedBytes"`
	UploadBudgetBytes   int64 `json:"uploadBudgetBytes"`
	Par2ActiveJobs      int64 `json:"par2ActiveJobs"`
	Par2QueuedJobs      int64 `json:"par2QueuedJobs"`
	Par2Capacity        int   `json:"par2Capacity"`
}

// Metrics returns a snapshot of current runtime resource usage. Safe on a nil
// Runtime (returns the zero value).
func (r *Runtime) Metrics() RuntimeMetrics {
	if r == nil {
		return RuntimeMetrics{}
	}

	var m RuntimeMetrics
	if r.uploadEngine != nil {
		em := r.uploadEngine.Metrics()
		m.UploadActiveWorkers = em.ActiveWorkers
		m.UploadQueuedWorkers = em.QueuedWorkers
		m.UploadWorkerCount = em.WorkerCount
		m.UploadReservedBytes = em.ReservedBytes
		m.UploadBudgetBytes = em.BudgetBytes
	}
	if r.par2Scheduler != nil {
		m.Par2ActiveJobs = r.par2Scheduler.Active()
		m.Par2QueuedJobs = r.par2Scheduler.Queued()
		m.Par2Capacity = r.par2Scheduler.Capacity()
	}
	return m
}

// Close releases runtime-owned resources. It is safe to call on a nil Runtime
// and safe to call more than once.
func (r *Runtime) Close() error {
	return nil
}

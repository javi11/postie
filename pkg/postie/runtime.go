package postie

import (
	"context"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/par2"
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
}

// NewRuntime builds the shared transfer runtime from cfg. It is safe to call
// with a cfg whose PAR2 settings are unset; the PAR2 scheduler then defaults to
// a single concurrent job.
func NewRuntime(ctx context.Context, cfg config.Config) (*Runtime, error) {
	maxJobs := 1
	if cfg != nil {
		par2Cfg, err := cfg.GetPar2Config(ctx)
		if err != nil {
			return nil, err
		}
		if par2Cfg != nil && par2Cfg.MaxConcurrentJobs > 0 {
			maxJobs = par2Cfg.MaxConcurrentJobs
		}
	}

	return &Runtime{
		par2Scheduler: par2.NewScheduler(maxJobs),
	}, nil
}

// Par2Scheduler returns the shared PAR2 scheduler, or nil if r is nil.
func (r *Runtime) Par2Scheduler() *par2.Scheduler {
	if r == nil {
		return nil
	}
	return r.par2Scheduler
}

// Close releases runtime-owned resources. It is safe to call on a nil Runtime
// and safe to call more than once.
func (r *Runtime) Close() error {
	return nil
}

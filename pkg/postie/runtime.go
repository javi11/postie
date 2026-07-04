package postie

import (
	"context"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/manifest"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/internal/transfercleaner"
	"github.com/javi11/postie/internal/transferstore"
	"github.com/javi11/postie/internal/transferwriter"
	"github.com/javi11/postie/internal/verification"
)

// poolStater adapts an NNTP client to the verification.Stater interface (a STAT
// that returns only an error: nil = article exists).
type poolStater struct {
	pool pool.NNTPClient
}

func (s poolStater) Stat(ctx context.Context, messageID string) error {
	_, err := s.pool.Stat(ctx, messageID)
	return err
}

// StatBatch checks the ids in one batched sweep; the verification service
// pre-chunks to its configured StatBatchSize, so the whole slice is one chunk.
func (s poolStater) StatBatch(ctx context.Context, messageIDs []string) (map[string]struct{}, error) {
	return pool.StatMissing(ctx, s.pool, messageIDs, len(messageIDs))
}

// newPostVerifyScriptRunner returns a transfercleaner.ScriptRunner that runs
// the post-upload script once a transfer is verified, resolving the NZB path
// from the completed item and the source path from the transfer's files. Returns
// nil when the script is disabled, so cleanup skips it entirely.
func newPostVerifyScriptRunner(store *transferstore.Store, cfg config.PostUploadScriptConfig, q QueueInterface) func(ctx context.Context, transferID string, files []transferstore.TransferFile) error {
	if !cfg.Enabled || cfg.Command == "" {
		return nil
	}
	return func(ctx context.Context, transferID string, files []transferstore.TransferFile) error {
		var itemID, sourcePath string
		for _, f := range files {
			if f.CompletedItemID != "" {
				itemID = f.CompletedItemID
			}
			if sourcePath == "" && f.FileRole == string(manifest.RoleOriginal) {
				sourcePath = f.SourcePath
			}
		}
		if sourcePath == "" && len(files) > 0 {
			sourcePath = files[0].SourcePath
		}
		nzbPath, err := store.GetCompletedItemNZBPath(ctx, itemID)
		if err != nil {
			return err
		}
		return runPostUploadScript(ctx, cfg, q, nzbPath, sourcePath, itemID)
	}
}

// verificationConfig maps the post-check config to the verification service's
// config; zero/auto values are normalised inside the service.
func verificationConfig(pc config.PostCheck) verification.Config {
	return verification.Config{
		MaxConcurrentChecks: pc.MaxConcurrentChecks,
		StatBatchSize:       pc.StatBatchSize,
		PropagationDelay:    pc.RetryDelay.ToDuration(),
		MaxReposts:          int(pc.MaxRePost),
		DeferredBackoff:     pc.DeferredCheckDelay.ToDuration(),
		MaxBackoff:          pc.DeferredMaxBackoff.ToDuration(),
		MaxDeferredChecks:   pc.DeferredMaxRetries,
		BatchSize:           pc.DeferredBatchSize,
	}
}

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
	store         *transferstore.Store
	manifestDir   string
	verifyService *verification.Service
}

// NewRuntime builds the shared transfer runtime from cfg. poolManager may be
// nil; when it is (or when no upload connections are advertised) the upload
// engine is left unset and jobs fall back to standalone, unbounded behaviour.
// store and manifestDir enable durable manifest recording; when store is nil
// jobs run without manifests (standalone). It is safe to call with a cfg whose
// PAR2 settings are unset; the PAR2 scheduler then defaults to a single job.
func NewRuntime(ctx context.Context, cfg config.Config, poolManager *pool.Manager, store *transferstore.Store, manifestDir string, scriptQueue QueueInterface) (*Runtime, error) {
	maxJobs := 1
	var uploadEngine *poster.Engine
	var verifyService *verification.Service

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

		// Build the durable verification service when a store + pool are present.
		// It re-posts through the shared engine and STATs through the verify pool.
		if store != nil && poolManager != nil {
			uploadPool := poolManager.GetUploadPool()
			verifyPool := poolManager.GetVerifyPool()
			if verifyPool == nil {
				verifyPool = uploadPool
			}
			if uploadPool != nil && verifyPool != nil {
				postingCfg := cfg.GetPostingConfig()
				reposter := poster.NewReposter(uploadPool, uploadEngine, postingCfg.ThrottleRate)
				verifyService = verification.New(
					store,
					poolStater{pool: verifyPool},
					reposter,
					verificationConfig(cfg.GetPostCheckConfig()),
					"postie",
				)
				// Post-verification cleanup: run the post-upload script, delete
				// originals per policy, remove generated PAR2 (unless maintained)
				// and manifests — only once the transfer is verified.
				maintainPar2 := par2Cfg != nil && par2Cfg.MaintainPar2Files != nil && *par2Cfg.MaintainPar2Files
				scriptCfg := cfg.GetPostUploadScriptConfig()
				runScript := newPostVerifyScriptRunner(store, scriptCfg, scriptQueue)
				verifyService.SetCleaner(transfercleaner.New(store, maintainPar2, runScript))
			}
		}
	}

	return &Runtime{
		par2Scheduler: par2.NewScheduler(maxJobs),
		uploadEngine:  uploadEngine,
		store:         store,
		manifestDir:   manifestDir,
		verifyService: verifyService,
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

// DurableVerificationEnabled reports whether durable verification is active
// (a verification service was created). When true, callers should defer
// destructive cleanup (delete_original, post-upload script) until the durable
// service marks the transfer verified, rather than acting at upload completion.
func (r *Runtime) DurableVerificationEnabled() bool {
	return r != nil && r.verifyService != nil
}

// RunVerification runs the durable verification service until ctx is cancelled.
// It is a no-op (returns immediately) when no service was created (standalone
// mode or missing store/pool). Intended to be started once as a goroutine.
func (r *Runtime) RunVerification(ctx context.Context) {
	if r == nil || r.verifyService == nil {
		return
	}
	r.verifyService.Run(ctx)
}

// TransferStore returns the shared durable transfer store, or nil if none.
func (r *Runtime) TransferStore() *transferstore.Store {
	if r == nil {
		return nil
	}
	return r.store
}

// NewManifestRecorder returns a per-job manifest recorder bound to transferID,
// or nil when the runtime has no store or transferID is empty (so the poster
// runs without manifest recording). The concrete type lets the caller both use
// it as a poster.ManifestSink and call CompleteUpload after posting.
func (r *Runtime) NewManifestRecorder(transferID string) *transferwriter.Recorder {
	if r == nil || r.store == nil || transferID == "" {
		return nil
	}
	return transferwriter.New(transferID, r.manifestDir, r.store)
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

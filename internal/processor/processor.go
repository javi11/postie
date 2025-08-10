package processor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/javi11/nntppool"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/pausable"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/javi11/postie/pkg/postie"
	"maragu.dev/goqite"
)

const maxRetries = 3

type Processor struct {
	queue        *queue.Queue
	config       config.Config
	cfg          config.QueueConfig
	poolManager  *pool.Manager
	outputFolder string
	isRunning    bool
	runningMux   sync.Mutex
	// Track running jobs and their contexts for cancellation
	runningJobs               map[string]*RunningJob
	jobsMux                   sync.RWMutex
	jobsWg                    sync.WaitGroup // WaitGroup to track running jobs
	deleteOriginalFile        bool
	maintainOriginalExtension bool
	watchFolder               string // Path to the watch folder for maintaining folder structure
	// Pause/resume functionality
	isPaused  bool
	pausedMux sync.RWMutex
	// Auto-pause functionality
	isAutoPaused        bool
	autoPauseReason     string
	autoPausedMux       sync.RWMutex
	providerCheckTicker *time.Ticker
	providerCheckCtx    context.Context
	providerCheckCancel context.CancelFunc
	// Callback to check if processor can start new items
	canProcessNextItem func() bool
}

type ProcessorOptions struct {
	Queue                     *queue.Queue
	Config                    config.Config
	QueueConfig               config.QueueConfig
	PoolManager               *pool.Manager
	OutputFolder              string
	DeleteOriginalFile        bool
	MaintainOriginalExtension bool
	WatchFolder               string
	CanProcessNextItem        func() bool // Callback to check if processor can start new items
}
type RunningJobDetails struct {
	ID       string                   `json:"id"`
	Path     string                   `json:"path"`
	FileName string                   `json:"fileName"`
	Size     int64                    `json:"size"`
	Progress []progress.ProgressState `json:"progress"`
}

type RunningJob struct {
	RunningJobDetails
	Progress    progress.JobProgress
	cancel      context.CancelFunc
	pausableCtx *pausable.Context
}

// RunningJobItem represents a running job for the frontend (kept for backward compatibility)
type RunningJobItem struct {
	ID string `json:"id"`
}

func New(opts ProcessorOptions) *Processor {
	// Create context for provider monitoring
	providerCtx, providerCancel := context.WithCancel(context.Background())

	processor := &Processor{
		queue:                     opts.Queue,
		config:                    opts.Config,
		cfg:                       opts.QueueConfig,
		poolManager:               opts.PoolManager,
		outputFolder:              opts.OutputFolder,
		runningJobs:               make(map[string]*RunningJob),
		deleteOriginalFile:        opts.DeleteOriginalFile,
		maintainOriginalExtension: opts.MaintainOriginalExtension,
		watchFolder:               opts.WatchFolder,
		providerCheckCtx:          providerCtx,
		providerCheckCancel:       providerCancel,
		canProcessNextItem:        opts.CanProcessNextItem,
	}

	// Start provider availability monitoring if we have a pool manager
	if opts.PoolManager != nil {
		go processor.monitorProviderAvailability()
	}

	return processor
}

// Start begins processing files from the queue
func (p *Processor) Start(ctx context.Context) error {
	processTicker := time.NewTicker(time.Second * 2) // Process queue frequently
	defer processTicker.Stop()

	// Main processing loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-processTicker.C:
			if err := p.processQueueItems(ctx); err != nil {
				slog.ErrorContext(ctx, "Error processing queue", "error", err)
			}
		}
	}
}

func (p *Processor) processQueueItems(ctx context.Context) error {
	// Check if processor is paused
	p.pausedMux.RLock()
	paused := p.isPaused
	p.pausedMux.RUnlock()

	if paused {
		return nil // Skip processing when paused
	}

	// Check if we can process next item (e.g., pending config changes)
	if p.canProcessNextItem != nil && !p.canProcessNextItem() {
		slog.DebugContext(ctx, "Processor waiting - cannot process new items (callback returned false)")
		return nil // Skip processing when callback indicates we should wait
	}

	if p.isRunning {
		return nil
	}

	p.runningMux.Lock()
	defer p.runningMux.Unlock()

	p.isRunning = true
	defer func() {
		p.isRunning = false
	}()

	// Process items with configurable concurrency
	semaphore := make(chan struct{}, p.cfg.MaxConcurrentUploads)
	var wg sync.WaitGroup

	// Process multiple items concurrently
	for range p.cfg.MaxConcurrentUploads {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }() // Release

			if err := p.processNextItem(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error processing item", "error", err)
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func (p *Processor) processNextItem(ctx context.Context) error {
	// Check if processor is shutting down
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if pool manager is still available before processing
	p.runningMux.Lock()
	poolManager := p.poolManager
	p.runningMux.Unlock()
	
	if poolManager == nil {
		slog.WarnContext(ctx, "Pool manager is not available, skipping item processing")
		return nil
	}

	// Get next item from queue
	msg, job, err := p.queue.ReceiveFile(ctx)
	if err != nil {
		return err
	}

	// If no message available, return without error
	if msg == nil {
		return nil
	}

	slog.Info("Processing file", "msg", msg.ID, "path", job.Path, "priority", job.Priority)

	// Process the file
	actualNzbPath, err := p.processFile(ctx, msg, job)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("Job cancelled", "msg", msg.ID, "path", job.Path)

			return nil
		}
		// Handle error with retry logic - re-add job to queue
		return p.handleProcessingError(ctx, msg, job, string(msg.ID), err)
	}

	// Use the actual NZB path returned by the postie.Post method
	// Mark as completed with the NZB path and job data
	if err := p.queue.CompleteFile(ctx, msg.ID, actualNzbPath, job); err != nil {
		slog.ErrorContext(ctx, "Error marking file as completed", "error", err, "path", job.Path)
		return err
	}

	return nil
}

func (p *Processor) processFile(ctx context.Context, msg *goqite.Message, job *queue.FileJob) (string, error) {
	fileName := getFileName(job.Path)
	jobID := string(msg.ID)

	// Create a context for this specific job that can be cancelled independently
	jobCtx, jobCancel := context.WithCancel(ctx)

	// Create a pausable context wrapper
	pausableCtx := pausable.NewContext(jobCtx)

	// Track this job for potential cancellation and pausing
	p.jobsMux.Lock()
	// Track detailed job information
	progressJob := progress.NewProgressJob(jobID)
	defer progressJob.Close()

	// Add to WaitGroup before adding to running jobs
	p.jobsWg.Add(1)

	p.runningJobs[jobID] = &RunningJob{
		RunningJobDetails: RunningJobDetails{
			ID:       jobID,
			Path:     job.Path,
			FileName: fileName,
			Size:     job.Size,
		},
		Progress:    progressJob,
		cancel:      jobCancel,
		pausableCtx: pausableCtx,
	}

	// Apply current pause state to new job
	if p.isPaused {
		pausableCtx.Pause()
		// Also set progress as paused for new jobs
		progressJob.SetAllPaused(true)
	}
	p.jobsMux.Unlock()

	// Cleanup function to remove from tracking
	defer func() {
		p.jobsMux.Lock()
		delete(p.runningJobs, jobID)
		p.jobsMux.Unlock()
		jobCancel()     // Ensure context is cancelled
		p.jobsWg.Done() // Signal job completion to WaitGroup
	}()

	// Create file info
	fileInfo := fileinfo.FileInfo{
		Path: job.Path,
		Size: uint64(job.Size),
	}

	// Double-check that pool manager is still available before creating postie
	p.runningMux.Lock()
	poolManager := p.poolManager
	p.runningMux.Unlock()
	
	if poolManager == nil {
		return "", fmt.Errorf("pool manager is not available for job %s", jobID)
	}

	// Create a postie instance for this job with progress manager
	jobPostie, err := postie.New(jobCtx, p.config, poolManager, progressJob)
	if err != nil {
		return "", fmt.Errorf("failed to create postie instance for job %s: %w", jobID, err)
	}
	defer jobPostie.Close()

	// Determine the input folder for maintaining folder structure
	var inputFolder string
	if p.watchFolder != "" && isWithinPath(job.Path, p.watchFolder) {
		// For files from the watcher, use the watch folder as root to maintain structure
		inputFolder = p.watchFolder
		slog.DebugContext(jobCtx, "Using watch folder as root for folder structure",
			"watchFolder", p.watchFolder, "filePath", job.Path)
	} else {
		// For manually added files, use the directory containing the file
		inputFolder = filepath.Dir(job.Path)
		slog.DebugContext(jobCtx, "Using file directory as root",
			"inputFolder", inputFolder, "filePath", job.Path)
	}

	// Post the file using the job-specific postie instance with pausable context
	actualNzbPath, err := jobPostie.Post(pausableCtx, []fileinfo.FileInfo{fileInfo}, inputFolder, p.outputFolder)
	if err != nil {
		return "", err
	}

	// Delete the original file
	if p.deleteOriginalFile {
		if err := os.Remove(job.Path); err != nil {
			slog.WarnContext(ctx, "Could not delete original file", "path", job.Path, "error", err)
		}
	}

	return actualNzbPath, nil
}

func (p *Processor) handleProcessingError(ctx context.Context, msg *goqite.Message, job *queue.FileJob, jobID string, err error) error {
	slog.ErrorContext(ctx, "Error processing file", "error", err, "path", job.Path, "retryCount", job.RetryCount)

	job.RetryCount++

	if job.RetryCount >= maxRetries {
		slog.ErrorContext(ctx, "Job failed permanently after reaching max retries", "path", job.Path)
		if err := p.queue.MarkAsError(ctx, msg.ID, job, err.Error()); err != nil {
			slog.ErrorContext(ctx, "Failed to mark job as error", "error", err, "path", job.Path)
			// Re-add to queue as a fallback
			if readdErr := p.queue.ReaddJob(ctx, job); readdErr != nil {
				slog.ErrorContext(ctx, "Failed to re-add job to queue", "error", readdErr, "path", job.Path)
			}
		}
	} else {
		// Re-add the job to the queue for retry
		if readdErr := p.queue.ReaddJob(ctx, job); readdErr != nil {
			slog.ErrorContext(ctx, "Failed to re-add job to queue for retry", "error", readdErr, "path", job.Path)
		}
	}

	return nil
}

// CancelJob cancels a running job by its ID
func (p *Processor) CancelJob(jobID string) error {
	p.jobsMux.Lock()

	rj, exists := p.runningJobs[jobID]
	if !exists {
		p.jobsMux.Unlock()
		return fmt.Errorf("job %s is not currently running", jobID)
	}

	// Remove from tracking first to prevent duplicate events
	delete(p.runningJobs, jobID)

	p.jobsMux.Unlock()

	// Cancel the job's context
	rj.cancel()

	slog.Info("Job cancelled", "jobID", jobID)
	return nil
}

// GetRunningJobs returns a map of currently running job IDs
func (p *Processor) GetRunningJobs() map[string]bool {
	p.jobsMux.RLock()
	defer p.jobsMux.RUnlock()

	result := make(map[string]bool)
	for jobID := range p.runningJobs {
		result[jobID] = true
	}
	return result
}

// GetRunningJobItems returns detailed information about currently running jobs
func (p *Processor) GetRunningJobItems() []RunningJobItem {
	p.jobsMux.RLock()
	defer p.jobsMux.RUnlock()

	var items []RunningJobItem
	for jobID := range p.runningJobs {
		items = append(items, RunningJobItem{
			ID: jobID,
		})
	}
	return items
}

// GetRunningJobDetails returns detailed information about currently running jobs
func (p *Processor) GetRunningJobDetails() map[string]RunningJobDetails {
	p.jobsMux.RLock()
	defer p.jobsMux.RUnlock()

	details := make(map[string]RunningJobDetails)
	for jobID, jobDetail := range p.runningJobs {
		details[jobID] = RunningJobDetails{
			ID:       jobID,
			Path:     jobDetail.Path,
			FileName: jobDetail.FileName,
			Size:     jobDetail.Size,
			Progress: jobDetail.Progress.GetAllProgressState(),
		}
	}

	return details
}

// IsPathBeingProcessed checks if a file path is currently being processed
func (p *Processor) IsPathBeingProcessed(path string) bool {
	p.jobsMux.RLock()
	defer p.jobsMux.RUnlock()

	for _, jobDetails := range p.runningJobs {
		if jobDetails.Path == path {
			return true
		}
	}
	return false
}

// PauseProcessing pauses the processor, preventing new jobs from starting and pausing active jobs
func (p *Processor) PauseProcessing() {
	p.pausedMux.Lock()
	defer p.pausedMux.Unlock()

	if !p.isPaused {
		p.isPaused = true

		// Pause all currently running jobs
		p.jobsMux.RLock()
		for jobID, job := range p.runningJobs {
			if job.pausableCtx != nil {
				job.pausableCtx.Pause()
				// Also set progress as paused
				if job.Progress != nil {
					job.Progress.SetAllPaused(true)
				}
				slog.Info("Paused running job", "jobID", jobID)
			}
		}
		p.jobsMux.RUnlock()

		slog.Info("Processor paused - new jobs blocked, active jobs suspended")
	}
}

// ResumeProcessing resumes the processor, allowing new jobs to start and resuming active jobs
func (p *Processor) ResumeProcessing() {
	p.pausedMux.Lock()
	defer p.pausedMux.Unlock()

	if p.isPaused {
		p.isPaused = false

		// Resume all currently running jobs
		p.jobsMux.RLock()
		for jobID, job := range p.runningJobs {
			if job.pausableCtx != nil {
				job.pausableCtx.Resume()
				// Also set progress as resumed
				if job.Progress != nil {
					job.Progress.SetAllPaused(false)
				}
				slog.Info("Resumed running job", "jobID", jobID)
			}
		}
		p.jobsMux.RUnlock()

		slog.Info("Processor resumed - new jobs allowed, active jobs resumed")
	}
}

// IsPaused returns whether the processor is currently paused
func (p *Processor) IsPaused() bool {
	p.pausedMux.RLock()
	defer p.pausedMux.RUnlock()
	return p.isPaused
}

func (p *Processor) Close() error {
	slog.Info("Processor shutdown initiated")

	// Set pool manager to nil to prevent new jobs from using it
	p.runningMux.Lock()
	p.poolManager = nil
	p.runningMux.Unlock()

	// Stop provider monitoring
	if p.providerCheckCancel != nil {
		p.providerCheckCancel()
	}

	// Get a snapshot of running jobs and cancel them
	p.jobsMux.Lock()
	runningJobsCount := len(p.runningJobs)

	// Cancel all running jobs
	for jobID, job := range p.runningJobs {
		job.cancel() // Cancel the job's context
		slog.Info("Cancelled running job", "jobID", jobID)
	}
	p.jobsMux.Unlock()

	if runningJobsCount == 0 {
		slog.Info("No running jobs to wait for")
		return nil
	}

	slog.Info("Waiting for running jobs to be cancelled", "count", runningJobsCount)

	// Wait for all jobs to complete with a timeout using WaitGroup
	timeout := 30 * time.Second
	done := make(chan struct{})

	go func() {
		defer close(done)
		p.jobsWg.Wait() // Wait for all jobs to call Done()
	}()

	// Wait for jobs to complete or timeout
	select {
	case <-done:
		slog.Info("All running jobs where cancelled successfully")
	case <-time.After(timeout):
		p.jobsMux.RLock()
		remainingJobs := len(p.runningJobs)
		p.jobsMux.RUnlock()

		slog.Warn("Timeout waiting for jobs to complete",
			"remainingJobs", remainingJobs,
			"timeout", timeout)
	}

	// Force clear any remaining jobs after timeout
	p.jobsMux.Lock()
	p.runningJobs = make(map[string]*RunningJob)
	p.jobsMux.Unlock()

	slog.Info("Processor shutdown completed")
	return nil
}

// IsAutoPaused returns true if the processor was automatically paused due to provider unavailability
func (p *Processor) IsAutoPaused() bool {
	p.autoPausedMux.RLock()
	defer p.autoPausedMux.RUnlock()
	return p.isAutoPaused
}

// GetAutoPauseReason returns the reason for automatic pause, if any
func (p *Processor) GetAutoPauseReason() string {
	p.autoPausedMux.RLock()
	defer p.autoPausedMux.RUnlock()
	return p.autoPauseReason
}

// monitorProviderAvailability monitors provider status and pauses/resumes processing accordingly
func (p *Processor) monitorProviderAvailability() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	p.providerCheckTicker = ticker

	slog.Info("Started provider availability monitoring")

	for {
		select {
		case <-p.providerCheckCtx.Done():
			slog.Info("Provider monitoring stopped")
			return
		case <-ticker.C:
			p.checkAndHandleProviderAvailability()
		}
	}
}

// checkAndHandleProviderAvailability checks provider availability and pauses/resumes as needed
func (p *Processor) checkAndHandleProviderAvailability() {
	// Get pool manager reference safely
	p.runningMux.Lock()
	poolManager := p.poolManager
	p.runningMux.Unlock()
	
	if poolManager == nil {
		return
	}

	// Get pool metrics to check provider status
	metrics, err := poolManager.GetMetrics()
	if err != nil {
		slog.Error("Failed to get pool metrics for provider check", "error", err)
		return
	}

	// Count connected/active providers
	activeProviders := 0
	totalProviders := len(metrics.ProviderMetrics)

	for _, provider := range metrics.ProviderMetrics {
		// Consider a provider active if it's connected or has a good state
		if provider.State == nntppool.ProviderStateActive {
			activeProviders++
		}
	}

	p.autoPausedMux.Lock()
	wasAutoPaused := p.isAutoPaused
	p.autoPausedMux.Unlock()

	slog.Debug("Provider availability check",
		"activeProviders", activeProviders,
		"totalProviders", totalProviders,
		"wasAutoPaused", wasAutoPaused)

	// If no providers are available and we haven't auto-paused yet
	if activeProviders == 0 && totalProviders > 0 && !wasAutoPaused {
		slog.Warn("No providers available - auto-pausing processing")
		p.PauseProcessing()

		p.autoPausedMux.Lock()
		p.isAutoPaused = true
		p.autoPauseReason = "All NNTP providers are unavailable"
		p.autoPausedMux.Unlock()
	}

	// If providers are available and we were auto-paused
	if activeProviders > 0 && wasAutoPaused {
		// Only resume if the processor was paused by us (auto-pause)
		// Check if it's still paused - if user manually resumed, don't override
		if p.IsPaused() {
			slog.Info("Providers available - auto-resuming processing",
				"activeProviders", activeProviders)
			p.ResumeProcessing()
		}

		p.autoPausedMux.Lock()
		p.isAutoPaused = false
		p.autoPauseReason = ""
		p.autoPausedMux.Unlock()
	}
}

func getFileName(path string) string {
	// Simple filename extraction
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

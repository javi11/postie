package processor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/javi11/postie/pkg/postie"
	"maragu.dev/goqite"
)

type Processor struct {
	queue        *queue.Queue
	postie       *postie.Postie
	cfg          config.QueueConfig
	outputFolder string
	eventEmitter func(eventName string, optionalData ...interface{})
	isRunning    bool
	runningMux   sync.Mutex
	// Track running jobs and their contexts for cancellation
	runningJobs map[string]context.CancelFunc
	// Track detailed information about running jobs
	runningJobDetails map[string]*RunningJobDetails
	jobsMux           sync.RWMutex
}

type ProcessorOptions struct {
	Queue        *queue.Queue
	Postie       *postie.Postie
	Config       config.QueueConfig
	OutputFolder string
	EventEmitter func(eventName string, optionalData ...interface{})
}

// RunningJobDetails stores detailed information about a running job
type RunningJobDetails struct {
	ID       string  `json:"id"`
	Path     string  `json:"path"`
	FileName string  `json:"fileName"`
	Size     int64   `json:"size"`
	Status   string  `json:"status"`
	Stage    string  `json:"stage"`
	Progress float64 `json:"progress"`
}

// RunningJobItem represents a running job for the frontend (kept for backward compatibility)
type RunningJobItem struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func New(opts ProcessorOptions) *Processor {
	return &Processor{
		queue:             opts.Queue,
		postie:            opts.Postie,
		cfg:               opts.Config,
		outputFolder:      opts.OutputFolder,
		eventEmitter:      opts.EventEmitter,
		runningJobs:       make(map[string]context.CancelFunc),
		runningJobDetails: make(map[string]*RunningJobDetails),
	}
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
				if err != context.Canceled {
					slog.ErrorContext(ctx, "Error processing item", "error", err)
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func (p *Processor) processNextItem(ctx context.Context) error {
	// Get next item from queue
	msg, job, err := p.queue.ReceiveFile(ctx)
	if err != nil {
		return err
	}

	// If no message available, return without error
	if msg == nil {
		return nil
	}

	slog.Info("Processing file", "msg", msg.ID, "path", job.Path)

	// Process the file
	if err := p.processFile(ctx, msg, job); err != nil {
		if err == context.Canceled {
			// Remove the job from the queue
			if err := p.queue.RemoveFromQueue(string(msg.ID)); err != nil {
				slog.Error("Failed to remove job from queue", "error", err, "msg", msg.ID, "path", job.Path)
			}

			slog.Info("Job cancelled", "msg", msg.ID, "path", job.Path)

			return nil
		}
		// Handle error with retry logic - re-add job to queue
		return p.handleProcessingError(ctx, msg, job, string(msg.ID), err)
	}

	// Generate the NZB path that would have been created by the postie.Post method
	// Since we're processing a single file, the NZB will be in the output folder with the same base name
	nzbPath := filepath.Join(p.outputFolder, filepath.Base(job.Path)+".nzb")

	// Mark as completed with the NZB path and job data
	if err := p.queue.CompleteFile(ctx, msg.ID, nzbPath, job); err != nil {
		slog.ErrorContext(ctx, "Error marking file as completed", "error", err, "path", job.Path)
		return err
	}

	return nil
}

func (p *Processor) processFile(ctx context.Context, msg *goqite.Message, job *queue.FileJob) error {
	fileName := getFileName(job.Path)
	jobID := string(msg.ID)

	// Create a context for this specific job that can be cancelled independently
	jobCtx, jobCancel := context.WithCancel(ctx)

	// Track this job for potential cancellation
	p.jobsMux.Lock()
	p.runningJobs[jobID] = jobCancel
	// Track detailed job information
	p.runningJobDetails[jobID] = &RunningJobDetails{
		ID:       jobID,
		Path:     job.Path,
		FileName: fileName,
		Size:     job.Size,
		Status:   "running",
		Stage:    "Starting",
		Progress: 0,
	}
	p.jobsMux.Unlock()

	// Cleanup function to remove from tracking
	defer func() {
		p.jobsMux.Lock()
		delete(p.runningJobs, jobID)
		delete(p.runningJobDetails, jobID)
		p.jobsMux.Unlock()
		jobCancel() // Ensure context is cancelled
	}()

	// Emit progress start event
	if p.eventEmitter != nil {
		p.eventEmitter("progress", map[string]interface{}{
			"currentFile":         fileName,
			"totalFiles":          1,
			"completedFiles":      0,
			"stage":               "Processing",
			"details":             fmt.Sprintf("Processing %s", fileName),
			"isRunning":           true,
			"lastUpdate":          time.Now().Unix(),
			"percentage":          0,
			"currentFileProgress": 0,
			"jobID":               jobID,
		})
	}

	// Create file info
	fileInfo := fileinfo.FileInfo{
		Path: job.Path,
		Size: uint64(job.Size),
	}

	// Set up progress callback for this file
	progressCallback := func(stage string, current, total int64, details string, speed float64, secondsLeft float64, elapsedTime float64) {
		var fileProgress float64
		if total > 0 {
			fileProgress = float64(current) / float64(total) * 100
		}

		// Update running job details
		p.jobsMux.Lock()
		if jobDetails, exists := p.runningJobDetails[jobID]; exists {
			jobDetails.Stage = stage
			jobDetails.Progress = fileProgress
		}
		p.jobsMux.Unlock()

		if p.eventEmitter != nil {
			p.eventEmitter("progress", map[string]interface{}{
				"currentFile":         fileName,
				"totalFiles":          1,
				"completedFiles":      0,
				"stage":               stage,
				"details":             details,
				"isRunning":           true,
				"lastUpdate":          time.Now().Unix(),
				"percentage":          fileProgress,
				"currentFileProgress": fileProgress,
				"jobID":               jobID,
				"speed":               speed,
				"secondsLeft":         secondsLeft,
			})
		}

		// Extend timeout for long-running operations
		if current > 0 && current%1000 == 0 { // Every 1000 progress units
			if err := p.queue.ExtendTimeout(ctx, msg.ID, time.Minute*5); err != nil {
				slog.WarnContext(ctx, "Failed to extend timeout", "error", err)
			}
		}
	}

	// Set progress callback on postie
	if p.postie != nil {
		p.postie.SetProgressCallback(progressCallback)
	}

	// Get the directory containing the file for input folder
	inputFolder := filepath.Dir(job.Path)

	// Post the file using the job-specific context
	if err := p.postie.Post(jobCtx, []fileinfo.FileInfo{fileInfo}, inputFolder, p.outputFolder); err != nil {
		// Check if this was a cancellation
		if err == context.Canceled {
			// Emit cancellation event
			if p.eventEmitter != nil {
				p.eventEmitter("progress", map[string]interface{}{
					"currentFile":         fileName,
					"totalFiles":          1,
					"completedFiles":      0,
					"stage":               "Cancelled",
					"details":             fmt.Sprintf("Cancelled %s", fileName),
					"isRunning":           false,
					"lastUpdate":          time.Now().Unix(),
					"percentage":          0,
					"currentFileProgress": 0,
					"jobID":               jobID,
				})
			}
			return context.Canceled
		}

		// Emit error progress event
		if p.eventEmitter != nil {
			p.eventEmitter("progress", map[string]interface{}{
				"currentFile":         fileName,
				"totalFiles":          1,
				"completedFiles":      0,
				"stage":               "Error",
				"details":             fmt.Sprintf("Error processing %s: %v", fileName, err),
				"isRunning":           false,
				"lastUpdate":          time.Now().Unix(),
				"percentage":          0,
				"currentFileProgress": 0,
				"jobID":               jobID,
			})
		}

		return err
	}

	// Emit completion progress event
	if p.eventEmitter != nil {
		p.eventEmitter("progress", map[string]interface{}{
			"currentFile":         fileName,
			"totalFiles":          1,
			"completedFiles":      1,
			"stage":               "Completed",
			"details":             fmt.Sprintf("Completed %s", fileName),
			"isRunning":           false,
			"lastUpdate":          time.Now().Unix(),
			"percentage":          100,
			"currentFileProgress": 100,
			"jobID":               jobID,
		})
	}

	// Delete the original file
	if err := os.Remove(job.Path); err != nil {
		slog.WarnContext(ctx, "Could not delete original file", "path", job.Path, "error", err)
		// Don't return error as the processing was successful
	}

	return nil
}

func (p *Processor) handleProcessingError(ctx context.Context, msg *goqite.Message, job *queue.FileJob, jobID string, err error) error {
	slog.ErrorContext(ctx, "Error processing file", "error", err, "path", job.Path)

	// Re-add the job to the queue for retry since it was already removed when processing started
	if readdErr := p.queue.ReaddJob(ctx, job); readdErr != nil {
		slog.ErrorContext(ctx, "Failed to re-add job to queue", "error", readdErr, "path", job.Path)
		// Return the original error, but log the re-add failure
	}

	// Emit error event
	if p.eventEmitter != nil {
		p.eventEmitter("progress", map[string]interface{}{
			"currentFile":         getFileName(job.Path),
			"totalFiles":          1,
			"completedFiles":      0,
			"stage":               "Error",
			"details":             fmt.Sprintf("Error processing %s: %v", getFileName(job.Path), err),
			"isRunning":           false,
			"lastUpdate":          time.Now().Unix(),
			"percentage":          0,
			"currentFileProgress": 0,
			"jobID":               jobID,
		})
	}

	return err
}

// CancelJob cancels a running job by its ID
func (p *Processor) CancelJob(jobID string) error {
	p.jobsMux.Lock()
	defer p.jobsMux.Unlock()

	cancelFunc, exists := p.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job %s is not currently running", jobID)
	}

	// Cancel the job's context
	cancelFunc()

	// Remove from tracking
	delete(p.runningJobs, jobID)
	delete(p.runningJobDetails, jobID)

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
			ID:     jobID,
			Status: "running",
		})
	}
	return items
}

// GetRunningJobDetails returns detailed information about currently running jobs
func (p *Processor) GetRunningJobDetails() []*RunningJobDetails {
	p.jobsMux.RLock()
	defer p.jobsMux.RUnlock()

	var details []*RunningJobDetails
	for _, jobDetail := range p.runningJobDetails {
		// Create a copy to avoid race conditions
		detailCopy := *jobDetail
		details = append(details, &detailCopy)
	}
	return details
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

package backend

import (
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProgressTracker struct {
	CurrentFile         string  `json:"currentFile"`
	TotalFiles          int     `json:"totalFiles"`
	CompletedFiles      int     `json:"completedFiles"`
	Stage               string  `json:"stage"`
	Details             string  `json:"details"`
	IsRunning           bool    `json:"isRunning"`
	LastUpdate          int64   `json:"lastUpdate"`
	Percentage          float64 `json:"percentage"`
	CurrentFileProgress float64 `json:"currentFileProgress"`
	JobID               string  `json:"jobID"`
	TotalBytes          int64   `json:"totalBytes"`
	TransferredBytes    int64   `json:"transferredBytes"`
	CurrentFileBytes    int64   `json:"currentFileBytes"`
	Speed               float64 `json:"speed"`       // Speed in KB/s
	SecondsLeft         float64 `json:"secondsLeft"` // Estimated seconds remaining
	ElapsedTime         float64 `json:"elapsedTime"` // Elapsed time in seconds
}

// ProgressUpdate contains the fields that can be updated in a progress update
type ProgressUpdate struct {
	// Core fields that are commonly updated together (non-nullable)
	Stage          string  `json:"stage"`          // Always provided for state changes
	IsRunning      bool    `json:"isRunning"`      // Always provided with stage changes
	CompletedFiles int     `json:"completedFiles"` // Frequently updated for progress
	Percentage     float64 `json:"percentage"`     // Frequently updated with completedFiles

	// Optional context fields (nullable)
	CurrentFile         *string  `json:"currentFile,omitempty"`         // Optional: file being processed
	TotalFiles          *int     `json:"totalFiles,omitempty"`          // Optional: set once at start
	Details             *string  `json:"details,omitempty"`             // Optional: detailed status from callbacks
	JobID               *string  `json:"jobID,omitempty"`               // Optional: only for queue operations
	TotalBytes          *int64   `json:"totalBytes,omitempty"`          // Optional: set once at start
	TransferredBytes    *int64   `json:"transferredBytes,omitempty"`    // Optional: periodic updates
	CurrentFileBytes    *int64   `json:"currentFileBytes,omitempty"`    // Optional: from callbacks
	CurrentFileProgress *float64 `json:"currentFileProgress,omitempty"` // Optional: from callbacks
	Speed               *float64 `json:"speed,omitempty"`               // Optional: upload speed in KB/s
	SecondsLeft         *float64 `json:"secondsLeft,omitempty"`         // Optional: estimated seconds remaining
	ElapsedTime         *float64 `json:"elapsedTime,omitempty"`         // Optional: elapsed time in seconds
}

// GetProgress returns the current progress
func (a *App) GetProgress() ProgressTracker {
	return a.getProgress()
}

func (a *App) getProgress() ProgressTracker {
	a.progressMux.RLock()
	defer a.progressMux.RUnlock()
	return *a.progress
}

// updateProgress is a helper function to update progress state and emit events
func (a *App) updateProgress(update ProgressUpdate) {
	a.progressMux.Lock()

	// Apply core fields (always provided)
	a.progress.Stage = update.Stage
	a.progress.IsRunning = update.IsRunning
	a.progress.CompletedFiles = update.CompletedFiles
	a.progress.Percentage = update.Percentage

	// Apply optional fields (only if not nil)
	if update.CurrentFile != nil {
		a.progress.CurrentFile = *update.CurrentFile
	}
	if update.TotalFiles != nil {
		a.progress.TotalFiles = *update.TotalFiles
	}
	if update.Details != nil {
		a.progress.Details = *update.Details
	}
	if update.CurrentFileProgress != nil {
		a.progress.CurrentFileProgress = *update.CurrentFileProgress
	}
	if update.TotalBytes != nil {
		a.progress.TotalBytes = *update.TotalBytes
	}
	if update.TransferredBytes != nil {
		a.progress.TransferredBytes = *update.TransferredBytes
	}
	if update.CurrentFileBytes != nil {
		a.progress.CurrentFileBytes = *update.CurrentFileBytes
	}
	if update.JobID != nil {
		a.progress.JobID = *update.JobID
	}
	if update.Speed != nil {
		a.progress.Speed = *update.Speed
	}
	if update.SecondsLeft != nil {
		a.progress.SecondsLeft = *update.SecondsLeft
	}
	if update.ElapsedTime != nil {
		a.progress.ElapsedTime = *update.ElapsedTime
	}

	// Always update timestamp
	a.progress.LastUpdate = time.Now().Unix()

	// Clear JobID for manual uploads (non-queue based) when not running
	if update.Stage == "Finished" || update.Stage == "Cancelled" || update.Stage == "Error" {
		a.progress.JobID = ""
	}

	a.progressMux.Unlock()

	// Emit progress event to frontend for both desktop and web modes
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "progress", a.getProgress())
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("progress", a.getProgress())
	}
}

// Helper functions to create pointers for ProgressUpdate fields
func stringPtr(s string) *string    { return &s }
func intPtr(i int) *int             { return &i }
func int64Ptr(i int64) *int64       { return &i }
func float64Ptr(f float64) *float64 { return &f }

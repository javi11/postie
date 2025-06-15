package poster

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressCallback defines the interface for progress notifications
type ProgressCallback func(stage string, current, total int64, details string, speed float64, secondsLeft float64, elapsedTime float64)

// ProgressManager handles progress tracking and display
type ProgressManager struct {
	bar           *progressbar.ProgressBar
	articlesTotal int64
	callback      ProgressCallback
	description   string
}

// NewFileProgress creates a new progress bar for a file
func NewFileProgress(
	description string,
	totalBytes int64,
	articlesTotal int64,
) *ProgressManager {
	return NewFileProgressWithCallback(description, totalBytes, articlesTotal, nil)
}

// NewFileProgressWithCallback creates a new progress bar for a file with a callback
func NewFileProgressWithCallback(
	description string,
	totalBytes int64,
	articlesTotal int64,
	callback ProgressCallback,
) *ProgressManager {
	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			_, _ = fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetMaxDetailRow(1),
	)

	return &ProgressManager{
		bar:           bar,
		articlesTotal: articlesTotal,
		callback:      callback,
		description:   description,
	}
}

// UpdateFileProgress updates the progress bar for a file
func (pm *ProgressManager) UpdateFileProgress(bytesProcessed int64, articlesProcessed int64, articleErrors int64) {
	speed := pm.bar.State().KBsPerSecond
	details := fmt.Sprintf("Articles: %d/%d | Errors: %d",
		articlesProcessed,
		pm.articlesTotal,
		articleErrors,
	)

	_ = pm.bar.AddDetail(details)
	_ = pm.bar.Set64(bytesProcessed)

	// Notify callback if available
	if pm.callback != nil {
		// Calculate seconds left from progress bar state
		secondsLeft := pm.bar.State().SecondsLeft
		elapsedTime := pm.bar.State().SecondsSince
		pm.callback("uploading", bytesProcessed, int64(pm.bar.GetMax64()), details, speed, secondsLeft, elapsedTime)
	}
}

// FinishFileProgress completes the progress bar for a file
func (pm *ProgressManager) FinishFileProgress() {
	_ = pm.bar.Finish()

	// Notify callback if available
	if pm.callback != nil {
		speed := pm.bar.State().KBsPerSecond
		details := "Upload completed"
		secondsLeft := 0.0 // No time left when completed
		elapsedTime := pm.bar.State().SecondsSince
		pm.callback("completed", int64(pm.bar.GetMax64()), int64(pm.bar.GetMax64()), details, speed, secondsLeft, elapsedTime)
	}
}

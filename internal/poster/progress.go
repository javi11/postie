package poster

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressManager handles progress tracking and display
type ProgressManager struct {
	bar           *progressbar.ProgressBar
	articlesTotal int64
}

// NewFileProgress creates a new progress bar for a file
func NewFileProgress(
	description string,
	totalBytes int64,
	articlesTotal int64,
) *ProgressManager {
	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
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
	}
}

// UpdateFileProgress updates the progress bar for a file
func (pm *ProgressManager) UpdateFileProgress(bytesProcessed int64, articlesProcessed int64, articleErrors int64) {
	pm.bar.AddDetail(fmt.Sprintf("Articles: %d/%d | Errors: %d",
		articlesProcessed,
		pm.articlesTotal,
		articleErrors,
	))
	pm.bar.Set64(bytesProcessed)
}

// FinishFileProgress completes the progress bar for a file
func (pm *ProgressManager) FinishFileProgress() {
	pm.bar.Finish()
}

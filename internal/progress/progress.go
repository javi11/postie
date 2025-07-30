package progress

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
)

type Callback func(
	jobID string,
	activeProgress map[uuid.UUID]Progress,
)

type ProgressType string

const (
	ProgressTypeUploading      ProgressType = "uploading"
	ProgressTypePar2Generation ProgressType = "par2_generation"
)

type ProgressManager interface {
	AddProgress(
		id uuid.UUID,
		name string,
		pType ProgressType,
		total int64,
	)
	FinishProgress(id uuid.UUID)
	RegisterListener(
		listener Callback,
	)
}

type Progress interface {
	UpdateProgress(
		processed int64,
	)
	Finish()
	GetState() progressbar.State
}

type progressManager struct {
	jobID          string
	activeProgress map[uuid.UUID]Progress
	listeners      []Callback
	notifier       chan uuid.UUID
}

func NewProgressManager(
	jobID string,
) ProgressManager {
	n := make(chan uuid.UUID, 100000)

	pm := &progressManager{
		jobID:    jobID,
		notifier: n,
	}

	go pm.notifierWorker()

	return pm
}

func (pm *progressManager) AddProgress(
	id uuid.UUID,
	name string,
	pType ProgressType,
	total int64,
) {
	progress := &pogress{
		ID:       id,
		Name:     name,
		notifier: pm.notifier,
		Progress: *progressbar.NewOptions64(
			total,
			progressbar.OptionSetDescription(name),
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
		),
	}

	pm.activeProgress[id] = progress
}

func (pm *progressManager) FinishProgress(
	id uuid.UUID,
) {
	pm.activeProgress[id].Finish()
	delete(pm.activeProgress, id)

	pm.notifier <- id
}

func (pm *progressManager) RegisterListener(
	cb Callback,
) {
	pm.listeners = append(pm.listeners, cb)
}

func (pm *progressManager) notifierWorker(ctx context.Context, notifier chan uuid.UUID) {

	for {
		select {
		case id <- notifier:
			for _, listener := range pm.listeners {
				listener()
			}
		}
	}
}

type pogress struct {
	ID       uuid.UUID
	Name     string
	Progress progressbar.ProgressBar
	notifier chan uuid.UUID
}

func (p *pogress) UpdateProgress(
	processed int64,
) {
	p.Progress.Set64(processed)

	p.notifier <- p.ID
}

func (p *pogress) Finish() {
	p.Progress.Finish()
}

func (p *pogress) GetState() progressbar.State {
	return p.Progress.State()
}

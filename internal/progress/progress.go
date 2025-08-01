package progress

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
)

type ProgressType string

const (
	ProgressTypeUploading      ProgressType = "uploading"
	ProgressTypePar2Generation ProgressType = "par2_generation"
	ProgressTypeChecking       ProgressType = "checking"
)

type ProgressState struct {
	Max            int64
	CurrentNum     int64
	CurrentPercent float64
	CurrentBytes   float64
	SecondsSince   float64
	SecondsLeft    float64
	KBsPerSecond   float64
	Description    string
	Type           ProgressType
	IsStarted      bool
}

// EventEmitter is a function type for emitting events to the frontend
type EventEmitter func(eventType string, data interface{})

type JobProgress interface {
	AddProgress(id uuid.UUID, name string, pType ProgressType, total int64) Progress
	FinishProgress(id uuid.UUID)
	GetProgress(id uuid.UUID) Progress
	GetAllProgress() map[uuid.UUID]Progress
	GetAllProgressState() []ProgressState
	GetJobID() string
	Close()
}

// Progress represents an individual progress indicator
type Progress interface {
	UpdateProgress(processed int64)
	Finish()
	GetState() ProgressState
	GetID() uuid.UUID
	GetName() string
	GetType() ProgressType
	GetCurrent() int64
	GetTotal() int64
	GetPercentage() float64
	IsComplete() bool
	GetStartTime() time.Time
	GetElapsedTime() time.Duration
}

type jobProgress struct {
	jobID          string
	activeProgress map[uuid.UUID]Progress
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewProgressJob(jobID string) JobProgress {
	ctx, cancel := context.WithCancel(context.Background())

	return &jobProgress{
		jobID:          jobID,
		activeProgress: make(map[uuid.UUID]Progress),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (pm *jobProgress) AddProgress(
	id uuid.UUID,
	name string,
	pType ProgressType,
	total int64,
) Progress {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var OptionShowBytes progressbar.Option
	if pType == ProgressTypeUploading {
		OptionShowBytes = progressbar.OptionShowBytes(true)
	} else {
		OptionShowBytes = progressbar.OptionShowBytes(false)
	}

	progress := &progress{
		id:        id,
		name:      name,
		pType:     pType,
		total:     total,
		startTime: time.Now(),
		progress: progressbar.NewOptions64(
			total,
			progressbar.OptionSetDescription(name),
			OptionShowBytes,
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
			progressbar.OptionSetMaxDetailRow(0),
			progressbar.OptionSetPredictTime(true),
		),
	}

	pm.activeProgress[id] = progress
	return progress
}

func (pm *jobProgress) FinishProgress(id uuid.UUID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if progress, exists := pm.activeProgress[id]; exists {
		progress.Finish()
		delete(pm.activeProgress, id)
	}
}

func (pm *jobProgress) GetProgress(id uuid.UUID) Progress {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.activeProgress[id]
}

func (pm *jobProgress) GetAllProgress() map[uuid.UUID]Progress {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[uuid.UUID]Progress)
	for k, v := range pm.activeProgress {
		result[k] = v
	}
	return result
}

func (pm *jobProgress) GetAllProgressState() []ProgressState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]ProgressState, 0, len(pm.activeProgress))
	for _, v := range pm.activeProgress {
		result = append(result, v.GetState())
	}

	sort.Slice(result, func(i, j int) bool {
		// Sort by current progress in descending order, then by description in ascending order
		if result[i].CurrentNum != result[j].CurrentNum {
			return result[i].CurrentNum > result[j].CurrentNum
		}

		return result[i].Description < result[j].Description
	})

	return result
}

func (pm *jobProgress) GetJobID() string {
	return pm.jobID
}

func (pm *jobProgress) Close() {
	pm.cancel()
}

type progress struct {
	id        uuid.UUID
	name      string
	pType     ProgressType
	total     int64
	startTime time.Time
	progress  *progressbar.ProgressBar
}

func (p *progress) UpdateProgress(processed int64) {
	if p.progress.IsFinished() {
		return
	}

	p.progress.Add64(processed)
}

func (p *progress) Finish() {
	p.progress.Finish()
	p.progress.Close()
}

func (p *progress) GetID() uuid.UUID {
	return p.id
}

func (p *progress) GetName() string {
	return p.name
}

func (p *progress) GetType() ProgressType {
	return p.pType
}

func (p *progress) GetState() ProgressState {
	s := p.progress.State()

	return ProgressState{
		Max:            s.Max,
		CurrentNum:     s.CurrentNum,
		CurrentPercent: s.CurrentPercent,
		CurrentBytes:   s.CurrentBytes,
		SecondsSince:   s.SecondsSince,
		SecondsLeft:    s.SecondsLeft,
		KBsPerSecond:   s.KBsPerSecond,
		Description:    s.Description,
		Type:           p.pType,
		IsStarted:      s.CurrentNum > 0,
	}
}

func (p *progress) GetCurrent() int64 {
	return p.progress.State().CurrentNum
}

func (p *progress) GetTotal() int64 {
	return p.total
}

func (p *progress) GetPercentage() float64 {
	return p.progress.State().CurrentPercent
}

func (p *progress) IsComplete() bool {
	return p.progress.IsFinished()
}

func (p *progress) GetStartTime() time.Time {
	return p.startTime
}

func (p *progress) GetElapsedTime() time.Duration {
	return time.Duration(p.progress.State().SecondsSince) * time.Second
}

func (p *progress) GetLeftTime() time.Duration {
	return time.Duration(p.progress.State().SecondsLeft) * time.Second
}

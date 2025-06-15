package poster

import (
	"sync"
	"time"
)

// Throttle handles rate limiting
type Throttle struct {
	rate     int64 // bytes per second
	interval time.Duration
	mu       sync.Mutex
	lastTime time.Time
	bytes    int64
	disabled bool
}

// NewThrottle creates a new throttle with the given rate and interval
func NewThrottle(rate int64, interval time.Duration) *Throttle {
	return &Throttle{
		rate:     rate,
		interval: interval,
		lastTime: time.Now(),
		disabled: rate <= 0,
	}
}

// Wait waits for bytes to be available
func (t *Throttle) Wait(bytes int64) {
	// If throttling is disabled, return immediately
	if t.disabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastTime)

	// Calculate how many bytes we can send
	bytesAvailable := int64(elapsed.Seconds() * float64(t.rate))
	t.bytes += bytesAvailable

	// If we don't have enough bytes, wait
	if t.bytes < bytes {
		waitTime := time.Duration(float64(bytes-t.bytes) / float64(t.rate) * float64(time.Second))
		time.Sleep(waitTime)
		t.bytes = 0
		t.lastTime = now.Add(waitTime)
	} else {
		t.bytes -= bytes
		t.lastTime = now
	}
}

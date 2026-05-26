package poster

import (
	"sync"
	"sync/atomic"
	"time"
)

// Throttle handles rate limiting using a token-bucket algorithm. Token-bucket
// state is protected by a mutex: the previous lock-free implementation allowed
// concurrent consumers to overdraw the bucket past zero, silently bypassing
// the configured rate under contention.
type Throttle struct {
	rate     int64         // bytes per second (immutable after creation)
	interval time.Duration // (immutable after creation)
	mu       sync.Mutex
	lastTime atomic.Int64 // last check time in nanoseconds (atomic for test inspection)
	tokens   atomic.Int64 // available tokens (bytes) (atomic for test inspection)
	disabled atomic.Bool  // fast-path check
	sleepFn  func(time.Duration)
}

// NewThrottle creates a new throttle with the given rate and interval
func NewThrottle(rate int64, interval time.Duration) *Throttle {
	t := &Throttle{
		rate:     rate,
		interval: interval,
		sleepFn:  time.Sleep,
	}

	if rate <= 0 {
		t.disabled.Store(true)
	} else {
		t.lastTime.Store(time.Now().UnixNano())
		t.tokens.Store(rate) // Start with 1 second worth of tokens
	}

	return t
}

// Wait waits until enough tokens are available for the given bytes.
// The mutex serialises bucket math so concurrent callers cannot both pass the
// availability check and double-spend the same tokens.
func (t *Throttle) Wait(bytes int64) {
	// Fast path: throttling disabled
	if t.disabled.Load() {
		return
	}

	t.mu.Lock()
	now := time.Now().UnixNano()
	last := t.lastTime.Load()
	elapsed := now - last

	tokensToAdd := (elapsed * t.rate) / int64(time.Second)
	t.lastTime.Store(now)

	current := min(t.tokens.Load()+tokensToAdd, t.rate)

	var waitNs int64
	if current < bytes {
		deficit := bytes - current
		waitNs = (deficit * int64(time.Second)) / t.rate
	}

	t.tokens.Store(current - bytes)
	t.mu.Unlock()

	if waitNs > 0 {
		t.sleepFn(time.Duration(waitNs))
	}
}

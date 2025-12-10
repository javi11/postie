package poster

import (
	"sync/atomic"
	"time"
)

// Throttle handles rate limiting using lock-free atomic operations
// for better performance under high concurrency.
type Throttle struct {
	rate     int64         // bytes per second (immutable after creation)
	interval time.Duration // (immutable after creation)
	lastTime atomic.Int64  // last check time in nanoseconds
	tokens   atomic.Int64  // available tokens (bytes)
	disabled atomic.Bool   // fast-path check
}

// NewThrottle creates a new throttle with the given rate and interval
func NewThrottle(rate int64, interval time.Duration) *Throttle {
	t := &Throttle{
		rate:     rate,
		interval: interval,
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
// Uses a token bucket algorithm with atomic operations for lock-free rate limiting.
func (t *Throttle) Wait(bytes int64) {
	// Fast path: throttling disabled
	if t.disabled.Load() {
		return
	}

	for {
		now := time.Now().UnixNano()
		last := t.lastTime.Load()
		elapsed := now - last

		// Calculate tokens to add based on elapsed time
		tokensToAdd := (elapsed * t.rate) / int64(time.Second)

		// Try to update the last time atomically
		if !t.lastTime.CompareAndSwap(last, now) {
			// Another goroutine updated it, retry
			continue
		}

		// Add new tokens (capped at 1 second worth to prevent burst)
		newTokens := t.tokens.Add(tokensToAdd)
		if newTokens > t.rate {
			t.tokens.Store(t.rate)
			newTokens = t.rate
		}

		// Try to consume tokens
		if newTokens >= bytes {
			t.tokens.Add(-bytes)
			return
		}

		// Not enough tokens, calculate wait time
		deficit := bytes - newTokens
		waitNs := (deficit * int64(time.Second)) / t.rate

		if waitNs > 0 {
			time.Sleep(time.Duration(waitNs))
		}

		// After sleeping, consume the tokens
		t.tokens.Add(-bytes)
		t.lastTime.Store(time.Now().UnixNano())
		return
	}
}

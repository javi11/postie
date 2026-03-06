package poster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewThrottle(t *testing.T) {
	rate := int64(1000)
	interval := time.Second

	// Test with normal rate
	throttle := NewThrottle(rate, interval)

	assert.Equal(t, rate, throttle.rate, "Rate should be initialized correctly")
	assert.Equal(t, interval, throttle.interval, "Interval should be initialized correctly")
	assert.NotZero(t, throttle.lastTime.Load(), "LastTime should be initialized")
	assert.False(t, throttle.disabled.Load(), "Throttle should not be disabled with positive rate")

	// Test with rate 0 (disabled)
	throttleDisabled := NewThrottle(0, interval)
	assert.True(t, throttleDisabled.disabled.Load(), "Throttle should be disabled when rate is 0")

	// Test with negative rate (disabled)
	throttleNegative := NewThrottle(-100, interval)
	assert.True(t, throttleNegative.disabled.Load(), "Throttle should be disabled when rate is negative")
}

func newTestThrottle(rate int64, interval time.Duration) (*Throttle, *[]time.Duration) {
	var slept []time.Duration
	th := NewThrottle(rate, interval)
	th.sleepFn = func(d time.Duration) { slept = append(slept, d) }
	return th, &slept
}

func TestThrottleWait(t *testing.T) {
	// Test case 1: Wait for less bytes than the rate — no sleep expected
	throttle, slept := newTestThrottle(1000, time.Second)
	throttle.Wait(100)
	assert.Empty(t, *slept, "Wait for bytes under rate limit should not sleep")

	// Test case 2: Wait for significantly more bytes than available (5x rate)
	// deficit = 5000 - 1000 = 4000 bytes → wait ≈ 4s
	throttle, slept = newTestThrottle(1000, time.Second)
	throttle.Wait(5000)
	assert.Len(t, *slept, 1, "Wait should sleep exactly once when over the rate limit")
	assert.Greater(t, (*slept)[0], 3500*time.Millisecond, "Sleep duration should be at least 3.5s")
	assert.Less(t, (*slept)[0], 5500*time.Millisecond, "Sleep duration should be at most 5.5s")

	// Test case 3: Accumulated calls exceeding the limit trigger a sleep
	throttle, slept = newTestThrottle(1000, time.Second)
	throttle.Wait(300)
	throttle.Wait(300)
	throttle.Wait(300) // 900 total — still within budget
	assert.Empty(t, *slept, "Three waits within rate should not sleep")
	throttle.Wait(500) // pushes over limit
	assert.NotEmpty(t, *slept, "Wait should sleep when rate is exceeded")
	assert.Greater(t, (*slept)[0], 300*time.Millisecond, "Sleep duration should be > 300ms")
}

func TestThrottleDisabled(t *testing.T) {
	// Test throttle with rate 0 (disabled)
	throttle := NewThrottle(0, time.Second)

	// All waits should be immediate when throttling is disabled
	start := time.Now()
	throttle.Wait(1000000) // Large number should not cause delay
	elapsed := time.Since(start)

	assert.Less(t, elapsed.Milliseconds(), int64(50), "Disabled throttle should not cause any delay")

	// Test multiple rapid calls
	for range 10 {
		start = time.Now()
		throttle.Wait(10000)
		elapsed = time.Since(start)
		assert.Less(t, elapsed.Milliseconds(), int64(50), "Disabled throttle should not delay multiple calls")
	}
}

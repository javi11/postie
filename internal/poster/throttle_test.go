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
	assert.False(t, throttle.lastTime.IsZero(), "LastTime should be initialized")
	assert.False(t, throttle.disabled, "Throttle should not be disabled with positive rate")

	// Test with rate 0 (disabled)
	throttleDisabled := NewThrottle(0, interval)
	assert.True(t, throttleDisabled.disabled, "Throttle should be disabled when rate is 0")

	// Test with negative rate (disabled)
	throttleNegative := NewThrottle(-100, interval)
	assert.True(t, throttleNegative.disabled, "Throttle should be disabled when rate is negative")
}

func TestThrottleWait(t *testing.T) {
	// Create a new throttle with a rate of 1000 bytes per second
	throttle := NewThrottle(1000, time.Second)

	// Test case 1: Wait for less bytes than the rate
	// This should not cause any significant delay
	start := time.Now()
	throttle.Wait(100)
	elapsed := time.Since(start)

	// The delay should be minimal when we're under the rate
	// Increased the threshold to handle slight processing delays
	assert.Less(t, elapsed.Milliseconds(), int64(200), "Wait for bytes under rate limit should be quick")

	// Test case 2: Wait for significantly more bytes than available
	// Forcing a more predictable delay
	start = time.Now()
	throttle.Wait(5000) // 5x our rate of 1000 bytes/sec
	elapsed = time.Since(start)

	// The delay should be roughly 4-5 seconds (5000-1000)/1000 ~= 4 seconds
	// Being more generous with the timing variance
	assert.Greater(t, elapsed.Milliseconds(), int64(3500), "Wait should delay when over the rate limit")
	assert.Less(t, elapsed.Milliseconds(), int64(5500), "Wait delay should be close to expected time")

	// Test case 3: Test with a fresh throttle and more controlled setup
	// Reset with new throttle
	throttle = NewThrottle(1000, time.Second)

	// This test is more focused on behavior than exact timing
	// First wait
	throttle.Wait(300)

	// Second wait
	throttle.Wait(300)

	// Third wait - should still be within limit
	start = time.Now()
	throttle.Wait(300)
	elapsed = time.Since(start)
	assert.Less(t, elapsed.Milliseconds(), int64(400), "Third wait within rate should be relatively quick")

	// Fourth wait - should cause delay
	start = time.Now()
	throttle.Wait(500) // Now exceeding our limit
	elapsed = time.Since(start)

	// Being more generous with timing expectations
	assert.Greater(t, elapsed.Milliseconds(), int64(300), "Wait should delay when rate is exceeded")
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
	for i := 0; i < 10; i++ {
		start = time.Now()
		throttle.Wait(10000)
		elapsed = time.Since(start)
		assert.Less(t, elapsed.Milliseconds(), int64(50), "Disabled throttle should not delay multiple calls")
	}
}

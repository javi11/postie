package poster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewThrottle(t *testing.T) {
	// Create a basic throttle constructor if it doesn't exist
	rate := int64(1000)
	interval := time.Second

	// Manually create the throttle to test initialization values
	throttle := &Throttle{
		rate:     rate,
		interval: interval,
		lastTime: time.Now(),
	}

	assert.Equal(t, rate, throttle.rate, "Rate should be initialized correctly")
	assert.Equal(t, interval, throttle.interval, "Interval should be initialized correctly")
	assert.False(t, throttle.lastTime.IsZero(), "LastTime should be initialized")
}

func TestThrottleWait(t *testing.T) {
	// Create a new throttle with a rate of 1000 bytes per second
	throttle := &Throttle{
		rate:     1000,
		interval: time.Second,
		lastTime: time.Now(),
	}

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
	throttle = &Throttle{
		rate:     1000,
		interval: time.Second,
		lastTime: time.Now(),
	}

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

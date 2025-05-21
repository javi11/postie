package poster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculateHash tests the CalculateHash function
func TestCalculateHash(t *testing.T) {
	data := []byte("test data for hashing")
	hash := CalculateHash(data)

	// Hash should not be empty
	assert.NotEmpty(t, hash, "hash should not be empty")

	// Hashing the same data again should produce the same hash
	hash2 := CalculateHash(data)
	assert.Equal(t, hash, hash2, "hashing the same data should produce the same hash")
}

// TODO: Add more comprehensive tests for other poster functionality
// This requires proper mocking of the nntppool interfaces.

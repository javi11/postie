package poster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileProgress(t *testing.T) {
	// Create a new progress bar
	desc := "Test Progress"
	totalBytes := int64(1000)
	articlesTotal := int64(10)

	pm := NewFileProgress(desc, totalBytes, articlesTotal)

	// Verify the progress manager is created correctly
	assert.NotNil(t, pm, "Progress manager should not be nil")
	assert.NotNil(t, pm.bar, "Progress bar should not be nil")
	assert.Equal(t, articlesTotal, pm.articlesTotal, "Articles total should match")
}

func TestUpdateFileProgress(t *testing.T) {
	// Create a new progress bar
	pm := NewFileProgress("Test Progress", 1000, 10)

	// Update the progress
	bytesProcessed := int64(500)
	articlesProcessed := int64(5)
	articleErrors := int64(1)

	// This shouldn't panic
	pm.UpdateFileProgress(bytesProcessed, articlesProcessed, articleErrors)

	// We can't directly check the progress bar's state, but we can check that
	// the function completes without errors
	assert.NotPanics(t, func() {
		pm.UpdateFileProgress(bytesProcessed, articlesProcessed, articleErrors)
	}, "UpdateFileProgress should not panic")
}

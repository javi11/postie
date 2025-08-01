package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/queue"
)

// mockProcessor implements ProcessorInterface for testing
type mockProcessor struct {
	processingPaths map[string]bool
}

func (m *mockProcessor) IsPathBeingProcessed(path string) bool {
	return m.processingPaths[path]
}

// mockQueueWithDuplicateCheck simulates queue behavior with duplicate checking
// It implements the queue.QueueInterface
type mockQueueWithDuplicateCheck struct {
	addFileCalls []string
	mu           sync.Mutex
}

func (m *mockQueueWithDuplicateCheck) AddFile(ctx context.Context, path string, size int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate duplicate checking - if file already in calls, ignore
	for _, existingPath := range m.addFileCalls {
		if existingPath == path {
			// Simulate queue.AddFile behavior - log and return nil for duplicates
			return nil
		}
	}

	// Add file to our mock queue
	m.addFileCalls = append(m.addFileCalls, path)
	return nil
}

func (m *mockQueueWithDuplicateCheck) GetQueueItems() ([]queue.QueueItem, error) {
	return []queue.QueueItem{}, nil
}

func (m *mockQueueWithDuplicateCheck) RemoveFromQueue(id string) error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) ClearQueue() error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) GetQueueStats() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *mockQueueWithDuplicateCheck) SetQueueItemPriorityWithReorder(ctx context.Context, id string, newPriority int) error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) GetMigrationStatus() (*queue.GooseMigrationStatus, error) {
	return nil, nil
}

func (m *mockQueueWithDuplicateCheck) RunMigrations() error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) RollbackMigration() error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) MigrateTo(version int64) error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) ResetDatabase() error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) IsLegacyDatabase() (bool, error) {
	return false, nil
}

func (m *mockQueueWithDuplicateCheck) RecreateDatabase() error {
	return nil
}

func (m *mockQueueWithDuplicateCheck) EnsureMigrationCompatibility() error {
	return nil
}

// Helper function to create a test watcher
func createTestWatcher(t *testing.T) (*Watcher, string) {
	tempDir := t.TempDir()

	cfg := config.WatcherConfig{
		Enabled:            true,
		WatchDirectory:     tempDir,
		SizeThreshold:      100,
		MinFileSize:        10,
		CheckInterval:      config.Duration("1s"),
		DeleteOriginalFile: false,
		IgnorePatterns:     []string{},
	}

	mockQueue := &mockQueueWithDuplicateCheck{
		addFileCalls: make([]string, 0),
	}
	mockProc := &mockProcessor{
		processingPaths: make(map[string]bool),
	}

	watcher := New(cfg, mockQueue, mockProc, tempDir)
	return watcher, tempDir
}

// Helper function to create a test file with specific content and modification time
func createTestFile(t *testing.T, dir, filename string, content []byte, modTime time.Time) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !modTime.IsZero() {
		err = os.Chtimes(filePath, modTime, modTime)
		if err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	return filePath
}

func TestIsFileStable_ModificationTime(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	tests := []struct {
		name           string
		modTime        time.Time
		expectedStable bool
	}{
		{
			name:           "recently modified file should not be stable",
			modTime:        time.Now().Add(-1 * time.Second),
			expectedStable: false,
		},
		{
			name:           "old file should be stable",
			modTime:        time.Now().Add(-10 * time.Second),
			expectedStable: true,
		},
		{
			name:           "file modified exactly 2 seconds ago should be stable",
			modTime:        time.Now().Add(-2 * time.Second),
			expectedStable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file with specific modification time
			content := []byte("test content for stability check")
			// Use unique filename for each test to avoid cache conflicts
			filePath := createTestFile(t, tempDir, tt.name+".txt", content, tt.modTime)

			// Get file info
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			// First call will populate the size cache (should be false)
			firstCheck := watcher.isFileStable(filePath, info)
			if firstCheck {
				t.Error("First stability check should return false due to size cache initialization")
			}

			// Second call tests the actual stability logic
			secondCheck := watcher.isFileStable(filePath, info)

			if secondCheck != tt.expectedStable {
				t.Errorf("Expected stability %v, got %v for file modified at %v",
					tt.expectedStable, secondCheck, tt.modTime)
			}
		})
	}
}

func TestCanOpenFileExclusively(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	t.Run("can open normal file exclusively", func(t *testing.T) {
		content := []byte("test content")
		filePath := createTestFile(t, tempDir, "normal_file.txt", content, time.Time{})

		canOpen := watcher.canOpenFileExclusively(filePath)
		if !canOpen {
			t.Error("Should be able to open normal file exclusively")
		}
	})

	t.Run("cannot open file being written to", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "writing_file.txt")

		// Open file for writing to simulate file being written
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		defer func() {
			_ = file.Close()
		}()

		// Write some content to make it realistic
		_, err = file.WriteString("content being written")
		if err != nil {
			t.Fatalf("Failed to write to file: %v", err)
		}

		// Test exclusive access - this should work on most systems
		// Note: This test might be platform-dependent
		canOpen := watcher.canOpenFileExclusively(filePath)
		// On some systems, multiple readers are allowed, so we just check it doesn't panic
		_ = canOpen // We mainly test that the function doesn't crash
	})

	t.Run("cannot open non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "does_not_exist.txt")

		canOpen := watcher.canOpenFileExclusively(nonExistentPath)
		if canOpen {
			t.Error("Should not be able to open non-existent file")
		}
	})
}

func TestIsFileSizeStable(t *testing.T) {
	watcher, _ := createTestWatcher(t)

	testPath := "/test/path/file.txt"

	t.Run("first encounter should not be stable", func(t *testing.T) {
		isStable := watcher.isFileSizeStable(testPath, 1000)
		if isStable {
			t.Error("First encounter with file should not be stable")
		}
	})

	t.Run("same size should be stable", func(t *testing.T) {
		// First call (not stable)
		watcher.isFileSizeStable(testPath+"_same", 1000)
		// Second call with same size (should be stable)
		isStable := watcher.isFileSizeStable(testPath+"_same", 1000)
		if !isStable {
			t.Error("File with same size should be stable")
		}
	})

	t.Run("different size should not be stable", func(t *testing.T) {
		// First call
		watcher.isFileSizeStable(testPath+"_diff", 1000)
		// Second call with different size (should not be stable)
		isStable := watcher.isFileSizeStable(testPath+"_diff", 1500)
		if isStable {
			t.Error("File with different size should not be stable")
		}
	})
}

func TestCleanupOldCacheEntries(t *testing.T) {
	watcher, _ := createTestWatcher(t)

	// Add some entries to the cache with different timestamps
	oldTime := time.Now().Add(-25 * time.Hour)   // Older than 24 hours
	recentTime := time.Now().Add(-1 * time.Hour) // Recent

	watcher.cacheMutex.Lock()
	watcher.fileSizeCache["old_file.txt"] = fileCacheEntry{
		size:      1000,
		timestamp: oldTime,
	}
	watcher.fileSizeCache["recent_file.txt"] = fileCacheEntry{
		size:      2000,
		timestamp: recentTime,
	}
	watcher.cacheMutex.Unlock()

	// Verify both entries exist
	watcher.cacheMutex.RLock()
	if len(watcher.fileSizeCache) != 2 {
		t.Errorf("Expected 2 cache entries, got %d", len(watcher.fileSizeCache))
	}
	watcher.cacheMutex.RUnlock()

	// Run cleanup
	watcher.cleanupOldCacheEntries()

	// Verify only recent entry remains
	watcher.cacheMutex.RLock()
	defer watcher.cacheMutex.RUnlock()

	if len(watcher.fileSizeCache) != 1 {
		t.Errorf("Expected 1 cache entry after cleanup, got %d", len(watcher.fileSizeCache))
	}

	if _, exists := watcher.fileSizeCache["old_file.txt"]; exists {
		t.Error("Old cache entry should have been removed")
	}

	if _, exists := watcher.fileSizeCache["recent_file.txt"]; !exists {
		t.Error("Recent cache entry should have been preserved")
	}
}

func TestShouldProcessFile_WithStabilityCheck(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	tests := []struct {
		name           string
		fileSize       int64
		modTime        time.Time
		ignorePattern  string
		expectedResult bool
	}{
		{
			name:           "stable file should be processed",
			fileSize:       1000,
			modTime:        time.Now().Add(-10 * time.Second),
			ignorePattern:  "",
			expectedResult: true, // Will be false on first run due to size cache, true on second
		},
		{
			name:           "recently modified file should not be processed",
			fileSize:       1000,
			modTime:        time.Now().Add(-1 * time.Second),
			ignorePattern:  "",
			expectedResult: false,
		},
		{
			name:           "file too small should not be processed",
			fileSize:       5, // Below MinFileSize of 10
			modTime:        time.Now().Add(-10 * time.Second),
			ignorePattern:  "",
			expectedResult: false,
		},
		{
			name:           "ignored pattern should not be processed",
			fileSize:       1000,
			modTime:        time.Now().Add(-10 * time.Second),
			ignorePattern:  "*.tmp",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update ignore patterns if specified
			if tt.ignorePattern != "" {
				watcher.cfg.IgnorePatterns = []string{tt.ignorePattern}
			} else {
				watcher.cfg.IgnorePatterns = []string{}
			}

			// Create test file
			content := make([]byte, tt.fileSize)
			filename := "test_process.txt"
			if tt.ignorePattern == "*.tmp" {
				filename = "test_process.tmp"
			}

			filePath := createTestFile(t, tempDir, filename, content, tt.modTime)

			// Get file info
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			// For stable files, we need to call twice due to size cache logic
			if tt.expectedResult && tt.fileSize >= watcher.cfg.MinFileSize {
				// First call to populate cache
				watcher.shouldProcessFile(filePath, info)
				// Second call should return true for stable file
			}

			// Test should process file
			result := watcher.shouldProcessFile(filePath, info)

			if result != tt.expectedResult {
				t.Errorf("Expected shouldProcessFile to return %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestIntegrationStabilityCheck(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	// Create a file that grows over time
	filePath := filepath.Join(tempDir, "growing_file.txt")

	// First write - make sure file is large enough to meet size requirements
	initialContent := make([]byte, 1500) // Larger than SizeThreshold (100) and MinFileSize (10)
	for i := range initialContent {
		initialContent[i] = 'a'
	}
	err := os.WriteFile(filePath, initialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	info1, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should not be stable (first encounter)
	shouldProcess1 := watcher.shouldProcessFile(filePath, info1)
	if shouldProcess1 {
		t.Error("File should not be processed on first encounter")
	}

	// Simulate file growth
	time.Sleep(100 * time.Millisecond)
	grownContent := make([]byte, 2000) // Make it even larger
	for i := range grownContent {
		grownContent[i] = 'b'
	}
	err = os.WriteFile(filePath, grownContent, 0644)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	info2, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should not be stable (size changed)
	shouldProcess2 := watcher.shouldProcessFile(filePath, info2)
	if shouldProcess2 {
		t.Error("Growing file should not be processed")
	}

	// Wait for modification time stability and check again with same size
	time.Sleep(3 * time.Second)

	info3, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// First check after stability wait - updates cache with stable size
	shouldProcess3a := watcher.shouldProcessFile(filePath, info3)
	if shouldProcess3a {
		t.Error("First check after stability should still return false due to size cache logic")
	}

	// Second check - now file should be stable (old mod time, same size as cached)
	shouldProcess3b := watcher.shouldProcessFile(filePath, info3)
	if !shouldProcess3b {
		t.Error("Second check of stable file should be processed")
	}
}

func TestDuplicatePrevention(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	// Create a stable file that meets all criteria
	content := make([]byte, 1500) // Large enough file
	for i := range content {
		content[i] = 'x'
	}
	filePath := createTestFile(t, tempDir, "duplicate_test.txt", content, time.Now().Add(-10*time.Second))

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Mock queue to track AddFile calls
	mockQueue := &mockQueueWithDuplicateCheck{
		addFileCalls: make([]string, 0),
	}
	watcher.queue = mockQueue

	// First scan - file should be processed
	watcher.shouldProcessFile(filePath, info)                   // Populate size cache
	shouldProcess1 := watcher.shouldProcessFile(filePath, info) // Now stable
	if !shouldProcess1 {
		t.Error("Stable file should be processed on first scan")
	}

	// Simulate adding to queue on first scan
	err = watcher.queue.AddFile(context.TODO(), filePath, info.Size())
	if err != nil {
		t.Fatalf("Failed to add file to queue: %v", err)
	}

	// Verify first call was made
	if len(mockQueue.addFileCalls) != 1 {
		t.Errorf("Expected 1 AddFile call, got %d", len(mockQueue.addFileCalls))
	}

	// Second scan - file should still be stable but queue should prevent duplicate
	shouldProcess2 := watcher.shouldProcessFile(filePath, info)
	if !shouldProcess2 {
		t.Error("File should still be considered processable")
	}

	// Simulate adding to queue on second scan (should be prevented by queue)
	err = watcher.queue.AddFile(context.TODO(), filePath, info.Size())
	if err != nil {
		t.Errorf("AddFile should not return error for duplicate: %v", err)
	}

	// Verify still only one call was actually processed by queue
	if len(mockQueue.addFileCalls) != 1 {
		t.Errorf("Expected queue to prevent duplicate, still got %d AddFile calls", len(mockQueue.addFileCalls))
	}
}

func TestMultipleScanCycles(t *testing.T) {
	watcher, tempDir := createTestWatcher(t)

	// Mock queue to track calls
	mockQueue := &mockQueueWithDuplicateCheck{
		addFileCalls: make([]string, 0),
	}
	watcher.queue = mockQueue

	// Create multiple files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range files {
		content := make([]byte, 1500)
		for i := range content {
			content[i] = 'y'
		}
		createTestFile(t, tempDir, filename, content, time.Now().Add(-10*time.Second))
	}

	// Simulate multiple scan cycles
	for cycle := 0; cycle < 3; cycle++ {
		t.Logf("Scan cycle %d", cycle+1)

		for _, filename := range files {
			filePath := filepath.Join(tempDir, filename)
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			// First call to populate size cache, second to check stability
			watcher.shouldProcessFile(filePath, info)
			if watcher.shouldProcessFile(filePath, info) {
				err = watcher.queue.AddFile(context.TODO(), filePath, info.Size())
				if err != nil {
					t.Errorf("Cycle %d: AddFile failed for %s: %v", cycle+1, filename, err)
				}
			}
		}
	}

	// Verify each file was only added once despite multiple scans
	if len(mockQueue.addFileCalls) != len(files) {
		t.Errorf("Expected %d unique files in queue, got %d calls: %v",
			len(files), len(mockQueue.addFileCalls), mockQueue.addFileCalls)
	}

	// Verify all expected files are present
	expectedFiles := make(map[string]bool)
	for _, filename := range files {
		expectedFiles[filepath.Join(tempDir, filename)] = true
	}

	for _, addedFile := range mockQueue.addFileCalls {
		if !expectedFiles[addedFile] {
			t.Errorf("Unexpected file added to queue: %s", addedFile)
		}
		delete(expectedFiles, addedFile)
	}

	if len(expectedFiles) > 0 {
		t.Errorf("Some files were not added to queue: %v", expectedFiles)
	}
}

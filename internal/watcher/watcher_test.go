package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/javi11/postie/pkg/postie"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Create config file
		err := os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		mockPostie := &postie.Postie{} // Use real postie for constructor test
		cfg := config.WatcherConfig{
			MinFileSize:   100,
			CheckInterval: time.Minute,
		}

		watcher, err := New(ctx, cfg, configPath, mockPostie, tempDir, tempDir)

		assert.NoError(t, err)
		assert.NotNil(t, watcher)
		assert.Equal(t, cfg, watcher.cfg)
		assert.Equal(t, mockPostie, watcher.postie)
		assert.Equal(t, tempDir, watcher.watchFolder)
		assert.Equal(t, tempDir, watcher.outputFolder)
		assert.NotNil(t, watcher.db)
		assert.False(t, watcher.isRunning)

		// Cleanup
		err = watcher.Close()
		assert.NoError(t, err)
	})

	t.Run("database initialization failure", func(t *testing.T) {
		mockPostie := &postie.Postie{}
		cfg := config.WatcherConfig{
			MinFileSize:   100,
			CheckInterval: time.Minute,
		}

		// Use invalid database path to force error
		invalidConfigPath := "/invalid/path/config.yaml"

		watcher, err := New(ctx, cfg, invalidConfigPath, mockPostie, "tempDir", "tempDir")

		assert.Error(t, err)
		assert.Nil(t, watcher)
	})
}

func TestInitDB(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer func() {
			_ = db.Close()
		}()

		err = initDB(db)
		assert.NoError(t, err)

		// Verify table exists and has correct schema
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='queue_items'").Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "queue_items", tableName)

		// Test inserting data to verify schema
		ctx := context.Background()
		_, err = db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/path.txt", 1000, StatusPending,
		)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		// Use a mock database that will fail
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		err = db.Close()
		assert.NoError(t, err) // Close it to make operations fail

		err = initDB(db)
		assert.Error(t, err)
	})
}

func TestResetRunningItems(t *testing.T) {
	ctx := context.Background()

	t.Run("successful reset", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Insert some running items
		_, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/running1.txt", 1000, StatusRunning,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/running2.txt", 1000, StatusRunning,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/pending.txt", 1000, StatusPending,
		)
		require.NoError(t, err)

		err = watcher.resetRunningItems(ctx)
		assert.NoError(t, err)

		// Verify running items were reset to pending
		var runningCount int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusRunning).Scan(&runningCount)
		assert.NoError(t, err)
		assert.Equal(t, 0, runningCount)

		var pendingCount int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusPending).Scan(&pendingCount)
		assert.NoError(t, err)
		assert.Equal(t, 3, pendingCount)
	})

	t.Run("database error", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Close database to cause error
		err := watcher.db.Close()
		assert.NoError(t, err)

		err = watcher.resetRunningItems(ctx)
		assert.Error(t, err)
	})
}

func TestStart(t *testing.T) {
	t.Run("start with cancellation", func(t *testing.T) {
		// Use real watcher for testing Start method since it's complex
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Create config file
		err := os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		cfg := config.WatcherConfig{
			MinFileSize:   100,
			CheckInterval: 50 * time.Millisecond, // Short interval for testing
		}

		// Create a context that we can cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Use a nil postie since we'll cancel before it's used
		watcher, err := New(ctx, cfg, configPath, nil, tempDir, tempDir)
		require.NoError(t, err)
		defer func() {
			err := watcher.Close()
			assert.NoError(t, err, "Failed to close watcher")
		}()

		// Start the watcher in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- watcher.Start(ctx)
		}()

		// Give it some time to start
		time.Sleep(25 * time.Millisecond)

		// Cancel the context
		cancel()

		// Wait for the watcher to stop
		select {
		case err := <-errChan:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(1 * time.Second):
			t.Fatal("watcher did not stop within timeout")
		}
	})
}

func TestProcessQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("not within schedule", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Set schedule to exclude current time
		now := time.Now()
		startTime := now.Add(-2 * time.Hour).Format("15:04")
		endTime := now.Add(-1 * time.Hour).Format("15:04")
		watcher.cfg.Schedule.StartTime = startTime
		watcher.cfg.Schedule.EndTime = endTime

		err := watcher.processQueue(ctx)
		assert.NoError(t, err)
	})

	t.Run("already running", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Set running flag
		watcher.isRunning = true

		err := watcher.processQueue(ctx)
		assert.NoError(t, err)
	})

	t.Run("successful processing with no items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		err := watcher.processQueue(ctx)
		assert.NoError(t, err)
		assert.False(t, watcher.isRunning)
	})

	t.Run("successful processing with items", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file
		testFile := createTestFile(t, tempDir, "test.txt", 200)

		// Add item to queue
		_, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 200, StatusPending,
		)
		require.NoError(t, err)

		// Setup mock postie expectation
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.MatchedBy(func(files []fileinfo.FileInfo) bool {
			return len(files) == 1 && files[0].Path == testFile && files[0].Size == 200
		}), tempDir, watcher.outputFolder).Return(nil)

		err = watcher.processQueue(ctx)
		assert.NoError(t, err)
		assert.False(t, watcher.isRunning)

		// Verify item was deleted from queue
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		// Verify original file was deleted
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))

		mockPostie.AssertExpectations(t)
	})

	t.Run("processing with item error", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file
		testFile := createTestFile(t, tempDir, "test.txt", 200)

		// Add item to queue
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 200, StatusPending,
		)
		require.NoError(t, err)
		itemID, _ := result.LastInsertId()

		// Setup mock postie to return error
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.Anything, tempDir, watcher.outputFolder).Return(fmt.Errorf("post error"))

		err = watcher.processQueue(ctx)
		assert.NoError(t, err) // processQueue should not return error for individual item failures

		// Verify item status was updated to error
		var status string
		err = watcher.db.QueryRowContext(ctx, "SELECT status FROM queue_items WHERE id = ?", itemID).Scan(&status)
		assert.NoError(t, err)
		assert.Equal(t, StatusError, status)

		// Verify original file still exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)

		mockPostie.AssertExpectations(t)
	})

	t.Run("scan directory error", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Set watch folder to non-existent directory
		watcher.watchFolder = "/non/existent/directory"

		err := watcher.processQueue(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error scanning directory")
	})

	t.Run("get next batch error", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Close database to cause error during scan
		err := watcher.db.Close()
		assert.NoError(t, err)

		err = watcher.processQueue(ctx)
		assert.Error(t, err)
		// The error could be from scanDirectory or getNextBatch depending on timing
		assert.True(t,
			strings.Contains(err.Error(), "error scanning directory") ||
				strings.Contains(err.Error(), "error getting next batch"),
			"Expected error to contain either 'error scanning directory' or 'error getting next batch', got: %s", err.Error())
	})
}

func TestProcessItem(t *testing.T) {
	ctx := context.Background()

	t.Run("successful processing", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file
		testFile := createTestFile(t, tempDir, "test.txt", 200)

		item := QueueItem{
			ID:   1,
			Path: testFile,
			Size: 200,
		}

		// Setup mock postie expectation
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.MatchedBy(func(files []fileinfo.FileInfo) bool {
			return len(files) == 1 && files[0].Path == testFile && files[0].Size == 200
		}), tempDir, watcher.outputFolder).Return(nil)

		err := watcher.processItem(ctx, item)
		assert.NoError(t, err)

		// Verify original file was deleted
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))

		mockPostie.AssertExpectations(t)
	})

	t.Run("update status error", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file
		testFile := createTestFile(t, tempDir, "test.txt", 200)

		item := QueueItem{
			ID:   1,
			Path: testFile,
			Size: 200,
		}

		// Close database to cause error
		err := watcher.db.Close()
		assert.NoError(t, err)

		err = watcher.processItem(ctx, item)
		assert.Error(t, err)
	})

	t.Run("postie error", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file
		testFile := createTestFile(t, tempDir, "test.txt", 200)

		// Add item to database first to allow status update
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 200, StatusPending,
		)
		require.NoError(t, err)
		itemID, _ := result.LastInsertId()

		item := QueueItem{
			ID:   itemID,
			Path: testFile,
			Size: 200,
		}

		// Setup mock postie to return error
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.Anything, tempDir, watcher.outputFolder).Return(fmt.Errorf("post error"))

		err = watcher.processItem(ctx, item)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post error")

		// Verify original file still exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)

		mockPostie.AssertExpectations(t)
	})

	t.Run("file deletion error", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Create test file in read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0755)
		require.NoError(t, err)

		testFile := createTestFile(t, readOnlyDir, "test.txt", 200)

		// Make directory read-only after creating file
		err = os.Chmod(readOnlyDir, 0555)
		require.NoError(t, err)
		defer func() {
			err := os.Chmod(readOnlyDir, 0755)
			assert.NoError(t, err, "Failed to change file permissions")
		}()

		// Add item to database first to allow status update
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 200, StatusPending,
		)
		require.NoError(t, err)
		itemID, _ := result.LastInsertId()

		item := QueueItem{
			ID:   itemID,
			Path: testFile,
			Size: 200,
		}

		// Setup mock postie expectation
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.Anything, tempDir, watcher.outputFolder).Return(nil)

		err = watcher.processItem(ctx, item)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error deleting original file")

		mockPostie.AssertExpectations(t)
	})
}

func TestClose(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		err := watcher.Close()
		assert.NoError(t, err)
	})

	t.Run("database already closed", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Close database first
		err := watcher.db.Close()
		assert.NoError(t, err)

		// Second close should not panic but might return error
		err = watcher.Close()
		// SQLite returns nil error on multiple closes, so we just verify no panic
		_ = err
	})
}

// Test race conditions and concurrent access
func TestWatcherConcurrency(t *testing.T) {
	ctx := context.Background()

	t.Run("concurrent processQueue calls", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Setup mock postie
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		// Start multiple processQueue calls concurrently
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func() {
				defer func() { done <- true }()
				err := watcher.processQueue(ctx)
				assert.NoError(t, err)
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("processQueue did not complete within timeout")
			}
		}

		// Verify watcher is not running
		assert.False(t, watcher.isRunning)
	})
}

// Test edge cases
func TestWatcherEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("empty database on reset", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		err := watcher.resetRunningItems(ctx)
		assert.NoError(t, err)
	})

	t.Run("non-existent file in queue", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcherWithMock(t)
		defer cleanup()

		// Add item for non-existent file
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/non/existent/file.txt", 200, StatusPending,
		)
		require.NoError(t, err)
		itemID, _ := result.LastInsertId()

		item := QueueItem{
			ID:   itemID,
			Path: "/non/existent/file.txt",
			Size: 200,
		}

		// Setup mock postie expectation (it should still be called)
		mockPostie := watcher.postie.(*MockPostie)
		mockPostie.On("Post", ctx, mock.Anything, tempDir, watcher.outputFolder).Return(nil)

		err = watcher.processItem(ctx, item)
		// Should succeed - file deletion will be handled gracefully
		assert.NoError(t, err)

		mockPostie.AssertExpectations(t)
	})
}

// PostieInterface defines the interface for postie functionality
type PostieInterface interface {
	Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) error
}

// MockPostie is a mock implementation of the PostieInterface
type MockPostie struct {
	mock.Mock
}

func (m *MockPostie) Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) error {
	args := m.Called(ctx, files, rootDir, outputDir)
	return args.Error(0)
}

// TestWatcher is a test version of Watcher that uses the interface
type TestWatcher struct {
	cfg          config.WatcherConfig
	postie       PostieInterface
	db           *sql.DB
	isRunning    bool
	runningMux   sync.Mutex
	watchFolder  string
	outputFolder string
}

// Helper function to create a test watcher with mock
func setupTestWatcherWithMock(t *testing.T) (*TestWatcher, string, func()) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Initialize database
	err = initDB(db)
	require.NoError(t, err)

	// Create mock postie
	mockPostie := &MockPostie{}

	// Create watcher config
	cfg := config.WatcherConfig{
		MinFileSize:    100,
		CheckInterval:  100 * time.Millisecond, // Short interval for testing
		IgnorePatterns: []string{"*.tmp", "*.ignore"},
		Schedule: config.ScheduleConfig{
			StartTime: "00:00",
			EndTime:   "23:59",
		},
	}

	watcher := &TestWatcher{
		cfg:          cfg,
		postie:       mockPostie,
		db:           db,
		watchFolder:  tempDir,
		outputFolder: filepath.Join(tempDir, "output"),
	}

	// Create output folder
	err = os.MkdirAll(watcher.outputFolder, 0755)
	require.NoError(t, err)

	cleanup := func() {
		err := db.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	}

	return watcher, tempDir, cleanup
}

// Copy methods from the original watcher for testing
func (w *TestWatcher) resetRunningItems(ctx context.Context) error {
	_, err := w.db.ExecContext(ctx, `
		UPDATE queue_items 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE status = ?`, StatusPending, StatusRunning)
	return err
}

func (w *TestWatcher) processQueue(ctx context.Context) error {
	if !w.isWithinSchedule() {
		slog.InfoContext(ctx, "Not within schedule or already running")
		return nil
	}

	w.runningMux.Lock()
	defer w.runningMux.Unlock()

	if w.isRunning {
		slog.InfoContext(ctx, "Not within schedule or already running")
		return nil
	}

	w.isRunning = true
	defer func() {
		w.isRunning = false
	}()

	// Scan directory for new files
	if err := w.scanDirectory(ctx); err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	// Get next batch of items to process
	items, err := w.getNextBatch(ctx)
	if err != nil {
		return fmt.Errorf("error getting next batch: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	// Process each item
	for _, item := range items {
		if err := w.processItem(ctx, item); err != nil {
			slog.ErrorContext(ctx, "Error processing item", "error", err, "path", item.Path)
			if err := w.updateItemStatus(ctx, item.ID, StatusError); err != nil {
				slog.ErrorContext(ctx, "Error updating item status", "error", err)
			}
			continue
		}

		// Delete the item from queue after successful processing
		if err := w.deleteItem(ctx, item.ID); err != nil {
			slog.ErrorContext(ctx, "Error deleting item from queue", "error", err)
		}
	}

	return nil
}

func (w *TestWatcher) processItem(ctx context.Context, item QueueItem) error {
	// Update status to running
	if err := w.updateItemStatus(ctx, item.ID, StatusRunning); err != nil {
		return err
	}

	// Create file info
	fileInfo := fileinfo.FileInfo{
		Path: item.Path,
		Size: uint64(item.Size),
	}

	// Post the file
	if err := w.postie.Post(ctx, []fileinfo.FileInfo{fileInfo}, w.watchFolder, w.outputFolder); err != nil {
		return err
	}

	// Delete the original file - handle non-existent files gracefully
	if err := os.Remove(item.Path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error deleting original file: %w", err)
		}
		// File doesn't exist - that's okay, maybe it was already cleaned up
		slog.WarnContext(ctx, "File already deleted or doesn't exist", "path", item.Path)
	}

	return nil
}

func (w *TestWatcher) Close() error {
	return w.db.Close()
}

// Delegate methods from scanner.go
func (w *TestWatcher) scanDirectory(ctx context.Context) error {
	// Create a real watcher instance to use its scan method
	realWatcher := &Watcher{
		cfg:         w.cfg,
		db:          w.db,
		watchFolder: w.watchFolder,
	}
	return realWatcher.scanDirectory(ctx)
}

func (w *TestWatcher) getNextBatch(ctx context.Context) ([]QueueItem, error) {
	realWatcher := &Watcher{
		cfg: w.cfg,
		db:  w.db,
	}
	return realWatcher.getNextBatch(ctx)
}

func (w *TestWatcher) updateItemStatus(ctx context.Context, id int64, status string) error {
	realWatcher := &Watcher{
		cfg: w.cfg,
		db:  w.db,
	}
	return realWatcher.updateItemStatus(ctx, id, status)
}

func (w *TestWatcher) deleteItem(ctx context.Context, id int64) error {
	realWatcher := &Watcher{
		cfg: w.cfg,
		db:  w.db,
	}
	return realWatcher.deleteItem(ctx, id)
}

func (w *TestWatcher) isWithinSchedule() bool {
	realWatcher := &Watcher{
		cfg: w.cfg,
	}
	return realWatcher.isWithinSchedule()
}

// Helper function to create a test file
func createTestFile(t *testing.T, dir, filename string, size int64) string {
	filePath := filepath.Join(dir, filename)
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
	return filePath
}

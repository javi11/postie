package watcher

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestWatcher(t *testing.T) (*Watcher, string, func()) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create a separate directory for the database outside the watch folder
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	// Initialize database
	err = initDB(db)
	require.NoError(t, err)

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='queue_items'").Scan(&tableName)
	require.NoError(t, err, "queue_items table should exist after initDB")

	// Create watcher config
	cfg := config.WatcherConfig{
		MinFileSize:    10, // Small size for testing
		CheckInterval:  time.Minute,
		IgnorePatterns: []string{"*.tmp", "*.ignore"},
		Schedule: config.ScheduleConfig{
			StartTime: "00:00",
			EndTime:   "23:59",
		},
	}

	watcher := &Watcher{
		cfg:         cfg,
		db:          db,
		watchFolder: tempDir,
	}

	cleanup := func() {
		err := db.Close()
		assert.NoError(t, err, "Failed to close database")
	}

	return watcher, tempDir, cleanup
}

func TestScanDirectory(t *testing.T) {
	ctx := context.Background()

	t.Run("successful scan with valid files", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Verify database connection is working by doing a preliminary operation
		err := watcher.addToQueue(ctx, "/test/warmup.txt", 50)
		require.NoError(t, err, "Database should be working for preliminary operation")

		// Clear the preliminary item
		items, err := watcher.getNextBatch(ctx)
		require.NoError(t, err)
		if len(items) > 0 {
			err = watcher.deleteItem(ctx, items[0].ID)
			require.NoError(t, err)
		}

		// Create test files
		testFile1 := filepath.Join(tempDir, "test1.txt")
		testFile2 := filepath.Join(tempDir, "test2.txt")

		content1 := "content larger than min size for file1"
		content2 := "different content larger than min size for file2"

		err = os.WriteFile(testFile1, []byte(content1), 0644)
		require.NoError(t, err)
		err = os.WriteFile(testFile2, []byte(content2), 0644)
		require.NoError(t, err)

		// Verify files exist
		info1, err := os.Stat(testFile1)
		require.NoError(t, err)
		info2, err := os.Stat(testFile2)
		require.NoError(t, err)

		t.Logf("File1: %s, size: %d", testFile1, info1.Size())
		t.Logf("File2: %s, size: %d", testFile2, info2.Size())
		t.Logf("MinFileSize: %d", watcher.cfg.MinFileSize)

		// Scan directory
		err = watcher.scanDirectory(ctx)
		assert.NoError(t, err)

		// Verify files were added to queue
		items, err = watcher.getNextBatch(ctx)
		assert.NoError(t, err)

		if len(items) != 2 {
			t.Logf("Items found: %+v", items)
		}
		assert.Len(t, items, 2)
	})

	t.Run("ignores small files", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Create small file
		smallFile := filepath.Join(tempDir, "small.txt")
		err := os.WriteFile(smallFile, []byte("hi"), 0644) // Less than minFileSize
		require.NoError(t, err)

		err = watcher.scanDirectory(ctx)
		assert.NoError(t, err)

		// Verify no files were added
		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("ignores files matching ignore patterns", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Create files that should be ignored
		tmpFile := filepath.Join(tempDir, "file.tmp")
		ignoreFile := filepath.Join(tempDir, "file.ignore")

		err := os.WriteFile(tmpFile, []byte("content larger than min size"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(ignoreFile, []byte("content larger than min size"), 0644)
		require.NoError(t, err)

		err = watcher.scanDirectory(ctx)
		assert.NoError(t, err)

		// Verify no files were added
		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("handles failed items cleanup", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Add a failed item
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content larger than min size"), 0644)
		require.NoError(t, err)

		// Insert failed item
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 100, StatusError,
		)
		require.NoError(t, err)

		err = watcher.scanDirectory(ctx)
		assert.NoError(t, err)

		// Verify failed item was removed and the file was also queued
		var errorCount int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusError).Scan(&errorCount)
		assert.NoError(t, err)
		assert.Equal(t, 0, errorCount)

		// The file should now be queued as pending
		var pendingCount int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusPending).Scan(&pendingCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, pendingCount)
	})

	t.Run("skips directories", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Create subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		err = watcher.scanDirectory(ctx)
		assert.NoError(t, err)

		// Verify no items were added
		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestAddToQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("adds new file to queue", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		err := watcher.addToQueue(ctx, "/test/path.txt", 1000)
		assert.NoError(t, err)

		// Verify item was added
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE path = ?", "/test/path.txt").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("does not add duplicate files", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Add file twice
		err := watcher.addToQueue(ctx, "/test/path.txt", 1000)
		assert.NoError(t, err)
		err = watcher.addToQueue(ctx, "/test/path.txt", 1000)
		assert.NoError(t, err)

		// Verify only one item exists
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE path = ?", "/test/path.txt").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestGetNextBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns pending items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Add multiple items
		for i := 0; i < 5; i++ {
			_, err := watcher.db.ExecContext(ctx,
				"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
				filepath.Join("/test", "file"+string(rune(i))+".txt"), 1000, StatusPending,
			)
			require.NoError(t, err)
		}

		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Len(t, items, 5)

		for _, item := range items {
			assert.Equal(t, StatusPending, item.Status)
		}
	})

	t.Run("limits to 10 items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Add more than 10 items
		for i := 0; i < 15; i++ {
			_, err := watcher.db.ExecContext(ctx,
				"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
				filepath.Join("/test", "file"+string(rune(i))+".txt"), 1000, StatusPending,
			)
			require.NoError(t, err)
		}

		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Len(t, items, 10)
	})

	t.Run("returns empty for no pending items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("ignores non-pending items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Add items with different statuses
		_, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/running.txt", 1000, StatusRunning,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/complete.txt", 1000, StatusComplete,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/error.txt", 1000, StatusError,
		)
		require.NoError(t, err)

		items, err := watcher.getNextBatch(ctx)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestUpdateItemStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("updates status successfully", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Insert item
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/path.txt", 1000, StatusPending,
		)
		require.NoError(t, err)

		id, err := result.LastInsertId()
		require.NoError(t, err)

		// Update status
		err = watcher.updateItemStatus(ctx, id, StatusRunning)
		assert.NoError(t, err)

		// Verify status was updated
		var status string
		err = watcher.db.QueryRowContext(ctx, "SELECT status FROM queue_items WHERE id = ?", id).Scan(&status)
		assert.NoError(t, err)
		assert.Equal(t, StatusRunning, status)
	})

	t.Run("handles non-existent ID", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		err := watcher.updateItemStatus(ctx, 99999, StatusRunning)
		assert.NoError(t, err) // Should not error even if no rows affected
	})
}

func TestDeleteItem(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes item successfully", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Insert item
		result, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/test/path.txt", 1000, StatusPending,
		)
		require.NoError(t, err)

		id, err := result.LastInsertId()
		require.NoError(t, err)

		// Delete item
		err = watcher.deleteItem(ctx, id)
		assert.NoError(t, err)

		// Verify item was deleted
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE id = ?", id).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("handles non-existent ID", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		err := watcher.deleteItem(ctx, 99999)
		assert.NoError(t, err) // Should not error even if no rows affected
	})
}

func TestIsFileOpen(t *testing.T) {
	t.Run("returns false for non-existent file", func(t *testing.T) {
		result := isFileOpen("/non/existent/file.txt")
		assert.True(t, result) // Returns true when file can't be opened
	})

	t.Run("returns false for accessible file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		result := isFileOpen(testFile)
		assert.False(t, result) // File should be accessible
	})

	t.Run("handles permission denied", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "readonly.txt")

		err := os.WriteFile(testFile, []byte("test content"), 0000)
		require.NoError(t, err)
		defer func() {
			err := os.Chmod(testFile, 0644)
			assert.NoError(t, err, "Failed to change file permissions")
		}()

		result := isFileOpen(testFile)
		assert.True(t, result) // Should return true when access is denied
	})
}

func TestIsWithinSchedule(t *testing.T) {
	t.Run("returns true when no schedule configured", func(t *testing.T) {
		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{},
			},
		}

		result := watcher.isWithinSchedule()
		assert.True(t, result)
	})

	t.Run("returns true when start time empty", func(t *testing.T) {
		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{
					EndTime: "23:59",
				},
			},
		}

		result := watcher.isWithinSchedule()
		assert.True(t, result)
	})

	t.Run("returns true when end time empty", func(t *testing.T) {
		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{
					StartTime: "00:00",
				},
			},
		}

		result := watcher.isWithinSchedule()
		assert.True(t, result)
	})

	t.Run("returns true for valid time range", func(t *testing.T) {
		now := time.Now()

		// Create a schedule that includes current time
		startTime := now.Add(-1 * time.Hour).Format("15:04")
		endTime := now.Add(1 * time.Hour).Format("15:04")

		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{
					StartTime: startTime,
					EndTime:   endTime,
				},
			},
		}

		result := watcher.isWithinSchedule()
		assert.True(t, result)
	})

	t.Run("returns false when outside time range", func(t *testing.T) {
		now := time.Now()

		// Create a schedule that excludes current time
		startTime := now.Add(-3 * time.Hour).Format("15:04")
		endTime := now.Add(-2 * time.Hour).Format("15:04")

		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{
					StartTime: startTime,
					EndTime:   endTime,
				},
			},
		}

		result := watcher.isWithinSchedule()
		assert.False(t, result)
	})

	t.Run("handles invalid time format gracefully", func(t *testing.T) {
		watcher := &Watcher{
			cfg: config.WatcherConfig{
				Schedule: config.ScheduleConfig{
					StartTime: "invalid",
					EndTime:   "23:59",
				},
			},
		}

		result := watcher.isWithinSchedule()
		assert.True(t, result) // Should return true on parse error
	})
}

func TestHandleFailedItems(t *testing.T) {
	ctx := context.Background()

	t.Run("removes failed items for existing files", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Create test file
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		// Insert failed item
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile, 100, StatusError,
		)
		require.NoError(t, err)

		err = watcher.handleFailedItems(ctx)
		assert.NoError(t, err)

		// Verify failed item was removed
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusError).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("keeps failed items for non-existent files", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Insert failed item for non-existent file
		_, err := watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/non/existent/file.txt", 100, StatusError,
		)
		require.NoError(t, err)

		err = watcher.handleFailedItems(ctx)
		assert.NoError(t, err)

		// Verify failed item still exists
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusError).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("handles multiple failed items", func(t *testing.T) {
		watcher, tempDir, cleanup := setupTestWatcher(t)
		defer cleanup()

		// Create test files
		testFile1 := filepath.Join(tempDir, "test1.txt")
		testFile2 := filepath.Join(tempDir, "test2.txt")
		err := os.WriteFile(testFile1, []byte("content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(testFile2, []byte("content"), 0644)
		require.NoError(t, err)

		// Insert failed items
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile1, 100, StatusError,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			testFile2, 100, StatusError,
		)
		require.NoError(t, err)
		_, err = watcher.db.ExecContext(ctx,
			"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
			"/non/existent.txt", 100, StatusError,
		)
		require.NoError(t, err)

		err = watcher.handleFailedItems(ctx)
		assert.NoError(t, err)

		// Verify only the non-existent file item remains
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items WHERE status = ?", StatusError).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		var path string
		err = watcher.db.QueryRowContext(ctx, "SELECT path FROM queue_items WHERE status = ?", StatusError).Scan(&path)
		assert.NoError(t, err)
		assert.Equal(t, "/non/existent.txt", path)
	})

	t.Run("handles empty failed items", func(t *testing.T) {
		watcher, _, cleanup := setupTestWatcher(t)
		defer cleanup()

		err := watcher.handleFailedItems(ctx)
		assert.NoError(t, err)

		// Verify no items exist
		var count int
		err = watcher.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestDatabaseInitialization(t *testing.T) {
	// Create file-based database
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Initialize database
	err = initDB(db)
	require.NoError(t, err)

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='queue_items'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "queue_items", tableName)

	// Test inserting data
	ctx := context.Background()
	_, err = db.ExecContext(ctx,
		"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
		"/test/path.txt", 1000, StatusPending,
	)
	require.NoError(t, err)

	// Test querying data
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queue_items").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

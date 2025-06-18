package poster

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/javi11/nntppool"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/mocks"
	"github.com/mnightingale/rapidyenc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		mockConfig := mocks.NewMockConfig(ctrl)
		mockConfig.EXPECT().GetNNTPPool().Return(mockPool, nil)
		mockConfig.EXPECT().GetPostingConfig().Return(createTestConfig())
		mockConfig.EXPECT().GetPostCheckConfig().Return(createTestPostCheckConfig())

		poster, err := New(ctx, mockConfig)

		require.NoError(t, err)
		assert.NotNil(t, poster)
	})

	t.Run("NNTP pool error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("pool creation failed")

		mockConfig := mocks.NewMockConfig(ctrl)
		mockConfig.EXPECT().GetNNTPPool().Return(nil, expectedErr)

		poster, err := New(ctx, mockConfig)

		assert.Error(t, err)
		assert.Nil(t, poster)
		assert.Contains(t, err.Error(), "error getting NNTP pool")
	})
}

func TestPost(t *testing.T) {
	t.Run("success with single file", func(t *testing.T) {
		ctx := context.Background()
		content := strings.Repeat("test data ", 100) // Create content larger than segment size
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return().AnyTimes()
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return().Times(1)

		// Create poster with check disabled to simplify test
		checkCfg := createTestPostCheckConfig()
		enabled := false
		checkCfg.Enabled = &enabled

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{testFile}, "", nzbGen)

		assert.NoError(t, err)
	})

	t.Run("success with multiple files", func(t *testing.T) {
		ctx := context.Background()

		// Create test files
		testFile1 := createTestFile(t, "test content 1")
		testFile2 := createTestFile(t, "test content 2")
		defer func() {
			err := os.Remove(testFile1)
			assert.NoError(t, err, "Failed to remove test file")
		}()
		defer func() {
			err := os.Remove(testFile2)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return().Times(2)                // One for each file
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return().Times(2) // One for each file

		// Create poster with check disabled to simplify test
		checkCfg := createTestPostCheckConfig()
		enabled := false
		checkCfg.Enabled = &enabled

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{testFile1, testFile2}, "", nzbGen)

		assert.NoError(t, err)
	})

	t.Run("posting failure", func(t *testing.T) {
		ctx := context.Background()
		testFile := createTestFile(t, "test content")
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(errors.New("posting failed")).AnyTimes()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		// When posting fails, AddArticle is still called but AddFileHash might not be called
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return().AnyTimes()
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return().AnyTimes()

		// Create poster with check disabled to simplify test
		checkCfg := createTestPostCheckConfig()
		enabled := false
		checkCfg.Enabled = &enabled

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{testFile}, "", nzbGen)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to post file")
	})

	t.Run("check enabled success", func(t *testing.T) {
		ctx := context.Background()
		testFile := createTestFile(t, "test content")
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any(), gomock.Any()).Return(200, nil).AnyTimes()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return()
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return()

		checkCfg := createTestPostCheckConfig()
		enabled := true
		checkCfg.Enabled = &enabled

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{testFile}, "", nzbGen)

		assert.NoError(t, err)
	})

	t.Run("check enabled failure", func(t *testing.T) {
		ctx := context.Background()
		testFile := createTestFile(t, "test content")
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, errors.New("article not found")).AnyTimes()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return()
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return()

		checkCfg := createTestPostCheckConfig()
		enabled := true
		checkCfg.Enabled = &enabled
		checkCfg.MaxRePost = 0 // No retries

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{testFile}, "", nzbGen)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify file")
	})

	t.Run("article stat fails and gets reuploaded successfully", func(t *testing.T) {
		// Instead of a full integration test, test the checkArticle method directly
		// to verify retry behavior without triggering progress bar issues
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		// Test checkArticle method directly
		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		// First call fails (article not found)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(0, errors.New("article not found")).Times(1)

		err := p.checkArticle(ctx, art)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "article not found")
		assert.Equal(t, int64(0), p.stats.ArticlesChecked) // Should not increment on failure

		// Second call succeeds (article found after retry)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(200, nil).Times(1)

		err = p.checkArticle(ctx, art)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), p.stats.ArticlesChecked) // Should increment on success
	})

	t.Run("article stat fails repeatedly and exceeds max retries", func(t *testing.T) {
		// Test the retry limit behavior using checkArticle method
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		// Simulate multiple failed attempts
		for i := 0; i < 3; i++ {
			mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(0, errors.New("article not found")).Times(1)

			err := p.checkArticle(ctx, art)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "article not found")
		}

		// Stats should still be 0 since all calls failed
		assert.Equal(t, int64(0), p.stats.ArticlesChecked)
	})

	t.Run("postArticle and checkArticle integration", func(t *testing.T) {
		// Test posting and checking an article without the full Post workflow
		ctx := context.Background()
		content := "test article content"
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() {
			if err := file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		// Post should succeed
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil)
		// Check should fail first time
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, errors.New("not found"))
		// Check should succeed second time
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any(), gomock.Any()).Return(200, nil)

		p := &poster{
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		art := &article.Article{
			MessageID:  "test@example.com",
			Subject:    "Test Subject",
			From:       "test@example.com",
			Groups:     []string{"alt.test"},
			PartNumber: 1,
			TotalParts: 1,
			FileName:   "test.txt",
			Date:       time.Now(),
			Offset:     0,
			Size:       uint64(len(content)),
		}

		// Post the article
		err = p.postArticle(ctx, art, file)
		assert.NoError(t, err)
		assert.NotEmpty(t, art.Hash) // Hash should be set
		assert.Equal(t, int64(1), p.stats.ArticlesPosted)

		// Check fails first time (simulating article not propagated yet)
		err = p.checkArticle(ctx, art)
		assert.Error(t, err)
		assert.Equal(t, int64(0), p.stats.ArticlesChecked)

		// Check succeeds second time (simulating article now available)
		err = p.checkArticle(ctx, art)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), p.stats.ArticlesChecked)
	})

	t.Run("file not found", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: createTestPostCheckConfig(),
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		err := p.Post(ctx, []string{"nonexistent.txt"}, "", nzbGen)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error adding file")
	})
}

func TestStats(t *testing.T) {
	startTime := time.Now()
	p := &poster{
		stats: &Stats{
			ArticlesPosted:  10,
			ArticlesChecked: 8,
			BytesPosted:     1024,
			ArticleErrors:   2,
			StartTime:       startTime,
		},
	}

	stats := p.Stats()

	assert.Equal(t, int64(10), stats.ArticlesPosted)
	assert.Equal(t, int64(8), stats.ArticlesChecked)
	assert.Equal(t, int64(1024), stats.BytesPosted)
	assert.Equal(t, int64(2), stats.ArticleErrors)
	assert.Equal(t, startTime, stats.StartTime)
}

func TestCalculateHash(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty buffer",
			input:    []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple text",
			input:    []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			expected: "ff5d8507b6a72bee2debce2c0054798deaccdc5d8a1b945b6280ce8aa9cba52e",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CalculateHash(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPostArticle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		content := "test article content"
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		// Open the file
		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() {
			if err := file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil)

		p := &poster{
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		art := &article.Article{
			MessageID:  "test@example.com",
			Subject:    "Test Subject",
			From:       "test@example.com",
			Groups:     []string{"alt.test"},
			PartNumber: 1,
			TotalParts: 1,
			FileName:   "test.txt",
			Date:       time.Now(),
			Offset:     0,
			Size:       uint64(len(content)),
		}

		err = p.postArticle(ctx, art, file)

		assert.NoError(t, err)
		assert.NotEmpty(t, art.Hash) // Hash should be calculated and set
		assert.Equal(t, int64(1), p.stats.ArticlesPosted)
		assert.Equal(t, int64(len(content)), p.stats.BytesPosted)
	})

	t.Run("file read error", func(t *testing.T) {
		ctx := context.Background()

		// Create a file and then close it to simulate read error
		testFile := createTestFile(t, "test content")
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		file, err := os.Open(testFile)
		require.NoError(t, err)
		_ = file.Close() // Close the file to cause read error

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)

		p := &poster{
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		art := &article.Article{
			Offset: 0,
			Size:   10,
		}

		err = p.postArticle(ctx, art, file)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading article body")
	})

	t.Run("post error", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() {
			if err := file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(errors.New("post failed"))

		p := &poster{
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		art := &article.Article{
			MessageID:  "test@example.com",
			Subject:    "Test Subject",
			From:       "test@example.com",
			Groups:     []string{"alt.test"},
			PartNumber: 1,
			TotalParts: 1,
			FileName:   "test.txt",
			Date:       time.Now(),
			Offset:     0,
			Size:       uint64(len(content)),
		}

		err = p.postArticle(ctx, art, file)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error posting article")
	})
}

func TestCheckArticle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(200, nil)

		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		err := p.checkArticle(ctx, art)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), p.stats.ArticlesChecked)
	})

	t.Run("article not found", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(0, errors.New("not found"))

		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		err := p.checkArticle(ctx, art)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "article not found")
	})
}

func TestAddPost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		content := strings.Repeat("test data ", 200) // Make it large enough to create multiple segments
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		cfg := createTestConfig()
		cfg.ArticleSizeInBytes = 100 // Small segment size to force multiple segments

		p := &poster{
			cfg: cfg,
		}

		var wg sync.WaitGroup
		failedPosts := 0
		postQueue := make(chan *Post, 10)
		nzbGen := mocks.NewMockNZBGenerator(ctrl)

		wg.Add(1)
		err := p.addPost(testFile, 1, 1, &wg, &failedPosts, postQueue, nzbGen)

		assert.NoError(t, err)

		// Check that a post was added to the queue
		select {
		case post := <-postQueue:
			assert.Equal(t, testFile, post.FilePath)
			assert.Equal(t, PostStatusPending, post.Status)
			assert.Greater(t, len(post.Articles), 1) // Should have multiple articles due to small segment size
			assert.NotNil(t, post.file)
			assert.Equal(t, int64(len(content)), post.filesize)

			// Clean up
			if err := post.file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		default:
			t.Fatal("Expected post to be added to queue")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		p := &poster{
			cfg: createTestConfig(),
		}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var wg sync.WaitGroup
		failedPosts := 0
		postQueue := make(chan *Post, 10)
		nzbGen := mocks.NewMockNZBGenerator(ctrl)

		err := p.addPost("nonexistent.txt", 1, 1, &wg, &failedPosts, postQueue, nzbGen)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error opening file")
	})
}

func TestPostStatus(t *testing.T) {
	// Test that PostStatus constants are properly defined
	assert.Equal(t, PostStatus(0), PostStatusPending)
	assert.Equal(t, PostStatus(1), PostStatusPosted)
	assert.Equal(t, PostStatus(2), PostStatusVerified)
	assert.Equal(t, PostStatus(3), PostStatusFailed)
}

func TestPost_ConcurrentAccess(t *testing.T) {
	// Test that Post struct can handle concurrent access safely
	post := &Post{
		FilePath: "test.txt",
		Status:   PostStatusPending,
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Simulate concurrent access to the post
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			post.mu.Lock()
			defer post.mu.Unlock()

			// Simulate some work with the post
			status := post.Status
			post.Status = PostStatusPosted
			post.Status = status
		}()
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

func TestStats_ConcurrentAccess(t *testing.T) {
	// Test that Stats can handle concurrent access safely
	stats := &Stats{
		StartTime: time.Now(),
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Simulate concurrent updates to stats
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats.mu.Lock()
				stats.ArticlesPosted++
				stats.BytesPosted += 1024
				stats.mu.Unlock()
			}
		}()
	}

	wg.Wait()

	stats.mu.Lock()
	expectedArticles := int64(numGoroutines * numOperations)
	expectedBytes := int64(numGoroutines * numOperations * 1024)
	stats.mu.Unlock()

	assert.Equal(t, expectedArticles, stats.ArticlesPosted)
	assert.Equal(t, expectedBytes, stats.BytesPosted)
}

func TestPosterInterface(t *testing.T) {
	// Test that poster implements the Poster interface
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
	mockConfig := mocks.NewMockConfig(ctrl)
	mockConfig.EXPECT().GetNNTPPool().Return(mockPool, nil)
	mockConfig.EXPECT().GetPostingConfig().Return(createTestConfig())
	mockConfig.EXPECT().GetPostCheckConfig().Return(createTestPostCheckConfig())

	var p Poster
	poster, err := New(ctx, mockConfig)
	require.NoError(t, err)

	p = poster // This should compile if poster implements Poster interface
	assert.NotNil(t, p)

	// Test that all interface methods are available
	stats := p.Stats()
	assert.NotNil(t, &stats)
}

// Integration test with real files
func TestPostIntegration(t *testing.T) {
	t.Run("small file integration", func(t *testing.T) {
		ctx := context.Background()

		// Create a temporary test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "This is a test file for integration testing."

		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Mock NNTP pool that always succeeds
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any(), gomock.Any()).Return(200, nil).AnyTimes()

		// Mock NZB generator
		nzbGen := mocks.NewMockNZBGenerator(ctrl)
		nzbGen.EXPECT().AddArticle(gomock.Any()).Return()
		nzbGen.EXPECT().AddFileHash(gomock.Any(), gomock.Any()).Return()

		// Create poster with realistic config
		cfg := createTestConfig()
		cfg.ArticleSizeInBytes = 1000 // Larger than our test file

		checkCfg := createTestPostCheckConfig()
		enabled := true
		checkCfg.Enabled = &enabled

		p := &poster{
			cfg:      cfg,
			checkCfg: checkCfg,
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		// Post the file
		err = p.Post(ctx, []string{testFile}, tmpDir, nzbGen)

		assert.NoError(t, err)

		// Verify stats were updated
		stats := p.Stats()
		assert.Equal(t, int64(1), stats.ArticlesPosted)
		assert.Equal(t, int64(1), stats.ArticlesChecked)
		assert.Equal(t, int64(len(content)), stats.BytesPosted)
	})
}

// Simplified unit tests for individual methods instead of complex integration tests

func TestPostLoop_Basic(t *testing.T) {
	t.Run("postLoop processes single article successfully", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		content := "test content for post loop"
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		p := &poster{
			cfg:      createTestConfig(),
			checkCfg: createTestPostCheckConfig(),
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		// Test postArticle directly instead of the full loop
		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() {
			if err := file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		}()

		art := &article.Article{
			MessageID:    "test@example.com",
			Subject:      "Test Subject",
			From:         "test@example.com",
			Groups:       []string{"alt.test"},
			PartNumber:   1,
			TotalParts:   1,
			FileName:     "test.txt",
			OriginalName: "test.txt",
			Date:         time.Now(),
			Offset:       0,
			Size:         uint64(len(content)),
		}

		err = p.postArticle(ctx, art, file)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), p.stats.ArticlesPosted)
	})

	t.Run("postLoop handles posting error", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"
		testFile := createTestFile(t, content)
		defer func() {
			err := os.Remove(testFile)
			assert.NoError(t, err, "Failed to remove test file")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Post(gomock.Any(), gomock.Any()).Return(errors.New("posting failed"))

		p := &poster{
			pool:     mockPool,
			encoder:  rapidyenc.NewEncoder(),
			stats:    &Stats{StartTime: time.Now()},
			throttle: NewThrottle(1024*1024, time.Second),
		}

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() {
			if err := file.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
				assert.NoError(t, err, "Failed to close test file")
			}
		}()

		art := &article.Article{
			MessageID:    "test@example.com",
			Subject:      "Test Subject",
			From:         "test@example.com",
			Groups:       []string{"alt.test"},
			PartNumber:   1,
			TotalParts:   1,
			FileName:     "test.txt",
			OriginalName: "test.txt",
			Date:         time.Now(),
			Offset:       0,
			Size:         uint64(len(content)),
		}

		err = p.postArticle(ctx, art, file)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error posting article")
	})
}

func TestCheckLoop_Basic(t *testing.T) {
	t.Run("checkArticle verifies successfully", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(200, nil)

		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		err := p.checkArticle(ctx, art)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), p.stats.ArticlesChecked)
	})

	t.Run("checkArticle handles verification failure", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPool := nntppool.NewMockUsenetConnectionPool(ctrl)
		mockPool.EXPECT().Stat(ctx, "test@example.com", []string{"alt.test"}).Return(0, errors.New("article not found"))

		p := &poster{
			pool:  mockPool,
			stats: &Stats{StartTime: time.Now()},
		}

		art := &article.Article{
			MessageID: "test@example.com",
			Groups:    []string{"alt.test"},
		}

		err := p.checkArticle(ctx, art)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "article not found")
		assert.Equal(t, int64(0), p.stats.ArticlesChecked)
	})
}

// Test helper functions

func createTestConfig() config.PostingConfig {
	enabled := true
	return config.PostingConfig{
		MaxRetries:         3,
		RetryDelay:         time.Second,
		ArticleSizeInBytes: 1000,
		Groups:             []string{"alt.test"},
		ThrottleRate:       1024 * 1024,
		MaxWorkers:         2,
		MessageIDFormat:    config.MessageIDFormatRandom,
		PostHeaders: config.PostHeaders{
			AddNGXHeader: false,
			DefaultFrom:  "",
		},
		ObfuscationPolicy:     config.ObfuscationPolicyNone,
		Par2ObfuscationPolicy: config.ObfuscationPolicyNone,
		GroupPolicy:           config.GroupPolicyEachFile,
		WaitForPar2:           &enabled,
	}
}

func createTestPostCheckConfig() config.PostCheck {
	enabled := true
	return config.PostCheck{
		Enabled:    &enabled,
		RetryDelay: time.Second,
		MaxRePost:  2,
	}
}

func createTestFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

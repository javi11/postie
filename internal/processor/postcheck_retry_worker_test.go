package processor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/mocks"
	"github.com/javi11/postie/internal/queue"
	"go.uber.org/mock/gomock"
)

// fakeQueue is a hand-rolled implementation of postCheckQueue for tests.
type fakeQueue struct {
	articles    []queue.PendingArticleCheck
	verified    []int64
	failed      []int64
	retried     []int64
	getErr      error
	verifyErr   error
	failErr     error
	retryErr    error
	countTotal  int
	countPend   int
	countFailed int
	countErr    error
	statusErr   error
}

func (f *fakeQueue) GetArticlesForCheck(_ context.Context, limit int) ([]queue.PendingArticleCheck, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(f.articles) <= limit {
		return f.articles, nil
	}
	return f.articles[:limit], nil
}

func (f *fakeQueue) MarkArticleVerified(_ context.Context, id int64) error {
	f.verified = append(f.verified, id)
	return f.verifyErr
}

func (f *fakeQueue) MarkArticleCheckFailed(_ context.Context, id int64) error {
	f.failed = append(f.failed, id)
	return f.failErr
}

func (f *fakeQueue) UpdateArticleCheckRetry(_ context.Context, id int64, _ int, _ time.Time) error {
	f.retried = append(f.retried, id)
	return f.retryErr
}

func (f *fakeQueue) GetPendingCheckCountForItem(_ context.Context, _ string) (total int, pending int, failed int, err error) {
	return f.countTotal, f.countPend, f.countFailed, f.countErr
}

func (f *fakeQueue) UpdateCompletedItemVerificationStatus(_ context.Context, _ string, _ string) error {
	return f.statusErr
}

// makeEnabled returns a pointer to a bool (helper for config.PostCheck.Enabled).
func makeEnabled(b bool) *bool { return &b }

// makeArticles builds n dummy PendingArticleCheck entries.
func makeArticles(n int, retryCount int) []queue.PendingArticleCheck {
	articles := make([]queue.PendingArticleCheck, n)
	for i := range articles {
		articles[i] = queue.PendingArticleCheck{
			ID:              int64(i + 1),
			CompletedItemID: fmt.Sprintf("item-%d", i+1),
			MessageID:       fmt.Sprintf("<msg-%d@test>", i+1),
			Groups:          `["alt.binaries.test"]`,
			Status:          "pending",
			RetryCount:      retryCount,
		}
	}
	return articles
}

// newWorker builds a PostCheckRetryWorker with sensible test defaults.
func newWorker(ctx context.Context, q postCheckQueue, pool *mocks.MockNNTPClient, batchSize int, maxRetries int) *PostCheckRetryWorker {
	enabled := makeEnabled(true)
	cfg := config.PostCheck{
		Enabled:               enabled,
		DeferredCheckInterval: config.Duration("1m"),
		DeferredCheckDelay:    config.Duration("5m"),
		DeferredMaxBackoff:    config.Duration("1h"),
		DeferredMaxRetries:    maxRetries,
		DeferredBatchSize:     batchSize,
	}
	w := NewPostCheckRetryWorker(ctx, q, pool, cfg)
	return w
}

func TestProcessRetries(t *testing.T) {
	t.Run("empty queue returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		q := &fakeQueue{articles: []queue.PendingArticleCheck{}}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		w := newWorker(context.Background(), q, mockPool, 3, 3)

		got := w.processRetries()
		if got {
			t.Error("expected false for empty queue, got true")
		}
	})

	t.Run("partial batch returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		articles := makeArticles(2, 0)
		q := &fakeQueue{articles: articles, countPend: 1}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		w := newWorker(context.Background(), q, mockPool, 3, 5)

		got := w.processRetries()
		if got {
			t.Error("expected false for partial batch (2 < batchSize 3), got true")
		}
	})

	t.Run("full batch returns true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		articles := makeArticles(2, 0)
		q := &fakeQueue{articles: articles, countTotal: 2, countPend: 0}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
		w := newWorker(context.Background(), q, mockPool, 2, 5)

		got := w.processRetries()
		if !got {
			t.Error("expected true for full batch (2 == batchSize 2), got false")
		}
	})

	t.Run("verified articles marked verified", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		articles := makeArticles(1, 0)
		q := &fakeQueue{articles: articles, countTotal: 1, countPend: 0}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		mockPool.EXPECT().Stat(gomock.Any(), articles[0].MessageID).Return(nil, nil).Times(1)
		w := newWorker(context.Background(), q, mockPool, 10, 5)

		w.processRetries()

		if len(q.verified) != 1 || q.verified[0] != articles[0].ID {
			t.Errorf("expected article %d to be marked verified, got verified=%v", articles[0].ID, q.verified)
		}
		if len(q.failed) != 0 {
			t.Errorf("expected no failed marks, got %v", q.failed)
		}
	})

	t.Run("failed STAT below maxRetries schedules retry", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		articles := makeArticles(1, 0) // retryCount=0, maxRetries=3 → newRetryCount=1 < 3
		q := &fakeQueue{articles: articles, countTotal: 1, countPend: 1}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		mockPool.EXPECT().Stat(gomock.Any(), articles[0].MessageID).Return(nil, errors.New("not found")).Times(1)
		w := newWorker(context.Background(), q, mockPool, 10, 3)

		w.processRetries()

		if len(q.retried) != 1 || q.retried[0] != articles[0].ID {
			t.Errorf("expected article %d to be scheduled for retry, got retried=%v", articles[0].ID, q.retried)
		}
		if len(q.failed) != 0 {
			t.Errorf("expected no failed marks, got %v", q.failed)
		}
	})

	t.Run("failed STAT at maxRetries marks failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// retryCount=2, maxRetries=3 → newRetryCount=3 >= 3 → mark failed
		articles := makeArticles(1, 2)
		q := &fakeQueue{articles: articles, countTotal: 1, countPend: 0, countFailed: 1}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		mockPool.EXPECT().Stat(gomock.Any(), articles[0].MessageID).Return(nil, errors.New("not found")).Times(1)
		w := newWorker(context.Background(), q, mockPool, 10, 3)

		w.processRetries()

		if len(q.failed) != 1 || q.failed[0] != articles[0].ID {
			t.Errorf("expected article %d to be marked failed, got failed=%v", articles[0].ID, q.failed)
		}
		if len(q.retried) != 0 {
			t.Errorf("expected no retry schedules, got %v", q.retried)
		}
	})

	t.Run("context cancel mid-batch returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		articles := makeArticles(2, 0)
		q := &fakeQueue{articles: articles}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		// With cancelled ctx, the worker should detect ctx.Err() before processing articles
		// Stat may or may not be called depending on timing, so allow any calls
		mockPool.EXPECT().Stat(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		w := newWorker(ctx, q, mockPool, 2, 5)

		got := w.processRetries()
		if got {
			t.Error("expected false when context cancelled, got true")
		}
	})

	t.Run("queue error returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		q := &fakeQueue{getErr: errors.New("db error")}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		w := newWorker(context.Background(), q, mockPool, 10, 5)

		got := w.processRetries()
		if got {
			t.Error("expected false on queue error, got true")
		}
	})

	t.Run("bad groups JSON marks failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		articles := []queue.PendingArticleCheck{
			{
				ID:              99,
				CompletedItemID: "item-bad",
				MessageID:       "<bad@test>",
				Groups:          `not-valid-json`,
				Status:          "pending",
				RetryCount:      0,
			},
		}
		q := &fakeQueue{articles: articles, countTotal: 1, countPend: 0, countFailed: 1}
		mockPool := mocks.NewMockNNTPClient(ctrl)
		// Stat should NOT be called since JSON parsing fails first
		w := newWorker(context.Background(), q, mockPool, 10, 5)

		w.processRetries()

		if len(q.failed) != 1 || q.failed[0] != 99 {
			t.Errorf("expected article 99 to be marked failed due to bad JSON, got failed=%v", q.failed)
		}
		if len(q.verified) != 0 {
			t.Errorf("expected no verified marks, got %v", q.verified)
		}
	})
}

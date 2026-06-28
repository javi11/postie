package poster

import (
	"context"
	"errors"
	"testing"

	"github.com/javi11/nntppool/v4"
	"go.uber.org/mock/gomock"

	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/mocks"
)

func TestFilterMissing_KeepsOnlyMissingPreservingOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := mocks.NewMockNNTPClient(ctrl)
	// m0 and m2 already present; m1 missing.
	mockPool.EXPECT().Stat(gomock.Any(), "m0").Return(&nntppool.StatResult{}, nil).AnyTimes()
	mockPool.EXPECT().Stat(gomock.Any(), "m1").Return(nil, errors.New("430 no such article")).AnyTimes()
	mockPool.EXPECT().Stat(gomock.Any(), "m2").Return(&nntppool.StatResult{}, nil).AnyTimes()

	p := &poster{verifyPool: mockPool}
	got := p.filterMissing(context.Background(), []*article.Article{
		{MessageID: "m0"}, {MessageID: "m1"}, {MessageID: "m2"},
	})

	if len(got) != 1 || got[0].MessageID != "m1" {
		t.Errorf("filterMissing = %v, want only m1", msgIDs(got))
	}
}

func TestFilterMissing_AllPresentReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPool := mocks.NewMockNNTPClient(ctrl)
	mockPool.EXPECT().Stat(gomock.Any(), gomock.Any()).Return(&nntppool.StatResult{}, nil).AnyTimes()

	p := &poster{verifyPool: mockPool}
	got := p.filterMissing(context.Background(), []*article.Article{{MessageID: "a"}, {MessageID: "b"}})
	if len(got) != 0 {
		t.Errorf("all present should yield no re-posts, got %v", msgIDs(got))
	}
}

func TestFilterMissing_NoVerifyPoolRepostsAll(t *testing.T) {
	p := &poster{verifyPool: nil}
	in := []*article.Article{{MessageID: "a"}, {MessageID: "b"}}
	got := p.filterMissing(context.Background(), in)
	if len(got) != 2 {
		t.Errorf("with no verify pool all articles should be re-posted, got %d", len(got))
	}
}

func msgIDs(arts []*article.Article) []string {
	ids := make([]string, len(arts))
	for i, a := range arts {
		ids[i] = a.MessageID
	}
	return ids
}

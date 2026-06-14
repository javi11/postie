package postie

import (
	"context"
	"testing"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewRuntime_Par2SchedulerCapacityFromConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := mocks.NewMockConfig(ctrl)
	cfg.EXPECT().GetPar2Config(gomock.Any()).Return(&config.Par2Config{MaxConcurrentJobs: 3}, nil)

	rt, err := NewRuntime(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer func() { _ = rt.Close() }()

	if got := rt.Par2Scheduler().Capacity(); got != 3 {
		t.Errorf("Par2Scheduler capacity = %d, want 3", got)
	}
}

func TestNewRuntime_DefaultsToOneWhenUnset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := mocks.NewMockConfig(ctrl)
	cfg.EXPECT().GetPar2Config(gomock.Any()).Return(&config.Par2Config{MaxConcurrentJobs: 0}, nil)

	rt, err := NewRuntime(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer func() { _ = rt.Close() }()

	if got := rt.Par2Scheduler().Capacity(); got != 1 {
		t.Errorf("Par2Scheduler capacity = %d, want 1 (default)", got)
	}
}

func TestRuntime_NilSafe(t *testing.T) {
	var rt *Runtime
	if rt.Par2Scheduler() != nil {
		t.Error("nil Runtime.Par2Scheduler() should return nil")
	}
	if err := rt.Close(); err != nil {
		t.Errorf("nil Runtime.Close() = %v, want nil", err)
	}
}

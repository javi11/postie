package postie

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/database"
	"github.com/javi11/postie/internal/mocks"
	"github.com/javi11/postie/internal/transferstore"
	"go.uber.org/mock/gomock"
)

func TestNewRuntime_Par2SchedulerCapacityFromConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := mocks.NewMockConfig(ctrl)
	cfg.EXPECT().GetPar2Config(gomock.Any()).Return(&config.Par2Config{MaxConcurrentJobs: 3}, nil)

	rt, err := NewRuntime(context.Background(), cfg, nil, nil, "")
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

	rt, err := NewRuntime(context.Background(), cfg, nil, nil, "")
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
	if rt.UploadEngine() != nil {
		t.Error("nil Runtime.UploadEngine() should return nil")
	}
	if (rt.Metrics() != RuntimeMetrics{}) {
		t.Errorf("nil Runtime.Metrics() = %+v, want zero value", rt.Metrics())
	}
	if err := rt.Close(); err != nil {
		t.Errorf("nil Runtime.Close() = %v, want nil", err)
	}
}

func TestRuntime_MetricsReflectsPar2Capacity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := mocks.NewMockConfig(ctrl)
	cfg.EXPECT().GetPar2Config(gomock.Any()).Return(&config.Par2Config{MaxConcurrentJobs: 4}, nil)

	// nil poolManager => no upload engine; PAR2 scheduler still present.
	rt, err := NewRuntime(context.Background(), cfg, nil, nil, "")
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer func() { _ = rt.Close() }()

	m := rt.Metrics()
	if m.Par2Capacity != 4 {
		t.Errorf("Par2Capacity = %d, want 4", m.Par2Capacity)
	}
	if m.UploadWorkerCount != 0 {
		t.Errorf("UploadWorkerCount = %d, want 0 (no engine)", m.UploadWorkerCount)
	}
}

func newTestTransferStore(t *testing.T) *transferstore.Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.New(ctx, config.DatabaseConfig{
		DatabaseType: "sqlite",
		DatabasePath: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.GetMigrationRunner().MigrateUp(); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	return transferstore.New(db.DB)
}

func TestRuntime_NewManifestRecorder(t *testing.T) {
	store := newTestTransferStore(t)
	rt := &Runtime{store: store, manifestDir: t.TempDir()}

	if rt.NewManifestRecorder("tid-1") == nil {
		t.Error("NewManifestRecorder with a store should return a recorder")
	}
	if rt.NewManifestRecorder("") != nil {
		t.Error("NewManifestRecorder with empty transferID should return nil")
	}

	var none *Runtime
	if none.NewManifestRecorder("tid-1") != nil {
		t.Error("nil Runtime should return nil recorder")
	}
	if (&Runtime{}).NewManifestRecorder("tid-1") != nil {
		t.Error("Runtime without a store should return nil recorder")
	}
}

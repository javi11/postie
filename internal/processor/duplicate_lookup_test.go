package processor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kipsilabs/postie/internal/config"
	"github.com/kipsilabs/postie/internal/database"
	"github.com/kipsilabs/postie/internal/pool"
	"github.com/kipsilabs/postie/internal/queue"
)

// Exercise the real queued durable path, where enabling automatic source
// deletion previously made successful jobs eligible for later cleanup.
func TestDuplicateLookupQueueRetainsSourceAndCompletesHooks(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "movie.mkv")
	if err := os.WriteFile(source, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	lookup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-nzb")
		w.Write([]byte(`<nzb><file subject="movie.mkv"><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">existing@test</segment></segments></file></nzb>`))
	}))
	defer lookup.Close()
	cfg := config.GetDefaultConfig()
	cfg.Database = config.DatabaseConfig{DatabaseType: "sqlite", DatabasePath: filepath.Join(root, "test.db")}
	cfg.Servers = []config.ServerConfig{{Host: "127.0.0.1", Port: 1, MaxConnections: 1, Role: config.ServerRoleUpload}}
	cfg.DuplicateLookup = config.DuplicateLookupConfig{Enabled: true, URL: lookup.URL, Timeout: "1s"}
	cfg.PostUploadScript = config.PostUploadScriptConfig{Enabled: true, Command: `touch "{source_path}.hook"`, Timeout: "1s"}
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.GetMigrationRunner().MigrateUp(); err != nil {
		t.Fatal(err)
	}
	q, err := queue.New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	if err := q.AddFile(ctx, source, 3); err != nil {
		t.Fatal(err)
	}
	pm, err := pool.New(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	p := New(ProcessorOptions{Queue: q, Config: &cfg, QueueConfig: cfg.Queue, PoolManager: pm, OutputFolder: filepath.Join(root, "output"), DeleteOriginalFile: true})
	defer p.Close()
	if !p.durableMode() {
		t.Fatal("expected durable mode")
	}
	if err := p.processNextItem(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("reused source was deleted", err)
	}
	if _, err := os.Stat(source + ".hook"); err != nil {
		t.Fatal("post-upload hook did not run", err)
	}
	result, err := q.GetQueueItems(queue.PaginationParams{Page: 1, Limit: 10, Status: "complete"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].VerificationStatus == nil || *result.Items[0].VerificationStatus != "reused" {
		t.Fatalf("completion=%+v", result)
	}
	var count int
	if err := q.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM transfer_files").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("fabricated transfer manifests: %d", count)
	}
}

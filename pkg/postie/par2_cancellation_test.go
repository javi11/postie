package postie

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kipsilabs/postie/internal/nzb"
	"github.com/kipsilabs/postie/internal/par2"
	"github.com/kipsilabs/postie/internal/pool"
	"github.com/kipsilabs/postie/internal/poster"
	"github.com/kipsilabs/postie/internal/transferstore"
	"github.com/kipsilabs/postie/pkg/fileinfo"
)

// Complete the main upload before cancelling the still-running PAR2 branch.
// This is the window where a cancellation used to become a successful upload.
type cancelAfterMainPoster struct {
	poster.Poster
	cancel       context.CancelFunc
	mainDone     chan struct{}
	duringUpload bool
}

func (p *cancelAfterMainPoster) Post(ctx context.Context, files []string, root string, gen nzb.NZBGenerator) error {
	if len(files) > 0 && par2.IsPar2File(files[0]) {
		<-p.mainDone
		p.cancel()
		return ctx.Err()
	}
	err := p.Poster.Post(ctx, files, root, gen)
	close(p.mainDone)
	if !p.duringUpload {
		p.cancel()
	}
	return err
}

func (p *cancelAfterMainPoster) PostWithRelativePaths(ctx context.Context, files []string, root string, gen nzb.NZBGenerator, _ map[string]string) error {
	return p.Post(ctx, files, root, gen)
}

type cancelPar2Executor struct {
	mockPar2Executor
	duringUpload bool
	paths        []string
}

func (p *cancelPar2Executor) CreateInDirectory(ctx context.Context, _ []fileinfo.FileInfo, _ string) (par2.Result, error) {
	if p.duringUpload {
		return par2.Result{Created: p.paths}, nil
	}
	<-ctx.Done()
	return par2.Result{}, ctx.Err()
}

func (p *cancelPar2Executor) CreateSet(ctx context.Context, files []fileinfo.FileInfo, out, _, _ string) (par2.Result, error) {
	return p.CreateInDirectory(ctx, files, out)
}

func TestParallelPar2CancellationDoesNotCompleteUpload(t *testing.T) {
	for _, folder := range []bool{false, true} {
		for _, duringUpload := range []bool{false, true} {
			t.Run(fmt.Sprintf("folder=%t/duringUpload=%t", folder, duringUpload), func(t *testing.T) {
				root := t.TempDir()
				files, _ := makeSourceFiles(t, root, "Release", "payload.rar")
				out := t.TempDir()
				parPath := filepath.Join(t.TempDir(), "Release.par2")
				if err := os.WriteFile(parPath, []byte("recovery source"), 0600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				p := newTestPostie(nil, false, false)
				p.par2runner = &cancelPar2Executor{duringUpload: duringUpload, paths: []string{parPath}}
				p.poster = &cancelAfterMainPoster{Poster: &mockPoster{}, cancel: cancel, mainDone: make(chan struct{}), duringUpload: duringUpload}
				path, err := p.Post(ctx, files, root, out, folder)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Post returned path=%q err=%v, want cancellation", path, err)
				}
				if err := filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !entry.IsDir() && filepath.Ext(path) == ".nzb" {
						t.Errorf("cancelled upload published NZB %s", path)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(parPath); err != nil {
					t.Errorf("recovery source removed after cancellation: %v", err)
				}
			})
		}
	}
}

// Exercise real NNTP posting, manifests and retry after cancellation. The PAR2
// executor waits for cancellation so the main upload has definitely finished.
func TestIntegration_ParallelPar2CancellationCanRetry(t *testing.T) {
	for _, folder := range []bool{false, true} {
		t.Run(fmt.Sprintf("folder=%t", folder), func(t *testing.T) {
			root := t.TempDir()
			files, _ := makeSourceFiles(t, root, "Release", "payload.rar")
			files[0] = writeBenchFile(t, filepath.Join(root, "Release"), "payload.rar", 1)
			files[0].RelativePath = "Release/payload.rar"
			out := t.TempDir()
			srv := startSinkNNTPServer(t, false)
			cfg := benchConfig(srv.port(), 2, true, false, t.TempDir())
			cfg.Servers[0].SSL = false
			cfg.Servers[0].InsecureSSL = false
			cfg.PostCheck.Enabled = boolPtr(true)
			pm, err := pool.New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pm.Close() }()
			store := newTestTransferStore(t)
			rt, err := NewRuntime(context.Background(), cfg, pm, store, t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rt.Close() }()
			// The poster has its own live context, just as the processor does; the
			// cancellable per-attempt context is passed to Post.
			p, err := NewWithRuntime(context.Background(), cfg, pm, noopProgress{}, nil, rt, "cancel-retry")
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			actualPoster := p.poster
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p.poster = &cancelAfterMainPoster{Poster: actualPoster, cancel: cancel, mainDone: make(chan struct{})}
			p.par2runner = &cancelPar2Executor{}
			if path, err := p.Post(ctx, files, root, out, folder); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled Post = %q, %v; want context.Canceled", path, err)
			}
			if srv.articles.Load() == 0 {
				t.Fatal("main upload never reached NNTP server")
			}
			rows, err := store.ListFilesByTransfer(context.Background(), "cancel-retry")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0].UploadState != transferstore.StatePlanned {
				t.Fatalf("cancelled transfer must remain planned for retry: %+v", rows)
			}
			p.Close()
			retry, err := NewWithRuntime(context.Background(), cfg, pm, noopProgress{}, nil, rt, "cancel-retry")
			if err != nil {
				t.Fatal(err)
			}
			defer retry.Close()
			path, err := retry.Post(context.Background(), files, root, out, folder)
			if err != nil {
				t.Fatalf("retry: %v", err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("retry NZB: %v", err)
			}
			rows, err = store.ListFilesByTransfer(context.Background(), "cancel-retry")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) < 2 {
				t.Fatalf("retry did not record PAR2 files: %+v", rows)
			}
			for _, row := range rows {
				if row.UploadState != transferstore.StateUploaded {
					t.Errorf("retry left %s in %s", row.SourcePath, row.UploadState)
				}
			}
		})
	}
}

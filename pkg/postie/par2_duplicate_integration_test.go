package postie

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/javi11/nzbparser"
	"github.com/kipsilabs/postie/internal/pool"
	"github.com/kipsilabs/postie/pkg/fileinfo"
)

// A preserved set can be part of the next source listing and also returned by
// the PAR2 executor. It must be posted once, with its source relative path.
func TestIntegration_PostFolderReusedSourcePar2PostedOnce(t *testing.T) {
	for _, wait := range []bool{true, false} {
		for _, skip := range []bool{true, false} {
			t.Run(fmt.Sprintf("wait=%t/skip=%t", wait, skip), func(t *testing.T) {
				ctx := context.Background()
				root := t.TempDir()
				folder := filepath.Join(root, "Release")
				if err := os.MkdirAll(folder, 0755); err != nil {
					t.Fatal(err)
				}
				file := writeBenchFile(t, folder, "payload.rar", 1)
				file.RelativePath = "Release/payload.rar"
				srv := startSinkNNTPServer(t, false)
				cfg := benchConfig(srv.port(), 2, true, wait, t.TempDir())
				cfg.Servers[0].SSL = false
				cfg.Servers[0].InsecureSSL = false
				cfg.Par2.MaintainPar2Files = boolPtr(true)
				cfg.Par2.SkipIfPar2Exists = boolPtr(skip)
				pm, err := pool.New(cfg)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = pm.Close() })
				files := []fileinfo.FileInfo{file}
				var wantFiles int
				var wantArticles int64
				for run := range 3 {
					p, err := New(ctx, cfg, pm, noopProgress{}, nil)
					if err != nil {
						t.Fatal(err)
					}
					before := srv.articles.Load()
					nzbPath, err := p.Post(ctx, files, root, root, true)
					p.Close()
					if err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(nzbPath)
					if err != nil {
						t.Fatal(err)
					}
					parsed, err := nzbparser.ParseString(string(data))
					if err != nil {
						t.Fatal(err)
					}
					posted := srv.articles.Load() - before
					if run == 0 {
						wantFiles = len(parsed.Files)
						wantArticles = posted
						if wantFiles < 2 {
							t.Fatal("first upload did not generate PAR2")
						}
						paths, err := filepath.Glob(filepath.Join(folder, "*.par2"))
						if err != nil {
							t.Fatal(err)
						}
						for _, path := range paths {
							info, err := os.Stat(path)
							if err != nil {
								t.Fatal(err)
							}
							files = append(files, fileinfo.FileInfo{Path: path, Size: uint64(info.Size()), RelativePath: "Release/" + filepath.Base(path)})
						}
					} else {
						if got := len(parsed.Files); got != wantFiles {
							t.Errorf("upload %d: NZB has %d files, want %d source files", run+1, got, wantFiles)
						}
						if posted != wantArticles {
							t.Errorf("upload %d: posted %d articles, want %d (each file once)", run+1, posted, wantArticles)
						}
						for _, f := range parsed.Files {
							if strings.HasPrefix(f.Filename, "Release/") {
								continue
							}
							t.Errorf("upload %d: file %q lost its source relative path", run+1, f.Filename)
						}
					}
					if err := os.Remove(nzbPath); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

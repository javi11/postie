package postie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kipsilabs/postie/internal/config"
	"github.com/kipsilabs/postie/pkg/fileinfo"
)

func TestDuplicateLookupBeforeUpload(t *testing.T) {
	for _, status := range []int{200, 204, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "movie.mkv")
			os.WriteFile(source, []byte("abc"), 0600)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/x-nzb")
				w.WriteHeader(status)
				if status == 200 {
					w.Write([]byte(`<nzb><file subject="movie.mkv"><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">reused@test</segment></segments></file></nzb>`))
				}
			}))
			defer s.Close()
			p := newTestPostie(&mockPar2Executor{}, true, false)
			p.duplicateLookupCfg = config.DuplicateLookupConfig{Enabled: true, URL: s.URL, Timeout: "1s"}
			p.SetDeleteOriginal(true)
			// nil poster and PAR2 runner make any upload work on a match panic.
			if status != 204 {
				p.poster = nil
				p.par2runner = nil
			}
			output, err := p.Post(context.Background(), []fileinfo.FileInfo{{Path: source, Size: 3}}, filepath.Dir(source), t.TempDir(), false)
			if status == 500 {
				if err == nil {
					t.Fatal("expected fail-closed error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.ReusedSource(source) != (status == 200) {
				t.Fatal("wrong reuse state")
			}
			if _, err := os.Stat(output); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(source); err != nil {
				t.Fatal("source removed", err)
			}
		})
	}
}

func TestDuplicateLookupSourceMutationStopsBeforePublication(t *testing.T) {
	source := filepath.Join(t.TempDir(), "movie.mkv")
	if err := os.WriteFile(source, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := os.WriteFile(source, []byte("changed"), 0600); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/x-nzb")
		w.Write([]byte(`<nzb><file subject="movie.mkv"><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">reused@test</segment></segments></file></nzb>`))
	}))
	defer server.Close()
	p := newTestPostie(nil, true, false)
	p.poster = nil
	p.duplicateLookupCfg = config.DuplicateLookupConfig{Enabled: true, URL: server.URL, Timeout: "1s"}
	p.SetDeleteOriginal(true)
	out := t.TempDir()
	if path, err := p.Post(context.Background(), []fileinfo.FileInfo{{Path: source, Size: 3}}, filepath.Dir(source), out, false); err == nil || path != "" {
		t.Fatalf("changed source reused: %q %v", path, err)
	}
	entries, err := os.ReadDir(out)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected NZB publication: %v %v", entries, err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("changed source removed", err)
	}
}

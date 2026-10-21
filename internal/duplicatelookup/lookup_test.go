package duplicatelookup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const validNZB = `<?xml version="1.0"?><nzb xmlns="http://www.newzbin.com/DTD/2003/nzb"><file poster="poster" date="1" subject='"movie.mkv" yEnc (1/1)'><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">test@example.com</segment></segments></file></nzb>`

func fixture(t *testing.T, size int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "movie.mkv")
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestFingerprintVectors(t *testing.T) {
	for _, tt := range []struct {
		size int
		hash string
	}{{3, "91faf454819e22599248399163fc7ae5a917c5e573b1fe2de6f7fe05872a6b9b"}, {0, "05a8b6e0a73e1eea7da7d93b345eb2e99c9e71b6c34e0ae42fde07302a5371d9"}, {196608, "20f87644845ab1ca1f301d85f83a1ca9e9ed9d5c6f86e589d02a945a39f8a740"}, {196609, "d2c3b849b00a87e9bb79ad28d2b820c74777911e4eb682d2c5b7b20ac1febbb9"}, {1000000, "6e9548af1d6275ae8aa740380df5e1c709e070124f81f9383832e6fff6109efb"}} {
		t.Run(strconv.Itoa(tt.size), func(t *testing.T) {
			r, err := Fingerprint(context.Background(), fixture(t, tt.size))
			if err != nil {
				t.Fatal(err)
			}
			if r.Hash != tt.hash {
				t.Fatalf("size %d: %s", tt.size, r.Hash)
			}
		})
	}
}
func TestLookupResponses(t *testing.T) {
	for _, tt := range []struct {
		name           string
		status         int
		body           string
		wantErr, match bool
	}{{"match", 200, validNZB, false, true}, {"miss", 204, "", false, false}, {"unauthorized", 401, "", true, false}, {"malformed", 200, "<nzb>", true, false}, {"empty", 200, "<nzb/>", true, false}, {"no segments", 200, "<nzb><file/></nzb>", true, false}, {"blank segment", 200, `<nzb><file><segments><segment bytes="1" number="1"> </segment></segments></file></nzb>`, true, false}} {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("X-API-Key") != "secret" {
					t.Error("request/auth")
				}
				var req Request
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name != "movie.mkv" || req.Size != 3 || req.Algorithm != Algorithm || req.Hash != "91faf454819e22599248399163fc7ae5a917c5e573b1fe2de6f7fe05872a6b9b" {
					t.Error("bad request", req, err)
				}
				w.Header().Set("Content-Type", "application/x-nzb")
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer s.Close()
			data, err := Lookup(context.Background(), s.URL, "secret", time.Second, fixture(t, 3))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v", err)
			}
			if (len(data) > 0) != tt.match {
				t.Fatal("match mismatch")
			}
		})
	}
}
func TestLookupCancellationAndTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(50 * time.Millisecond) }))
	defer s.Close()
	p := fixture(t, 3)
	_, err := Lookup(context.Background(), s.URL, "", 10*time.Millisecond, p)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Lookup(ctx, s.URL, "", time.Second, p)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestLookupRejectsRedirectContentTypeAndNetworkFailure(t *testing.T) {
	path := fixture(t, 3)
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect must not forward credentials") }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer redirect.Close()
	if data, err := Lookup(context.Background(), redirect.URL, "secret", time.Second, path); err == nil || len(data) > 0 {
		t.Fatal("redirect accepted")
	}
	wrongType := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(validNZB))
	}))
	defer wrongType.Close()
	if data, err := Lookup(context.Background(), wrongType.URL, "", time.Second, path); err == nil || len(data) > 0 {
		t.Fatal("wrong media type accepted")
	}
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()
	if data, err := Lookup(context.Background(), closed.URL, "", time.Second, path); err == nil || len(data) > 0 {
		t.Fatal("network failure treated as miss")
	}
}

func TestLookupRejectsSourceChangedDuringRequest(t *testing.T) {
	path := fixture(t, 3)
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/x-nzb")
		w.Write([]byte(validNZB))
	}))
	defer service.Close()
	if data, err := Lookup(context.Background(), service.URL, "", time.Second, path); err == nil || len(data) > 0 {
		t.Fatal("changed source accepted as a duplicate")
	}
}

func TestValidateNZBRequiresSubjectAndGroups(t *testing.T) {
	for _, data := range []string{
		`<nzb><file><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">test@example.com</segment></segments></file></nzb>`,
		`<nzb><file subject="movie"><segments><segment bytes="3" number="1">test@example.com</segment></segments></file></nzb>`,
		`<nzb><file subject="movie"><groups><group> </group></groups><segments><segment bytes="3" number="1">test@example.com</segment></segments></file></nzb>`,
	} {
		if err := ValidateNZB([]byte(data)); err == nil {
			t.Fatal("unusable file metadata accepted", data)
		}
	}
}

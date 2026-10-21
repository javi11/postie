package nzb

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kipsilabs/postie/internal/config"
)

func TestSaveExistingNamingCompressionAndCancellation(t *testing.T) {
	data := []byte(`<nzb><file subject='"movie.mkv" yEnc (1/1)'><groups><group>alt.test</group></groups><segments><segment bytes="3" number="1">existing@test</segment></segments></file></nzb>`)
	for _, compression := range []string{"none", "zstd", "brotli", "zip"} {
		for _, keep := range []bool{false, true} {
			t.Run(compression+map[bool]string{true: "/keep", false: "/strip"}[keep], func(t *testing.T) {
				target := filepath.Join(t.TempDir(), "nested", "movie.mkv")
				cfg := config.NzbCompressionConfig{Enabled: compression != "none", Type: config.CompressionType(compression), Level: 3}
				path, err := SaveExisting(context.Background(), data, target, cfg, keep)
				if err != nil {
					t.Fatal(err)
				}
				suffix := map[string]string{"none": "", "zstd": ".zst", "brotli": ".br", "zip": ".zip"}[compression]
				name := "movie.nzb"
				if keep {
					name = "movie.mkv.nzb"
				}
				if filepath.Base(path) != name+suffix {
					t.Fatalf("path=%s", path)
				}
				stat, err := os.Stat(path)
				if err != nil || stat.Size() == 0 {
					t.Fatalf("empty output: %v", err)
				}
				encoded, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var decoded []byte
				switch compression {
				case "zstd":
					decoder, err := zstd.NewReader(nil)
					if err != nil {
						t.Fatal(err)
					}
					decoded, err = decoder.DecodeAll(encoded, nil)
					decoder.Close()
					if err != nil {
						t.Fatal(err)
					}
				case "brotli":
					decoded, err = io.ReadAll(brotli.NewReader(bytes.NewReader(encoded)))
					if err != nil {
						t.Fatal(err)
					}
				case "zip":
					reader, err := zip.OpenReader(path)
					if err != nil {
						t.Fatal(err)
					}
					if len(reader.File) != 1 || reader.File[0].Name != name {
						t.Fatal("invalid archive entry")
					}
					entry, err := reader.File[0].Open()
					if err != nil {
						t.Fatal(err)
					}
					decoded, err = io.ReadAll(entry)
					entry.Close()
					reader.Close()
					if err != nil {
						t.Fatal(err)
					}
				default:
					decoded = encoded
				}
				if !bytes.Equal(decoded, data) {
					t.Fatal("NZB bytes changed during publication")
				}
				matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".duplicate-nzb-*"))
				if err != nil || len(matches) != 0 {
					t.Fatalf("temporary files leaked: %v %v", matches, err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := SaveExisting(ctx, data, target+"-cancel", cfg, keep); err != context.Canceled {
					t.Fatalf("cancel error=%v", err)
				}
			})
		}
	}
}

package nzb

import (
	"archive/zip"
	"compress/flate"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/andybalholm/brotli"
	"github.com/kipsilabs/postie/internal/config"
	"github.com/kipsilabs/postie/internal/duplicatelookup"
	"github.com/klauspost/compress/zstd"
)

// SaveExisting publishes a validated provider NZB atomically using normal naming
// and compression. It never synthesizes article records or manifests.
func SaveExisting(ctx context.Context, data []byte, target string, cfg config.NzbCompressionConfig, maintainExtension bool) (string, error) {
	if err := duplicatelookup.ValidateNZB(data); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	g := &Generator{maintainOriginalExtension: maintainExtension}
	target = g.generateFinalNzbPath(target)
	uncompressed := target
	if cfg.Enabled {
		switch cfg.Type {
		case config.CompressionTypeZstd:
			target += ".zst"
		case config.CompressionTypeBrotli:
			target += ".br"
		case config.CompressionTypeZip:
			target += ".zip"
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(target), ".duplicate-nzb-*")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	defer f.Close()
	if err := writeExisting(f, data, filepath.Base(uncompressed), cfg); err != nil {
		return "", err
	}
	if err := f.Sync(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, target); err != nil {
		return "", fmt.Errorf("publish reused NZB: %w", err)
	}
	return target, nil
}

func writeExisting(out io.Writer, data []byte, name string, cfg config.NzbCompressionConfig) error {
	if !cfg.Enabled {
		_, err := out.Write(data)
		return err
	}
	var writer io.WriteCloser
	switch cfg.Type {
	case config.CompressionTypeZstd:
		enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.EncoderLevel(cfg.Level)))
		if err != nil {
			return err
		}
		writer = enc
	case config.CompressionTypeBrotli:
		writer = brotli.NewWriterLevel(out, cfg.Level)
	case config.CompressionTypeZip:
		enc := zip.NewWriter(out)
		enc.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) { return flate.NewWriter(w, cfg.Level) })
		entry, err := enc.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			enc.Close()
			return err
		}
		if _, err := entry.Write(data); err != nil {
			enc.Close()
			return err
		}
		return enc.Close()
	default:
		_, err := out.Write(data)
		return err
	}
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

package postie

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/pkg/fileinfo"
	"golang.org/x/sync/errgroup"
)

type Postie struct {
	par2Cfg        *config.Par2Config
	postingCfg     config.PostingConfig
	par2runner     *par2.Par2CmdExecutor
	poster         poster.Poster
	compressionCfg config.NzbCompressionConfig
}

func New(
	ctx context.Context,
	cfg config.Config,
) (*Postie, error) {
	// Ensure par2 executable exists and get its path
	par2Cfg, err := cfg.GetPar2Config(ctx)
	if err != nil {
		return nil, err
	}

	postingConfig := cfg.GetPostingConfig()
	compressionConfig := cfg.GetNzbCompressionConfig()

	// Create par2 runner
	par2runner := par2.New(ctx, postingConfig.ArticleSizeInBytes, par2Cfg)

	// Create poster
	p, err := poster.New(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Error creating poster", "error", err)

		return nil, err
	}

	return &Postie{
		par2Cfg:        par2Cfg,
		par2runner:     par2runner,
		poster:         p,
		postingCfg:     postingConfig,
		compressionCfg: compressionConfig,
	}, nil
}

func (p *Postie) Close() {
	p.poster.Close()
}

// SetProgressCallback sets the progress callback function for both poster and par2runner
func (p *Postie) SetProgressCallback(callback poster.ProgressCallback) {
	p.poster.SetProgressCallback(callback)

	// Convert poster callback to par2 callback (they have the same signature)
	par2Callback := par2.ProgressCallback(callback)
	p.par2runner.SetProgressCallback(par2Callback)
}

func (p *Postie) Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to post")
	}

	// Start posting
	startTime := time.Now()

	for _, f := range files {
		slog.InfoContext(ctx, "Posting file", "file", f.Path)

		if *p.postingCfg.WaitForPar2 {
			if err := p.post(ctx, f, rootDir, outputDir); err != nil {
				return err
			}
		} else {
			if err := p.postInParallel(ctx, f, rootDir, outputDir); err != nil {
				return err
			}
		}
	}

	// Print final statistics
	stats := p.poster.Stats()
	elapsed := time.Since(startTime)

	slog.InfoContext(ctx, "Upload completed in", "elapsed", elapsed.Round(time.Second))
	slog.InfoContext(ctx, "Articles posted", "count", stats.ArticlesPosted)
	slog.InfoContext(ctx, "Articles checked", "count", stats.ArticlesChecked)
	slog.InfoContext(ctx, "Total bytes", "count", stats.BytesPosted)
	slog.InfoContext(ctx, "Errors", "count", stats.ArticleErrors)

	return nil
}

func (p *Postie) postInParallel(
	ctx context.Context,
	f fileinfo.FileInfo,
	rootDir string,
	outputDir string,
) error {
	var (
		createdPar2Paths []string
		err              error
	)
	defer func() {
		for _, p := range createdPar2Paths {
			if err := os.Remove(p); err != nil {
				slog.ErrorContext(ctx, "Error during par2 cleanup", "error", err)
			}
		}
	}()

	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg)

	errg := errgroup.Group{}

	errg.Go(func() error {
		createdPar2Paths, err = p.par2runner.Create(ctx, []fileinfo.FileInfo{f})
		if err != nil {
			if err != context.Canceled {
				slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
			}

			return nil
		}

		if err := p.poster.Post(ctx, createdPar2Paths, rootDir, nzbGen); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Error during upload of par2 files: %s. Upload will continue without par2.", createdPar2Paths), "error", err)

			return nil
		}

		return nil
	})

	errg.Go(func() error {
		if err := p.poster.Post(ctx, []string{f.Path}, rootDir, nzbGen); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", f.Path), "error", err)

			return err
		}

		return nil
	})

	if err := errg.Wait(); err != nil {
		return err
	}

	// Generate single NZB file for all files
	dirPath := filepath.Dir(f.Path)
	dirPath = strings.TrimPrefix(dirPath, rootDir)

	nzbPath := filepath.Join(outputDir, dirPath, filepath.Base(f.Path)+".nzb")
	if err := nzbGen.Generate(nzbPath); err != nil {
		return fmt.Errorf("error generating NZB file: %w", err)
	}

	return nil
}

func (p *Postie) post(
	ctx context.Context,
	f fileinfo.FileInfo,
	rootDir string,
	outputDir string,
) error {
	var (
		createdPar2Paths []string
		err              error
	)

	defer func() {
		for _, p := range createdPar2Paths {
			if err := os.Remove(p); err != nil {
				slog.ErrorContext(ctx, "Error during par2 cleanup", "error", err)
			}
		}
	}()

	filesPath := []string{f.Path}
	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg)

	if *p.par2Cfg.Enabled {
		createdPar2Paths, err = p.par2runner.Create(ctx, []fileinfo.FileInfo{f})
		if err != nil {
			if err != context.Canceled {
				slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
			}

			return err
		}

		filesPath = append(filesPath, createdPar2Paths...)
	}

	if err := p.poster.Post(ctx, filesPath, rootDir, nzbGen); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", filesPath), "error", err)

		return err
	}

	for _, p := range createdPar2Paths {
		if err := os.Remove(p); err != nil {
			slog.ErrorContext(ctx, "Error during par2 cleanup", "error", err)
		}
	}

	// Generate single NZB file for all files
	dirPath := filepath.Dir(f.Path)
	dirPath = strings.TrimPrefix(dirPath, rootDir)

	nzbPath := filepath.Join(outputDir, dirPath, filepath.Base(f.Path)+".nzb")
	if err := nzbGen.Generate(nzbPath); err != nil {
		return fmt.Errorf("error generating NZB file: %w", err)
	}

	return nil
}

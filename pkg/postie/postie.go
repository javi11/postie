package postie

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/pkg/fileinfo"
)

type Postie struct {
	cfg        config.Config
	par2runner *par2.Par2CmdExecutor
	poster     poster.Poster
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

	// Create par2 runner
	par2runner := par2.New(ctx, postingConfig.ArticleSizeInBytes, par2Cfg)

	// Create poster
	p, err := poster.New(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Error creating poster", "error", err)

		return nil, err
	}

	return &Postie{cfg: cfg, par2runner: par2runner, poster: p}, nil
}

func (p *Postie) Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to post")
	}

	// Start posting
	startTime := time.Now()

	for _, f := range files {
		slog.InfoContext(ctx, "Posting file", "file", f.Path)

		createdPar2Paths, err := p.par2runner.Create(ctx, []fileinfo.FileInfo{f})
		if err != nil {
			slog.ErrorContext(ctx, "Error during par2 creation", "error", err)

			return err
		}

		filesPath := []string{f.Path}
		filesPath = append(filesPath, createdPar2Paths...)

		if err := p.poster.Post(ctx, filesPath, rootDir, outputDir); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", filesPath), "error", err)

			for _, p := range createdPar2Paths {
				if err := os.Remove(p); err != nil {
					slog.ErrorContext(ctx, "Error during par2 cleanup", "error", err)
				}
			}

			return err
		}

		for _, p := range createdPar2Paths {
			if err := os.Remove(p); err != nil {
				slog.ErrorContext(ctx, "Error during par2 cleanup", "error", err)
			}
		}
	}

	// Print final statistics
	stats := p.poster.GetStats()
	elapsed := time.Since(startTime)

	slog.InfoContext(ctx, "Upload completed in", "elapsed", elapsed.Round(time.Second))
	slog.InfoContext(ctx, "Articles posted", "count", stats.ArticlesPosted)
	slog.InfoContext(ctx, "Articles checked", "count", stats.ArticlesChecked)
	slog.InfoContext(ctx, "Total bytes", "count", stats.BytesPosted)
	slog.InfoContext(ctx, "Errors", "count", stats.ArticleErrors)

	return nil
}

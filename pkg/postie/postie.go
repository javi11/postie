package postie

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/pkg/fileinfo"
	"golang.org/x/sync/errgroup"
)

type Postie struct {
	par2Cfg                   *config.Par2Config
	postingCfg                config.PostingConfig
	par2runner                *par2.Par2CmdExecutor
	poster                    poster.Poster
	compressionCfg            config.NzbCompressionConfig
	postUploadScriptCfg       config.PostUploadScriptConfig
	maintainOriginalExtension bool
	jobProgress               progress.JobProgress
}

func New(
	ctx context.Context,
	cfg config.Config,
	jobProgress progress.JobProgress,
) (*Postie, error) {
	// Ensure par2 executable exists and get its path
	par2Cfg, err := cfg.GetPar2Config(ctx)
	if err != nil {
		return nil, err
	}

	postingConfig := cfg.GetPostingConfig()
	compressionConfig := cfg.GetNzbCompressionConfig()
	postUploadScriptConfig := cfg.GetPostUploadScriptConfig()
	maintainOriginalExtension := cfg.GetMaintainOriginalExtension()

	// Create par2 runner with progress manager
	par2runner := par2.New(ctx, postingConfig.ArticleSizeInBytes, par2Cfg, jobProgress)

	// Create poster with progress manager
	p, err := poster.New(ctx, cfg, jobProgress)
	if err != nil {
		slog.ErrorContext(ctx, "Error creating poster", "error", err)

		return nil, err
	}

	return &Postie{
		par2Cfg:                   par2Cfg,
		par2runner:                par2runner,
		poster:                    p,
		postingCfg:                postingConfig,
		compressionCfg:            compressionConfig,
		postUploadScriptCfg:       postUploadScriptConfig,
		maintainOriginalExtension: maintainOriginalExtension,
		jobProgress:               jobProgress,
	}, nil
}

func (p *Postie) Close() {
	p.poster.Close()
	if p.jobProgress != nil {
		p.jobProgress.Close()
	}
}

// safeRemoveFile attempts to remove a file with retry logic for Windows compatibility
func safeRemoveFile(ctx context.Context, filePath string) {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		if err := os.Remove(filePath); err == nil {
			return // Success
		}

		// On Windows, files might still be locked. Wait and retry.
		if i < maxRetries-1 {
			delay := baseDelay * time.Duration(i+1)
			select {
			case <-ctx.Done():
				slog.ErrorContext(ctx, "File cleanup cancelled", "file", filePath)
				return
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	// Final attempt with detailed error logging
	if err := os.Remove(filePath); err != nil {
		slog.ErrorContext(ctx, "Failed to remove file after retries",
			"file", filePath,
			"error", err,
			"os", runtime.GOOS)
	}
}

func (p *Postie) Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "Panic in Postie.Post",
				"panic", r,
				"files", len(files),
				"rootDir", rootDir,
				"outputDir", outputDir,
				"os", runtime.GOOS)
		}
	}()

	if len(files) == 0 {
		return "", fmt.Errorf("no files to post")
	}

	// Start posting
	startTime := time.Now()
	var lastNzbPath string

	for _, f := range files {
		slog.InfoContext(ctx, "Posting file", "file", f.Path)

		if *p.postingCfg.WaitForPar2 {
			nzbPath, err := p.post(ctx, f, rootDir, outputDir)
			if err != nil {
				return "", err
			}
			lastNzbPath = nzbPath
		} else {
			nzbPath, err := p.postInParallel(ctx, f, rootDir, outputDir)
			if err != nil {
				return "", err
			}
			lastNzbPath = nzbPath
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

	return lastNzbPath, nil
}

func (p *Postie) postInParallel(
	ctx context.Context,
	f fileinfo.FileInfo,
	rootDir string,
	outputDir string,
) (string, error) {
	var (
		createdPar2Paths []string
		err              error
	)
	defer func() {
		for _, p := range createdPar2Paths {
			safeRemoveFile(ctx, p)
		}
	}()

	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg, p.maintainOriginalExtension)

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
		return "", err
	}

	// Generate single NZB file for all files
	dirPath := filepath.Dir(f.Path)
	dirPath = strings.TrimPrefix(dirPath, rootDir)

	// Use the original filename as input for NZB generation
	nzbPath := filepath.Join(outputDir, dirPath, filepath.Base(f.Path))
	finalPath, err := nzbGen.Generate(nzbPath)
	if err != nil {
		return "", fmt.Errorf("error generating NZB file: %w", err)
	}

	// Execute post upload script if configured
	if err := p.executePostUploadScript(ctx, finalPath); err != nil {
		slog.ErrorContext(ctx, "Post upload script execution failed", "error", err, "nzbPath", finalPath)
		// Note: We don't return the error here to avoid failing the upload if the script fails
	}

	return finalPath, nil
}

func (p *Postie) post(
	ctx context.Context,
	f fileinfo.FileInfo,
	rootDir string,
	outputDir string,
) (string, error) {
	var (
		createdPar2Paths []string
		err              error
	)

	defer func() {
		for _, p := range createdPar2Paths {
			safeRemoveFile(ctx, p)
		}
	}()

	filesPath := []string{f.Path}
	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg, p.maintainOriginalExtension)

	if *p.par2Cfg.Enabled {
		createdPar2Paths, err = p.par2runner.Create(ctx, []fileinfo.FileInfo{f})
		if err != nil {
			if err != context.Canceled {
				slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
			}

			return "", err
		}

		filesPath = append(filesPath, createdPar2Paths...)
	}

	if err := p.poster.Post(ctx, filesPath, rootDir, nzbGen); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", filesPath), "error", err)

		return "", err
	}

	for _, p := range createdPar2Paths {
		safeRemoveFile(ctx, p)
	}

	// Generate single NZB file for all files
	dirPath := filepath.Dir(f.Path)
	dirPath = strings.TrimPrefix(dirPath, rootDir)

	// Use the original filename as input for NZB generation
	nzbPath := filepath.Join(outputDir, dirPath, filepath.Base(f.Path))
	finalPath, err := nzbGen.Generate(nzbPath)
	if err != nil {
		return "", fmt.Errorf("error generating NZB file: %w", err)
	}

	// Execute post upload script if configured
	if err := p.executePostUploadScript(ctx, finalPath); err != nil {
		slog.ErrorContext(ctx, "Post upload script execution failed", "error", err, "nzbPath", finalPath)
		// Note: We don't return the error here to avoid failing the upload if the script fails
	}

	return finalPath, nil
}

func (p *Postie) executePostUploadScript(ctx context.Context, nzbPath string) error {
	if !p.postUploadScriptCfg.Enabled || p.postUploadScriptCfg.Command == "" {
		return nil
	}

	slog.InfoContext(ctx, "Executing post upload script", "command", p.postUploadScriptCfg.Command, "nzbPath", nzbPath)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.postUploadScriptCfg.Timeout.ToDuration())
	defer cancel()

	// Replace {nzb_path} placeholder with actual NZB path
	command := strings.ReplaceAll(p.postUploadScriptCfg.Command, "{nzb_path}", nzbPath)

	// Parse command using appropriate shell for the OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", command)
	}
	cmd.Dir = filepath.Dir(nzbPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.ErrorContext(ctx, "Error executing post upload script", "error", err, "output", string(output), "command", command)
		return fmt.Errorf("post upload script failed: %w", err)
	}

	slog.InfoContext(ctx, "Post upload script executed successfully", "command", command, "output", string(output))
	return nil
}

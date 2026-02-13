package postie

import (
	"context"
	"errors"
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
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/internal/progress"
	"github.com/javi11/postie/pkg/fileinfo"
	"golang.org/x/sync/errgroup"
)

type Postie struct {
	par2Cfg                   *config.Par2Config
	postingCfg                config.PostingConfig
	par2runner                *par2.NativeExecutor
	poster                    poster.Poster
	compressionCfg            config.NzbCompressionConfig
	postUploadScriptCfg       config.PostUploadScriptConfig
	maintainOriginalExtension bool
	jobProgress               progress.JobProgress
	queue                     QueueInterface
}

// QueueInterface defines the queue methods needed by Postie
type QueueInterface interface {
	UpdateScriptStatus(ctx context.Context, itemID string, status string, retryCount int, lastError string, nextRetryAt *time.Time, firstFailureAt *time.Time) error
	MarkScriptCompleted(ctx context.Context, itemID string) error
}

func New(
	ctx context.Context,
	cfg config.Config,
	poolManager *pool.Manager,
	jobProgress progress.JobProgress,
	queue QueueInterface,
) (*Postie, error) {
	// Get PAR2 configuration
	par2Cfg, err := cfg.GetPar2Config(ctx)
	if err != nil {
		return nil, err
	}

	postingConfig := cfg.GetPostingConfig()
	compressionConfig := cfg.GetNzbCompressionConfig()
	postUploadScriptConfig := cfg.GetPostUploadScriptConfig()
	maintainOriginalExtension := cfg.GetMaintainOriginalExtension()

	// Create par2 runner with progress manager
	par2runner := par2.New(postingConfig.ArticleSizeInBytes, par2Cfg, jobProgress)

	// Create poster with progress manager
	p, err := poster.New(ctx, cfg, poolManager, jobProgress)
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
		queue:                     queue,
	}, nil
}

func (p *Postie) Close() {
	p.poster.Close()
	if p.jobProgress != nil {
		p.jobProgress.Close()
	}
}

// CleanupPar2Files removes PAR2 files for the given source file
// This method can be called when a job fails permanently to clean up orphaned PAR2 files
func (p *Postie) CleanupPar2Files(ctx context.Context, sourceFile fileinfo.FileInfo) {
	var dirPath string
	if p.par2Cfg != nil && p.par2Cfg.TempDir != "" {
		dirPath = p.par2Cfg.TempDir
	} else {
		dirPath = filepath.Dir(sourceFile.Path)
	}

	baseName := filepath.Base(sourceFile.Path)
	par2FileName := baseName + ".par2"
	mainPar2Path := filepath.Join(dirPath, par2FileName)

	// Remove main PAR2 file
	if _, err := os.Stat(mainPar2Path); err == nil {
		safeRemoveFile(ctx, mainPar2Path)
		slog.DebugContext(ctx, "Cleaned up main PAR2 file", "file", mainPar2Path)
	}

	// Find and remove all volume files
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		slog.WarnContext(ctx, "Failed to read directory for PAR2 cleanup", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match patterns like .vol0+1.par2, .vol1+1.par2, etc.
		if strings.HasPrefix(name, baseName) && strings.Contains(name, ".vol") && strings.HasSuffix(name, ".par2") {
			volumePath := filepath.Join(dirPath, name)
			safeRemoveFile(ctx, volumePath)
			slog.DebugContext(ctx, "Cleaned up PAR2 volume file", "file", volumePath)
		}
	}

	slog.InfoContext(ctx, "PAR2 cleanup completed", "sourceFile", sourceFile.Path)
}

// safeRemoveFile attempts to remove a file with retry logic
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

	// Final attempt if error just ignore it is a tmp file it will be deleted automatically
	_ = os.Remove(filePath)
}

func (p *Postie) Post(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string, forceFolderMode bool) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to post")
	}

	// Use folder mode (single NZB) if explicitly requested via forceFolderMode
	// This is set to true for:
	// - Add Folder button (explicit folder upload)
	// - Watch mode with SingleNzbPerFolder enabled (FOLDER: prefix in queue)
	if forceFolderMode {
		// Post all files as a single unit (folder mode)
		return p.postFolder(ctx, files, rootDir, outputDir)
	}

	// Start posting (one NZB per file - traditional mode)
	startTime := time.Now()
	var lastNzbPath string
	var lastDeferredErr *poster.DeferredCheckError

	for _, f := range files {
		slog.InfoContext(ctx, "Posting file", "file", f.Path)

		var nzbPath string
		var err error
		if *p.postingCfg.WaitForPar2 {
			nzbPath, err = p.post(ctx, f, rootDir, outputDir)
		} else {
			nzbPath, err = p.postInParallel(ctx, f, rootDir, outputDir)
		}

		if err != nil {
			var de *poster.DeferredCheckError
			if errors.As(err, &de) {
				// Non-fatal - NZB was generated, collect deferred articles
				lastDeferredErr = de
			} else {
				return "", err
			}
		}
		lastNzbPath = nzbPath
	}

	// Print final statistics
	stats := p.poster.Stats()
	elapsed := time.Since(startTime)

	slog.InfoContext(ctx, "Upload completed in", "elapsed", elapsed.Round(time.Second))
	slog.InfoContext(ctx, "Articles posted", "count", stats.ArticlesPosted)
	slog.InfoContext(ctx, "Articles checked", "count", stats.ArticlesChecked)
	slog.InfoContext(ctx, "Total bytes", "count", stats.BytesPosted)
	slog.InfoContext(ctx, "Errors", "count", stats.ArticleErrors)

	// Return deferred check error if present (non-fatal, NZBs were generated)
	if lastDeferredErr != nil {
		return lastNzbPath, lastDeferredErr
	}
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
		postingSucceeded bool
	)
	defer func() {
		// Only process PAR2 files if posting was successful
		if !postingSucceeded {
			// Keep PAR2 files on failure for retry attempts
			return
		}

		// Only clean up if maintain_par2_files is disabled
		// If enabled, files are already in output directory - no action needed
		if p.par2Cfg.MaintainPar2Files == nil || !*p.par2Cfg.MaintainPar2Files {
			// Clean up PAR2 files when maintain_par2_files is disabled (default)
			for _, path := range createdPar2Paths {
				safeRemoveFile(ctx, path)
			}
		}
	}()

	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg, p.maintainOriginalExtension)

	errg := errgroup.Group{}

	errg.Go(func() error {
		// Determine PAR2 output directory based on maintain_par2_files setting
		var par2OutputDir string
		if p.par2Cfg.MaintainPar2Files != nil && *p.par2Cfg.MaintainPar2Files {
			// Generate PAR2 files directly in output directory
			dirPath := filepath.Dir(f.Path)
			relativePath := strings.TrimPrefix(dirPath, rootDir)
			par2OutputDir = filepath.Join(outputDir, relativePath)

			slog.DebugContext(ctx, "Generating PAR2 files directly in output directory",
				"sourceFile", f.Path, "outputDir", par2OutputDir)
		}
		// If par2OutputDir is empty, CreateInDirectory will use default behavior (temp/source dir)

		createdPar2Paths, err = p.par2runner.CreateInDirectory(ctx, []fileinfo.FileInfo{f}, par2OutputDir)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
			}

			return nil
		}

		if err := p.poster.Post(ctx, createdPar2Paths, rootDir, nzbGen); err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, fmt.Sprintf("Error during upload of par2 files: %s. Upload will continue without par2.", createdPar2Paths), "error", err)
			}

			return nil
		}

		return nil
	})

	var deferredErr *poster.DeferredCheckError

	errg.Go(func() error {
		if err := p.poster.Post(ctx, []string{f.Path}, rootDir, nzbGen); err != nil {
			// Check if this is a non-fatal deferred check error
			var de *poster.DeferredCheckError
			if errors.As(err, &de) {
				deferredErr = de
				return nil // Non-fatal, continue to NZB generation
			}
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", f.Path), "error", err)
			}

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

	// Mark posting as successful so PAR2 files get cleaned up
	postingSucceeded = true

	// Return deferred check error if present (non-fatal, NZB was generated)
	if deferredErr != nil {
		return finalPath, deferredErr
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
		postingSucceeded bool
	)

	defer func() {
		// Only process PAR2 files if posting was successful
		if !postingSucceeded {
			// Keep PAR2 files on failure for retry attempts
			return
		}

		// Only clean up if maintain_par2_files is disabled
		// If enabled, files are already in output directory - no action needed
		if p.par2Cfg.MaintainPar2Files == nil || !*p.par2Cfg.MaintainPar2Files {
			// Clean up PAR2 files when maintain_par2_files is disabled (default)
			for _, path := range createdPar2Paths {
				safeRemoveFile(ctx, path)
			}
		}
	}()

	filesPath := []string{f.Path}
	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg, p.maintainOriginalExtension)

	if *p.par2Cfg.Enabled {
		// Determine PAR2 output directory based on maintain_par2_files setting
		var par2OutputDir string
		if p.par2Cfg.MaintainPar2Files != nil && *p.par2Cfg.MaintainPar2Files {
			// Generate PAR2 files directly in output directory
			dirPath := filepath.Dir(f.Path)
			relativePath := strings.TrimPrefix(dirPath, rootDir)
			par2OutputDir = filepath.Join(outputDir, relativePath)

			slog.DebugContext(ctx, "Generating PAR2 files directly in output directory",
				"sourceFile", f.Path, "outputDir", par2OutputDir)
		}
		// If par2OutputDir is empty, CreateInDirectory will use default behavior (temp/source dir)

		createdPar2Paths, err = p.par2runner.CreateInDirectory(ctx, []fileinfo.FileInfo{f}, par2OutputDir)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
			}

			return "", err
		}

		filesPath = append(filesPath, createdPar2Paths...)
	}

	var deferredErr *poster.DeferredCheckError
	if err := p.poster.Post(ctx, filesPath, rootDir, nzbGen); err != nil {
		// Check if this is a non-fatal deferred check error
		if errors.As(err, &deferredErr) {
			slog.InfoContext(ctx, "Some articles deferred for later verification", "file", f.Path, "deferred", len(deferredErr.FailedArticles))
		} else {
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, fmt.Sprintf("Error during upload: %s", filesPath), "error", err)
			}
			return "", err
		}
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

	// Mark posting as successful so PAR2 files get cleaned up
	postingSucceeded = true

	// Return deferred check error if present (non-fatal, NZB was generated)
	if deferredErr != nil {
		return finalPath, deferredErr
	}
	return finalPath, nil
}

// postFolder posts all files from a folder as a single NZB
func (p *Postie) postFolder(ctx context.Context, files []fileinfo.FileInfo, rootDir string, outputDir string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to post in folder")
	}

	startTime := time.Now()

	// Determine the folder name from the first file's path
	// This will be used as the NZB filename
	firstFilePath := files[0].Path
	folderPath := filepath.Dir(firstFilePath)
	folderName := filepath.Base(folderPath)
	if folderName == "." || folderName == "/" {
		// If files are in root, use a default name
		folderName = "upload"
	}

	slog.InfoContext(ctx, "Posting folder as single NZB", "folder", folderName, "files", len(files))

	var (
		createdPar2Paths []string
		err              error
		postingSucceeded bool
	)

	defer func() {
		// Only process PAR2 files if posting was successful
		if !postingSucceeded {
			// Keep PAR2 files on failure for retry attempts
			return
		}

		// Only clean up if maintain_par2_files is disabled
		// If enabled, files are already in output directory - no action needed
		if p.par2Cfg.MaintainPar2Files == nil || !*p.par2Cfg.MaintainPar2Files {
			// Clean up PAR2 files when maintain_par2_files is disabled (default)
			for _, path := range createdPar2Paths {
				safeRemoveFile(ctx, path)
			}
		}
	}()

	// Create a single NZB generator for all files
	nzbGen := nzb.NewGenerator(p.postingCfg.ArticleSizeInBytes, p.compressionCfg, p.maintainOriginalExtension)

	// Collect all file paths and build relative paths map for subject generation
	var allFilePaths []string
	relativePaths := make(map[string]string)
	for _, f := range files {
		allFilePaths = append(allFilePaths, f.Path)
		// Use RelativePath for subject if available, otherwise use filename
		if f.RelativePath != "" {
			relativePaths[f.Path] = f.RelativePath
		}
	}

	if *p.postingCfg.WaitForPar2 {
		// Create PAR2 files for all files in the folder
		if *p.par2Cfg.Enabled {
			// Determine PAR2 output directory based on maintain_par2_files setting
			var par2OutputDir string
			if p.par2Cfg.MaintainPar2Files != nil && *p.par2Cfg.MaintainPar2Files {
				// For folder posting, PAR2 files go directly in outputDir
				par2OutputDir = outputDir

				slog.DebugContext(ctx, "Generating PAR2 files directly in output directory",
					"folder", folderName, "outputDir", par2OutputDir)
			}
			// If par2OutputDir is empty, CreateInDirectory will use default behavior (temp/source dir)

			createdPar2Paths, err = p.par2runner.CreateInDirectory(ctx, files, par2OutputDir)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
				}
				// Continue without PAR2 files
			} else {
				allFilePaths = append(allFilePaths, createdPar2Paths...)
			}
		}

		var deferredErr *poster.DeferredCheckError
		// Post all files (including PAR2) together with relative paths for subjects
		if err := p.poster.PostWithRelativePaths(ctx, allFilePaths, rootDir, nzbGen, relativePaths); err != nil {
			if errors.As(err, &deferredErr) {
				slog.InfoContext(ctx, "Some articles deferred for later verification", "folder", folderName, "deferred", len(deferredErr.FailedArticles))
			} else {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error during folder upload", "folder", folderName, "error", err)
				}
				return "", err
			}
		}

		// Generate NZB and return with deferred error if present
		nzbPath := filepath.Join(outputDir, folderName)
		finalPath, nzbErr := nzbGen.Generate(nzbPath)
		if nzbErr != nil {
			return "", fmt.Errorf("error generating NZB file for folder: %w", nzbErr)
		}
		postingSucceeded = true

		if deferredErr != nil {
			return finalPath, deferredErr
		}
		return finalPath, nil
	}

	// Post files and PAR2 in parallel
	var deferredErr *poster.DeferredCheckError
	{
		errg := errgroup.Group{}

		// Create PAR2 files in parallel
		errg.Go(func() error {
			if !*p.par2Cfg.Enabled {
				return nil
			}

			// Determine PAR2 output directory based on maintain_par2_files setting
			var par2OutputDir string
			if p.par2Cfg.MaintainPar2Files != nil && *p.par2Cfg.MaintainPar2Files {
				// For folder posting, PAR2 files go directly in outputDir
				par2OutputDir = outputDir

				slog.DebugContext(ctx, "Generating PAR2 files directly in output directory",
					"folder", folderName, "outputDir", par2OutputDir)
			}

			createdPar2Paths, err = p.par2runner.CreateInDirectory(ctx, files, par2OutputDir)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error during par2 creation. Upload will continue without par2.", "error", err)
				}
				return nil
			}

			// PAR2 files don't need relative paths - use standard Post
			if err := p.poster.Post(ctx, createdPar2Paths, rootDir, nzbGen); err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error during upload of par2 files. Upload will continue without par2.", "error", err)
				}
				return nil
			}
			return nil
		})

		// Post main files with relative paths for subjects
		errg.Go(func() error {
			if err := p.poster.PostWithRelativePaths(ctx, allFilePaths, rootDir, nzbGen, relativePaths); err != nil {
				// Check if this is a non-fatal deferred check error
				var de *poster.DeferredCheckError
				if errors.As(err, &de) {
					deferredErr = de
					return nil // Non-fatal, continue to NZB generation
				}
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "Error during folder upload", "folder", folderName, "error", err)
				}
				return err
			}
			return nil
		})

		if err := errg.Wait(); err != nil {
			return "", err
		}
	}

	// Generate single NZB file for the entire folder
	// Use folder name as the base for NZB filename
	nzbPath := filepath.Join(outputDir, folderName)
	finalPath, err := nzbGen.Generate(nzbPath)
	if err != nil {
		return "", fmt.Errorf("error generating NZB file for folder: %w", err)
	}

	// Mark posting as successful so PAR2 files get cleaned up
	postingSucceeded = true

	// Print final statistics
	stats := p.poster.Stats()
	elapsed := time.Since(startTime)

	slog.InfoContext(ctx, "Folder upload completed", "folder", folderName, "elapsed", elapsed.Round(time.Second))
	slog.InfoContext(ctx, "Articles posted", "count", stats.ArticlesPosted)
	slog.InfoContext(ctx, "Articles checked", "count", stats.ArticlesChecked)
	slog.InfoContext(ctx, "Total bytes", "count", stats.BytesPosted)
	slog.InfoContext(ctx, "Errors", "count", stats.ArticleErrors)

	// Return deferred check error if present (non-fatal, NZB was generated)
	if deferredErr != nil {
		return finalPath, deferredErr
	}
	return finalPath, nil
}

// ExecutePostUploadScript executes the post-upload script for a completed item
// This should be called after the file has been marked as completed in the database
func (p *Postie) ExecutePostUploadScript(ctx context.Context, nzbPath string, itemID string) error {
	if !p.postUploadScriptCfg.Enabled || p.postUploadScriptCfg.Command == "" {
		return nil
	}

	slog.InfoContext(ctx, "Executing post upload script", "command", p.postUploadScriptCfg.Command, "nzb_path", nzbPath, "item_id", itemID)

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
		errorMsg := fmt.Sprintf("script failed: %v, output: %s", err, string(output))
		slog.ErrorContext(ctx, "Error executing post upload script", "error", err, "output", string(output), "command", command)

		// Track failure in database for retry if queue is available
		if p.queue != nil {
			// Calculate next retry time with exponential backoff
			baseDelay := p.postUploadScriptCfg.RetryDelay.ToDuration()
			now := time.Now()
			nextRetry := now.Add(baseDelay)

			// This is the first failure, so set firstFailureAt to now
			if updateErr := p.queue.UpdateScriptStatus(ctx, itemID, "pending_retry", 0, errorMsg, &nextRetry, &now); updateErr != nil {
				slog.ErrorContext(ctx, "Failed to track script failure", "error", updateErr)
			}
		}

		return fmt.Errorf("post upload script failed: %w", err)
	}

	// Mark script as completed in database if queue is available
	if p.queue != nil {
		if updateErr := p.queue.MarkScriptCompleted(ctx, itemID); updateErr != nil {
			slog.ErrorContext(ctx, "Failed to mark script as completed", "error", updateErr)
		}
	}

	slog.InfoContext(ctx, "Post upload script executed successfully", "command", command, "output", string(output))
	return nil
}

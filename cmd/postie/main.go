package postie

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/poster"
	"github.com/javi11/postie/pkg/par2cmdlinedownloader"
)

var (
	defaultPar2Exe = "./par2cmd"
)

func Execute() {
	ctx := context.Background()

	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	dirPath := flag.String("dir", ".", "Directory containing files to upload")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.ErrorContext(ctx, "Error loading configuration", "error", err)
		os.Exit(1)
	}

	setupLogging(*verbose)

	// Ensure par2 executable exists and get its path
	par2ExePath, err := ensurePar2Executable(ctx, cfg)
	if err != nil {

	}

	// Create par2 runner
	par2runner := par2.New(ctx, par2ExePath, cfg.Posting.ArticleSizeInBytes)

	// Create poster
	p, err := poster.New(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Error creating poster", "error", err)
		os.Exit(1)
	}

	// Start posting
	slog.InfoContext(ctx, "Starting upload from directory", "dir", *dirPath)
	startTime := time.Now()

	// Walk directory
	files := make([]string, 0)
	err = filepath.Walk(*dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		slog.ErrorContext(ctx, "Error during directory walk", "error", err)
		os.Exit(1)
	}

	createdPar2Paths, err := par2runner.Create(ctx, files)
	if err != nil {
		slog.ErrorContext(ctx, "Error during par2 creation", "error", err)
		os.Exit(1)
	}

	files = append(files, createdPar2Paths...)

	if err := p.Post(ctx, files); err != nil {
		slog.ErrorContext(ctx, "Error during upload", "error", err)
		os.Exit(1)
	}

	// Print final statistics
	stats := p.GetStats()
	elapsed := time.Since(startTime)

	slog.InfoContext(ctx, "Upload completed in", "elapsed", elapsed.Round(time.Second))
	slog.InfoContext(ctx, "Articles posted", "count", stats.ArticlesPosted)
	slog.InfoContext(ctx, "Articles checked", "count", stats.ArticlesChecked)
	slog.InfoContext(ctx, "Total bytes", "count", stats.BytesPosted)
	slog.InfoContext(ctx, "Errors", "count", stats.ArticleErrors)
}

func setupLogging(verbose bool) {
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

// ensurePar2Executable checks if a par2 executable is configured, downloads one if necessary,
// and returns the final path to the executable.
func ensurePar2Executable(ctx context.Context, cfg *config.Config) (string, error) {
	if cfg.Par2Exe != "" {
		slog.DebugContext(ctx, "Using configured Par2 executable", "path", cfg.Par2Exe)
		// Verify it exists?
		if _, err := os.Stat(cfg.Par2Exe); err == nil {
			return cfg.Par2Exe, nil
		} else {
			slog.WarnContext(ctx, "Configured Par2 executable not found, proceeding to check default/download", "path", cfg.Par2Exe, "error", err)
			// Fall through to check default/download
		}
	}

	// Check default path
	if _, err := os.Stat(defaultPar2Exe); err == nil {
		slog.InfoContext(ctx, "Par2 executable found in default path, using it.", "path", defaultPar2Exe)
		// Update the config in memory if we found it here? Might not be necessary if only path is returned.
		// cfg.Par2Exe = defaultPar2Exe // Avoid modifying cfg directly here, just return the path
		return defaultPar2Exe, nil
	} else if !os.IsNotExist(err) {
		// Log unexpected error checking default path, but proceed to download
		slog.WarnContext(ctx, "Unexpected error checking for par2 executable at default path", "path", defaultPar2Exe, "error", err)
	}

	// Download if not configured and not found in default path
	slog.InfoContext(ctx, "No par2 executable configured or found, downloading animetosho/par2cmdline-turbo...")
	execPath, err := par2cmdlinedownloader.DownloadPar2Cmd()
	if err != nil {
		return "", fmt.Errorf("failed to download par2cmd: %w", err)
	}

	slog.InfoContext(ctx, "Downloaded Par2 executable", "path", execPath)
	// Update the config in memory? Again, maybe just return the path.
	// cfg.Par2Exe = execPath
	return execPath, nil
}

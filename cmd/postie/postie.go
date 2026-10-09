package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/kipsilabs/postie/internal/config"
	"github.com/kipsilabs/postie/internal/pool"
	"github.com/kipsilabs/postie/internal/progress"
	"github.com/kipsilabs/postie/pkg/fileinfo"
	"github.com/kipsilabs/postie/pkg/postie"
	"github.com/spf13/cobra"
)

var (
	configPath     string
	dirPath        string
	inputFile      string
	verbose        bool
	outputDir      string
	ignorePatterns []string
)

var rootCmd = &cobra.Command{
	Use:   "postie",
	Short: "Postie is a tool for uploading files",
	Long: `Postie is a command-line tool for uploading files to various destinations.
It supports configuration via a YAML file and can process multiple files in a directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		files := make([]fileinfo.FileInfo, 0)
		if inputFile != "" {
			info, err := os.Stat(inputFile)
			if err != nil {
				return err
			}
			files = append(files, fileinfo.FileInfo{Path: inputFile, Size: uint64(info.Size())})
			dirPath = filepath.Dir(inputFile)
		} else {
			var err error
			files, err = collectDirectoryFiles(dirPath, ignorePatterns)
			if err != nil {
				return err
			}
		}
		if len(files) == 0 {
			return fmt.Errorf("no files to post")
		}

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			slog.ErrorContext(ctx, "Error loading configuration", "error", err)
			return err
		}

		setupLogging(verbose)

		// Initialize connection pool manager
		poolManager, err := pool.New(cfg)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating connection pool manager", "error", err)
			return err
		}
		defer func() {
			if err := poolManager.Close(); err != nil {
				slog.ErrorContext(ctx, "Error closing connection pool manager", "error", err)
			}
		}()

		jobProgress := progress.NewProgressJob("postie-job")
		defer jobProgress.Close()

		poster, err := postie.New(ctx, cfg, poolManager, jobProgress, nil)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating postie", "error", err)
			return err
		}

		// Start posting
		slog.InfoContext(ctx, "Starting upload from directory", "dir", dirPath)

		// CLI mode: use config-based folder mode decision (not forced)
		_, err = poster.Post(ctx, files, dirPath, outputDir, inputFile == "")
		return err
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&dirPath, "dir", "d", ".", "Directory containing files to upload")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to output files to")

	rootCmd.PersistentFlags().StringVarP(&inputFile, "input-file", "i", "", "File to upload. If provided, the directory will be ignored.")
	rootCmd.Flags().StringArrayVar(&ignorePatterns, "ignore-pattern", nil, "Ignore files matching a basename glob during directory uploads (repeatable; ignored with --input-file)")
}

func collectDirectoryFiles(dir string, patterns []string) ([]fileinfo.FileInfo, error) {
	for _, pattern := range patterns {
		if err := validateIgnorePattern(pattern); err != nil {
			return nil, fmt.Errorf("invalid ignore pattern %q: %w", pattern, err)
		}
	}
	files := make([]fileinfo.FileInfo, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				return nil
			}
		}
		files = append(files, fileinfo.FileInfo{Path: path, Size: uint64(info.Size())})
		return nil
	})
	return files, err
}

func validateIgnorePattern(pattern string) error {
	// Removing star chunk boundaries makes Match validate the entire pattern even when a prefix doesn't match.
	_, err := filepath.Match(strings.ReplaceAll(pattern, "*", "?"), "")
	return err
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

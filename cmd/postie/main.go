package postie

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/javi11/postie/pkg/postie"
	"github.com/spf13/cobra"
)

var (
	configPath string
	dirPath    string
	verbose    bool
	outputDir  string
)

var rootCmd = &cobra.Command{
	Use:   "postie",
	Short: "Postie is a tool for uploading files",
	Long: `Postie is a command-line tool for uploading files to various destinations.
It supports configuration via a JSON file and can process multiple files in a directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			slog.ErrorContext(ctx, "Error loading configuration", "error", err)
			return err
		}

		setupLogging(verbose)

		poster, err := postie.New(ctx, cfg)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating postie", "error", err)
			return err
		}

		// Start posting
		slog.InfoContext(ctx, "Starting upload from directory", "dir", dirPath)

		// Walk directory
		files := make([]fileinfo.FileInfo, 0)
		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			files = append(files, fileinfo.FileInfo{Path: path, Size: uint64(info.Size())})
			return nil
		})
		if err != nil {
			slog.ErrorContext(ctx, "Error during directory walk", "error", err)
			return err
		}

		return poster.Post(ctx, files, dirPath, outputDir)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.json", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&dirPath, "dir", "d", ".", "Directory containing files to upload")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to output files to")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
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

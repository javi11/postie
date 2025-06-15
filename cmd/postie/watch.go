package postie

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/processor"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/internal/watcher"
	"github.com/javi11/postie/pkg/postie"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a directory for new files and upload them",
	Long: `Watch a directory for new files and automatically upload them when they meet the criteria.
The watch command will monitor the configured directory and upload files according to the settings in the configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			slog.ErrorContext(ctx, "Error loading configuration", "error", err)
			return err
		}

		setupLogging(verbose)

		// Create postie instance
		p, err := postie.New(ctx, cfg)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating postie instance", "error", err)
			return err
		}

		// Get configurations
		watcherCfg := cfg.GetWatcherConfig()
		queueCfg := cfg.GetQueueConfig()

		// Set up directories
		watchDir := dirPath
		if watchDir == "" {
			watchDir = "./watch"
		}

		outputFolder := outputDir
		if outputFolder == "" {
			outputFolder = "./output"
		}

		// Ensure directories exist
		if err := os.MkdirAll(watchDir, 0755); err != nil {
			slog.ErrorContext(ctx, "Error creating watch directory", "error", err)
			return err
		}
		if err := os.MkdirAll(outputFolder, 0755); err != nil {
			slog.ErrorContext(ctx, "Error creating output directory", "error", err)
			return err
		}

		// Initialize queue
		q, err := queue.New(ctx, queueCfg)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating queue", "error", err)
			return err
		}
		defer func() {
			if err := q.Close(); err != nil {
				slog.ErrorContext(ctx, "Error closing queue", "error", err)
			}
		}()

		// Create no-op event emitter for CLI
		noopEventEmitter := func(eventName string, optionalData ...interface{}) {
			// No-op for CLI, events are only used in GUI mode
		}

		// Initialize processor
		proc := processor.New(processor.ProcessorOptions{
			Queue:              q,
			Postie:             p,
			Config:             queueCfg,
			OutputFolder:       outputFolder,
			EventEmitter:       noopEventEmitter,
			DeleteOriginalFile: watcherCfg.DeleteOriginalFile,
		})

		// Start processor in background
		go func() {
			if err := proc.Start(ctx); err != nil && err != context.Canceled {
				slog.ErrorContext(ctx, "Processor error", "error", err)
			}
		}()

		// Create watcher
		w := watcher.New(watcherCfg, q, watchDir, noopEventEmitter)

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start watcher in a goroutine
		go func() {
			if err := w.Start(ctx); err != nil && err != context.Canceled {
				slog.ErrorContext(ctx, "Error running watcher", "error", err)
				cancel()
			}
		}()

		slog.Info("Watching directory", "dir", watchDir, "output", outputFolder)

		// Wait for shutdown signal
		<-sigChan
		slog.Info("Shutting down...")
		cancel()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

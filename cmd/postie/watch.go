package postie

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/watcher"
	"github.com/javi11/postie/pkg/postie"
	"github.com/spf13/cobra"
)

var (
	watchFolder  string
	outputFolder string
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

		// Create watcher
		w, err := watcher.New(ctx, cfg.GetWatcherConfig(), configPath, p, watchFolder, outputFolder)
		if err != nil {
			slog.ErrorContext(ctx, "Error creating watcher", "error", err)
			return err
		}
		defer w.Close()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start watcher in a goroutine
		go func() {
			if err := w.Start(ctx); err != nil {
				slog.ErrorContext(ctx, "Error running watcher", "error", err)
				cancel()
			}
		}()

		// Wait for shutdown signal
		<-sigChan
		slog.Info("Shutting down...")
		cancel()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().StringVar(&watchFolder, "watch-folder", "", "Directory to watch for new files")
	watchCmd.Flags().StringVar(&outputFolder, "output-folder", "", "Directory where processed files will be moved")
}

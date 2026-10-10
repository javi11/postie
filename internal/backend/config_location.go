package backend

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kipsilabs/postie/internal/config"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Keep the location preference in the original config directory so it can be
// discovered before the YAML is loaded. Database and data paths remain unchanged.
func configLocationFile(defaultPath string) string {
	return filepath.Join(filepath.Dir(defaultPath), "config-location")
}

func resolveConfigPath(defaultPath string) (string, error) {
	data, err := os.ReadFile(configLocationFile(defaultPath))
	if os.IsNotExist(err) {
		return defaultPath, nil
	}
	if err != nil {
		return "", fmt.Errorf("read configuration location: %w", err)
	}
	path := string(data)
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("saved configuration location must be an absolute path")
	}
	return filepath.Clean(path), nil
}

// SelectConfigDirectory copies the saved configuration to a chosen directory.
// Cancellation leaves the active path unchanged.
func (a *App) SelectConfigDirectory() (string, error) {
	if a.isWebMode {
		return "", fmt.Errorf("configuration relocation is only available in the desktop app")
	}
	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose configuration directory",
		DefaultDirectory: filepath.Dir(a.GetConfigPath()),
	})
	if err != nil || directory == "" {
		return "", err
	}
	return a.relocateConfig(directory)
}

func (a *App) relocateConfig(directory string) (string, error) {
	if a.isWebMode {
		return "", fmt.Errorf("configuration relocation is only available in the desktop app")
	}
	if directory == "" {
		return "", fmt.Errorf("configuration directory is required")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	destination := filepath.Join(directory, "config.yaml")

	// Serialize copying with saves so pending configuration bytes cannot be lost.
	a.configFileMux.Lock()
	defer a.configFileMux.Unlock()
	source, err := filepath.Abs(a.configPath)
	if err != nil {
		return "", err
	}
	if source == destination {
		return a.configPath, nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read current configuration: %w", err)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create configuration at %s (choose a directory without config.yaml): %w", destination, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(destination)
		}
	}()
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return "", fmt.Errorf("copy configuration: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close configuration: %w", closeErr)
	}
	if err := persistConfigLocation(configLocationFile(a.appPaths.Config), destination); err != nil {
		return "", err
	}
	a.configPath = destination
	committed = true
	return destination, nil
}

func persistConfigLocation(path, destination string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".config-location-*")
	if err != nil {
		return fmt.Errorf("remember configuration location: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, writeErr := file.WriteString(destination)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("remember configuration location: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("remember configuration location: %w", closeErr)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("remember configuration location: %w", err)
	}
	return nil
}

func (a *App) saveConfigFile(cfg *config.ConfigData) error {
	a.configFileMux.Lock()
	defer a.configFileMux.Unlock()
	if a.configLocationErr != nil {
		return a.configLocationErr
	}
	if err := config.SaveConfig(cfg, a.configPath); err != nil {
		return err
	}
	a.configLoadError = ""
	return nil
}

// configurationError distinguishes recoverable config failures from unrelated
// critical errors. Retrying a restored config must not clear other failures.
func (a *App) configurationError() string {
	a.configFileMux.RLock()
	defer a.configFileMux.RUnlock()
	if a.configLocationErr != nil {
		return a.configLocationErr.Error()
	}
	return a.configLoadError
}

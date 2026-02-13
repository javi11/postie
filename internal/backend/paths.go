package backend

import (
	"os"
	"path/filepath"
	"runtime"
)

// AppPaths holds the various paths needed by the application
type AppPaths struct {
	Config   string
	Database string
	Data     string
	Log      string
}

// isRunningInDocker checks if the application is running inside a Docker container
func isRunningInDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for docker in cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		if len(data) > 0 && (filepath.Base(string(data)) == "docker" || filepath.Base(string(data)) == "containerd") {
			return true
		}
	}

	return false
}

// GetAppPaths returns the appropriate paths for the current operating system
func GetAppPaths() (*AppPaths, error) {
	var configDir, dataDir string
	var err error

	// Check if running in Docker first
	if isRunningInDocker() {
		configDir = "/config"
		dataDir = "/config"
	} else {
		switch runtime.GOOS {
		case "darwin": // macOS
			configDir, err = getMacOSConfigDir()
			if err != nil {
				return nil, err
			}
			dataDir = configDir // On macOS, we use the same directory for both config and data
		case "windows":
			configDir, err = getWindowsConfigDir()
			if err != nil {
				return nil, err
			}
			dataDir = configDir // On Windows, we use the same directory for both config and data
		default: // Linux and other Unix-like systems
			configDir, err = getLinuxConfigDir()
			if err != nil {
				return nil, err
			}
			dataDir, err = getLinuxDataDir()
			if err != nil {
				return nil, err
			}
		}
	}

	// Ensure directories exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}
	if configDir != dataDir {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, err
		}
	}

	logPath := filepath.Join(dataDir, "postie.log")

	return &AppPaths{
		Config:   filepath.Join(configDir, "config.yaml"),
		Database: filepath.Join(configDir, "postie.db"),
		Data:     dataDir,
		Log:      logPath,
	}, nil
}

// getMacOSConfigDir returns the appropriate configuration directory for macOS
func getMacOSConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Library", "Application Support", "Postie"), nil
}

// getWindowsConfigDir returns the appropriate configuration directory for Windows
func getWindowsConfigDir() (string, error) {
	appDataDir := os.Getenv("APPDATA")
	if appDataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appDataDir = filepath.Join(homeDir, "AppData", "Roaming")
	}
	return filepath.Join(appDataDir, "Postie"), nil
}

// getLinuxConfigDir returns the appropriate configuration directory for Linux
func getLinuxConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configHome, "postie"), nil
}

// getLinuxDataDir returns the appropriate data directory for Linux
func getLinuxDataDir() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}
	return filepath.Join(dataHome, "postie"), nil
}

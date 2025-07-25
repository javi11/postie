package parpardownloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/mholt/archiver"
)

// Release represents the structure of the GitHub release JSON response
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// DownloadParParCmd downloads the latest parpar executable from GitHub releases
// for the current operating system and architecture.
//
// It performs the following steps:
// 1. Fetches latest release information from GitHub
// 2. Determines system OS and architecture
// 3. Finds appropriate release asset for the system
// 4. Downloads the executable file
//
// Returns:
//   - string: The name of the downloaded executable ("parpar")
//   - error: Any error encountered during the download process
func DownloadParParCmd(executablePath string) (string, error) {

	// Fetch the latest release information from GitHub API
	release, err := fetchLatestRelease()
	if err != nil {
		slog.With("err", err).Error("Error fetching latest release")

		return "", err
	}

	// Determine the system's OS and architecture
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map system details to the appropriate asset
	asset, err := findAssetForSystem(release, goos, goarch)
	if err != nil {
		slog.With("err", err).Error("Error finding asset for system")

		return "", err
	}

	slog.Info(fmt.Sprintf("Downloading %s for %s/%s...", asset.Name, goos, goarch))

	// Download the asset
	err = downloadFile(executablePath, asset.BrowserDownloadURL)
	if err != nil {
		slog.With("err", err).Error("Error downloading file")

		return "", err
	}

	slog.Info(fmt.Sprintf("Downloaded %s successfully.\n", asset.Name))

	return executablePath, nil
}

// fetchLatestRelease retrieves the latest release information from GitHub
func fetchLatestRelease() (*Release, error) {
	url := "https://api.github.com/repos/animetosho/ParPar/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release Release
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

// findAssetForSystem matches the system's OS and architecture to an asset in the release
func findAssetForSystem(release *Release, goos, goarch string) (*struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}, error) {
	var assetName string
	// Detect actual runtime architecture for containers
	actualArch := goarch
	if goos == "linux" {
		// Check if we're running in a Docker container and detect actual platform
		if containerArch := detectContainerArch(); containerArch != "" {
			actualArch = containerArch
		}
	}
	switch goos {
	case "linux":
		switch actualArch {
		case "amd64":
			assetName = "linux-static-amd64.xz"
		case "arm64":
			assetName = "linux-static-aarch64.xz"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", actualArch)
		}
	case "darwin":
		// macOS only has x64 builds available, but they work on both Intel and Apple Silicon via Rosetta
		assetName = "macos-x64.xz"
	case "windows":
		// Windows only has x64 builds available, but they work on both amd64 and arm64
		assetName = "win64.7z"
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", goos)
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, assetName) {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no asset found for %s/%s", goos, goarch)
}

// downloadFile downloads a file from the specified URL and extracts it if it's compressed
func downloadFile(filename, url string) error {
	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the entire response into a buffer
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Create the output file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	// Determine the archive type and decompress accordingly
	if strings.HasSuffix(url, ".xz") {
		// Handle XZ compression
		xzReader := archiver.NewXz()
		err = xzReader.Decompress(buf, out)
		if err != nil {
			return fmt.Errorf("error decompressing XZ file: %w", err)
		}
	} else if strings.HasSuffix(url, ".7z") {
		// For 7z files, we need to use the sevenzip library
		// First save the compressed file temporarily
		tempFile := filename + ".7z"
		tempOut, err := os.Create(tempFile)
		if err != nil {
			return fmt.Errorf("error creating temp file: %w", err)
		}

		_, err = io.Copy(tempOut, buf)
		_ = tempOut.Close()
		if err != nil {
			_ = os.Remove(tempFile)
			return fmt.Errorf("error writing temp file: %w", err)
		}

		// Open the 7z file for reading
		reader, err := sevenzip.OpenReader(tempFile)
		if err != nil {
			_ = os.Remove(tempFile)
			return fmt.Errorf("error opening 7z file: %w", err)
		}
		defer func() {
			_ = reader.Close()
		}()

		// Find the executable file in the archive (should be the first/only file)
		if len(reader.File) == 0 {
			_ = reader.Close()
			_ = os.Remove(tempFile)
			return fmt.Errorf("no files found in 7z archive")
		}

		// Extract the first file (which should be parpar.exe)
		file := reader.File[0]
		rc, err := file.Open()
		if err != nil {
			_ = reader.Close()
			_ = os.Remove(tempFile)
			return fmt.Errorf("error opening file in archive: %w", err)
		}

		// Copy the executable content to the output file
		_, err = io.Copy(out, rc)
		_ = rc.Close()
		_ = reader.Close()

		// Clean up temp file
		_ = os.Remove(tempFile)

		if err != nil {
			return fmt.Errorf("error extracting file from 7z archive: %w", err)
		}
	} else {
		// If no compression detected, just copy the content directly
		_, err = io.Copy(out, buf)
		if err != nil {
			return fmt.Errorf("error copying file content: %w", err)
		}
	}

	// Add execute permissions to the downloaded file
	err = os.Chmod(filename, 0755)
	if err != nil {
		return fmt.Errorf("error setting execute permission for %s: %w", filename, err)
	}

	return nil
}

// detectContainerArch detects the actual container architecture at runtime
// This helps with cross-platform Docker builds where compile-time arch differs from runtime arch
func detectContainerArch() string {
	// Method 1: Check /proc/cpuinfo for processor info (Linux-specific)
	if cpuInfo, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		cpuInfoStr := string(cpuInfo)

		// Look for x86_64/amd64 indicators
		if strings.Contains(cpuInfoStr, "x86_64") ||
			strings.Contains(cpuInfoStr, "Intel") ||
			strings.Contains(cpuInfoStr, "AMD") {
			return "amd64"
		}

		// Look for ARM64/aarch64 indicators
		if strings.Contains(cpuInfoStr, "aarch64") ||
			strings.Contains(cpuInfoStr, "ARM64") ||
			strings.Contains(cpuInfoStr, "arm64") {
			return "arm64"
		}
	}

	// Method 2: Check platform-specific environment variables
	if arch := os.Getenv("TARGETARCH"); arch != "" {
		return arch
	}
	if platform := os.Getenv("TARGETPLATFORM"); platform != "" {
		if strings.Contains(platform, "amd64") {
			return "amd64"
		}
		if strings.Contains(platform, "arm64") || strings.Contains(platform, "aarch64") {
			return "arm64"
		}
	}
	// Method 3: For Docker environments, default to amd64 if detection fails
	// This is safer as most container images run in amd64 mode even on ARM hosts
	if _, err := os.Stat("/.dockerenv"); err == nil {
		slog.Warn("Running in Docker but couldn't detect arch, defaulting to amd64")
		return "amd64"
	}

	// If all methods fail, return empty to use runtime.GOARCH
	return ""
}

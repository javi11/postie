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
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			assetName = "linux-static-amd64.xz"
		case "arm64":
			assetName = "linux-static-aarch64.xz"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", goarch)
		}
	case "darwin":
		switch goarch {
		case "amd64":
			assetName = "macos-x64.xz"
		case "arm64":
			assetName = "macos-x64.xz"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", goarch)
		}
	case "windows":
		switch goarch {
		case "amd64":
			assetName = "win64.7z"
		case "arm64":
			assetName = "win64.7z"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", goarch)
		}
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

// downloadFile downloads a file from the specified URL and extracts it if it's a zip file
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

	// Create a xz reader from the buffer
	xzReader := archiver.NewXz()

	// Create the output file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	// Copy the executable to the output file
	err = xzReader.Decompress(buf, out)
	if err != nil {
		return err
	}

	// Add execute permissions to the downloaded file
	err = os.Chmod(filename, 0755)
	if err != nil {
		return fmt.Errorf("error setting execute permission for %s: %w", filename, err)
	}

	return nil
}

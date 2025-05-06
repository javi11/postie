package par2cmdlinedownloader

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// Release represents the structure of the GitHub release JSON response
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// DownloadPar2Cmd downloads the latest par2cmd executable from GitHub releases
// for the current operating system and architecture.
//
// It performs the following steps:
// 1. Fetches latest release information from GitHub
// 2. Determines system OS and architecture
// 3. Finds appropriate release asset for the system
// 4. Downloads the executable file
//
// Returns:
//   - string: The name of the downloaded executable ("par2cmd")
//   - error: Any error encountered during the download process
func DownloadPar2Cmd() (string, error) {
	executable := "./par2cmd"

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
	err = downloadFile("par2cmd", asset.BrowserDownloadURL)
	if err != nil {
		slog.With("err", err).Error("Error downloading file")

		return "", err
	}

	slog.Info(fmt.Sprintf("Downloaded %s successfully.\n", asset.Name))

	return executable, nil
}

// fetchLatestRelease retrieves the latest release information from GitHub
func fetchLatestRelease() (*Release, error) {
	url := "https://api.github.com/repos/Parchive/par2cmdline/releases/latest"
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
			assetName = "linux-amd64.zip"
		case "arm64":
			assetName = "linux-arm64.zip"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", goarch)
		}
	case "darwin":
		switch goarch {
		case "amd64":
			assetName = "macos-amd64.zip"
		case "arm64":
			assetName = "macos-arm64.zip"
		default:
			return nil, fmt.Errorf("unsupported architecture: %s", goarch)
		}
	case "windows":
		switch goarch {
		case "amd64":
			assetName = "win-x64.zip"
		case "arm64":
			assetName = "win-arm64.zip"
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
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Create a zip reader from the buffer
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return fmt.Errorf("error creating zip reader: %w", err)
	}

	// Find the par2cmd executable in the zip file
	var executableFile *zip.File
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "par2") || strings.HasSuffix(file.Name, "par2.exe") {
			executableFile = file
			break
		}
	}

	if executableFile == nil {
		return fmt.Errorf("no par2cmd executable found in zip file")
	}

	// Create the output file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	// Open the file in the zip
	rc, err := executableFile.Open()
	if err != nil {
		return fmt.Errorf("error opening file in zip: %w", err)
	}
	defer rc.Close()

	// Copy the executable to the output file
	_, err = io.Copy(out, rc)
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

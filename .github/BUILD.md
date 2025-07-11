# Build Configuration

This project has been configured to build three separate components:

## Components

### 1. CLI Application (`postie-cli`)
- **Source**: `cmd/main.go`
- **Config**: `.goreleaser-cli.yml`
- **Platforms**: Linux, macOS, Windows (amd64, arm64)
- **Features**: Command-line interface, no GUI dependencies
- **CGO**: Disabled for maximum portability

### 2. GUI Application (`postie-gui`)
- **Source**: `main.go`
- **Built with**: Wails framework
- **Platforms**: macOS (Universal), Windows (amd64)
- **Features**: Cross-platform desktop application with web-based UI
- **Frontend**: Svelte-based UI in `frontend/` directory

### 3. Server Application (`postie-server`)
- **Source**: `cmd/main.go`
- **Config**: `.goreleaser-server.yml`
- **Platforms**: Linux (amd64, arm64)
- **Features**: Docker images, server deployment
- **CGO**: Enabled with static linking

## GitHub Workflows

### Release Workflow (`.github/workflows/release.yml`)
Triggered on version tags (`v*.*.*`):

- **`build-cli`**: Builds CLI for all platforms using GoReleaser
- **`build-gui-windows`**: Builds Windows GUI using Wails on Windows runner
- **`build-gui-macos`**: Builds macOS GUI using Wails on macOS runner
- **`build-server`**: Builds Linux server and Docker images

### Pull Request Workflow (`.github/workflows/pull-request.yml`)
Triggered on PRs to main:

- **`build-cli-test`**: Tests CLI builds (snapshot)
- **`build-gui-test`**: Tests GUI builds (Linux only for CI)
- **`build-server-test`**: Tests server builds (Linux only)

## Artifacts

### Release Artifacts
- `postie-cli_v{version}_{os}_{arch}.{tar.gz|zip}` - CLI binaries
- `postie-gui-windows-amd64.exe` - Windows GUI executable
- `postie-gui-macos-universal` - macOS Universal binary
- `postie-server_v{version}_linux_{arch}.tar.gz` - Server binaries
- Docker images: `ghcr.io/{repo}:v{version}`

### Development Artifacts (PR builds)
- `cli-snapshot` - CLI snapshot builds
- `gui-test` - GUI test build (Linux)
- `server-snapshot` - Server snapshot builds

## Configuration Files

- `.goreleaser-cli.yml` - CLI application config (all platforms)
- `.goreleaser-server.yml` - Server application config (Linux + Docker)
- `.gorelease-dev.yml` - Development/testing config (Linux only)
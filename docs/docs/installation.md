---
sidebar_position: 2
---

# Installation

There are several ways to install and run Postie. Choose the method that best suits your environment and needs.

**💡 All installation methods include access to the modern web interface for easy configuration and monitoring.**

## Quick Download

Download the latest release for your platform:

<a href="https://github.com/javi11/postie/releases/latest/download/postie-gui-windows-amd64.zip">
  <img src="/img/download-for-windows.webp" alt="Download for Windows" width="200" height="60" />
</a>
<a href="https://github.com/javi11/postie/releases/latest/download/postie-gui-macos-universal.zip">
  <img src="/img/download-for-mac.png" alt="Download for macOS" width="200" height="60" />
</a>

[![View All Releases](https://img.shields.io/badge/All%20Releases-View-6c757d?style=for-the-badge&logo=github)](https://github.com/javi11/postie/releases/latest)

## Binary Installation

The easiest way to get started is by downloading a pre-built binary:

1. Go to the [releases page](https://github.com/javi11/postie/releases) or use the download buttons above
2. Download the appropriate binary for your platform
3. Extract the archive
4. Run Postie to start the web interface:

```bash
./postie
```

5. Open your browser to `http://localhost:8080` and configure through the web UI

### Alternative: Command Line Configuration

If you prefer to use a configuration file:

1. Create a config.yaml file ([see configuration guide](./configuration.md))
2. Run Postie with the configuration file:

```bash
./postie -config config.yaml -d ./upload -o ./output
```

## Docker Installation

### Using Docker Compose (Recommended)

Postie provides a complete Docker setup that includes the web interface for easy management.

1. Create a `docker-compose.yml` file:

```yaml
services:
  postie:
    container_name: postie
    image: ghcr.io/javi11/postie:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config:/config
      - ./watch:/watch
      - ./output:/output
    environment:
      - PORT=8080
      - HOST=0.0.0.0
      - PUID=1000
      - PGID=1000
    restart: unless-stopped
```

1. Create the following directories:

   - `config`: Contains your configuration file
   - `watch`: Directory to watch for new files
   - `output`: Directory for output files

1. Create a `config.yaml` file in the `config` directory ([see configuration guide](./configuration.md))

1. Start the container:

```bash
docker-compose up -d
```

1. Access the web interface at `http://localhost:8080`

### Using Docker Run

Alternatively, you can run Postie directly with Docker:

```bash
docker run -d \
  --name postie \
  -p 8080:8080 \
  -v ./config:/config \
  -v ./watch:/watch \
  -v ./output:/output \
  -e PUID=1000 \
  -e PGID=1000 \
  ghcr.io/javi11/postie:latest
```

### Web Interface Access

After starting the Docker container, you can access the Postie web interface by:

1. Opening your web browser
2. Navigating to `http://localhost:8080`
3. Using the web interface to:
   - Monitor upload progress
   - Manage the upload queue
   - Configure settings
   - View logs and statistics

The web interface provides a modern, user-friendly way to interact with Postie without needing to use command-line tools.

## Using the Binary Version

After downloading and extracting the binary for your platform:

### Windows

1. Extract the downloaded zip file
2. Create a `config.yaml` file ([see configuration guide](./configuration.md))
3. Open Command Prompt or PowerShell
4. Navigate to the extracted folder
5. Run Postie:

```cmd
postie.exe -config config.yaml -d ./upload -o ./output
```

### macOS

1. Extract the downloaded tar.gz file
2. Create a `config.yaml` file ([see configuration guide](./configuration.md))
3. Open Terminal
4. Navigate to the extracted folder
5. Make the binary executable:

```bash
chmod +x postie
```

1. Run Postie:

```bash
./postie -config config.yaml -d ./upload -o ./output
```

### Linux

1. Extract the downloaded tar.gz file
2. Create a `config.yaml` file ([see configuration guide](./configuration.md))
3. Open Terminal
4. Navigate to the extracted folder
5. Make the binary executable:

```bash
chmod +x postie
```

1. Run Postie:

```bash
./postie -config config.yaml -d ./upload -o ./output
```

#### Web Interface with Binary

1. Download [postie-web](https://github.com/javi11/postie/releases/latest/download/postie-web-linux-amd64.tar.gz)
2. Extract the file
3. Start Postie with the web server enabled `./postie-web --port 80080`
4. Open your web browser
5. Navigate to `http://localhost:8080` (or the port specified in your configuration)
6. Use the same web interface features as the Docker version

## Building from Source

If you prefer to build from source:

```bash
git clone https://github.com/javi11/postie.git
cd postie
go build
```

## Installing with Go

If you have Go installed, you can install Postie directly:

```bash
go install github.com/javi11/postie@latest
```

## System Requirements

- **OS**: Linux, macOS, or Windows
- **Disk Space**: Varies based on file sizes being processed
- **Network**: Reliable internet connection with sufficient bandwidth for Usenet posting

## Next Steps

After installation, you'll need to [configure Postie](./configuration) before you can start posting files.

---
sidebar_position: 2
---

# Installation

There are several ways to install and run Postie. Choose the method that best suits your environment and needs.

## Binary Installation

The easiest way to get started is by downloading a pre-built binary:

1. Go to the [releases page](https://github.com/javi11/postie/releases)
2. Download the appropriate binary for your platform
3. Extract the archive
4. Create config.yaml file [see](./configuration.md)
5. Run Postie with a configuration file:

```bash
./postie -config config.yaml -d ./upload -o ./output
```

## Docker Installation

You can run Postie using Docker Compose for a containerized setup:

1. Create a `docker-compose.yml` file:

```yaml
services:
  postie:
    container_name: postie
    platform: linux/amd64
    image: ghcr.io/javi11/postie:latest
    volumes:
      - ./config:/config
      - ./watch:/watch
      - ./output:/output
    environment:
      - PUID=1000
      - PGID=1000
    restart: unless-stopped
```

2. Create the following directories:

   - `config`: Contains your configuration file
   - `watch`: Directory to watch for new files
   - `output`: Directory for output files

3. Start the container:

```bash
docker-compose up -d
```

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

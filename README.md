# Postie

[![Build](https://github.com/javi11/postie/actions/workflows/pull-request.yml/badge.svg)](https://github.com/javi11/postie/actions/workflows/pull-request.yml)
[![Coverage](https://github.com/javi11/postie/actions/workflows/coverage.yml/badge.svg)](https://github.com/javi11/postie/actions/workflows/coverage.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/javi11/postie)](https://goreportcard.com/report/github.com/javi11/postie)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<a href="https://www.buymeacoffee.com/qbt52hh7sjd"><img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=☕&slug=qbt52hh7sjd&button_colour=FFDD00&font_colour=000000&font_family=Cookie&outline_colour=000000&coffee_colour=ffffff" /></a>

![logo](./docs/static/img/full_logo.jpeg)

A high-performance Usenet binary poster written in Go, inspired by Nyuu-Obfuscation.

## Features

- Multi-server support with automatic failover
- Yenc encoding using rapidyenc for high performance
- Post checking with multiple attempts
- Configurable segment size
- Automatic retry on failure
- SSL/TLS support
- Connection pooling for better performance
- PAR2 support with configurable redundancy
- Multiple obfuscation policies
- Configurable group posting policies
- File watching and automatic posting
- Configurable posting schedules

## Quick Start

### Docker (Recommended)

```bash
# Create docker-compose.yml
curl -o docker-compose.yml https://raw.githubusercontent.com/javi11/postie/main/docker-compose.yml

# Start Postie
docker-compose up -d

# Access web interface at http://localhost:8080
```

### Binary Download

[![Download for Windows](https://img.shields.io/badge/Windows-Download-0078d4?style=for-the-badge&logo=windows)](https://github.com/javi11/postie/releases/latest/download/postie_windows_amd64.zip)
[![Download for macOS](https://img.shields.io/badge/macOS-Download-0078d4?style=for-the-badge&logo=apple)](https://github.com/javi11/postie/releases/latest/download/postie_darwin_amd64.zip)
[![Download for Linux](https://img.shields.io/badge/Linux-Download-0078d4?style=for-the-badge&logo=linux)](https://github.com/javi11/postie/releases/latest/download/postie_linux_amd64.zip)

## Screenshots

![Postie Dashboard](./docs/static/examples/dashboard.png)

Web interface dashboard showing upload progress and queue management

## Documentation

For detailed documentation, installation instructions, configuration options, and usage examples, please visit the [Postie Documentation Site](https://postie.kipsilabs.top).

## Quick Links

- [Installation Guide](https://javi11.github.io/postie/docs/installation)
- [Quick Start](https://javi11.github.io/postie/docs/quick-start)
- [Configuration Guide](https://javi11.github.io/postie/docs/configuration)
- [Obfuscation Policies](https://javi11.github.io/postie/docs/obfuscation)
- [File Watcher](https://javi11.github.io/postie/docs/watcher)

## For Developers

### Building from Source

```bash
git clone https://github.com/javi11/postie.git
cd postie
go build
```

### Installing with Go

```bash
go install github.com/javi11/postie@latest
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

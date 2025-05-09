# Postie

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

## Installation

```bash
go install github.com/javi11/postie@latest
```

## Configuration

Create a configuration file (e.g., `config.yaml`) based on the example in `config.yaml.example`:

```yaml
servers:
  - host: news.example.com
    port: 119
    username: user
    password: pass
    ssl: false
    max_connections: 10
  - host: ssl.example.com
    port: 563
    username: user
    password: pass
    ssl: true
    max_connections: 10

posting:
  max_retries: 3
  retry_delay: 5s
  check_interval: 1m
  max_check_retries: 3
  article_size_in_bytes: 750000
  groups:
    - alt.binaries.example
  throttle_rate: 1048576 # 1MB/s
  message_id_format: random # or "ngx"
  obfuscation_policy: full # or "partial", "none"
  group_policy: each_file # or "all"
  post_headers:
    add_ngx_header: true
    default_from: ""
    custom_headers: []

post_check:
  enabled: true
  delay: 10s
  max_retries: 3
  max_reposts: 1

par2:
  par2_path: ./par2cmd
  redundancy: 10 # 10%
  volume_size: 153600000 # 150MB
```

### Configuration Options

#### Servers Configuration

- `servers`: Array of NNTP servers to use
  - `host`: Server hostname
  - `port`: Server port (119 for non-SSL, 563 for SSL)
  - `username`: Username for authentication
  - `password`: Password for authentication
  - `ssl`: Whether to use SSL/TLS
  - `max_connections`: Maximum number of concurrent connections per server

#### Posting Configuration

- `posting`: Main posting configuration
  - `max_retries`: Maximum number of retry attempts for posting
  - `retry_delay`: Delay between retry attempts
  - `check_interval`: Interval between post checks
  - `max_check_retries`: Maximum number of post check attempts
  - `article_size_in_bytes`: Size of each article in bytes
  - `groups`: List of newsgroups to post to
  - `throttle_rate`: Upload speed limit in bytes per second
  - `message_id_format`: Format of message IDs ("random" or "ngx")
  - `obfuscation_policy`: Level of obfuscation ("full", "partial", or "none")
  - `group_policy`: How to distribute posts across groups ("all" or "each_file")
  - `post_headers`: Additional headers configuration
    - `add_ngx_header`: Whether to add X-NXG header
    - `default_from`: Default poster name
    - `custom_headers`: List of custom headers to add

#### Post Check Configuration

- `post_check`: Configuration for verifying posts
  - `enabled`: Whether to check posts after uploading
  - `delay`: Delay between check attempts
  - `max_retries`: Maximum number of check attempts
  - `max_reposts`: Maximum number of reposts if check fails

#### PAR2 Configuration

- `par2`: PAR2 recovery file configuration
  - `par2_path`: Path to par2 executable
  - `redundancy`: Percentage of redundancy (default: 10%)
  - `volume_size`: Size of each volume in bytes (default: 150MB)

## Usage

```bash
postie -config config.yaml
```

The tool will post all files in the current directory to the specified newsgroups.

## Building from Source

```bash
git clone https://github.com/javi11/postie.git
cd postie
go build
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

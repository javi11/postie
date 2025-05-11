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
- File watching and automatic posting
- Configurable posting schedules

## Installation

```bash
go install github.com/javi11/postie@latest
```

## Configuration

Create a configuration file (e.g., `config.yaml`) based on the example in `config.yaml.example`:

```yaml
servers:
  - host: news.server.com
    port: 1199
    username: ""
    password: ""
    ssl: false
    max_connections: 50
    max_connection_idle_time_in_seconds: 300
    max_connection_ttl_in_seconds: 3600
    insecure_ssl: false

connection_pool:
  min_connections: 5
  health_check_interval: 1m
  skip_providers_verification_on_creation: false

posting:
  max_retries: 3
  retry_delay: 5s
  max_check_retries: 3
  article_size_in_bytes: 750000
  groups:
    - alt.binaries.example
  throttle_rate: 1048576
  message_id_format: random
  obfuscation_policy: full
  par2_obfuscation_policy: full
  group_policy: each_file
  post_headers:
    add_ngx_header: true
    default_from: "user@example.com"
    custom_headers:
      - name: "X-Custom-Header"
        value: "value"

post_check:
  enabled: true
  delay: 10s
  max_reposts: 1

par2:
  par2_path: ./par2cmd
  redundancy: "1n*1.2"
  volume_size: 153600000
  max_input_slices: 4000
  recovery_slices: ""
  extra_par2_options: []

watcher:
  watch_dir: /path/to/watch
  output_dir: /path/to/output
  size_threshold: 104857600
  schedule:
    start_time: "00:00"
    end_time: "23:59"
  ignore_patterns:
    - "*.tmp"
    - "*.part"
  min_file_size: 1048576
  check_interval: 5m
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
  - `max_connection_idle_time_in_seconds`: Maximum time a connection can be idle (default: 300)
  - `max_connection_ttl_in_seconds`: Maximum time a connection can live (default: 3600)
  - `insecure_ssl`: Whether to skip SSL certificate verification

#### Connection Pool Configuration

- `connection_pool`: Configuration for the NNTP connection pool
  - `min_connections`: Minimum number of connections to maintain (default: 5)
  - `health_check_interval`: How often to check connection health (default: 1m)
  - `skip_providers_verification_on_creation`: Skip initial server verification

#### Posting Configuration

- `posting`: Main posting configuration
  - `max_retries`: Maximum number of retry attempts for posting
  - `retry_delay`: Delay between retry attempts
  - `max_check_retries`: Maximum number of post check attempts
  - `article_size_in_bytes`: Size of each article in bytes (default: 750KB)
  - `groups`: List of newsgroups to post to
  - `throttle_rate`: Upload speed limit in bytes per second
  - `max_workers`: Maximum number of concurrent workers (auto-set based on server connections)
  - `message_id_format`: Format of message IDs ("random" or "ngx")
  - `obfuscation_policy`: Level of obfuscation ("full", "partial", or "none")
  - `par2_obfuscation_policy`: Level of obfuscation for par2 files ("full", "partial", or "none")
  - `group_policy`: How to distribute posts across groups ("all" or "each_file")
  - `post_headers`: Additional headers configuration
    - `add_ngx_header`: Whether to add X-NXG header
    - `default_from`: Default poster name
    - `custom_headers`: List of custom headers to add

#### Post Check Configuration

- `post_check`: Configuration for verifying posts
  - `enabled`: Whether to check posts after uploading
  - `delay`: Delay between check attempts
  - `max_reposts`: Maximum number of reposts if check fails

#### PAR2 Configuration

- `par2`: PAR2 recovery file configuration
  - `par2_path`: Path to par2 executable
  - `redundancy`: Redundancy level (default: "1n\*1.2" for 10%)
  - `volume_size`: Size of each volume in bytes (default: 150MB)
  - `max_input_slices`: Maximum number of input slices (default: 4000)
  - `recovery_slices`: Custom recovery slices configuration
  - `extra_par2_options`: Additional par2 command line options

#### Watcher Configuration

- `watcher`: File watching configuration
  - `watch_dir`: Directory to watch for new files
  - `output_dir`: Directory for output files
  - `size_threshold`: Minimum size for files to be processed
  - `schedule`: Posting schedule
    - `start_time`: When to start posting (24h format)
    - `end_time`: When to stop posting (24h format)
  - `ignore_patterns`: File patterns to ignore
  - `min_file_size`: Minimum file size to process
  - `check_interval`: How often to check for new files

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

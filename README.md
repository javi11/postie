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

## Installation

```bash
go install github.com/javi11/postie@latest
```

## Configuration

Create a configuration file (e.g., `config.json`) based on the example in `config.json.example`:

```json
{
  "servers": [
    {
      "host": "news.example.com",
      "port": 119,
      "username": "user",
      "password": "pass",
      "ssl": false
    }
  ],
  "posting": {
    "max_retries": 3,
    "retry_delay": "5s",
    "check_interval": "10s",
    "max_check_retries": 5,
    "segment_size": 750000,
    "group": "alt.binaries.example"
  }
}
```

### Configuration Options

- `servers`: Array of NNTP servers to use

  - `host`: Server hostname
  - `port`: Server port (119 for non-SSL, 563 for SSL)
  - `username`: Username for authentication
  - `password`: Password for authentication
  - `ssl`: Whether to use SSL/TLS

- `posting`: Posting configuration
  - `max_retries`: Maximum number of retry attempts for posting
  - `retry_delay`: Delay between retry attempts
  - `check_interval`: Interval between post checks
  - `max_check_retries`: Maximum number of post check attempts
  - `segment_size`: Size of each segment in bytes
  - `group`: Newsgroup to post to

## Usage

```bash
postie -config config.json
```

The tool will post all files in the current directory to the specified newsgroup.

## Building from Source

```bash
git clone https://github.com/javi11/postie.git
cd postie
go build
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

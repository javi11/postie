---
sidebar_position: 4
---

# Configuration Guide

This guide explains all the configuration options available in Postie's YAML configuration file.

## Complete Configuration Example

Below is a complete example of a configuration file with all available options:

```yaml
servers:
  - host: news.example.com
    port: 119
    username: user
    password: pass
    ssl: true
    max_connections: 10
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
  article_size_in_bytes: 750000
  groups:
    - alt.bin.test
  throttle_rate: 1048576 # 1MB/s
  message_id_format: random # Options: random, ngx
  obfuscation_policy: full # Options: full, partial, none
  par2_obfuscation_policy: full # Options: full, partial, none
  group_policy: each_file # Options: all, each_file
  post_headers:
    add_ngx_header: false
    default_from: ""

post_check:
  enabled: true
  delay: 10s
  max_reposts: 1

par2:
  par2_path: ./parpar
  redundancy: "1n*1.2" # [redundancy](https://github.com/animetosho/ParPar/blob/6feee4dd94bb18480f0bf08cd9d17ffc7e671b69/help-full.txt#L75)
  volume_size: 153600000 # 150MB
  max_input_slices: 4000
  extra_par2_options: []

watcher:
  size_threshold: 104857600 # 100MB
  schedule:
    start_time: "00:00"
    end_time: "23:59"
  ignore_patterns:
    - "*.tmp"
    - "*.part"
    - "*.!ut"
  min_file_size: 1048576 # 1MB
  check_interval: 5m
```

## Configuration Sections

### Usenet Servers

Configure one or more Usenet providers:

```yaml
servers:
  - host: news.example.com # Server hostname
    port: 119 # Server port (typically 119 for non-SSL, 563 for SSL)
    username: user # Your username
    password: pass # Your password
    ssl: true # Whether to use SSL/TLS
    max_connections: 10 # Maximum concurrent connections to this server
    max_connection_idle_time_in_seconds: 300 # Max idle time before recycling a connection
    max_connection_ttl_in_seconds: 3600 # Max total lifetime of a connection
    insecure_ssl: false # Set to true to skip SSL certificate verification
```

You can add multiple servers for redundancy. Postie will automatically fail over to another server if one becomes unavailable. It will also randomly post to any of the servers specified, so be aware if you use different backbones.

### Connection Pool

Configure how connections are managed:

```yaml
connection_pool:
  min_connections: 5 # Minimum number of connections to maintain alive in watch mode.
  health_check_interval: 1m # How often to check connection health.
  skip_providers_verification_on_creation: false # Skip initial server verification.
```

### Posting Settings

Configure how files are posted:

```yaml
posting:
  max_retries: 3 # Maximum retry attempts for posting
  retry_delay: 5s # Delay between retry attempts
  article_size_in_bytes: 750000 # Size of each article (default: 750KB)
  groups: # Newsgroups to post to
    - alt.bin.test
  throttle_rate: 1048576 # Upload speed limit in bytes/sec (1MB/s)
  message_id_format: random # Format of message IDs ("random" or "[ngx](https://github.com/javi11/nxg)")
  obfuscation_policy: full # Level of obfuscation ("full", "partial", or "none")
  par2_obfuscation_policy: full # Obfuscation for PAR2 files
  group_policy: each_file # How to distribute posts ("to all groups" or "or one group for each_file")
  post_headers: # Additional headers configuration
    add_ngx_header: false # Whether to add [X-NXG](https://github.com/javi11/nxg) header
    default_from: "" # Default poster name
    custom_headers: # Custom headers to add to the post
      - name: "X-Custom-Header"
        value: "value"
```

#### Obfuscation Policies

- **full**: Maximum obfuscation (randomized filenames, subjects, dates, and poster)
- **partial**: Medium obfuscation (obfuscated filenames and subjects, consistent poster)
- **none**: No obfuscation

#### Group Policies

- **all**: Post to all specified groups simultaneously
- **each_file**: Post each file to a different group from the list

### Post Verification

Configure post verification:

```yaml
post_check:
  enabled: true # Whether to verify posts after uploading using STAT method
  delay: 5s # Delay between check attempts
  max_reposts: 1 # Maximum number of reposts if check fails
```

### PAR2 Recovery Files

The default par binary used is [parpar](https://github.com/animetosho/ParPar/blob/master/help-full.txt) but you can also specify [par2cmd](https://github.com/Parchive/par2cmdline) by specifying the path to the executable and naming it as par2cmd

See [parpar](https://github.com/animetosho/ParPar/blob/master/help-full.txt) for more info about the options.
Configure PAR2 recovery file generation:

```yaml
par2:
  par2_path: ./parpar # Path to PAR2 executable
  redundancy: "1n*1.2" # [Redundancy level](https://github.com/animetosho/ParPar/blob/6feee4dd94bb18480f0bf08cd9d17ffc7e671b69/help-full.txt#L75)
  volume_size: 153600000 # Size of each volume (150MB)
  max_input_slices: 4000 # Maximum number of input slices
  extra_par2_options: [] # Additional PAR2 command line options
```

### File Watcher

Configure automatic file watching and posting:

```yaml
watcher:
  size_threshold: 104857600 # 100MB
  schedule:
    start_time: "00:00" # When to start posting (24h format)
    end_time: "23:59" # When to stop posting (24h format)
  ignore_patterns:
    - "*.tmp"
    - "*.part"
    - "*.!ut"
  min_file_size: 1048576 # 1MB
  check_interval: 5m # How often to check for new files
```

> **Note:** For information about file hashing and verification, please see the [File Hash and Verification](file-hash.md) documentation.

## Command Line Parameters

In addition to the configuration file, Postie accepts several command line parameters:

- `-config`: Path to the configuration file
- `-d`: Path to a directory containing files to post
- `-o`: Output directory for processed files
- `-v`: Enable verbose logging
- `-i`: Path to a single file to post. If provided the dir is ignored.
- `-version`: Display version information

Example:

```bash
./postie -config config.yaml -d ./upload -o ./output
```

```bash
./postie watch -config config.yaml -d ./upload -o ./output
```

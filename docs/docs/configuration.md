---
sidebar_position: 4
---

# Configuration Guide

This guide explains all the configuration options available in Postie. **We strongly recommend using the web UI to configure Postie**, as it provides a user-friendly interface with validation, real-time feedback, and organized sections for all configuration options.

## Using the Web UI (Recommended)

The easiest way to configure Postie is through the web interface:

1. Start Postie (using Docker or the binary)
2. Open your browser and navigate to `http://localhost:8080` (or your configured port)
3. Click on the "Settings" tab
4. Configure all options through the intuitive interface with organized tabs:
   - **Core Configuration**: General settings and server configuration
   - **Upload Settings**: Posting configuration, connection pool, and post-check settings
   - **File Processing**: PAR2 and NZB compression settings
   - **Automation**: File watcher and post-upload script configuration

The web UI automatically validates your configuration, provides helpful descriptions for each option, and saves changes instantly. It also shows your current configuration status and any errors that need to be resolved.

## Manual YAML Configuration

If you prefer to manually edit the YAML configuration file, this section explains all available options.

## Complete Configuration Example

Below is a complete example of a configuration file with all available options:

```yaml
# Global output directory for processed files and NZB files
output_dir: "./output"

# Whether to maintain the original file extension in the NZB filename
maintain_original_extension: false

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
    enabled: true

connection_pool:
  min_connections: 5
  health_check_interval: 1m
  skip_providers_verification_on_creation: false

posting:
  # Wait for par2 generation before uploading the file.
  # Setting this to false could improve the posting speed but you risk into posting
  # a something without par2 if this fails, also it will increase the resource usage
  wait_for_par2: true
  max_retries: 3
  retry_delay: 5s
  article_size_in_bytes: 750000
  groups:
    - alt.bin.test
  throttle_rate: 0 # unlimited (bytes per second)
  message_id_format: random # Options: random, ngx
  obfuscation_policy: full # Options: full, partial, none
  par2_obfuscation_policy: full # Options: full, partial, none
  group_policy: each_file # Options: all, each_file
  post_headers:
    add_ngx_header: false
    default_from: ""
    custom_headers:
      - name: "X-Custom-Header"
        value: "custom-value"

post_check:
  enabled: true
  delay: 10s
  max_reposts: 1

par2:
  enabled: true
  par2_path: ./parpar
  redundancy: "1n*1.2" # [redundancy](https://github.com/animetosho/ParPar/blob/6feee4dd94bb18480f0bf08cd9d17ffc7e671b69/help-full.txt#L75)
  volume_size: 153600000 # 150MB
  max_input_slices: 4000
  extra_par2_options: []
  temp_dir: "" # Optional temporary directory for PAR2 operations

nzb_compression:
  enabled: false # Whether to enable compression of the output NZB file
  type: none # Options: none, zstd, brotli
  level: 0 # Compression level (zstd: 1-22, brotli: 0-11)

watcher:
  enabled: false # Whether to enable the file watcher
  watch_directory: "" # Directory to watch for new files
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
  delete_original_file: false # Delete source files after successful upload

# Queue configuration for upload management
queue:
  database_type: sqlite # Options: sqlite, postgres, mysql (only sqlite implemented currently)
  database_path: ./postie_queue.db # Database file path or connection string
  max_concurrent_uploads: 1 # Maximum concurrent uploads from queue

# Post upload script configuration
post_upload_script:
  enabled: false # Whether to enable post upload script execution
  command: "" # Command to execute, use {{nzb_path}} placeholder for NZB file path
  timeout: 30s # Maximum time to wait for command execution
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
    enabled: true # Whether this server is enabled
```

You can add multiple servers for redundancy. Postie will automatically fail over to another server if one becomes unavailable. It will also randomly post to any of the servers specified, so be aware if you use different backbones.

**💡 Tip: Use the web UI to easily add, remove, and test server configurations with real-time validation.**

### Connection Pool

Configure how connections are managed:

```yaml
connection_pool:
  min_connections: 5 # Minimum number of connections to maintain alive in watch mode.
  health_check_interval: 1m # How often to check connection health.
  skip_providers_verification_on_creation: false # Skip initial server verification.
```

**💡 Tip: The web UI provides real-time connection status and allows you to test your configuration before saving.**

### Posting Settings

Configure how files are posted:

```yaml
posting:
  wait_for_par2: true # Wait for PAR2 generation before uploading
  max_retries: 3 # Maximum retry attempts for posting
  retry_delay: 5s # Delay between retry attempts
  article_size_in_bytes: 750000 # Size of each article (default: 750KB)
  groups: # Newsgroups to post to
    - alt.bin.test
  throttle_rate: 0 # Upload speed limit in bytes/sec (0 = unlimited)
  message_id_format: random # Format of message IDs ("random" or "[ngx](https://github.com/javi11/nxg)")
  obfuscation_policy: full # Level of obfuscation ("full", "partial", or "none")
  par2_obfuscation_policy: full # Obfuscation for PAR2 files
  group_policy: each_file # How to distribute posts ("all" or "each_file")
  post_headers: # Additional headers configuration
    add_ngx_header: false # Whether to add [X-NXG](https://github.com/javi11/nxg) header
    default_from: "" # Default poster name
    custom_headers: # Custom headers to add to the post
      - name: "X-Custom-Header"
        value: "value"
```

**💡 Tip: The web UI provides helpful tooltips and validation for all posting options, making it easy to understand the impact of each setting.**

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
  enabled: true
  par2_path: ./parpar # Path to PAR2 executable
  redundancy: "1n*1.2" # [Redundancy level](https://github.com/animetosho/ParPar/blob/6feee4dd94bb18480f0bf08cd9d17ffc7e671b69/help-full.txt#L75)
  volume_size: 153600000 # Size of each volume (150MB)
  max_input_slices: 4000 # Maximum number of input slices
  extra_par2_options: [] # Additional PAR2 command line options
  temp_dir: "" # Optional temporary directory for PAR2 operations
```

**💡 Tip: The web UI automatically downloads and configures the PAR2 executable for you, and provides easy-to-use controls for redundancy settings.**

### NZB Compression

Configure NZB file compression:

```yaml
nzb_compression:
  enabled: false # Whether to enable compression of the output NZB file
  type: none # Compression algorithm to use (options: none, zstd, brotli)
  level: 0 # Compression level (zstd: 1-22, brotli: 0-11)
```

When compression is enabled, the generated NZB files will be compressed using the specified algorithm and will have the appropriate file extension added (`.nzb.zst` for zstd or `.nzb.br` for brotli). This can significantly reduce the size of NZB files, especially for large uploads with many segments.

#### Compression Types

- **none**: No compression (default)
- **zstd**: [Zstandard compression](https://github.com/facebook/zstd) - fast compression with good ratios
- **brotli**: [Brotli compression](https://github.com/google/brotli) - higher compression ratios but slower

#### Compression Levels

- **zstd**: 1-22 (higher = better compression but slower, default: 3)
- **brotli**: 0-11 (higher = better compression but slower, default: 4)

**💡 Tip: The web UI provides compression size estimates and helps you choose the optimal settings for your use case.**

### File Watcher

Configure automatic file watching and posting:

```yaml
watcher:
  enabled: false # Whether to enable the file watcher
  watch_directory: "" # Directory to watch for new files
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
  delete_original_file: false # Delete source files after upload
```

**💡 Tip: The web UI provides an easy way to select directories, test ignore patterns, and validate schedule configurations.**

### Queue Management

Configure the upload queue system:

```yaml
queue:
  database_type: sqlite # Database type (sqlite, postgres, mysql - only sqlite implemented currently)
  database_path: ./postie_queue.db # Database file path or connection string
  max_concurrent_uploads: 1 # Maximum concurrent uploads from queue
```

### Post Upload Script

Configure commands to run after successful uploads:

```yaml
post_upload_script:
  enabled: false # Whether to enable post upload script execution
  command: "" # Command to execute, use {{nzb_path}} placeholder for NZB file path
  timeout: 30s # Maximum time to wait for command execution
```

Example webhook notification:

```yaml
post_upload_script:
  enabled: true
  command: 'curl -X POST -H "Content-Type: application/json" -d "{\"nzb_path\": \"{{nzb_path}}\"}" https://your-webhook-url.com/notify'
  timeout: 30s
```

**💡 Tip: The web UI allows you to test your post-upload scripts and provides examples for common use cases.**

### Global Settings

Additional global configuration options:

```yaml
# Global output directory for processed files and NZB files
output_dir: "./output"

# Whether to maintain the original file extension in the NZB filename
maintain_original_extension: false
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

## Getting Started

**For new users, we strongly recommend starting with the web UI:**

1. Start Postie with minimal or no configuration
2. Open the web interface at `http://localhost:8080`
3. Use the Settings page to configure all options through the intuitive interface
4. The UI will guide you through required settings and validate your configuration
5. Once configured, you can use either the web interface or command line tools

The web UI provides immediate feedback, configuration validation, and organized sections that make setup much easier than manually editing YAML files.

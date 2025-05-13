---
sidebar_position: 3
---

# Quick Start

This guide will help you get up and running with Postie quickly.

## Prerequisites

Before you begin, make sure you have:

- Installed Postie using one of the [installation methods](./installation)
- Access to at least one Usenet server with posting permissions
- Files you want to post to Usenet

## Basic Configuration

1. Create a basic configuration file named `config.yaml`:

```yaml
servers:
  - host: news.example.com
    port: 119 # or 563 for SSL
    username: your_username
    password: your_password
    ssl: true
    max_connections: 10

posting:
  max_retries: 3
  retry_delay: 5s
  article_size_in_bytes: 750000 # ~750KB per article
  groups:
    - alt.binaries.test
  obfuscation_policy: full # Options: full, partial, none

post_check:
  enabled: true
  delay: 10s
  max_reposts: 1
```

2. Save this file in the same directory as your Postie executable or in a designated config directory if using Docker.

## Basic Usage

### Posting a Single File

To post a single file:

```bash
./postie -config config.yaml -i /path/to/your/file.mp4 -o ./output
```

### Posting a Directory

To post all files in a directory:

```bash
./postie -config config.yaml -d /path/to/directory -o ./output
```

### Watch Mode

To continuously watch a directory for new files to post:

```bash
./postie watch -config config.yaml -d /path/to/watch_dir -o ./output
```

## Verifying Posts

If you've enabled post checking in your configuration (`post_check.enabled: true`), Postie will automatically verify that your posts were successful after a specified delay.

## Next Steps

- Explore the [detailed configuration options](./configuration)
- Learn about [obfuscation policies](./obfuscation)
- Set up [automated posting with the file watcher](./watcher)

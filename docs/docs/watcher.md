---
sidebar_position: 6
---

# File Watcher

The file watcher is a powerful feature in Postie that allows you to automatically post files as they appear in a designated directory. This is particularly useful for automated workflows where files are downloaded or created and need to be posted to Usenet without manual intervention.

## How It Works

The file watcher scan in a regular bases the configured directory, when a file appears that meets the configured criteria (size, name pattern, etc.), Postie will automatically process and post it to Usenet.

## Configuration

To use the file watcher, configure the following section in your `config.yaml`:

```yaml
watcher:
  size_threshold: 104857600 # Minimum size for batch processing (100MB)
  schedule: # Optional posting schedule
    start_time: "00:00" # When to start posting (24h format)
    end_time: "23:59" # When to stop posting (24h format)
  ignore_patterns: # File patterns to ignore
    - "*.tmp"
    - "*.part"
    - "*.!ut"
  min_file_size: 1048576 # Minimum file size to process (1MB)
  check_interval: 5m # How often to check for new files
```

### Configuration Options

- **size_threshold**: Minimum accumulated size of files before batch processing
- **schedule**: Optional time window when posting is allowed
  - **start_time**: When to start posting each day (24h format)
  - **end_time**: When to stop posting each day (24h format)
- **ignore_patterns**: File patterns to ignore (uses glob syntax)
- **min_file_size**: Minimum size of individual files to process
- **check_interval**: How frequently to scan the watch directory

## Starting the Watcher

To start Postie in watch mode:

```bash
./postie watch -config config.yaml -d /path/to/watch_dir -o ./output
```

If you're using Docker, the watcher will automatically start if you've configured the watcher section in your config file.

## Behavior

1. **File Detection**: Postie scans the watch directory at the interval specified by `check_interval`
2. **File Filtering**: Files are filtered based on `min_file_size` and `ignore_patterns`
3. **Batch Processing**: Files are accumulated until they reach the `size_threshold`
4. **Schedule Checking**: If a schedule is configured, Postie will only post during the allowed time window
5. **Processing**: Files are processed according to your configuration (PAR2, obfuscation, etc.)
6. **Moving**: After successful posting, files are moved to the `output_dir`
7. **Deleting**: The original files are deleted after the nzb are created in the `output_dir`

## Use Cases

The file watcher is particularly useful for:

1. **Automated Pipelines**: Automatically post files as they are downloaded
2. **Scheduled Posting**: Only post during off-peak hours to maximize bandwidth
3. **Batch Processing**: Accumulate small files for more efficient posting

## Tips and Best Practices

1. **Set Appropriate Thresholds**: Adjust `min_file_size` and `size_threshold` based on your typical file sizes
2. **Use Ignore Patterns**: Prevent partial downloads from being processed using patterns like `*.part` or `*.incomplete`
3. **Schedule During Off-Peak Hours**: If your ISP has peak/off-peak hours, schedule posting during off-peak times
4. **Monitor Disk Space**: Ensure both watch and output directories have sufficient space
5. **Logging**: When running in watch mode, consider redirecting output to a log file for monitoring:

```bash
./postie watch -config config.yaml -d /path/to/watch_dir -o ./output > postie.log 2>&1
```

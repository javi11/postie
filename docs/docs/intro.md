---
sidebar_position: 1
---

# Introduction to Postie

Postie is a high-performance Usenet binary poster written in Go, inspired by Nyuu-Obfuscation.

## What is Postie?

Postie is designed to efficiently upload binary files to Usenet with a focus on:

- **Performance**: Optimized for high-speed posting
- **Reliability**: Automatic retry and post verification
- **Flexibility**: Multiple configuration options for different posting strategies
- **Security**: Various obfuscation policies to protect your posts

## Key Features

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

## Why Use Postie?

Postie provides a robust solution for uploading content to Usenet with features that ensure your posts are:

1. **Fast**: Optimized for high-throughput posting
2. **Reliable**: Post verification and automatic reposting of failed segments
3. **Secure**: Multiple obfuscation options to protect your content
4. **Flexible**: Extensive configuration options to meet your specific needs

## Getting Started

To get started with Postie, check out the [Installation Guide](./installation) and [Quick Start](./quick-start) sections.

For detailed configuration options, see the [Configuration Guide](./configuration).

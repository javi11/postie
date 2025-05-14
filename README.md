# Postie

<a href="https://www.buymeacoffee.com/qbt52hh7sjd"><img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=☕&slug=qbt52hh7sjd&button_colour=FFDD00&font_colour=000000&font_family=Comic&outline_colour=000000&coffee_colour=ffffff" /></a>

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

## Documentation

For detailed documentation, installation instructions, configuration options, and usage examples, please visit the [Postie Documentation Site](https://postie.nzbtools.top).

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

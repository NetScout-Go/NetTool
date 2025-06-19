# NetScout Plugin: Port Scanner

Host, Port, Range, Timeout

This is a plugin for the NetScout-Go network diagnostics tool. It provides Scans for open ports on a target host
The hostname or IP address to scan
Range of ports to scan (e.g. 1-65535 or 80,443)
Timeout in seconds for each port.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_port_scanner.git ~/.netscout/plugins/port_scanner
host
portRange
timeout
```

Or use the NetScout-Go plugin manager to install it:

```go
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_port_scanner")
```

## Features

- Network diagnostics for port_scanner
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

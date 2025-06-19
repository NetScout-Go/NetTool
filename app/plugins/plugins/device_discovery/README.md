# NetScout Plugin: Network Device Discovery
Network Range
Scan Timeout
Resolve Hostnames
Identify Device Types

This is a plugin for the NetScout-Go network diagnostics tool. It provides Discovers devices on the local network and shows IP addresses
CIDR notation of the network range to scan (e.g.
Timeout in seconds for the scan
Attempt to resolve hostnames for discovered devices
Attempt to identify device types by port scanning.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_device_discovery.git ~/.netscout/plugins/device_discovery
network_range
scan_timeout
resolve_hostnames
identify_devices
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_device_discovery")
```

## Features

- Network diagnostics for device_discovery
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

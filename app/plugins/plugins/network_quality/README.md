# NetScout Plugin: Network Quality Monitor

Target Host
Test Duration
Sampling Interval
Packet Size

This is a plugin for the NetScout-Go network diagnostics tool. It provides Continuously monitors network quality metrics including jitter
The hostname or IP address to monitor
Duration of the test in seconds
Time between samples in seconds
Size of the ICMP packets in bytes.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_network_quality.git ~/.netscout/plugins/network_quality
target
duration
interval
packet_size
```

Or use the NetScout-Go plugin manager to install it:

```go
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_network_quality")
```

## Features

- Network diagnostics for network_quality
- Easy integration with NetScout-Go

## License

MIT License

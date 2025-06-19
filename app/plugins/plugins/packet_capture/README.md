# NetScout Plugin: Network Packet Capture
Network Interface
Capture Filter
Packet Count
Capture Timeout
Include Headers

This is a plugin for the NetScout-Go network diagnostics tool. It provides Captures and analyzes network traffic using tcpdump
The network interface to capture packets from
BPF filter expression to filter packets (e.g.
Number of packets to capture (0 for unlimited)
Time in seconds to capture packets (0 for unlimited)
Include packet headers in output.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_packet_capture.git ~/.netscout/plugins/packet_capture
interface
capture_filter
packet_count
capture_timeout
include_headers
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_packet_capture")
```

## Features

- Network diagnostics for packet_capture
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

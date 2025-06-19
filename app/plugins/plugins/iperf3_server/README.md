# NetScout Plugin: iPerf3 Server
Port
Bind Address
Protocol
Server Duration
Action

This is a plugin for the NetScout-Go network diagnostics tool. It provides Start an iPerf3 server on this device to enable bandwidth testing from other devices
Port to listen on for iPerf3 connections
IP address to bind the server to (empty for all interfaces)
Protocol to use for the server
How long to run the server (minutes
Start or stop the iPerf3 server.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_iperf3_server.git ~/.netscout/plugins/iperf3_server
port
bind_address
protocol
duration
action
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_iperf3_server")
```

## Features

- Network diagnostics for iperf3_server
- Easy integration with NetScout-Go

## License

MIT License

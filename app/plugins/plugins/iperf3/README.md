# NetScout Plugin: iPerf3 Throughput Test

iPerf3 Server
Port
Test Duration
Protocol
Reverse Mode
Parallel Connections

This is a plugin for the NetScout-Go network diagnostics tool. It provides Test network throughput with iPerf3 between this device and a remote iPerf3 server
IP or hostname of the iPerf3 server
iPerf3 server port
Duration of test in seconds
Protocol to use for the test
Run in reverse mode (server sends
Number of parallel connections to use.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_iperf3.git ~/.netscout/plugins/iperf3
server
port
duration
protocol
reverse
parallel
```

Or use the NetScout-Go plugin manager to install it:

```go
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_iperf3")
```

## Features

- Network diagnostics for iperf3
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

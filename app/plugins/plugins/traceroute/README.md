# NetScout Plugin: Traceroute
Host
Max Hops

This is a plugin for the NetScout-Go network diagnostics tool. It provides Traces the route packets take to a network host
The hostname or IP address to trace
Maximum number of hops to trace.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_traceroute.git ~/.netscout/plugins/traceroute
host
maxHops
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_traceroute")
```

## Features

- Network diagnostics for traceroute
- Easy integration with NetScout-Go

## License

MIT License

# NetScout Plugin: Ping
Host
Count

This is a plugin for the NetScout-Go network diagnostics tool. It provides Tests connectivity to a host by sending ICMP echo requests
The hostname or IP address to ping
Number of packets to send.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_ping.git ~/.netscout/plugins/ping
host
count
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_ping")
```

## Features

- Network diagnostics for ping
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

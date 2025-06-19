# NetScout Plugin: Wi-Fi Scanner
Wi-Fi Interface
Scan Time
Show Hidden Networks

This is a plugin for the NetScout-Go network diagnostics tool. It provides Scans for nearby Wi-Fi networks and displays signal strength
The wireless interface to use for scanning
Time in seconds to scan for networks
Include networks that don't broadcast their SSID.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_wifi_scanner.git ~/.netscout/plugins/wifi_scanner
interface
scan_time
show_hidden
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_wifi_scanner")
```

## Features

- Network diagnostics for wifi_scanner
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

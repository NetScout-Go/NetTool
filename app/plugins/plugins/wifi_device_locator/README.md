# NetScout Plugin: Wi-Fi Device Locator
Wi-Fi Interface
Scan Time
Scan Mode
Minimum Signal Strength
Continuous Scan

This is a plugin for the NetScout-Go network diagnostics tool. It provides Helps locate nearby Wi-Fi devices (access points and clients) by displaying their signal strength and other information
The wireless interface to use for scanning
Time in seconds to scan for devices
Type of devices to scan for
Minimum signal strength to display (dBm)
Continuously update signal strength (useful for locating devices).

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_wifi_device_locator.git ~/.netscout/plugins/wifi_device_locator
interface
scan_time
scan_mode
min_signal_strength
continuous_scan
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_wifi_device_locator")
```

## Features

- Network diagnostics for wifi_device_locator
- Easy integration with NetScout-Go

## License

MIT License

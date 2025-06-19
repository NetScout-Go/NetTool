# NetScout Plugin: MTU Size Tester
Target Host
Minimum MTU
Maximum MTU
Step Size

This is a plugin for the NetScout-Go network diagnostics tool. It provides Determines the optimal MTU size for a connection to help diagnose packet fragmentation issues
The hostname or IP address to test MTU size for
Minimum MTU size to test
Maximum MTU size to test
Step size for MTU testing.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_mtu_tester.git ~/.netscout/plugins/mtu_tester
target
min_mtu
max_mtu
step_size
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_mtu_tester")
```

## Features

- Network diagnostics for mtu_tester
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

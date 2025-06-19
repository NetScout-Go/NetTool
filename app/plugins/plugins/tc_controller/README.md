# NetScout Plugin: Traffic Control (QoS)
Network Interface
Mode
Bandwidth Limit (Kbps)
Latency (ms)
Packet Loss (%)
Jitter (ms)
Duration (seconds)

This is a plugin for the NetScout-Go network diagnostics tool. It provides Simulate network conditions and test Quality of Service using Linux Traffic Control (tc)
Network interface to apply traffic control rules
Traffic control mode
Maximum bandwidth in Kbps (1000 = 1Mbps)
Added latency in milliseconds
Percentage of packets to drop
Latency variation in milliseconds
Duration to apply traffic control rules (0 for indefinite).

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_tc_controller.git ~/.netscout/plugins/tc_controller
interface
mode
bandwidth
latency
packet_loss
jitter
duration
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_tc_controller")
```

## Features

- Network diagnostics for tc_controller
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

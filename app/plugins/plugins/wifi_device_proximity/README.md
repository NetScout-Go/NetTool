# WiFi Device Proximity Locator Plugin

This plugin helps you locate nearby WiFi devices (access points and clients) by displaying their signal strength, proximity level, and estimated distance. It's especially useful for finding specific devices or assessing the proximity of WiFi devices in your area.

## Features

- **Real-time signal strength tracking**: Monitor WiFi signal strength in real time
- **Proximity indicators**: Visual indicators showing how close devices are (very close, close, medium, far, very far)
- **Device targeting**: Focus on a specific device to track its proximity
- **Manufacturer identification**: Identify device manufacturers through OUI database lookup
- **Distance estimation**: Approximate distance calculation (in meters) based on signal strength
- **Signal trend analysis**: Track whether a device is moving closer or farther away
- **Proximity alerts**: Receive alerts when devices come within a specified signal threshold

## Requirements

- Linux system with WiFi interface that supports monitor mode
- Installed packages: `aircrack-ng`, `tcpdump`, `iw`

## Parameters

- **Wi-Fi Interface**: The wireless interface to use for scanning (default: wlan0)
- **Scan Time**: Time in seconds to scan for devices (default: 10)
- **Scan Mode**: Type of devices to scan for (All Devices, Access Points Only, Client Devices Only)
- **Minimum Signal Strength**: Minimum signal strength to display in dBm (default: -80)
- **Target Device**: Optional MAC address of a specific device to track
- **Continuous Scan**: Continuously scan and update signal strength readings
- **Proximity Alerts**: Enable alerts when a device comes within close proximity
- **Proximity Threshold**: Signal strength threshold to trigger proximity alert

## Usage Tips

1. **Finding a specific device**: Enter the MAC address in the "Target Device" field
2. **Locating an access point**: Use "Access Points Only" scan mode and move around to find the strongest signal
3. **Tracking signal changes**: Enable "Continuous Scan" to see signal trends as you move
4. **Finding hidden devices**: Set "Minimum Signal Strength" to a lower value to detect weaker signals

## Interpreting Results

- **Signal Strength**: Measured in dBm, typically ranges from -30 (very strong) to -90 (very weak)
- **Signal Quality**: Percentage value (0-100%) indicating connection quality
- **Proximity Level**: Categorization of distance (very_close, close, medium, far, very_far)
- **Distance Estimate**: Approximate distance in meters (most accurate in open spaces)
- **Signal Trend**: Shows if signal is increasing (getting closer), decreasing (moving away), or stable

## Notes

- Distance estimates are approximate and work best in open spaces without obstructions
- Signal strength can be affected by walls, furniture, and other obstacles
- For best results, hold the device at a consistent height and orientation while scanning

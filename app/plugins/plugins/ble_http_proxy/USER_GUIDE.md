# NetTool BLE HTTP Proxy User Guide

## Overview

The BLE HTTP Proxy feature allows you to connect to your NetTool device using Bluetooth Low Energy (BLE) instead of WiFi or Ethernet. This is particularly useful when:

- You're in a location without WiFi access
- You can't connect to the network where NetTool is connected
- You need to keep your mobile device on its current network

## Getting Started

### Setup on NetTool Device (One-time setup)

1. Install required dependencies on your Pi Zero 2 W:

   ```bash
   sudo apt-get update
   sudo apt-get install -y bluez bluez-tools python3-pip
   sudo pip3 install dbus-next requests
   ```

2. Give your user proper permissions:

   ```bash
   sudo usermod -a -G bluetooth $USER
   ```

3. Reboot the device:

   ```bash
   sudo reboot
   ```

### On NetTool Device

1. Make sure your NetTool device (Pi Zero 2 W) has Bluetooth enabled
2. Start the BLE HTTP Proxy plugin from the dashboard or command line
3. The device will begin advertising as "NetTool" over Bluetooth

### On Your Mobile Device

1. Install a BLE HTTP Proxy client app
   - For Android: [BLE HTTP Proxy](https://play.google.com/store/apps/details?id=com.example.blehttpproxy)
   - For iOS: [NetTool Connect](https://apps.apple.com/app/nettool-connect/id123456789)
   - For desktop: Use the included Go client in the `client` directory

2. Enable Bluetooth on your device and scan for "NetTool"

3. Connect to the device through the app

4. Access the NetTool dashboard via the local URL provided by the app (typically `http://localhost:8000/`)

## Features

- **Full Dashboard Access**: All NetTool features are available through the BLE connection
- **Secure Connection**: BLE provides a direct, secure link between devices
- **Battery Efficient**: BLE is designed for low power consumption
- **Auto-Reconnect**: Client apps can automatically reconnect when in range

## Troubleshooting

### Device Not Found

- Make sure Bluetooth is enabled on both devices
- Verify the BLE HTTP Proxy plugin is running on NetTool
- Restart Bluetooth on your mobile device
- Move closer to the NetTool device (BLE range is typically 10-30 meters)

### Slow Performance

- BLE has limited bandwidth compared to WiFi
- Large operations like packet captures may be slower
- Use the dashboard's "lightweight mode" for faster performance

### Connection Drops

- Stay within range of the NetTool device
- Keep your mobile device's Bluetooth enabled
- Update to the latest version of the client app
- Restart the BLE HTTP Proxy service if problems persist

## Technical Specifications

- **BLE Version**: 5.0
- **Max Theoretical Throughput**: ~1.4 Mbps (with optimal conditions)
- **Typical Range**: 10-30 meters (depends on environment)
- **Service UUID**: 00001234-0000-1000-8000-00805f9b34fb
- **Default MTU Size**: 23 bytes (negotiable up to 512 bytes on supported devices)
- **Auto-reconnect**: Supported
- **Status Monitoring**: CPU, Memory, Network latency, Connection statistics

## Advanced Features

The BLE HTTP Proxy includes several advanced features to improve performance and reliability:

### MTU Negotiation

The proxy automatically negotiates the optimal MTU (Maximum Transmission Unit) size with connected devices, which can significantly improve performance on newer Bluetooth 5.0+ devices that support larger MTU sizes.

### Chunked Transfers

Large HTTP responses are automatically chunked to work efficiently over BLE, with each chunk containing:

- A 16-byte UUID to match request/response pairs
- A 1-byte flags field (bit 0 = end marker)
- The actual HTTP data payload

### Status Monitoring

The proxy provides real-time status information through a dedicated characteristic, including:

- Connected client count
- Data transfer statistics
- CPU and memory usage
- Network latency to gateway
- Service uptime
- Current MTU size

## Test Client Tools

NetTool includes Python-based client tools for testing the BLE HTTP Proxy functionality:

### Python Test Client

If you're using Linux or another Pi, you can use the included Python test client:

1. Install the required dependencies:

   ```bash
   sudo apt-get install python3-pip libglib2.0-dev
   sudo pip3 install bluepy
   ```

2. Scan for NetTool devices:

   ```bash
   python3 client/test_ble_client.py --scan
   ```

3. Check the status of a specific device:

   ```bash
   python3 client/test_ble_client.py --status XX:XX:XX:XX:XX:XX
   ```

4. Send a GET request to the dashboard:

   ```bash
   python3 client/test_ble_client.py --get XX:XX:XX:XX:XX:XX --path /
   ```

Replace `XX:XX:XX:XX:XX:XX` with the Bluetooth MAC address of your NetTool device.

## Battery Impact

Using the BLE HTTP Proxy will increase power consumption on the NetTool device. When running on battery power:

- Expected runtime: ~10 hours with occasional use
- ~8 hours with continuous BLE connection
- ~6 hours with heavy data transfer

For extended operations, consider connecting external power.

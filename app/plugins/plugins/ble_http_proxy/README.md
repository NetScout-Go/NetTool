# BLE HTTP Proxy Plugin for NetTool

This plugin enables a Bluetooth Low Energy (BLE) GATT server that allows mobile phones and PCs to connect to the NetTool dashboard over Bluetooth.

## Features

- **BLE GATT Server**: Implements a custom HTTP proxy service over BLE
- **Automatic MTU Negotiation**: Optimizes data transfer speed
- **Chunked Transfers**: Handles large HTTP requests and responses
- **Notification Support**: Alerts clients when responses are ready
- **Easy Connection**: Simple service discovery and connection process
- **Exposes NetTool's web interface**: Provides full dashboard access over BLE
- **No Wi-Fi Required**: Works when traditional networking is unavailable

## Requirements

When deploying to a Raspberry Pi Zero 2 W:

1. BlueZ 5.50 or higher
2. `bluez-tools` package installed
3. Bluetooth service enabled and running

## Installation

1. Install the required dependencies:
```bash
sudo apt-get update
sudo apt-get install bluez bluez-tools
```

2. Enable and start the Bluetooth service:
```bash
sudo systemctl enable bluetooth
sudo systemctl start bluetooth
```

3. Ensure the plugin is available in NetTool's plugin directory.

## Configuration

In the NetTool dashboard, you can configure the following parameters:

- **Device Name**: The Bluetooth device name that will be advertised (default: NetTool)
- **HTTP Port**: The local HTTP port to proxy (default: 8080)
- **Action**: The action to perform (start, stop, status)

## Usage with Mobile Devices

### Android

1. Install a BLE HTTP Proxy client app from the Google Play Store
2. Scan for the NetTool device
3. Connect to the device
4. Open the web browser in the app to access the NetTool dashboard

### iOS

1. Most modern iOS HTTP proxy apps support BLE connections
2. Scan for and connect to the NetTool device
3. Access the dashboard through the proxy app's browser

## Implementation Notes

This plugin uses the `go-bluetooth` library to create a GATT server with the following:

- Custom HTTP Proxy Service UUID: `00001234-0000-1000-8000-00805f9b34fb`
- HTTP Request Characteristic: `00001235-0000-1000-8000-00805f9b34fb`
- HTTP Response Characteristic: `00001236-0000-1000-8000-00805f9b34fb`

The implementation follows a client-server model where:
1. The client sends HTTP requests via the Request characteristic
2. The server processes the request and forwards it to the local HTTP server
3. The response is made available via the Response characteristic

## Troubleshooting

If you encounter issues:

1. Check if the Bluetooth service is running: `sudo systemctl status bluetooth`
2. Verify the plugin status in NetTool
3. Check BlueZ logs: `sudo journalctl -u bluetooth`
4. Try restarting the Bluetooth service: `sudo systemctl restart bluetooth`

## License

This plugin is licensed under the GNU General Public License v3.0.

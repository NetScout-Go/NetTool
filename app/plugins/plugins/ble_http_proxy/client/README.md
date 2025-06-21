# NetTool BLE HTTP Proxy Client

This directory contains client tools for interacting with the NetTool BLE HTTP Proxy service running on a Raspberry Pi Zero 2 W.

## Python Test Client

The `test_ble_client.py` script provides a command-line tool for testing the BLE HTTP Proxy functionality.

### Dependencies

Install the required dependencies:

```bash
pip install bluepy
```

Note: `bluepy` requires Linux and may need additional system packages. Install with:

```bash
sudo apt-get install python3-pip libglib2.0-dev
```

### Usage

The client provides several commands for interacting with the BLE HTTP Proxy:

1. **Scanning for devices**:

   ```bash
   python3 test_ble_client.py --scan
   ```

2. **Testing connection**:

   ```bash
   python3 test_ble_client.py --connect XX:XX:XX:XX:XX:XX
   ```

3. **Getting status information**:

   ```bash
   python3 test_ble_client.py --status XX:XX:XX:XX:XX:XX
   ```

4. **Sending HTTP GET request**:

   ```bash
   python3 test_ble_client.py --get XX:XX:XX:XX:XX:XX --path /
   python3 test_ble_client.py --get XX:XX:XX:XX:XX:XX --path /api/networkinfo
   ```

Replace `XX:XX:XX:XX:XX:XX` with the MAC address of your Raspberry Pi Zero.

## Go Client

The `ble_http_client.go` file contains a Go implementation of a BLE HTTP client. This is still under development and may require additional dependencies.

## Troubleshooting

If you encounter issues:

1. Make sure Bluetooth is enabled on your system:

   ```bash
   sudo systemctl status bluetooth
   sudo hciconfig hci0 up
   ```

2. Check that you have permission to use Bluetooth:

   ```bash
   sudo setcap 'cap_net_raw,cap_net_admin+eip' $(which python3)
   ```

3. Try running the client with elevated privileges:

   ```bash
   sudo python3 test_ble_client.py --scan
   ```

4. Verify the Raspberry Pi is correctly advertising the BLE service:

   ```bash
   sudo bluetoothctl
   scan on
   # Look for a device with "NetTool" in the name
   ```

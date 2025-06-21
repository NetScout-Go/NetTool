#!/bin/bash

# Script to test BLE HTTP Proxy functionality
# This script simulates what would happen on a real Pi Zero with Bluetooth

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}BLE HTTP Proxy Test Script${NC}"
echo -e "${YELLOW}This script simulates BLE HTTP Proxy functionality for testing.${NC}"
echo

# Check for bluetoothctl
if ! command -v bluetoothctl &> /dev/null; then
    echo -e "${RED}bluetoothctl command not found.${NC}"
    echo -e "${YELLOW}This is a simulation script, but we still need bluetoothctl for proper testing.${NC}"
    echo -e "${YELLOW}Try: sudo apt-get install bluez bluez-tools${NC}"
    exit 1
fi

# Check if BlueZ DBus service is running
if ! busctl --system list | grep -q org.bluez; then
    echo -e "${RED}BlueZ DBus service not found.${NC}"
    echo -e "${YELLOW}Make sure the Bluetooth service is running: sudo systemctl start bluetooth${NC}"
    exit 1
fi

echo -e "${BLUE}BlueZ service detected. Bluetooth should be working.${NC}"

# Check Python dependencies
PYTHON_DEPS=("dbus-next" "requests")
MISSING_DEPS=()

for dep in "${PYTHON_DEPS[@]}"; do
    if ! python3 -c "import $dep" &>/dev/null; then
        MISSING_DEPS+=("$dep")
    fi
done

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    echo -e "${YELLOW}Missing Python dependencies: ${MISSING_DEPS[*]}${NC}"
    echo -e "${YELLOW}Install them with: pip3 install ${MISSING_DEPS[*]}${NC}"
    read -p "Do you want to install them now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        pip3 install "${MISSING_DEPS[@]}"
    else
        echo -e "${YELLOW}The BLE service may not work correctly without these dependencies.${NC}"
    fi
fi

# Create a simple HTTP server for testing
TEMP_DIR="/tmp/nettool_ble_test"
mkdir -p "$TEMP_DIR"

cat > "$TEMP_DIR/index.html" << 'EOL'
<!DOCTYPE html>
<html>
<head>
    <title>NetTool BLE Test</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 40px;
            line-height: 1.6;
        }
        h1 {
            color: #2c3e50;
        }
        .success {
            color: #27ae60;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <h1>NetTool BLE HTTP Proxy Test</h1>
    <p class="success">SUCCESS! If you're seeing this page, the BLE HTTP Proxy is working correctly.</p>
    <p>This test page is being served through the BLE HTTP Proxy Service.</p>
    <p>Now you can access the full NetTool dashboard through this connection.</p>
</body>
</html>
EOL

echo -e "${BLUE}Starting test HTTP server on port 8088...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the test when finished.${NC}"
echo

# Check if Python3 is installed
if command -v python3 &> /dev/null; then
    (cd "$TEMP_DIR" && python3 -m http.server 8088) &
    SERVER_PID=$!
else
    echo -e "${RED}Python3 not found. Cannot start test server.${NC}"
    exit 1
fi

# Launch the BLE service
echo -e "${BLUE}Starting BLE HTTP Proxy service...${NC}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
python3 "$SCRIPT_DIR/pi_zero_ble_service.py" --device-name "NetTool-Test" --port 8088 &
BLE_PID=$!

# Setup BLE advertising
echo -e "${BLUE}Setting up BLE advertising...${NC}"
echo -e "${GREEN}Advertising NetTool BLE HTTP Proxy service on Bluetooth LE${NC}"
echo -e "${GREEN}Service UUID: 00001234-0000-1000-8000-00805f9b34fb${NC}"

# Get Bluetooth adapter info
echo -e "${BLUE}Bluetooth adapter information:${NC}"
ADAPTER_INFO=$(bluetoothctl show)
echo "$ADAPTER_INFO"

# Get the adapter address
ADAPTER_ADDR=$(echo "$ADAPTER_INFO" | grep "Controller" | awk '{print $2}')

echo
echo -e "${GREEN}Bluetooth device address: ${ADAPTER_ADDR}${NC}"
echo -e "${GREEN}BLE HTTP Proxy service is ready!${NC}"
echo
echo -e "${YELLOW}On your mobile device:${NC}"
echo -e "1. Install a BLE HTTP Proxy client app"
echo -e "2. Scan for device with name 'NetTool-Test'"
echo -e "3. Connect to the device"
echo -e "4. Try accessing http://localhost/ through the proxy"
echo
echo -e "${YELLOW}Press Ctrl+C to stop the test server.${NC}"

# Wait for Ctrl+C
trap "kill $SERVER_PID $BLE_PID; rm -rf $TEMP_DIR; echo -e '\n${GREEN}Test server stopped.${NC}'" EXIT

# Keep the script running
while true; do
    sleep 1
done

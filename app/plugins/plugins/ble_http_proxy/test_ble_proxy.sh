#!/bin/bash

# Script to test BLE HTTP Proxy functionality
# This script tests the BLE HTTP Proxy service on a Raspberry Pi

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}BLE HTTP Proxy Test Script${NC}"
echo -e "${YELLOW}This script tests the BLE HTTP Proxy service on a Raspberry Pi.${NC}"
echo

# Check for dependencies using our Python script
echo -e "${BLUE}Checking dependencies...${NC}"
if [ -f "${SCRIPT_DIR}/check_dependencies.py" ]; then
    python3 "${SCRIPT_DIR}/check_dependencies.py"
    if [ $? -ne 0 ]; then
        echo -e "${YELLOW}Dependency check found issues. You may need to install missing packages.${NC}"
        echo -e "${YELLOW}Run setup_pi_zero.sh to install all required dependencies.${NC}"
    fi
else
    # Fall back to basic checks if the dependency checker isn't available
    # Check for bluetoothctl
    if ! command -v bluetoothctl &> /dev/null; then
        echo -e "${RED}bluetoothctl command not found.${NC}"
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

# Run using the debug wrapper for better error reporting
if [ -f "${SCRIPT_DIR}/debug_ble_service.sh" ]; then
    bash "${SCRIPT_DIR}/debug_ble_service.sh" --device-name "NetTool-Test" --port 8088 &
    BLE_PID=$!
else
    # Fall back to direct execution if debug wrapper isn't available
    python3 "$SCRIPT_DIR/pi_zero_ble_service.py" --device-name "NetTool-Test" --port 8088 &
    BLE_PID=$!
fi

# Setup BLE advertising
echo -e "${BLUE}BLE service started with PID: ${BLE_PID}${NC}"
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
echo -e "${YELLOW}Testing Options:${NC}"
echo -e "1. Use a BLE client app on your mobile device"
echo -e "   - Scan for device with name 'NetTool-Test'"
echo -e "   - Connect to the device"
echo -e "   - Try accessing http://localhost/ through the proxy"
echo
echo -e "2. Use the test client script in the client directory:"
echo -e "   - Run: python3 ${SCRIPT_DIR}/client/test_ble_client.py --scan"
echo -e "   - Run: python3 ${SCRIPT_DIR}/client/test_ble_client.py --get XX:XX:XX:XX:XX:XX --path /"
echo
echo -e "${YELLOW}Press Ctrl+C to stop the test server.${NC}"

# Function to clean up on script exit
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    
    # Kill the BLE service
    if [ ! -z "$BLE_PID" ]; then
        echo -e "Stopping BLE service (PID: $BLE_PID)..."
        kill $BLE_PID 2>/dev/null || true
    fi
    
    # Kill the HTTP server
    if [ ! -z "$SERVER_PID" ]; then
        echo -e "Stopping HTTP server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
    fi
    
    # Remove temporary files
    if [ -d "$TEMP_DIR" ]; then
        echo -e "Removing temporary files..."
        rm -rf "$TEMP_DIR"
    fi
    
    echo -e "${GREEN}Done!${NC}"
}

# Register cleanup function to run on script exit
trap cleanup EXIT INT TERM

# Wait for Ctrl+C
trap "kill $SERVER_PID $BLE_PID; rm -rf $TEMP_DIR; echo -e '\n${GREEN}Test server stopped.${NC}'" EXIT

# Keep the script running
while true; do
    sleep 1
done

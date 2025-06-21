#!/bin/bash

# Script to install BLE HTTP Proxy prerequisites on a Raspberry Pi Zero 2 W
# Run this script on the actual Pi Zero hardware, not in the development environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}BLE HTTP Proxy Setup for NetTool on Raspberry Pi Zero 2 W${NC}"
echo -e "${YELLOW}This script will install prerequisites for the BLE HTTP Proxy plugin.${NC}"
echo -e "${YELLOW}Make sure you are running this on the Pi Zero hardware.${NC}"
echo

# Check if running on a Raspberry Pi
if ! grep -q "Raspberry Pi" /proc/device-tree/model 2>/dev/null; then
    echo -e "${RED}This script should be run on a Raspberry Pi.${NC}"
    echo -e "${YELLOW}You can ignore this warning if you're deliberately running it elsewhere.${NC}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root (use sudo).${NC}"
    exit 1
fi

echo -e "${BLUE}Updating package lists...${NC}"
apt-get update

echo -e "${BLUE}Installing Bluetooth dependencies...${NC}"
apt-get install -y bluez bluez-tools libbluetooth-dev

echo -e "${BLUE}Installing Go dependencies...${NC}"
apt-get install -y build-essential

# Enable Bluetooth service
echo -e "${BLUE}Enabling Bluetooth service...${NC}"
systemctl enable bluetooth
systemctl start bluetooth

# Configure D-Bus permissions for Bluetooth
echo -e "${BLUE}Configuring D-Bus permissions...${NC}"
cat > /etc/dbus-1/system.d/nettool-bluetooth.conf << 'EOL'
<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <policy user="pi">
    <allow own="org.bluez"/>
    <allow send_destination="org.bluez"/>
    <allow send_interface="org.bluez.GattCharacteristic1"/>
    <allow send_interface="org.bluez.GattDescriptor1"/>
    <allow send_interface="org.bluez.LEAdvertisement1"/>
    <allow send_interface="org.freedesktop.DBus.ObjectManager"/>
    <allow send_interface="org.freedesktop.DBus.Properties"/>
  </policy>
</busconfig>
EOL

# Restart D-Bus for permissions to take effect
echo -e "${BLUE}Restarting D-Bus service...${NC}"
systemctl restart dbus

# Add user to bluetooth group
if getent passwd pi > /dev/null; then
    echo -e "${BLUE}Adding pi user to bluetooth group...${NC}"
    usermod -a -G bluetooth pi
else
    echo -e "${YELLOW}User 'pi' not found. Please add your user to the bluetooth group manually:${NC}"
    echo -e "${YELLOW}  sudo usermod -a -G bluetooth YOUR_USERNAME${NC}"
fi

echo -e "${GREEN}Installation completed successfully!${NC}"
echo -e "${YELLOW}Notes:${NC}"
echo -e "1. You may need to reboot for all changes to take effect."
echo -e "2. To test BLE functionality, run: 'sudo bluetoothctl'"
echo -e "3. Within bluetoothctl, try: 'show', 'power on', 'scan on'"
echo
echo -e "${GREEN}The BLE HTTP Proxy plugin should now work properly on your Pi Zero 2 W.${NC}"

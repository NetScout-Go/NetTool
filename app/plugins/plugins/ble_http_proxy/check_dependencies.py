#!/usr/bin/env python3
"""
NetTool BLE HTTP Proxy Dependency Checker

This script checks if all required Python dependencies for the BLE HTTP Proxy
are installed on the Raspberry Pi Zero 2 W.
"""

import sys
import importlib
import subprocess
import platform

# Required Python packages
REQUIRED_PACKAGES = {
    "dbus-next": "dbus_next",
    "requests": "requests",
    "json": "json",
    "array": "array",
    "uuid": "uuid",
    "time": "time",
    "threading": "threading",
    "logging": "logging",
    "socket": "socket",
    "subprocess": "subprocess",
    "psutil": "psutil"
}

# System requirements
SYSTEM_REQUIREMENTS = [
    ("bluez", "bluetoothd --version", "BlueZ is required for Bluetooth functionality"),
    ("d-bus", "dbus-daemon --version", "D-Bus is required for BLE GATT service")
]

def check_python_version():
    """Check if Python version is at least 3.7"""
    version = sys.version_info
    if version.major < 3 or (version.major == 3 and version.minor < 7):
        print(f"Python version {version.major}.{version.minor} is not supported.")
        print("Please use Python 3.7 or newer.")
        return False
    else:
        print(f"Python version: {version.major}.{version.minor}.{version.micro} ✓")
        return True

def check_packages():
    """Check if required packages are installed"""
    missing_packages = []
    
    for package_name, module_name in REQUIRED_PACKAGES.items():
        try:
            importlib.import_module(module_name)
            print(f"Package {package_name}: ✓")
        except ImportError:
            print(f"Package {package_name}: ✗ (not installed)")
            missing_packages.append(package_name)
    
    return missing_packages

def check_system_requirements():
    """Check if system requirements are met"""
    missing_requirements = []
    
    for name, command, message in SYSTEM_REQUIREMENTS:
        try:
            subprocess.check_output(command.split(), stderr=subprocess.STDOUT)
            print(f"System requirement {name}: ✓")
        except (subprocess.CalledProcessError, FileNotFoundError):
            print(f"System requirement {name}: ✗ (not installed)")
            missing_requirements.append((name, message))
    
    return missing_requirements

def check_bluetooth_permissions():
    """Check if user has permissions to use Bluetooth"""
    try:
        # Try to open a Bluetooth socket
        import socket
        sock = socket.socket(socket.AF_BLUETOOTH, socket.SOCK_RAW, socket.BTPROTO_HCI)
        sock.close()
        print("Bluetooth permissions: ✓")
        return True
    except PermissionError:
        print("Bluetooth permissions: ✗ (insufficient permissions)")
        return False
    except ImportError:
        print("Bluetooth permissions: ? (could not check, socket module not available)")
        return False
    except Exception as e:
        print(f"Bluetooth permissions: ? (unknown error: {e})")
        return False

def check_system_info():
    """Check system information"""
    # Check operating system
    print(f"Operating system: {platform.system()} {platform.release()}")
    
    # Check CPU architecture
    print(f"CPU architecture: {platform.machine()}")
    
    # Check if running on a Raspberry Pi
    try:
        with open('/proc/device-tree/model', 'r') as f:
            model = f.read().strip()
            print(f"Device model: {model}")
            if "Raspberry Pi Zero 2" in model:
                print("Running on Raspberry Pi Zero 2 W: ✓")
            else:
                print("Warning: Not running on Raspberry Pi Zero 2 W")
    except FileNotFoundError:
        print("Warning: Could not determine if running on Raspberry Pi")

def main():
    """Main function"""
    print("NetTool BLE HTTP Proxy Dependency Checker")
    print("=========================================")
    print()
    
    print("System Information:")
    print("-----------------")
    check_system_info()
    print()
    
    print("Python Version:")
    print("--------------")
    python_ok = check_python_version()
    print()
    
    print("Required Packages:")
    print("-----------------")
    missing_packages = check_packages()
    print()
    
    print("System Requirements:")
    print("------------------")
    missing_requirements = check_system_requirements()
    print()
    
    print("Bluetooth Permissions:")
    print("--------------------")
    bt_permissions = check_bluetooth_permissions()
    print()
    
    # Summary
    print("Summary:")
    print("-------")
    if not python_ok:
        print("✗ Python version is not supported")
    
    if missing_packages:
        print("✗ Missing Python packages:")
        for package in missing_packages:
            print(f"  - {package}")
        print("\nInstall with:")
        print(f"  pip3 install {' '.join(missing_packages)}")
    else:
        print("✓ All required Python packages are installed")
    
    if missing_requirements:
        print("✗ Missing system requirements:")
        for name, message in missing_requirements:
            print(f"  - {name}: {message}")
    else:
        print("✓ All system requirements are met")
    
    if not bt_permissions:
        print("✗ Insufficient Bluetooth permissions")
        print("  Run the following command:")
        print("  sudo setcap 'cap_net_raw,cap_net_admin+eip' $(which python3)")
        print("  or run the script with sudo")
    else:
        print("✓ Bluetooth permissions are set correctly")

if __name__ == "__main__":
    main()

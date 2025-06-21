#!/usr/bin/env python3
"""
BLE Diagnostic Test Script for NetTool

This script runs diagnostic tests on the Raspberry Pi's Bluetooth subsystem
to verify that everything is working correctly. It's designed to be run
on startup to help troubleshoot any BLE issues.

Tests performed:
1. Check if BlueZ is installed and running
2. Verify the Bluetooth adapter is present and can be powered on
3. Attempt to discover nearby devices
4. Check if BLE advertisements can be created
5. Verify D-Bus permissions are set correctly
"""

import os
import sys
import subprocess
import time
import argparse
import logging
import json

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('nettool-ble-diagnostics')

def check_command_exists(command):
    """Check if a command exists in the system path"""
    try:
        subprocess.check_output(["which", command], stderr=subprocess.STDOUT)
        return True
    except subprocess.CalledProcessError:
        return False

def check_service_running(service):
    """Check if a systemd service is running"""
    try:
        output = subprocess.check_output(["systemctl", "is-active", service], stderr=subprocess.STDOUT)
        return output.decode('utf-8').strip() == "active"
    except subprocess.CalledProcessError:
        return False

def check_dbus_permissions():
    """Check if proper D-Bus permissions are set for BlueZ"""
    try:
        # Check if dbus configuration file exists
        dbus_conf = "/etc/dbus-1/system.d/nettool-bluetooth.conf"
        if not os.path.exists(dbus_conf):
            return False, f"D-Bus configuration file not found: {dbus_conf}"
        
        # Check if current user can access BlueZ over D-Bus
        current_user = os.getenv('USER', os.getenv('LOGNAME', 'unknown'))
        cmd = ["busctl", "--system", "call", "org.bluez", "/", "org.freedesktop.DBus.Introspectable", "Introspect"]
        subprocess.check_output(cmd, stderr=subprocess.STDOUT)
        
        return True, f"User {current_user} has proper D-Bus permissions"
    except subprocess.CalledProcessError as e:
        return False, f"D-Bus permission check failed: {e.output.decode('utf-8')}"

def check_bluetooth_adapter():
    """Check if Bluetooth adapter is present and operational"""
    try:
        output = subprocess.check_output(["hciconfig"], stderr=subprocess.STDOUT)
        output_str = output.decode('utf-8')
        
        if "hci0" not in output_str:
            return False, "No Bluetooth adapter found"
        
        # Check if adapter can be powered on
        subprocess.check_output(["hciconfig", "hci0", "up"], stderr=subprocess.STDOUT)
        
        # Get adapter info
        adapter_info = subprocess.check_output(["hciconfig", "hci0"], stderr=subprocess.STDOUT).decode('utf-8')
        
        # Extract address and other information
        address = None
        for line in adapter_info.splitlines():
            if "BD Address:" in line:
                address = line.split("BD Address:")[1].strip().split()[0]
        
        return True, f"Bluetooth adapter found and operational (Address: {address})"
    except subprocess.CalledProcessError as e:
        return False, f"Bluetooth adapter check failed: {e.output.decode('utf-8')}"

def test_ble_advertisement():
    """Test if BLE advertisements can be created"""
    try:
        # Create a temporary Python script for advertisement test
        test_script = "/tmp/ble_adv_test.py"
        with open(test_script, "w") as f:
            f.write("""
import sys
import time
import dbus
from dbus.mainloop.glib import DBusGMainLoop
from gi.repository import GLib

def register_ad_cb():
    print("Advertisement registered")

def register_ad_error_cb(error):
    print(f"Failed to register advertisement: {error}")
    sys.exit(1)

def main():
    DBusGMainLoop(set_as_default=True)
    bus = dbus.SystemBus()
    
    # Get adapter interface
    adapter_path = "/org/bluez/hci0"
    adapter = dbus.Interface(bus.get_object("org.bluez", adapter_path),
                           "org.bluez.Adapter1")
    
    # Power on the adapter
    adapter_props = dbus.Interface(bus.get_object("org.bluez", adapter_path),
                                "org.freedesktop.DBus.Properties")
    adapter_props.Set("org.bluez.Adapter1", "Powered", dbus.Boolean(1))
    
    # Get advertisement manager interface
    adv_manager = dbus.Interface(bus.get_object("org.bluez", adapter_path),
                               "org.bluez.LEAdvertisingManager1")
    
    # Create a test advertisement
    service_uuids = ["1234"]
    manufacturer_data = {0x004C: [0x02, 0x15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]}
    
    # Create advertisement properties
    adv_props = {
        "Type": "peripheral",
        "ServiceUUIDs": dbus.Array(service_uuids, signature='s'),
        "ManufacturerData": dbus.Dictionary(manufacturer_data, signature='qv')
    }
    
    adv_path = "/org/bluez/test/advertisement0"
    
    # Create advertisement object
    adv_obj = bus.get_object("org.bluez", adv_path)
    
    # Register advertisement
    adv_manager.RegisterAdvertisement(adv_path, {}, 
                                   reply_handler=register_ad_cb,
                                   error_handler=register_ad_error_cb)
    
    # Run for 2 seconds
    mainloop = GLib.MainLoop()
    GLib.timeout_add(2000, mainloop.quit)
    mainloop.run()
    
    print("Advertisement test completed successfully")
    return 0

if __name__ == "__main__":
    sys.exit(main())
""")
        
        # Run the test script
        try:
            output = subprocess.check_output(["python3", test_script], stderr=subprocess.STDOUT, timeout=5)
            output_str = output.decode('utf-8')
            
            if "Advertisement test completed successfully" in output_str:
                return True, "BLE advertisement test passed"
            else:
                return False, f"BLE advertisement test failed: {output_str}"
        except subprocess.TimeoutExpired:
            return False, "BLE advertisement test timed out"
    except Exception as e:
        return False, f"BLE advertisement test failed: {str(e)}"
    finally:
        # Clean up
        if os.path.exists(test_script):
            os.remove(test_script)

def scan_for_devices():
    """Scan for nearby Bluetooth devices"""
    try:
        # Turn on the adapter
        subprocess.check_output(["hciconfig", "hci0", "up"], stderr=subprocess.STDOUT)
        
        # Scan for devices
        logger.info("Scanning for nearby Bluetooth devices (5 seconds)...")
        scan_proc = subprocess.Popen(["timeout", "5", "hcitool", "lescan"], 
                                    stdout=subprocess.PIPE, 
                                    stderr=subprocess.STDOUT)
        
        # Give it time to scan
        time.sleep(5)
        
        # Get output
        output, _ = scan_proc.communicate()
        output_str = output.decode('utf-8')
        
        # Count discovered devices
        lines = output_str.strip().split('\n')
        
        # First line is the header, so we skip it
        devices = set()
        for line in lines[1:]:
            if line and ":" in line:
                parts = line.split(' ', 1)
                if len(parts) > 0:
                    devices.add(parts[0])
        
        if devices:
            return True, f"Found {len(devices)} Bluetooth devices nearby"
        else:
            return False, "No Bluetooth devices found nearby. This might be normal if no devices are in range."
    except subprocess.CalledProcessError as e:
        return False, f"Device scan failed: {e.output.decode('utf-8')}"

def run_all_tests():
    """Run all BLE diagnostic tests"""
    results = {
        "tests": [],
        "overall_result": "PASS",
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "system_info": {}
    }
    
    # Get system information
    try:
        # Get Raspberry Pi model
        with open("/proc/device-tree/model", "r") as f:
            results["system_info"]["model"] = f.read().strip('\0')
    except:
        results["system_info"]["model"] = "Unknown"
    
    # Get OS information
    try:
        with open("/etc/os-release", "r") as f:
            for line in f:
                if line.startswith("PRETTY_NAME="):
                    results["system_info"]["os"] = line.split("=")[1].strip().strip('"')
                    break
    except:
        results["system_info"]["os"] = "Unknown"
    
    # Get kernel version
    try:
        results["system_info"]["kernel"] = subprocess.check_output(["uname", "-r"]).decode('utf-8').strip()
    except:
        results["system_info"]["kernel"] = "Unknown"
    
    # Test 1: Check if BlueZ is installed
    success, message = check_command_exists("bluetoothd")
    results["tests"].append({
        "name": "BlueZ Installation",
        "result": "PASS" if success else "FAIL",
        "message": "BlueZ is installed" if success else "BlueZ is not installed"
    })
    if not success:
        results["overall_result"] = "FAIL"
    
    # Test 2: Check if Bluetooth service is running
    success = check_service_running("bluetooth")
    results["tests"].append({
        "name": "Bluetooth Service",
        "result": "PASS" if success else "FAIL",
        "message": "Bluetooth service is running" if success else "Bluetooth service is not running"
    })
    if not success:
        results["overall_result"] = "FAIL"
    
    # Test 3: Check D-Bus permissions
    success, message = check_dbus_permissions()
    results["tests"].append({
        "name": "D-Bus Permissions",
        "result": "PASS" if success else "FAIL",
        "message": message
    })
    if not success:
        results["overall_result"] = "FAIL"
    
    # Test 4: Check Bluetooth adapter
    success, message = check_bluetooth_adapter()
    results["tests"].append({
        "name": "Bluetooth Adapter",
        "result": "PASS" if success else "FAIL",
        "message": message
    })
    if not success:
        results["overall_result"] = "FAIL"
    
    # Test 5: Scan for devices
    success, message = scan_for_devices()
    results["tests"].append({
        "name": "Device Scan",
        "result": "PASS" if success else "WARNING",
        "message": message
    })
    # Not finding devices is a warning, not a failure
    
    # Test 6: Test BLE advertisement
    success, message = test_ble_advertisement()
    results["tests"].append({
        "name": "BLE Advertisement",
        "result": "PASS" if success else "FAIL",
        "message": message
    })
    if not success:
        results["overall_result"] = "FAIL"
    
    return results

def main():
    parser = argparse.ArgumentParser(description="BLE Diagnostic Test for NetTool")
    parser.add_argument("--output", help="Output file for results (JSON format)")
    parser.add_argument("--quiet", action="store_true", help="Suppress console output")
    args = parser.parse_args()
    
    if args.quiet:
        logger.setLevel(logging.WARNING)
    
    logger.info("Starting BLE diagnostic tests...")
    
    results = run_all_tests()
    
    # Print results to console
    if not args.quiet:
        print("\n=== BLE Diagnostic Test Results ===")
        print(f"Overall result: {results['overall_result']}")
        print(f"Timestamp: {results['timestamp']}")
        print("\nSystem Information:")
        for key, value in results["system_info"].items():
            print(f"  {key}: {value}")
        
        print("\nTest Results:")
        for test in results["tests"]:
            result_color = "\033[92m" if test["result"] == "PASS" else \
                          "\033[93m" if test["result"] == "WARNING" else \
                          "\033[91m"
            print(f"  {test['name']}: {result_color}{test['result']}\033[0m")
            print(f"    {test['message']}")
    
    # Save results to file if requested
    if args.output:
        with open(args.output, "w") as f:
            json.dump(results, f, indent=2)
        logger.info(f"Results saved to {args.output}")
    
    # Return non-zero exit code if overall result is not PASS
    return 0 if results["overall_result"] == "PASS" else 1

if __name__ == "__main__":
    sys.exit(main())

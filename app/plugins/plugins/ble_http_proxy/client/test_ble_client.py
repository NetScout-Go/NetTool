#!/usr/bin/env python3
"""
NetTool BLE HTTP Proxy Test Client

This script implements a test client for the NetTool BLE HTTP Proxy service.
It connects to a Raspberry Pi Zero running the BLE HTTP Proxy service,
sends HTTP requests through BLE, and processes the responses.

Dependencies:
- Python 3.7+
- bluepy (pip install bluepy)
- argparse
- json
- uuid

Usage:
  python3 test_ble_client.py --scan           # Scan for devices
  python3 test_ble_client.py --connect MAC    # Connect to specific device
  python3 test_ble_client.py --status MAC     # Get status from device
  python3 test_ble_client.py --get MAC /path  # Send GET request
"""

import argparse
import json
import time
import uuid
import sys
from bluepy import btle
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('nettool-ble-client')

# BLE Service and Characteristic UUIDs
BLE_SERVICE_UUID = "00001234-0000-1000-8000-00805f9b34fb"
BLE_REQUEST_CHAR_UUID = "00001235-0000-1000-8000-00805f9b34fb"
BLE_RESPONSE_CHAR_UUID = "00001236-0000-1000-8000-00805f9b34fb"
BLE_STATUS_CHAR_UUID = "00001237-0000-1000-8000-00805f9b34fb"

class NotificationDelegate(btle.DefaultDelegate):
    def __init__(self):
        btle.DefaultDelegate.__init__(self)
        self.response_data = bytearray()
        self.response_complete = False
        self.current_uuid = None

    def handleNotification(self, cHandle, data):
        # First 16 bytes are UUID to match request with response
        if len(data) >= 17:
            uuid_bytes = data[0:16]
            received_uuid = uuid.UUID(bytes=bytes(uuid_bytes))
            flags = data[16]
            payload = data[17:]
            
            logger.info(f"Received notification: UUID={received_uuid}, Flags={flags}, Payload length={len(payload)}")
            
            # If this is a new response or continuation of current one
            if self.current_uuid is None:
                self.current_uuid = received_uuid
                self.response_data = bytearray(payload)
                if flags & 0x01 == 0:  # If end flag is not set
                    self.response_complete = False
                else:
                    self.response_complete = True
            elif received_uuid == self.current_uuid:
                self.response_data.extend(payload)
                if flags & 0x01 != 0:  # If end flag is set
                    self.response_complete = True
            else:
                logger.warning(f"Received notification for different UUID: {received_uuid} vs {self.current_uuid}")

def scan_for_devices(timeout=10):
    """Scan for BLE devices"""
    logger.info(f"Scanning for BLE devices for {timeout} seconds...")
    scanner = btle.Scanner()
    devices = scanner.scan(timeout)
    
    logger.info(f"Found {len(devices)} devices")
    for dev in devices:
        logger.info(f"Device {dev.addr} ({dev.addrType}), RSSI={dev.rssi} dB")
        for (adtype, desc, value) in dev.getScanData():
            if desc == "Complete Local Name" and "NetTool" in value:
                logger.info(f"  {desc}: {value} ** NETTOOL DEVICE **")
            else:
                logger.info(f"  {desc}: {value}")
    
    return devices

def connect_to_device(address):
    """Connect to a specific device by MAC address"""
    logger.info(f"Connecting to {address}...")
    try:
        peripheral = btle.Peripheral(address)
        peripheral.setDelegate(NotificationDelegate())
        logger.info("Connected successfully")
        return peripheral
    except Exception as e:
        logger.error(f"Failed to connect: {e}")
        return None

def get_status(peripheral):
    """Get status information from the BLE HTTP Proxy"""
    try:
        service = peripheral.getServiceByUUID(BLE_SERVICE_UUID)
        status_char = service.getCharacteristics(BLE_STATUS_CHAR_UUID)[0]
        
        # Enable notifications for status updates
        peripheral.writeCharacteristic(status_char.valHandle + 1, b"\\x01\\x00", True)
        logger.info("Enabling status notifications...")
        
        # Read the status
        logger.info("Reading status characteristic...")
        status_bytes = status_char.read()
        status_json = status_bytes.decode('utf-8')
        status = json.loads(status_json)
        
        # Print status information in a formatted way
        logger.info("Status information:")
        logger.info("-" * 40)
        logger.info(f"Version:            {status.get('version', 'unknown')}")
        logger.info(f"Connected clients:  {status.get('connected_clients', 0)}")
        logger.info(f"Total connections:  {status.get('total_connections', 0)}")
        logger.info(f"Total requests:     {status.get('total_requests', 0)}")
        logger.info(f"Bytes sent:         {status.get('total_bytes_sent', 0):,} bytes")
        logger.info(f"Bytes received:     {status.get('total_bytes_received', 0):,} bytes")
        
        uptime_start = status.get('uptime_start', 0)
        if uptime_start > 0:
            uptime_seconds = int(time.time() - uptime_start)
            hours, remainder = divmod(uptime_seconds, 3600)
            minutes, seconds = divmod(remainder, 60)
            logger.info(f"Uptime:             {hours}h {minutes}m {seconds}s")
        
        logger.info(f"Service start:      {status.get('service_start_time', 'unknown')}")
        logger.info(f"MTU size:           {status.get('mtu_size', 23)} bytes")
        logger.info(f"Network latency:    {status.get('network_latency', 0):.2f} ms")
        logger.info(f"CPU usage:          {status.get('cpu_percent', 0):.1f}%")
        logger.info(f"Memory usage:       {status.get('memory_percent', 0):.1f}%")
        logger.info("-" * 40)
        
        features = status.get('supported_features', [])
        if features:
            logger.info("Supported features: " + ", ".join(features))
        
        return status
    except Exception as e:
        logger.error(f"Failed to get status: {e}")
        return None

def send_http_request(peripheral, method, path, headers=None, body=None):
    """Send an HTTP request over BLE"""
    try:
        service = peripheral.getServiceByUUID(BLE_SERVICE_UUID)
        request_char = service.getCharacteristics(BLE_REQUEST_CHAR_UUID)[0]
        response_char = service.getCharacteristics(BLE_RESPONSE_CHAR_UUID)[0]
        
        # Enable notifications for responses
        peripheral.writeCharacteristic(response_char.valHandle + 1, b"\\x01\\x00", True)
        
        # Generate a request UUID
        request_uuid = uuid.uuid4()
        logger.info(f"Sending {method} request to {path} with UUID {request_uuid}")
        
        # Construct HTTP request
        http_request = f"{method} {path} HTTP/1.1\\r\\n"
        if headers:
            for key, value in headers.items():
                http_request += f"{key}: {value}\\r\\n"
        http_request += "\\r\\n"
        if body:
            http_request += body
        
        # Prepare request data with UUID and flags
        request_data = bytearray(request_uuid.bytes)
        request_data.append(0x01)  # Flags: 0x01 = last chunk
        request_data.extend(http_request.encode('utf-8'))
        
        # Write the request
        request_char.write(request_data, True)
        logger.info(f"Request sent: {len(request_data)} bytes")
        
        # Wait for response (with timeout)
        start_time = time.time()
        delegate = peripheral.delegate
        delegate.response_complete = False
        delegate.current_uuid = request_uuid
        
        while not delegate.response_complete and (time.time() - start_time) < 30:
            if peripheral.waitForNotifications(1.0):
                logger.info("Received notification")
            else:
                logger.debug("Waiting for notification...")
        
        if delegate.response_complete:
            response_text = delegate.response_data.decode('utf-8', errors='replace')
            logger.info(f"Response received ({len(delegate.response_data)} bytes)")
            
            # Parse HTTP response
            headers_end = response_text.find("\\r\\n\\r\\n")
            if headers_end > 0:
                headers = response_text[:headers_end]
                body = response_text[headers_end + 4:]
                
                logger.info("Response headers:")
                for header in headers.split("\\r\\n"):
                    logger.info(f"  {header}")
                
                logger.info(f"Response body length: {len(body)} bytes")
                if len(body) < 1000:
                    logger.info(f"Response body: {body}")
                else:
                    logger.info(f"Response body: {body[:500]}... (truncated)")
            else:
                logger.info(f"Full response: {response_text}")
            
            return response_text
        else:
            logger.error("Timed out waiting for response")
            return None
    except Exception as e:
        logger.error(f"Failed to send request: {e}")
        return None

def main():
    parser = argparse.ArgumentParser(description='NetTool BLE HTTP Proxy Client')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--scan', action='store_true', help='Scan for BLE devices')
    group.add_argument('--connect', type=str, help='Connect to a specific device by MAC address')
    group.add_argument('--status', type=str, help='Get status from a specific device')
    group.add_argument('--get', type=str, help='Send GET request to a specific device')
    
    parser.add_argument('--path', type=str, default='/', help='HTTP path for request (default: /)')
    parser.add_argument('--timeout', type=int, default=10, help='Timeout in seconds (default: 10)')
    
    args = parser.parse_args()
    
    if args.scan:
        scan_for_devices(args.timeout)
        return
    
    if args.connect:
        peripheral = connect_to_device(args.connect)
        if peripheral:
            logger.info("Successfully connected to device")
            peripheral.disconnect()
        return
    
    if args.status:
        peripheral = connect_to_device(args.status)
        if peripheral:
            get_status(peripheral)
            peripheral.disconnect()
        return
    
    if args.get:
        peripheral = connect_to_device(args.get)
        if peripheral:
            send_http_request(peripheral, "GET", args.path)
            peripheral.disconnect()
        return

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    except Exception as e:
        logger.error(f"Error: {e}")
        sys.exit(1)

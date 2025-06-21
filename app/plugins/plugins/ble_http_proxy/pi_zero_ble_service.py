#!/usr/bin/env python3
"""
NetTool BLE HTTP Proxy Service for Raspberry Pi Zero 2 W

This script implements a Bluetooth Low Energy GATT service that allows
clients to connect to the NetTool dashboard over BLE.

It uses the B    

@method(in_signature='aya{sv}', out_signature='')
    def WriteValue(self, value, options):
        """
        Handle client writes to this characteristic
        """
        logger.info(f"Received write request with {len(value)} bytes")
        
        # Check if we have a device path in the options
        device_path = None
        if 'device' in options:
            device_path = options['device'].value
            # Register device as connected
            self.service.update_connection_stats(device_path, connected=True)
        
        # Update statistics for bytes received
        self.service.update_connection_stats(bytes_received=len(value))
        
        # Ensure we have at least 17 bytes (16 for UUID + 1 for flags)
        if len(value) < 17:
            logger.error("Request too short")
            returnAPI through the dbus-next library to:
1. Set up a GATT server
2. Create an HTTP proxy service
3. Handle HTTP requests and forward them to the local web server
4. Return responses to clients

Requirements:
- Python 3.7+
- dbus-next
- requests

Install dependencies:
  pip3 install dbus-next requests
"""

import asyncio
import dbus_next
from dbus_next.aio import MessageBus
from dbus_next.service import ServiceInterface, method, property, signal
from dbus_next.constants import BusType
from dbus_next import Variant, DBusError
import array
import struct
import uuid
import logging
import sys
import os
import requests
import threading
import signal
import time
import argparse
import json
from urllib.parse import urlparse

# Optional imports - wrap in try/except
try:
    import psutil  # For monitoring system resources
except ImportError:
    # Create a fallback if psutil is not available
    class PsutilFallback:
        def cpu_percent(self):
            return 0
        
        def virtual_memory(self):
            class Memory:
                def __init__(self):
                    self.percent = 0
            return Memory()
    
    psutil = PsutilFallback()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('nettool-ble-proxy')

# BLE Service UUIDs
BLE_HTTP_PROXY_SERVICE_UUID = '00001234-0000-1000-8000-00805f9b34fb'
BLE_HTTP_REQUEST_CHAR_UUID = '00001235-0000-1000-8000-00805f9b34fb'
BLE_HTTP_RESPONSE_CHAR_UUID = '00001236-0000-1000-8000-00805f9b34fb'
BLE_STATUS_CHAR_UUID = '00001237-0000-1000-8000-00805f9b34fb'

# BlueZ D-Bus constants
BLUEZ_SERVICE_NAME = 'org.bluez'
ADAPTER_INTERFACE = 'org.bluez.Adapter1'
DEVICE_INTERFACE = 'org.bluez.Device1'
GATT_MANAGER_INTERFACE = 'org.bluez.GattManager1'
GATT_SERVICE_INTERFACE = 'org.bluez.GattService1'
GATT_CHARACTERISTIC_INTERFACE = 'org.bluez.GattCharacteristic1'
DBUS_OM_INTERFACE = 'org.freedesktop.DBus.ObjectManager'
DBUS_PROP_INTERFACE = 'org.freedesktop.DBus.Properties'
LE_ADVERTISING_MANAGER_INTERFACE = 'org.bluez.LEAdvertisingManager1'
LE_ADVERTISEMENT_INTERFACE = 'org.bluez.LEAdvertisement1'

# HTTP Request data store
request_data = {}
response_data = {}
data_lock = threading.Lock()

# Connection tracking
connected_devices = set()
connection_stats = {
    "total_connections": 0,
    "total_requests": 0,
    "total_bytes_sent": 0,
    "total_bytes_received": 0,
    "uptime_start": time.time(),
    "connected_clients": 0
}
stats_lock = threading.Lock()

class InvalidArgsException(DBusError):
    """Exception for invalid arguments on D-Bus methods."""
    def __init__(self, message):
        super().__init__('org.freedesktop.DBus.Error.InvalidArgs', message)

class HTTPProxyService(ServiceInterface):
    """
    Implementation of the HTTP Proxy Service over BLE
    """
    def __init__(self, http_port):
        super().__init__(GATT_SERVICE_INTERFACE)
        self.http_port = http_port
        self.request_notifying = set()
        self.response_notifying = set()
        self.path = '/org/bluez/nettool/service0'
        self.mtu = 23  # Default MTU
        self._uuid = BLE_HTTP_PROXY_SERVICE_UUID
        self._characteristics = [
            '/org/bluez/nettool/service0/char0',  # Request characteristic
            '/org/bluez/nettool/service0/char1',  # Response characteristic
            '/org/bluez/nettool/service0/char2',  # Status characteristic
        ]
        self._primary = True
        
        # Store connected devices and connection stats
        self.connected_devices = set()
        self.last_status_update = 0
        
    def update_connection_stats(self, device_path=None, connected=None, bytes_sent=0, bytes_received=0):
        """Update connection statistics"""
        with stats_lock:
            # Update device connection status if provided
            if device_path is not None and connected is not None:
                if connected and device_path not in connected_devices:
                    connected_devices.add(device_path)
                    connection_stats["total_connections"] += 1
                    connection_stats["connected_clients"] = len(connected_devices)
                elif not connected and device_path in connected_devices:
                    connected_devices.remove(device_path)
                    connection_stats["connected_clients"] = len(connected_devices)
            
            # Update data transfer stats
            connection_stats["total_bytes_sent"] += bytes_sent
            connection_stats["total_bytes_received"] += bytes_received
            if bytes_received > 0:
                connection_stats["total_requests"] += 1
                
            # Get system resource usage
            connection_stats["cpu_percent"] = psutil.cpu_percent()
            connection_stats["memory_percent"] = psutil.virtual_memory().percent
            
            # Check if we should notify status change
            current_time = time.time()
            if (current_time - self.last_status_update) > 2.0:  # Update every 2 seconds max
                self.last_status_update = current_time
                self.notify_status_change()
    
    def notify_status_change(self):
        """Notify status characteristic subscribers of changes"""
        if hasattr(self, 'status_char'):
            try:
                # Convert stats to JSON and set as value
                status_json = json.dumps(connection_stats).encode('utf-8')
                self.status_char.set_value(array.array('B', status_json))
                logger.debug("Sent status update notification")
            except Exception as e:
                logger.error(f"Failed to send status notification: {e}")
        
    @property(signature='s')
    def UUID(self) -> str:
        return self._uuid
    
    @property(signature='ao')
    def Characteristics(self) -> list:
        return self._characteristics
    
    @property(signature='b')
    def Primary(self) -> bool:
        return self._primary
        
    def set_mtu(self, mtu):
        """
        Set the MTU for the BLE connection
        """
        self.mtu = mtu
        logger.info(f"MTU set to {mtu} bytes")
        
    def get_max_attribute_size(self):
        """
        Get the maximum attribute size (MTU - 3)
        """
        return self.mtu - 3

class HTTPRequestCharacteristic(ServiceInterface):
    """
    Characteristic for receiving HTTP requests from clients
    """
    def __init__(self, service):
        super().__init__(GATT_CHARACTERISTIC_INTERFACE)
        self.service = service
        self.path = '/org/bluez/nettool/service0/char0'
        self._value = array.array('B', [0] * 512)
        self.http_handlers = {}
        self._uuid = BLE_HTTP_REQUEST_CHAR_UUID
        self._service_path = service.path
        self._flags = ['write', 'write-without-response']
        
    @property(signature='s')
    def UUID(self) -> str:
        return self._uuid
    
    @property(signature='o')
    def Service(self) -> str:
        return self._service_path
    
    @property(signature='ay')
    def Value(self) -> list:
        return self._value
    
    @property(signature='as')
    def Flags(self) -> list:
        return self._flags
    
    @method(GATT_CHARACTERISTIC_INTERFACE, in_signature='aya{sv}', out_signature='')
    def WriteValue(self, value, options):
        """
        Handle client writes to this characteristic
        """
        logger.info(f"Received write request with {len(value)} bytes")
        
        # Check if we have a device path in the options
        device_path = None
        if 'device' in options:
            device_path = options['device'].value
            # Register device as connected
            self.service.update_connection_stats(device_path, connected=True)
        
        # Update statistics for bytes received
        self.service.update_connection_stats(bytes_received=len(value))
        
        # Ensure we have at least 17 bytes (16 for UUID + 1 for flags)
        if len(value) < 17:
            logger.error("Request too short")
            return
        
        # Extract request ID (first 16 bytes)
        req_id_bytes = bytes(value[:16])
        req_id = req_id_bytes.hex()
        
        # Extract flags (1 byte)
        flags = value[16]
        is_new = (flags & 0x1) != 0
        is_final = (flags & 0x2) != 0
        
        # Extract data
        data = bytes(value[17:])
        
        with data_lock:
            # For new requests, initialize the buffer
            if is_new:
                request_data[req_id] = data
            else:
                # Append to existing request
                if req_id in request_data:
                    request_data[req_id] += data
                else:
                    logger.error(f"Continuation for unknown request ID: {req_id}")
            
            # If this is the final chunk, process the request
            if is_final:
                # Process the complete HTTP request
                threading.Thread(target=self._process_http_request, 
                                args=(req_id, request_data[req_id])).start()
    
    def _process_http_request(self, req_id, req_data):
        """
        Process a complete HTTP request and prepare the response
        """
        try:
            # Parse HTTP request
            req_str = req_data.decode('utf-8')
            lines = req_str.split('\r\n')
            if not lines:
                logger.error("Invalid HTTP request format")
                return
            
            # Extract method, path and HTTP version
            request_line = lines[0].split(' ')
            if len(request_line) != 3:
                logger.error("Invalid HTTP request line")
                return
            
            method, path, http_ver = request_line
            
            # Extract headers
            headers = {}
            body = ""
            
            # Find the empty line that separates headers from body
            header_end = lines.index('')
            for i in range(1, header_end):
                header_line = lines[i]
                key, value = header_line.split(':', 1)
                headers[key.strip()] = value.strip()
            
            # Extract body if present
            if header_end < len(lines) - 1:
                body = '\r\n'.join(lines[header_end + 1:])
            
            # Build the URL
            url = f"http://localhost:{self.service.http_port}{path}"
            
            logger.info(f"Forwarding {method} request to {url}")
            
            # Forward the request to the local HTTP server
            response = requests.request(
                method=method,
                url=url,
                headers=headers,
                data=body
            )
            
            # Build HTTP response
            resp_str = f"{http_ver} {response.status_code} {response.reason}\r\n"
            
            # Add headers
            for key, value in response.headers.items():
                resp_str += f"{key}: {value}\r\n"
            
            resp_str += "\r\n"
            
            # Add body
            resp_str += response.text
            
            # Store the response
            with data_lock:
                response_data[req_id] = resp_str.encode('utf-8')
            
            # Notify the client that response is ready
            self._notify_response_ready(req_id)
            
        except Exception as e:
            logger.error(f"Error processing HTTP request: {e}")
            
            # Create an error response
            error_resp = f"HTTP/1.1 500 Internal Server Error\r\n"
            error_resp += "Content-Type: text/plain\r\n\r\n"
            error_resp += f"Error: {str(e)}"
            
            with data_lock:
                response_data[req_id] = error_resp.encode('utf-8')
            
            self._notify_response_ready(req_id)
    
    def _notify_response_ready(self, req_id):
        """
        Notify client that a response is ready
        """
        # Get the response characteristic
        if hasattr(self.service, 'response_char'):
            response_char = self.service.response_char
            
            # Create a notification packet (16 bytes for req_id + 1 byte for status)
            notify_data = bytes.fromhex(req_id) + bytes([1])  # 1 = response ready
            
            # Set the value and notify clients
            if response_char:
                try:
                    # Set the initial value to trigger notification
                    response_char.set_value(notify_data)
                    logger.info(f"Sent notification for request ID: {req_id}")
                except Exception as e:
                    logger.error(f"Failed to send notification: {e}")
        else:
            logger.warning("Response characteristic not available for notification")

class HTTPResponseCharacteristic(ServiceInterface):
    """
    Characteristic for sending HTTP responses to clients
    """
    def __init__(self, service):
        super().__init__(GATT_CHARACTERISTIC_INTERFACE)
        self.service = service
        self.path = '/org/bluez/nettool/service0/char1'
        self.notifying = set()
        self.current_value = array.array('B', [0] * 0)
        self._uuid = BLE_HTTP_RESPONSE_CHAR_UUID
        self._service_path = service.path
        self._flags = ['read', 'notify']
        
    @property(signature='s')
    def UUID(self) -> str:
        return self._uuid
    
    @property(signature='o')
    def Service(self) -> str:
        return self._service_path
    
    @property(signature='ay')
    def Value(self) -> list:
        # Return current value
        return self.current_value
    
    @property(signature='as')
    def Flags(self) -> list:
        return self._flags
    
    def set_value(self, value):
        """
        Set the characteristic value and notify subscribers
        """
        # Convert to array if bytes
        if isinstance(value, bytes):
            value = array.array('B', value)
        
        self.current_value = value
        
        # Notify all subscribers
        for client in self.notifying:
            try:
                self.emit_properties_changed({'Value': self.current_value})
                logger.info(f"Notified client {client}")
            except Exception as e:
                logger.error(f"Failed to notify client {client}: {e}")
                self.notifying.remove(client)
    
    @method(GATT_CHARACTERISTIC_INTERFACE, in_signature='a{sv}', out_signature='ay')
    def ReadValue(self, options):
        """
        Handle client reads from this characteristic
        """
        offset = 0
        if 'offset' in options:
            offset = options['offset'].value
        
        # Check if we have a device path in the options
        device_path = None
        if 'device' in options:
            device_path = options['device'].value
            # Register device as connected
            self.service.update_connection_stats(device_path, connected=True)
        
        # Check if we have a request ID
        req_id = None
        if 'req_id' in options:
            req_id = options['req_id'].value
        
        # Get MTU if available
        mtu = 0
        if 'mtu' in options:
            mtu = options['mtu'].value
            if mtu > 23 and mtu <= 517:  # Valid MTU range
                self.service.set_mtu(mtu)
        
        max_chunk_size = self.service.get_max_attribute_size()
        logger.info(f"Read request from device {device_path}, offset {offset}, req_id {req_id}, MTU: {self.service.mtu}")
        
        # If no specific request ID, return an error
        if not req_id:
            return array.array('B', [0] * 0)
        
        with data_lock:
            if req_id not in response_data:
                logger.error(f"No response data for request ID: {req_id}")
                return array.array('B', [0] * 0)
            
            resp_data = response_data[req_id]
            
            # Return the data starting from the requested offset
            if offset >= len(resp_data):
                return array.array('B', [0] * 0)
            
            # Return a chunk of the response
            chunk = resp_data[offset:offset+max_chunk_size]
            
            # Update bytes sent statistics
            self.service.update_connection_stats(bytes_sent=len(chunk))
            
            return array.array('B', chunk)
    
    @method(GATT_CHARACTERISTIC_INTERFACE, in_signature='', out_signature='')
    def StartNotify(self):
        """
        Start notifications for this characteristic
        """
        # Get the current client path
        sender = self.get_sender()
        logger.info(f"StartNotify from {sender}")
        self.notifying.add(sender)
    
    @method(GATT_CHARACTERISTIC_INTERFACE, in_signature='', out_signature='')
    def StopNotify(self):
        """
        Stop notifications for this characteristic
        """
        sender = self.get_sender()
        logger.info(f"StopNotify from {sender}")
        if sender in self.notifying:
            self.notifying.remove(sender)

class StatusCharacteristic(ServiceInterface):
    """
    Characteristic for providing BLE proxy status information
    """
    def __init__(self, service):
        super().__init__(GATT_CHARACTERISTIC_INTERFACE)
        self.service = service
        self.path = '/org/bluez/nettool/service0/char2'
        self.notifying = set()
        
    @property(signature='s')
    def UUID(self) -> str:
        return BLE_STATUS_CHAR_UUID
    
    @property(signature='o')
    def Service(self) -> str:
        return self.service.path
    
    @property(signature='ay')
    def Value(self) -> list:
        # Return current status as JSON
        with stats_lock:
            status_json = json.dumps(connection_stats).encode('utf-8')
            return array.array('B', status_json)
    
    @property(signature='as')
    def Flags(self) -> list:
        return ['read', 'notify']
    
    @method(in_signature='a{sv}', out_signature='ay')
    def ReadValue(self, options):
        """
        Handle client reads from this characteristic
        """
        offset = 0
        if 'offset' in options:
            offset = options['offset'].value
        
        # Check if we have a device path in the options
        device_path = None
        if 'device' in options:
            device_path = options['device'].value
            # Register device as connected when it reads status
            self.service.update_connection_stats(device_path, connected=True)
        
        # Get the current status as JSON
        with stats_lock:
            status_json = json.dumps(connection_stats).encode('utf-8')
        
        # Return the data starting from the requested offset
        if offset >= len(status_json):
            return array.array('B', [0] * 0)
        
        # Return the status data
        chunk = status_json[offset:offset+self.service.get_max_attribute_size()]
        return array.array('B', chunk)
    
    @method(in_signature='', out_signature='')
    def StartNotify(self):
        """
        Start notifications for this characteristic
        """
        # Get the current client path
        sender = self.get_sender()
        logger.info(f"StartNotify for status from {sender}")
        self.notifying.add(sender)
    
    @method(in_signature='', out_signature='')
    def StopNotify(self):
        """
        Stop notifications for this characteristic
        """
        sender = self.get_sender()
        logger.info(f"StopNotify for status from {sender}")
        if sender in self.notifying:
            self.notifying.remove(sender)
    
    def set_value(self, value):
        """
        Set the value and notify subscribers
        """
        # Only notify if we have subscribers
        if not self.notifying:
            return
        
        # Notify all subscribers
        for client in self.notifying:
            logger.debug(f"Notifying client {client} of status change")
            self.PropertiesChanged(GATT_CHARACTERISTIC_INTERFACE, {"Value": value}, [])

class Advertisement(ServiceInterface):
    """
    BLE Advertisement to broadcast the HTTP Proxy Service
    """
    def __init__(self, device_name):
        super().__init__(LE_ADVERTISEMENT_INTERFACE)
        self.path = '/org/bluez/nettool/advertisement0'
        self.device_name = device_name
        self._type = 'peripheral'
        self._service_uuids = [BLE_HTTP_PROXY_SERVICE_UUID]
        self._manufacturer_data = array.array('B', [0x59, 0x00, 0x01])
        
    @property(signature='s')
    def Type(self) -> str:
        return self._type
    
    @property(signature='as')
    def ServiceUUIDs(self) -> list:
        return self._service_uuids
    
    @property(signature='s')
    def LocalName(self) -> str:
        return self.device_name
        
    @property(signature='a{sv}')
    def ServiceData(self) -> dict:
        # Add service data to help with discovery
        return {
            BLE_HTTP_PROXY_SERVICE_UUID: Variant('ay', [0x01])  # Version 1
        }
        
    @property(signature='ay')
    def ManufacturerData(self) -> list:
        # Use NetTool manufacturer ID (using 0x0059 for example)
        # Format: [0x59, 0x00, 0x01] - ID 0x0059, protocol version 0x01
        return self._manufacturer_data
    
    @method(LE_ADVERTISEMENT_INTERFACE, in_signature='', out_signature='')
    def Release(self):
        logger.info("Advertisement released")

async def find_adapter(bus):
    """Find the first available Bluetooth adapter"""
    manager = bus.get_proxy_object(BLUEZ_SERVICE_NAME, '/',
                                  DBUS_OM_INTERFACE).get_interface(DBUS_OM_INTERFACE)
    
    objects = await manager.call_get_managed_objects()
    for path, interfaces in objects.items():
        if ADAPTER_INTERFACE in interfaces:
            return path
    
    return None

async def main(device_name, http_port):
    """Main application entry point"""
    logger.info(f"Starting NetTool BLE HTTP Proxy Service ({device_name}) on port {http_port}")
    
    # Create PID file
    with open('/tmp/nettool_ble_proxy.status', 'w') as f:
        f.write('running\n')
        f.write(f'PID: {os.getpid()}\n')
    
    # Connect to the D-Bus system bus
    bus = await MessageBus(bus_type=BusType.SYSTEM).connect()
    
    # Find the Bluetooth adapter
    adapter_path = await find_adapter(bus)
    if not adapter_path:
        logger.error("No Bluetooth adapter found")
        sys.exit(1)
    
    logger.info(f"Using Bluetooth adapter: {adapter_path}")
    
    # Get the adapter interface
    adapter = bus.get_proxy_object(BLUEZ_SERVICE_NAME, adapter_path,
                                  ADAPTER_INTERFACE).get_interface(ADAPTER_INTERFACE)
    
    # Power on the adapter
    props_iface = bus.get_proxy_object(BLUEZ_SERVICE_NAME, adapter_path,
                                      DBUS_PROP_INTERFACE).get_interface(DBUS_PROP_INTERFACE)
    
    # Power on the adapter if it's not already powered on
    powered = await props_iface.call_get('org.bluez.Adapter1', 'Powered')
    if not powered.value:
        await props_iface.call_set('org.bluez.Adapter1', 'Powered', Variant('b', True))
        logger.info("Powered on Bluetooth adapter")
    
    # Create the GATT service
    service = HTTPProxyService(http_port)
    request_char = HTTPRequestCharacteristic(service)
    response_char = HTTPResponseCharacteristic(service)
    status_char = StatusCharacteristic(service)
    
    # Store response_char and status_char in service for notifications
    service.response_char = response_char
    service.status_char = status_char
    
    # Export the interfaces to DBus
    bus.export('/org/bluez/nettool/service0', service)
    bus.export('/org/bluez/nettool/service0/char0', request_char)
    bus.export('/org/bluez/nettool/service0/char1', response_char)
    bus.export('/org/bluez/nettool/service0/char2', status_char)
    
    # Register the GATT service
    gatt_manager = bus.get_proxy_object(BLUEZ_SERVICE_NAME, adapter_path,
                                       GATT_MANAGER_INTERFACE).get_interface(GATT_MANAGER_INTERFACE)
    
    await gatt_manager.call_register_application('/org/bluez/nettool', {})
    logger.info("Registered GATT application")
    
    # Create and register the advertisement
    adv = Advertisement(device_name)
    bus.export('/org/bluez/nettool/advertisement0', adv)
    
    adv_manager = bus.get_proxy_object(BLUEZ_SERVICE_NAME, adapter_path,
                                      LE_ADVERTISING_MANAGER_INTERFACE).get_interface(LE_ADVERTISING_MANAGER_INTERFACE)
    
    await adv_manager.call_register_advertisement('/org/bluez/nettool/advertisement0', {})
    logger.info(f"Started advertising as '{device_name}'")
    
    # Set up signal handlers for graceful shutdown
    loop = asyncio.get_event_loop()
    
    def signal_handler():
        logger.info("Shutting down...")
        loop.stop()
    
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, signal_handler)
    
    try:
        # Keep the service running
        while True:
            await asyncio.sleep(1)
    except asyncio.CancelledError:
        pass
    finally:
        # Unregister the advertisement
        try:
            await adv_manager.call_unregister_advertisement('/org/bluez/nettool/advertisement0')
            logger.info("Unregistered advertisement")
        except Exception as e:
            logger.error(f"Error unregistering advertisement: {e}")
        
        # Unregister the GATT service
        try:
            await gatt_manager.call_unregister_application('/org/bluez/nettool')
            logger.info("Unregistered GATT application")
        except Exception as e:
            logger.error(f"Error unregistering GATT application: {e}")
        
        # Update status file
        with open('/tmp/nettool_ble_proxy.status', 'w') as f:
            f.write('stopped\n')
        
        logger.info("BLE HTTP Proxy Service stopped")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="NetTool BLE HTTP Proxy Service")
    parser.add_argument('--device-name', default='NetTool', help='Bluetooth device name to advertise')
    parser.add_argument('--port', type=int, default=8080, help='HTTP port to proxy')
    args = parser.parse_args()
    
    # Run the async main function
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(main(args.device_name, args.port))
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    finally:
        loop.close()

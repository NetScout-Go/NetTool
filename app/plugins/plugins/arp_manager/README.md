# ARP Manager Plugin

The ARP Manager plugin provides a comprehensive interface for viewing and managing the ARP (Address Resolution Protocol) table on your device. This plugin allows you to view the ARP cache, add static entries, delete entries, and flush the entire ARP cache.

## What is ARP?

Address Resolution Protocol (ARP) is a protocol used for mapping an IP address to a physical machine address (MAC address) recognized in the local network. ARP is essential for network communication as it links IP addresses to physical hardware addresses.

## Features

- **View ARP Table**: See all entries in the ARP cache with their associated details
- **Get Specific ARP Entry**: Look up a specific IP address in the ARP cache
- **Add ARP Entry**: Add a static mapping between an IP address and a MAC address
- **Delete ARP Entry**: Remove an entry from the ARP cache
- **Flush ARP Cache**: Clear all dynamic entries from the ARP cache
- **Manufacturer Identification**: Identify device manufacturers through MAC address OUI lookup

## Parameters

- **Action**: The ARP operation to perform
  - Show ARP Table: View all entries in the ARP cache
  - Get ARP Entry: Look up a specific IP address
  - Add ARP Entry: Add a static mapping
  - Delete ARP Entry: Remove an entry
  - Flush ARP Cache: Clear all dynamic entries
  
- **Network Interface**: The network interface to use (e.g., eth0, wlan0)
  
- **IP Address**: IP address for the ARP entry (required for get, add, and delete operations)
  
- **MAC Address**: MAC address for the ARP entry (required for add operation)
  
- **Entry Type**: Type of ARP entry when adding entries
  - Permanent: Static entry that doesn't expire
  - Temporary: Entry that will expire according to normal timeout
  - Published: Entry that will be used for proxy ARP
  
- **Verbose Output**: Show detailed output including reachability status and last seen time
  
- **Numeric Output**: Show numeric addresses without hostname resolution

## Usage Examples

### Viewing the ARP Table

Set "Action" to "Show ARP Table" to view all current entries in the ARP cache. Optionally specify a network interface to filter entries by interface.

### Adding a Static ARP Entry

1. Set "Action" to "Add ARP Entry"
2. Enter the IP Address (e.g., 192.168.1.100)
3. Enter the MAC Address (e.g., 00:11:22:33:44:55)
4. Select the Entry Type (usually "Permanent" for static entries)
5. Optionally specify the Network Interface

### Deleting an ARP Entry

1. Set "Action" to "Delete ARP Entry"
2. Enter the IP Address to delete
3. Optionally specify the Network Interface

### Flushing the ARP Cache

1. Set "Action" to "Flush ARP Cache"
2. Optionally specify the Network Interface to flush only entries for that interface

## Technical Notes

- Adding and deleting ARP entries requires root/sudo privileges
- Static ARP entries persist until they are explicitly deleted or the system is rebooted
- The "Flush ARP Cache" action only removes dynamic entries; static entries remain
- Manufacturer information is provided based on the MAC address OUI (first 6 digits)

## Troubleshooting

- **Operation Failed**: Make sure you have sufficient permissions (try running NetScout-Pi with sudo)
- **Invalid MAC Format**: Ensure MAC addresses are in the format 00:11:22:33:44:55 or 00-11-22-33-44-55
- **Entry Not Added/Deleted**: Verify the command output in the result for specific error messages
- **Missing Manufacturer Info**: This is normal if the OUI database is not available on your system

## Use Cases

- **Network Troubleshooting**: Identify MAC-to-IP mappings on your network
- **Security Monitoring**: Check for unexpected devices or MAC address changes
- **ARP Spoofing Protection**: Add static ARP entries for critical network devices
- **Network Management**: Manage ARP entries for network optimization

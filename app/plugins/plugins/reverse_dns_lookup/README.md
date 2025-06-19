# NetScout Plugin: Reverse DNS Lookup

IP Addresses
Timeout
DNS Server
Concurrent Lookups

This is a plugin for the NetScout-Go network diagnostics tool. It provides Performs reverse DNS lookups to get hostnames from IP addresses with batch processing capabilities
List of IP addresses for reverse lookup (one per line or comma-separated)
Timeout in seconds for each lookup
DNS server to use for lookups (leave blank for system default)
Number of lookups to perform concurrently.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_reverse_dns_lookup.git ~/.netscout/plugins/reverse_dns_lookup
ip_addresses
timeout
dns_server
concurrent_lookups
```

Or use the NetScout-Go plugin manager to install it:

```go
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_reverse_dns_lookup")
```

## Features

- Network diagnostics for reverse_dns_lookup
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

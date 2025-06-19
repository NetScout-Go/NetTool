# NetScout Plugin: DNS Propagation Checker
Domain
Record Type
Timeout
Custom DNS Servers

This is a plugin for the NetScout-Go network diagnostics tool. It provides Checks DNS propagation of a domain across multiple DNS servers worldwide
The domain name to check propagation for
The DNS record type to check
Timeout in seconds for each DNS query
Comma-separated list of additional DNS servers to check (e.g..

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_dns_propagation.git ~/.netscout/plugins/dns_propagation
domain
record_type
timeout
custom_dns_servers
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_dns_propagation")
```

## Features

- Network diagnostics for dns_propagation
- Easy integration with NetScout-Go

## License

MIT License

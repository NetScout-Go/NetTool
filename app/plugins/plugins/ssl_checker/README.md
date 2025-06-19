# NetScout Plugin: SSL/TLS Certificate Checker
Hostname
Port
Check Certificate Chain
Show Certificate Details

This is a plugin for the NetScout-Go network diagnostics tool. It provides Validates SSL/TLS certificates of websites and displays expiration dates
The hostname or domain to check (e.g.
The port to connect to
Validate the entire certificate chain
Show detailed certificate information.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_ssl_checker.git ~/.netscout/plugins/ssl_checker
hostname
port
check_chain
show_details
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_ssl_checker")
```

## Features

- Network diagnostics for ssl_checker
- Easy integration with NetScout-Go

## License

GNU GENERAL PUBLIC LICENSE, Version 3, 29 June 2007

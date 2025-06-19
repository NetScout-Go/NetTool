# Subnet Calculator Plugin

A comprehensive IP subnet calculator for NetTool that helps with analyzing network addresses, calculating subnet masks, and planning network segmentation.

## Features

- **Subnet Calculation**: Calculate detailed information about IPv4 and IPv6 subnets including network address, broadcast address, usable hosts, and more.
- **Subnet Division**: Divide a subnet into multiple smaller equal-sized subnets.
- **Supernetting**: Calculate the smallest supernet that contains multiple IP addresses or subnets.
- **Binary Representation**: View binary representations of IP addresses and subnet masks.
- **Reverse DNS Lookup Information**: Get reverse DNS lookup zone information for IP addresses.

## Usage

### Calculate Subnet Information

```json
{
  "action": "calculate",
  "address": "192.168.1.0/24"
}
```

Additional parameters:
- `mask`: Optional subnet mask (e.g., "255.255.255.0" or "24") if not included in the address
- `subnet_bits`: Optional additional subnet bits to add to the prefix length

### Divide a Subnet

```json
{
  "action": "divide",
  "address": "192.168.1.0/24",
  "num_subnets": 4
}
```

This will divide the 192.168.1.0/24 subnet into 4 equal-sized subnets.

### Calculate Supernet

```json
{
  "action": "supernet",
  "ip_list": [
    "192.168.1.0/24",
    "192.168.2.0/24",
    "192.168.3.0/24",
    "192.168.4.0/24"
  ]
}
```

This will calculate the smallest supernet (e.g., 192.168.0.0/22) that contains all the specified subnets.

## Response Example

```json
{
  "action": "calculate",
  "address": "192.168.1.0/24",
  "info": {
    "input_address": "192.168.1.0",
    "cidr": "192.168.1.0/24",
    "network_address": "192.168.1.0",
    "broadcast_address": "192.168.1.255",
    "netmask": "255.255.255.0",
    "wildcard_mask": "0.0.0.255",
    "first_host": "192.168.1.1",
    "last_host": "192.168.1.254",
    "total_hosts": 256,
    "usable_hosts": 254,
    "prefix_length": 24,
    "mask_bits": "11111111111111111111111100000000",
    "mask_decimal": 4294967040,
    "address_class": "C",
    "is_private": true,
    "type": "IPv4",
    "binary_address": "11000000101010000000000100000000",
    "binary_mask": "11111111111111111111111100000000",
    "subnet_bits": 24,
    "host_bits": 8,
    "address_range": "192.168.1.0 - 192.168.1.255",
    "reverse_dns_lookup": "0.1.168.192.in-addr.arpa",
    "reverse_dns_postfix": "1.168.192.in-addr.arpa"
  },
  "mask": "24",
  "subnet_bits": 0,
  "timestamp": "2025-06-19T12:34:56Z"
}
```

## License

MIT

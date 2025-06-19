# Subnet Calculator Plugin

The Subnet Calculator is a comprehensive networking tool for analyzing, planning, and managing network addressing. It provides detailed subnet information and allows for subnet division and supernetting operations.

## Features

### Subnet Calculation

- Calculate detailed information about IPv4 and IPv6 subnets
- Display network/broadcast addresses, first/last hosts, and usable host count
- Show binary representations, address class, and reverse DNS information
- Determine if an address is private or public

### Subnet Division

- Divide a network into multiple equal-sized subnets
- Support for both IPv4 and IPv6 subnets
- Automatic calculation of required prefix length

### Supernetting

- Calculate the smallest supernet that contains multiple IP addresses or networks
- Find common network prefix across disparate networks
- Support for IPv4 (full) and IPv6 (basic)

### CIDR Aggregation

- Find the minimum number of CIDR blocks that can represent a list of IP addresses
- Optimize route tables by reducing the number of routes needed
- Combine adjacent networks where possible
- Support for both IPv4 and IPv6 (basic)

### Network Conflict Detection

- Identify overlapping networks in a list of subnets
- Detect potential IP address conflicts
- Validate network segmentation designs
- Support for both IPv4 and IPv6

## Usage

### Calculate Subnet

```bash
Action: calculate
IP Address/CIDR: 192.168.1.0/24
Subnet Mask: (optional if CIDR is provided)
Subnet Bits: 0 (optional, for additional subnetting)
```

### Divide Subnet

```bash
Action: divide
IP Address/CIDR: 192.168.1.0/24
Subnet Mask: (optional if CIDR is provided)
Number of Subnets: 4 (how many subnets to create)
```

### Calculate Supernet

```bash
Action: supernet
IP Address List: 192.168.1.0/24,192.168.2.0/24
```

### Aggregate CIDRs

```bash
Action: aggregate
IP Address List: 192.168.1.0/24,192.168.2.0/24,192.168.3.0/24,192.168.4.0/24
```

### Detect Network Conflicts

```bash
Action: conflict_detect
IP Address List: 192.168.1.0/24,192.168.1.128/25,192.168.2.0/24
```

## Technical Details

- Supports both IPv4 and IPv6 addresses
- Handles CIDR notation and traditional subnet masks
- Provides detailed subnet analysis including:
  - Network/broadcast addresses
  - First/last usable hosts
  - Total and usable host counts
  - Binary representations
  - Address class and type information
  - Reverse DNS lookup zones

## Examples

### Calculate a Subnet

For IP 192.168.1.0/24:

- Network: 192.168.1.0
- Broadcast: 192.168.1.255
- Netmask: 255.255.255.0
- First Host: 192.168.1.1
- Last Host: 192.168.1.254
- Usable Hosts: 254

### Divide a Subnet

Dividing 192.168.1.0/24 into 4 subnets:

- 192.168.1.0/26 (64 hosts)
- 192.168.1.64/26 (64 hosts)
- 192.168.1.128/26 (64 hosts)
- 192.168.1.192/26 (64 hosts)

### Calculate a Supernet

For 192.168.1.0/24 and 192.168.2.0/24:

- Supernet: 192.168.0.0/23

### Aggregate CIDRs

For 192.168.1.0/24, 192.168.2.0/24, 192.168.3.0/24, and 192.168.4.0/24:

- Aggregated CIDRs: 192.168.0.0/22 (single CIDR block)

### Detect Network Conflicts

For 192.168.1.0/24, 192.168.1.128/25, and 192.168.2.0/24:

- Conflict detected: 192.168.1.0/24 overlaps with 192.168.1.128/25
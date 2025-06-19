package subnet_calculator

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/bits"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SubnetInfo represents subnet calculation results
type SubnetInfo struct {
	InputAddress      string   `json:"input_address"`
	CIDR              string   `json:"cidr"`
	NetworkAddress    string   `json:"network_address"`
	BroadcastAddress  string   `json:"broadcast_address"`
	Netmask           string   `json:"netmask"`
	WildcardMask      string   `json:"wildcard_mask"`
	FirstHost         string   `json:"first_host"`
	LastHost          string   `json:"last_host"`
	TotalHosts        int      `json:"total_hosts"`
	UsableHosts       int      `json:"usable_hosts"`
	PrefixLength      int      `json:"prefix_length"`
	MaskBits          string   `json:"mask_bits"`
	MaskDecimal       uint32   `json:"mask_decimal"`
	AddressClass      string   `json:"address_class"`
	IsPrivate         bool     `json:"is_private"`
	Type              string   `json:"type"` // IPv4 or IPv6
	BinaryAddress     string   `json:"binary_address"`
	BinaryMask        string   `json:"binary_mask"`
	SubnetBits        int      `json:"subnet_bits"`
	HostBits          int      `json:"host_bits"`
	Subnets           []string `json:"subnets,omitempty"`          // If subnet division requested
	SupernetCIDR      string   `json:"supernet_cidr,omitempty"`    // If supernetting requested
	SupernetNetmask   string   `json:"supernet_netmask,omitempty"` // If supernetting requested
	SupernetAddress   string   `json:"supernet_address,omitempty"` // If supernetting requested
	AddressRange      string   `json:"address_range"`
	ReverseDNSLookup  string   `json:"reverse_dns_lookup"`
	ReverseDNSPostfix string   `json:"reverse_dns_postfix"`
}

// Execute handles the subnet calculator plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	action, ok := params["action"].(string)
	if !ok {
		action = "calculate" // Default action
	}

	address, _ := params["address"].(string)
	mask, _ := params["mask"].(string)
	subnetBits, _ := params["subnet_bits"].(float64)
	numSubnets, _ := params["num_subnets"].(float64)
	ipList, _ := params["ip_list"].([]interface{})

	// Default timestamp
	timestamp := time.Now().Format(time.RFC3339)

	// Execute the requested subnet calculation action
	switch action {
	case "calculate":
		if address == "" {
			return nil, fmt.Errorf("IP address is required for calculate action")
		}
		return calculateSubnet(address, mask, int(subnetBits), timestamp)
	case "divide":
		if address == "" {
			return nil, fmt.Errorf("IP address is required for divide action")
		}
		return divideSubnet(address, mask, int(numSubnets), timestamp)
	case "supernet":
		// Convert interface slice to string slice
		ips := make([]string, 0, len(ipList))
		for _, ip := range ipList {
			if ipStr, ok := ip.(string); ok {
				ips = append(ips, ipStr)
			}
		}
		if len(ips) < 2 {
			return nil, fmt.Errorf("at least two IP addresses are required for supernet action")
		}
		return calculateSupernet(ips, timestamp)
	case "aggregate":
		// Convert interface slice to string slice
		ips := make([]string, 0, len(ipList))
		for _, ip := range ipList {
			if ipStr, ok := ip.(string); ok {
				ips = append(ips, ipStr)
			}
		}
		if len(ips) < 2 {
			return nil, fmt.Errorf("at least two IP addresses are required for aggregate action")
		}
		return aggregateCIDRs(ips, timestamp)
	case "conflict_detect":
		// Convert interface slice to string slice
		ips := make([]string, 0, len(ipList))
		for _, ip := range ipList {
			if ipStr, ok := ip.(string); ok {
				ips = append(ips, ipStr)
			}
		}
		if len(ips) < 2 {
			return nil, fmt.Errorf("at least two IP addresses are required for conflict detection")
		}
		return networkConflictDetector(ips, timestamp)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// calculateSubnet performs subnet calculations
func calculateSubnet(ipAddress, mask string, subnetBits int, timestamp string) (interface{}, error) {
	// Normalize the input and handle CIDR notation
	var prefixLen int
	var err error
	var ipNet *net.IPNet
	var ip net.IP

	// Check if the address already contains CIDR notation
	if strings.Contains(ipAddress, "/") {
		ip, ipNet, err = net.ParseCIDR(ipAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR address: %s", err.Error())
		}
		ones, _ := ipNet.Mask.Size()
		prefixLen = ones
	} else {
		// IP address without CIDR notation
		ip = net.ParseIP(ipAddress)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
		}

		// Determine IP version and set default mask if not provided
		ipv4 := ip.To4() != nil
		if mask == "" {
			if ipv4 {
				mask = "24" // Default to /24 for IPv4
			} else {
				mask = "64" // Default to /64 for IPv6
			}
		}

		// Parse the mask (could be in CIDR format or dotted decimal)
		if strings.Contains(mask, ".") {
			// Dotted decimal notation (IPv4 only)
			if !ipv4 {
				return nil, fmt.Errorf("dotted decimal mask can only be used with IPv4 addresses")
			}
			netmask := net.ParseIP(mask).To4()
			if netmask == nil {
				return nil, fmt.Errorf("invalid netmask: %s", mask)
			}
			prefixLen, _ = net.IPv4Mask(netmask[0], netmask[1], netmask[2], netmask[3]).Size()
		} else {
			// CIDR prefix length
			prefixLen, err = strconv.Atoi(mask)
			if err != nil {
				return nil, fmt.Errorf("invalid prefix length: %s", mask)
			}

			// Validate prefix length based on IP version
			if ipv4 && (prefixLen < 0 || prefixLen > 32) {
				return nil, fmt.Errorf("IPv4 prefix length must be between 0 and 32")
			} else if !ipv4 && (prefixLen < 0 || prefixLen > 128) {
				return nil, fmt.Errorf("IPv6 prefix length must be between 0 and 128")
			}
		}

		// Create the IPNet with the parsed IP and mask
		var maskSize int
		if ipv4 {
			maskSize = 32
			ip = ip.To4()
		} else {
			maskSize = 128
		}
		mask := net.CIDRMask(prefixLen, maskSize)
		ipNet = &net.IPNet{
			IP:   ip.Mask(mask),
			Mask: mask,
		}
	}

	// Apply subnet bits if provided
	if subnetBits > 0 {
		ones, bits := ipNet.Mask.Size()
		newPrefixLen := ones + subnetBits
		if newPrefixLen > bits {
			return nil, fmt.Errorf("subnet bits too large, resulting prefix length would be %d/%d", newPrefixLen, bits)
		}
		prefixLen = newPrefixLen
		ipNet.Mask = net.CIDRMask(prefixLen, bits)
		ipNet.IP = ipNet.IP.Mask(ipNet.Mask)
	}

	// Create the subnet info structure
	info := calculateSubnetInfo(ip, ipNet, prefixLen)

	// Build the response
	result := map[string]interface{}{
		"action":      "calculate",
		"address":     ipAddress,
		"mask":        mask,
		"subnet_bits": subnetBits,
		"info":        info,
		"timestamp":   timestamp,
	}

	return result, nil
}

// divideSubnet divides a subnet into multiple equal-sized subnets
func divideSubnet(ipAddress, mask string, numSubnets int, timestamp string) (interface{}, error) {
	// First, calculate the base subnet
	baseResult, err := calculateSubnet(ipAddress, mask, 0, timestamp)
	if err != nil {
		return nil, err
	}

	// Get the base subnet info
	resultMap, ok := baseResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result format from subnet calculation")
	}

	info, ok := resultMap["info"].(SubnetInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected subnet info format")
	}

	// Determine how many additional bits we need for the subnets
	requiredBits := int(math.Ceil(math.Log2(float64(numSubnets))))

	// Check if we have enough host bits to create these subnets
	if requiredBits > info.HostBits {
		return nil, fmt.Errorf("cannot create %d subnets, need %d bits but only have %d host bits available",
			numSubnets, requiredBits, info.HostBits)
	}

	// Calculate the new prefix length
	newPrefixLength := info.PrefixLength + requiredBits

	// Calculate the subnets
	subnets := make([]string, 0)

	// Parse the base CIDR
	_, baseIPNet, _ := net.ParseCIDR(info.CIDR)
	baseIP := baseIPNet.IP

	// IPv4 specific processing
	if info.Type == "IPv4" {
		// Convert IP to uint32 for easy manipulation
		ipInt := ipToUint32(baseIP)

		// Calculate the size of each subnet
		subnetSize := uint32(1) << (32 - newPrefixLength)

		// Generate the subnets
		for i := 0; i < 1<<requiredBits && i < numSubnets; i++ {
			subnetIP := uint32ToIP(ipInt + uint32(i)*subnetSize)
			subnets = append(subnets, fmt.Sprintf("%s/%d", subnetIP.String(), newPrefixLength))
		}
	} else {
		// IPv6 processing
		_, bits := baseIPNet.Mask.Size()

		// Create a subnet template
		subnetTemplate := make(net.IP, len(baseIP))
		copy(subnetTemplate, baseIP)

		// Generate the subnets
		for i := 0; i < 1<<requiredBits && i < numSubnets; i++ {
			// Create a copy of the template for this subnet
			subnetIP := make(net.IP, len(subnetTemplate))
			copy(subnetIP, subnetTemplate)

			// Calculate the subnet offset and apply it to the IP
			// We need to identify which bytes and bits within those bytes to modify
			byteIndex := (bits - newPrefixLength) / 8
			bitOffset := (bits - newPrefixLength) % 8

			// If we need to modify multiple bytes
			offset := i
			// Start from the least significant bytes that need modification
			for j := len(subnetIP) - 1; j >= 0 && byteIndex > 0; j-- {
				if offset == 0 {
					break
				}

				// Calculate how many bits we can modify in this byte
				bitsInByte := 8
				if j == len(subnetIP)-byteIndex {
					bitsInByte = bitOffset
				}

				// Calculate the value to add to this byte
				byteMask := (1 << bitsInByte) - 1
				byteValue := offset & byteMask
				subnetIP[j] |= byte(byteValue)

				// Shift the offset for the next byte
				offset >>= bitsInByte
			}

			subnets = append(subnets, fmt.Sprintf("%s/%d", subnetIP.String(), newPrefixLength))
		}
	}

	// Add subnets to the info
	info.Subnets = subnets
	info.SubnetBits = requiredBits

	// Build the result
	result := map[string]interface{}{
		"action":          "divide",
		"address":         ipAddress,
		"mask":            mask,
		"num_subnets":     numSubnets,
		"subnet_bits":     requiredBits,
		"new_prefix":      newPrefixLength,
		"info":            info,
		"subnets":         subnets,
		"subnets_created": len(subnets),
		"timestamp":       timestamp,
	}

	return result, nil
}

// calculateSupernet calculates the smallest supernet that contains all given IPs
func calculateSupernet(ipAddresses []string, timestamp string) (interface{}, error) {
	if len(ipAddresses) < 2 {
		return nil, fmt.Errorf("at least two IP addresses are required for supernetting")
	}

	// Verify all IPs are the same version
	firstIP := net.ParseIP(strings.Split(ipAddresses[0], "/")[0])
	if firstIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddresses[0])
	}

	isIPv4 := firstIP.To4() != nil

	// Convert all IPs to their binary representation
	ips := make([]net.IP, 0, len(ipAddresses))
	for _, addrStr := range ipAddresses {
		addr := strings.Split(addrStr, "/")[0] // Remove CIDR notation if present
		ip := net.ParseIP(addr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", addr)
		}

		// Ensure all IPs are of same version
		if (ip.To4() != nil) != isIPv4 {
			return nil, fmt.Errorf("mix of IPv4 and IPv6 addresses not supported")
		}

		if isIPv4 {
			ips = append(ips, ip.To4())
		} else {
			ips = append(ips, ip)
		}
	}

	// Find the common prefix bits
	var prefixLen int
	if isIPv4 {
		prefixLen = findCommonPrefixLength(ips, 32)
	} else {
		prefixLen = findCommonPrefixLength(ips, 128)
	}

	// Create the supernet
	var supernetIP net.IP
	if isIPv4 {
		mask := net.CIDRMask(prefixLen, 32)
		supernetIP = ips[0].Mask(mask)
	} else {
		mask := net.CIDRMask(prefixLen, 128)
		supernetIP = ips[0].Mask(mask)
	}

	// Create CIDR
	supernetCIDR := fmt.Sprintf("%s/%d", supernetIP.String(), prefixLen)

	// Calculate netmask in dotted decimal (for IPv4)
	var supernetNetmask string
	if isIPv4 {
		mask := net.CIDRMask(prefixLen, 32)
		supernetNetmask = fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
	} else {
		// IPv6 doesn't typically use dotted decimal for masks
		supernetNetmask = fmt.Sprintf("/%d", prefixLen)
	}

	// Get full subnet info for the supernet
	_, supernetNet, _ := net.ParseCIDR(supernetCIDR)
	info := calculateSubnetInfo(supernetIP, supernetNet, prefixLen)

	// Add supernet specific info
	info.SupernetCIDR = supernetCIDR
	info.SupernetNetmask = supernetNetmask
	info.SupernetAddress = supernetIP.String()

	// Build the result
	result := map[string]interface{}{
		"action":        "supernet",
		"ip_addresses":  ipAddresses,
		"supernet_cidr": supernetCIDR,
		"supernet_mask": supernetNetmask,
		"supernet_bits": prefixLen,
		"info":          info,
		"ip_count":      len(ipAddresses),
		"timestamp":     timestamp,
	}

	return result, nil
}

// aggregateCIDRs implements CIDR aggregation to find the minimum set of CIDR blocks
// that can represent a list of IP addresses or networks
func aggregateCIDRs(ipAddresses []string, timestamp string) (interface{}, error) {
	if len(ipAddresses) < 1 {
		return nil, fmt.Errorf("at least one IP address is required for CIDR aggregation")
	}

	// Parse and validate IP addresses or networks
	ipNets := make([]*net.IPNet, 0, len(ipAddresses))
	isIPv4 := true

	for _, addrStr := range ipAddresses {
		// Ensure the address has CIDR notation
		if !strings.Contains(addrStr, "/") {
			// Assume it's a single IP and add the max prefix
			if net.ParseIP(addrStr).To4() != nil {
				addrStr = addrStr + "/32"
			} else {
				addrStr = addrStr + "/128"
				isIPv4 = false
			}
		}

		_, ipNet, err := net.ParseCIDR(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid IP address or CIDR: %s (%s)", addrStr, err.Error())
		}

		// Check for IP version consistency
		if (ipNet.IP.To4() != nil) != isIPv4 {
			return nil, fmt.Errorf("mixing IPv4 and IPv6 addresses is not supported")
		}

		ipNets = append(ipNets, ipNet)
	}

	// Sort networks by IP address
	sort.Slice(ipNets, func(i, j int) bool {
		return bytes.Compare(ipNets[i].IP, ipNets[j].IP) < 0
	})

	// Merge adjacent or overlapping networks
	aggregatedNets := aggregateAdjacentNetworks(ipNets, isIPv4)

	// Create result strings for each aggregated CIDR
	resultCIDRs := make([]string, 0, len(aggregatedNets))
	for _, n := range aggregatedNets {
		ones, _ := n.Mask.Size()
		resultCIDRs = append(resultCIDRs, fmt.Sprintf("%s/%d", n.IP.String(), ones))
	}

	// Build the response
	result := map[string]interface{}{
		"action":            "aggregate",
		"input_addresses":   ipAddresses,
		"aggregated_cidrs":  resultCIDRs,
		"original_count":    len(ipAddresses),
		"aggregated_count":  len(resultCIDRs),
		"reduction_percent": 100 - (float64(len(resultCIDRs)) / float64(len(ipAddresses)) * 100),
		"ip_version":        map[bool]string{true: "IPv4", false: "IPv6"}[isIPv4],
		"timestamp":         timestamp,
	}

	return result, nil
}

// aggregateAdjacentNetworks combines adjacent networks into larger ones where possible
func aggregateAdjacentNetworks(nets []*net.IPNet, isIPv4 bool) []*net.IPNet {
	if len(nets) <= 1 {
		return nets
	}

	// Function to check if two networks can be merged
	canMerge := func(a, b *net.IPNet) bool {
		// They must have the same prefix length
		aOnes, aBits := a.Mask.Size()
		bOnes, bBits := b.Mask.Size()

		if aOnes != bOnes || aBits != bBits {
			return false
		}

		// For networks to be mergeable, they must be adjacent
		// and their combined network must align with a CIDR boundary

		// Convert IPs to integers for easier comparison
		var aInt, bInt uint64
		if isIPv4 {
			aInt = uint64(ipToUint32(a.IP.To4()))
			bInt = uint64(ipToUint32(b.IP.To4()))
		} else {
			// For IPv6, we'll use the first 8 bytes for simplicity
			// This is not a complete implementation for IPv6
			aInt = binary.BigEndian.Uint64(a.IP[:8])
			bInt = binary.BigEndian.Uint64(b.IP[:8])
		}

		// Calculate size of each network (in address space)
		networkSize := uint64(1) << (uint64(aBits) - uint64(aOnes))

		// Networks are adjacent if one starts where the other ends
		isAdjacent := (aInt+networkSize == bInt) || (bInt+networkSize == aInt)
		if !isAdjacent {
			return false
		}

		// To be mergeable into a valid CIDR, the lower address must be
		// a multiple of twice the network size
		lowerInt := aInt
		if bInt < aInt {
			lowerInt = bInt
		}

		return (lowerInt % (networkSize * 2)) == 0
	}

	// Function to merge two networks
	mergeNets := func(a, b *net.IPNet) *net.IPNet {
		// When merging, we decrease the prefix length by 1
		aOnes, aBits := a.Mask.Size()
		newOnes := aOnes - 1

		// Find the lower of the two networks
		var lowerIP net.IP
		if bytes.Compare(a.IP, b.IP) < 0 {
			lowerIP = a.IP
		} else {
			lowerIP = b.IP
		}

		// Create a new network with the combined prefix
		newMask := net.CIDRMask(newOnes, aBits)
		return &net.IPNet{
			IP:   lowerIP.Mask(newMask),
			Mask: newMask,
		}
	}

	// Continue merging networks until no more merges are possible
	result := make([]*net.IPNet, len(nets))
	copy(result, nets)

	for {
		merged := false

		for i := 0; i < len(result)-1; i++ {
			if canMerge(result[i], result[i+1]) {
				// Merge these two networks
				merged = true

				// Replace the first with the merged network and remove the second
				mergedNet := mergeNets(result[i], result[i+1])
				result[i] = mergedNet
				result = append(result[:i+1], result[i+2:]...)

				// Start over since we modified the list
				break
			}
		}

		// If no merges happened in this pass, we're done
		if !merged {
			break
		}
	}

	return result
}

// calculateSubnetInfo calculates detailed information about a subnet
func calculateSubnetInfo(ip net.IP, ipNet *net.IPNet, prefixLen int) SubnetInfo {
	info := SubnetInfo{}

	// Determine if it's IPv4 or IPv6
	ipv4 := ip.To4() != nil
	if ipv4 {
		info.Type = "IPv4"
	} else {
		info.Type = "IPv6"
	}

	// Set basic info
	info.InputAddress = ip.String()
	info.CIDR = fmt.Sprintf("%s/%d", ipNet.IP.String(), prefixLen)
	info.NetworkAddress = ipNet.IP.String()
	info.PrefixLength = prefixLen

	// Calculate additional IPv4-specific info
	if ipv4 {
		// Get mask bits
		ones, bits := ipNet.Mask.Size()
		info.SubnetBits = ones
		info.HostBits = bits - ones

		// Binary representations
		ipInt := ipToUint32(ip.To4())
		netMask := net.CIDRMask(prefixLen, 32)
		// Convert mask to IP format and then to uint32
		maskIP := net.IPv4(netMask[0], netMask[1], netMask[2], netMask[3])
		maskInt := ipToUint32(maskIP.To4())
		info.BinaryAddress = fmt.Sprintf("%032b", ipInt)
		info.BinaryMask = fmt.Sprintf("%032b", maskInt)
		info.MaskDecimal = maskInt

		// Netmask in dotted decimal
		mask := ipNet.Mask
		info.Netmask = fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])

		// Wildcard mask (inverse of netmask)
		wildcard := []byte{^mask[0], ^mask[1], ^mask[2], ^mask[3]}
		info.WildcardMask = fmt.Sprintf("%d.%d.%d.%d", wildcard[0], wildcard[1], wildcard[2], wildcard[3])

		// Network and broadcast addresses
		networkIP := ipInt & maskInt
		broadcastIP := networkIP | ^maskInt
		info.NetworkAddress = uint32ToIP(networkIP).String()
		info.BroadcastAddress = uint32ToIP(broadcastIP).String()

		// First and last usable host addresses
		if prefixLen < 31 { // Standard subnets
			info.FirstHost = uint32ToIP(networkIP + 1).String()
			info.LastHost = uint32ToIP(broadcastIP - 1).String()
			info.UsableHosts = int(broadcastIP - networkIP - 1)
		} else if prefixLen == 31 { // RFC 3021 - point-to-point links
			info.FirstHost = uint32ToIP(networkIP).String()
			info.LastHost = uint32ToIP(broadcastIP).String()
			info.UsableHosts = 2
		} else { // /32 - single host
			info.FirstHost = uint32ToIP(networkIP).String()
			info.LastHost = uint32ToIP(networkIP).String()
			info.UsableHosts = 1
		}

		// Total hosts (including network and broadcast addresses)
		info.TotalHosts = int(math.Pow(2, float64(32-prefixLen)))

		// Mask bits representation
		info.MaskBits = strings.Repeat("1", ones) + strings.Repeat("0", bits-ones)

		// Address range
		info.AddressRange = fmt.Sprintf("%s - %s", info.NetworkAddress, info.BroadcastAddress)

		// Determine address class
		firstOctet := ip[0]
		switch {
		case firstOctet < 128:
			info.AddressClass = "A"
		case firstOctet < 192:
			info.AddressClass = "B"
		case firstOctet < 224:
			info.AddressClass = "C"
		case firstOctet < 240:
			info.AddressClass = "D (Multicast)"
		default:
			info.AddressClass = "E (Reserved)"
		}

		// Check if it's a private address
		private := false
		privateRanges := []struct{ start, end uint32 }{
			{ipToUint32(net.ParseIP("10.0.0.0").To4()), ipToUint32(net.ParseIP("10.255.255.255").To4())},
			{ipToUint32(net.ParseIP("172.16.0.0").To4()), ipToUint32(net.ParseIP("172.31.255.255").To4())},
			{ipToUint32(net.ParseIP("192.168.0.0").To4()), ipToUint32(net.ParseIP("192.168.255.255").To4())},
		}
		for _, r := range privateRanges {
			if ipInt >= r.start && ipInt <= r.end {
				private = true
				break
			}
		}
		info.IsPrivate = private

		// Reverse DNS lookup zone
		octets := strings.Split(info.NetworkAddress, ".")
		info.ReverseDNSPostfix = fmt.Sprintf("%s.%s.%s.in-addr.arpa", octets[2], octets[1], octets[0])
		info.ReverseDNSLookup = fmt.Sprintf("%d.%s", ip[3], info.ReverseDNSPostfix)

	} else {
		// IPv6 specific calculations
		// Many of the fields above would need different calculations for IPv6
		// For now we'll provide basic info and mark the rest as not applicable
		info.UsableHosts = int(math.Pow(2, float64(128-prefixLen)))
		info.TotalHosts = info.UsableHosts
		info.IsPrivate = ip.IsPrivate()
		info.AddressClass = "Not applicable to IPv6"

		// For IPv6, the first and last hosts depend on the prefix length
		if prefixLen == 128 {
			// Single host
			info.FirstHost = ip.String()
			info.LastHost = ip.String()
			info.UsableHosts = 1
		} else {
			// Calculate first and last hosts for IPv6
			// This is simplified and might not be accurate for all use cases
			ones, bits := ipNet.Mask.Size()
			hostBits := bits - ones

			// First host - add 1 to network address
			firstHostIP := make(net.IP, len(ipNet.IP))
			copy(firstHostIP, ipNet.IP)
			firstHostIP[len(firstHostIP)-1]++
			info.FirstHost = firstHostIP.String()

			// Last host calculation would be complex for IPv6
			// We'll simplify and just show the theoretical count
			info.LastHost = "Complex calculation for IPv6"

			// For IPv6, the usable host count is essentially 2^hostBits
			info.UsableHosts = int(math.Pow(2, float64(hostBits)))
		}

		// Basic IPv6 reverse DNS (simplified)
		expandedIP := expandIPv6(ip)
		reversedNibbles := reverseIPv6Nibbles(expandedIP)
		info.ReverseDNSPostfix = reversedNibbles + ".ip6.arpa"
		info.ReverseDNSLookup = info.ReverseDNSPostfix
	}

	return info
}

// ipToUint32 converts an IPv4 address to uint32
func ipToUint32(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

// uint32ToIP converts a uint32 to an IPv4 address
func uint32ToIP(ipInt uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipInt)
	return ip
}

// findCommonPrefixLength finds the common prefix length of a list of IPs
func findCommonPrefixLength(ips []net.IP, maxBits int) int {
	if len(ips) == 0 {
		return 0
	}

	// Use first IP as reference
	refIP := ips[0]

	// Start with maximum possible prefix length
	prefixLen := maxBits

	// Compare each IP to the reference and reduce prefixLen as needed
	for _, ip := range ips[1:] {
		// Compare byte by byte
		commonBits := 0
		for i := 0; i < len(refIP) && i < len(ip); i++ {
			// XOR the bytes - bits that differ will be 1
			xor := refIP[i] ^ ip[i]
			if xor == 0 {
				// All bits match
				commonBits += 8
			} else {
				// Count leading zeros in XOR result (matching bits)
				commonBits += bits.LeadingZeros8(xor)
				break
			}
		}

		// Update prefix length if this IP has fewer matching bits
		if commonBits < prefixLen {
			prefixLen = commonBits
		}
	}

	return prefixLen
}

// expandIPv6 expands an IPv6 address to its full form
func expandIPv6(ip net.IP) string {
	dst := make([]byte, hex.EncodedLen(len(ip)))
	_ = hex.Encode(dst, ip)
	return string(dst)
}

// reverseIPv6Nibbles returns the nibbles of an IPv6 address in reverse order
func reverseIPv6Nibbles(expandedIP string) string {
	var reversed strings.Builder
	for i := len(expandedIP) - 1; i >= 0; i-- {
		if i != len(expandedIP)-1 {
			reversed.WriteByte('.')
		}
		reversed.WriteByte(expandedIP[i])
	}
	return reversed.String()
}

// networkConflictDetector identifies overlapping networks
func networkConflictDetector(ipAddresses []string, timestamp string) (interface{}, error) {
	if len(ipAddresses) < 2 {
		return nil, fmt.Errorf("at least two IP addresses are required for conflict detection")
	}

	// Parse and validate IP addresses or networks
	ipNets := make([]*net.IPNet, 0, len(ipAddresses))
	ipNetStrings := make([]string, 0, len(ipAddresses))
	isIPv4 := true

	for _, addrStr := range ipAddresses {
		// Ensure the address has CIDR notation
		if !strings.Contains(addrStr, "/") {
			// Assume it's a single IP and add the max prefix
			if net.ParseIP(addrStr).To4() != nil {
				addrStr = addrStr + "/32"
			} else {
				addrStr = addrStr + "/128"
				isIPv4 = false
			}
		}

		_, ipNet, err := net.ParseCIDR(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid IP address or CIDR: %s (%s)", addrStr, err.Error())
		}

		// Check for IP version consistency
		if (ipNet.IP.To4() != nil) != isIPv4 {
			return nil, fmt.Errorf("mixing IPv4 and IPv6 addresses is not supported")
		}

		ipNets = append(ipNets, ipNet)
		ipNetStrings = append(ipNetStrings, addrStr)
	}

	// Find conflicts
	conflicts := make([]map[string]interface{}, 0)

	for i := 0; i < len(ipNets); i++ {
		for j := i + 1; j < len(ipNets); j++ {
			// Check if networks overlap
			if networksOverlap(ipNets[i], ipNets[j]) {
				conflict := map[string]interface{}{
					"network1": ipNetStrings[i],
					"network2": ipNetStrings[j],
					"overlap":  true,
					"details":  fmt.Sprintf("%s overlaps with %s", ipNetStrings[i], ipNetStrings[j]),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	// Build the response
	result := map[string]interface{}{
		"action":         "conflict_detect",
		"input_networks": ipNetStrings,
		"conflicts":      conflicts,
		"conflict_count": len(conflicts),
		"has_conflicts":  len(conflicts) > 0,
		"networks_count": len(ipNetStrings),
		"ip_version":     map[bool]string{true: "IPv4", false: "IPv6"}[isIPv4],
		"timestamp":      timestamp,
	}

	return result, nil
}

// networksOverlap checks if two networks have any overlapping IP addresses
func networksOverlap(n1, n2 *net.IPNet) bool {
	// If one network contains the IP of the other, they overlap
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}

// ExecuteAdapter adapts the input from the dashboard to the format expected by the plugin
func executeAdapter(params map[string]interface{}) (interface{}, error) {
	// Copy the parameters to avoid modifying the original
	adaptedParams := make(map[string]interface{})
	for k, v := range params {
		adaptedParams[k] = v
	}

	// Handle action
	action, _ := params["action"].(string)
	adaptedParams["action"] = action

	// Handle the IP list for supernet, aggregate, and conflict_detect actions
	if action == "supernet" || action == "aggregate" || action == "conflict_detect" {
		ipListStr, _ := params["ip_list"].(string)
		if ipListStr != "" {
			// Split the comma-separated string into a slice
			ipList := strings.Split(ipListStr, ",")
			// Convert to []interface{} as required by Execute
			ipListInterface := make([]interface{}, len(ipList))
			for i, ip := range ipList {
				ipListInterface[i] = strings.TrimSpace(ip)
			}
			adaptedParams["ip_list"] = ipListInterface
		}
	}

	// Call the original Execute function
	return Execute(adaptedParams)
}

// Plugin registers this plugin with the plugin system
// This is called by the plugin loader
func Plugin() interface{} {
	return map[string]interface{}{
		"id":          "subnet_calculator",
		"name":        "Subnet Calculator",
		"description": "IP subnet calculator for analyzing network addresses, calculating subnet masks, and planning network segmentation",
		"version":     "1.0.0",
		"execute":     executeAdapter,
	}
}

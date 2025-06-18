package device_discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Execute handles the device discovery plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	networkRange, ok := params["network_range"].(string)
	if !ok || networkRange == "" {
		networkRange = "192.168.1.0/24" // Default network range
	}

	scanTimeoutFloat, ok := params["scan_timeout"].(float64)
	if !ok {
		scanTimeoutFloat = 30 // Default timeout in seconds
	}
	scanTimeout := time.Duration(scanTimeoutFloat) * time.Second

	resolveHostnames, ok := params["resolve_hostnames"].(bool)
	if !ok {
		resolveHostnames = true // Default to resolve hostnames
	}

	identifyDevices, ok := params["identify_devices"].(bool)
	if !ok {
		identifyDevices = true // Default to identify device types
	}

	// Initialize result structure
	result := map[string]interface{}{
		"network_range":     networkRange,
		"scan_timeout":      scanTimeoutFloat,
		"resolve_hostnames": resolveHostnames,
		"identify_devices":  identifyDevices,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	// First try native methods for discovery
	devices, err := discoverDevices(networkRange, scanTimeout, resolveHostnames, identifyDevices)
	if err != nil {
		result["error"] = fmt.Sprintf("Device discovery failed: %s", err.Error())

		// Try fallback method using system tools
		fallbackDevices, fallbackErr := discoverDevicesFallback(networkRange, scanTimeout)
		if fallbackErr != nil {
			result["fallback_error"] = fallbackErr.Error()
		} else {
			devices = fallbackDevices
		}
	}

	// Add devices to result
	result["devices"] = devices
	result["device_count"] = len(devices)

	return result, nil
}

// discoverDevices discovers devices on the network using native Go methods
func discoverDevices(networkRange string, scanTimeout time.Duration, resolveHostnames, identifyDevices bool) ([]map[string]interface{}, error) {
	// Parse CIDR notation
	_, ipnet, err := net.ParseCIDR(networkRange)
	if err != nil {
		return nil, fmt.Errorf("invalid network range: %w", err)
	}

	// Get all IP addresses in the range
	ips, err := getAllIPsInRange(ipnet)
	if err != nil {
		return nil, fmt.Errorf("failed to determine IP range: %w", err)
	}

	// Get current ARP table for quick lookups
	arpTable, _ := getARPTable()

	// Scan for devices
	var devices []map[string]interface{}
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	// Limit concurrency to avoid overwhelming the network
	semaphore := make(chan struct{}, 20)

	// Start scanning
	for _, ip := range ips {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire

		go func(ip string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release

			// Check if context is canceled
			select {
			case <-ctx.Done():
				return
			default:
				// Continue
			}

			// First check if we already have the MAC from ARP table
			macFromARP := arpTable[ip]

			// If not in ARP table, ping to populate ARP table
			if macFromARP == "" {
				pingCtx, pingCancel := context.WithTimeout(ctx, 1*time.Second)
				defer pingCancel()
				pingCmd := exec.CommandContext(pingCtx, "ping", "-c", "1", "-W", "1", ip)
				_ = pingCmd.Run()

				// Check ARP table again
				arpTable, _ = getARPTable()
				macFromARP = arpTable[ip]
			}

			// Skip if not alive and no MAC address
			if macFromARP == "" {
				return
			}

			// Create device entry
			device := map[string]interface{}{
				"ip":          ip,
				"mac_address": macFromARP,
			}

			// Resolve hostname if requested
			if resolveHostnames {
				if hostname, err := resolveHostname(ip); err == nil && hostname != "" {
					device["hostname"] = hostname
				}
			}

			// Identify device type if requested
			if identifyDevices {
				deviceType, err := identifyDeviceType(ip)
				if err == nil && deviceType != "" {
					device["device_type"] = deviceType
				}
			}

			// Add vendor information based on MAC address
			if vendor := lookupVendor(macFromARP); vendor != "" {
				device["vendor"] = vendor
			}

			// Add to devices list
			mutex.Lock()
			devices = append(devices, device)
			mutex.Unlock()
		}(ip)
	}

	// Wait for all scans to complete or timeout
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// All scans completed
	case <-ctx.Done():
		// Timeout occurred
	}

	return devices, nil
}

// discoverDevicesFallback discovers devices using system tools like arp-scan
func discoverDevicesFallback(networkRange string, scanTimeout time.Duration) ([]map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	// Try arp-scan first
	var devices []map[string]interface{}

	// Check if arp-scan is available
	arpScanCmd := exec.CommandContext(ctx, "which", "arp-scan")
	if arpScanErr := arpScanCmd.Run(); arpScanErr == nil {
		// Use arp-scan
		cmd := exec.CommandContext(ctx, "sudo", "arp-scan", "--localnet")
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Parse arp-scan output
			lines := strings.Split(string(output), "\n")
			ipMacRegex := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)\s+([0-9a-f:]{17})\s+(.*)`)

			for _, line := range lines {
				matches := ipMacRegex.FindStringSubmatch(line)
				if len(matches) >= 3 {
					device := map[string]interface{}{
						"ip":          matches[1],
						"mac_address": matches[2],
					}

					if len(matches) > 3 && matches[3] != "" {
						device["vendor"] = strings.TrimSpace(matches[3])
					}

					// Try to get hostname
					if hostname, err := resolveHostname(matches[1]); err == nil && hostname != "" {
						device["hostname"] = hostname
					}

					devices = append(devices, device)
				}
			}

			return devices, nil
		}
	}

	// If arp-scan failed, try nmap
	nmapCmd := exec.CommandContext(ctx, "which", "nmap")
	if nmapErr := nmapCmd.Run(); nmapErr == nil {
		// Use nmap
		cmd := exec.CommandContext(ctx, "nmap", "-sP", networkRange)
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Parse nmap output
			lines := strings.Split(string(output), "\n")
			ipRegex := regexp.MustCompile(`Nmap scan report for ([^\s]+) \((\d+\.\d+\.\d+\.\d+)\)`)
			macRegex := regexp.MustCompile(`MAC Address: ([0-9A-F:]{17}) \(([^)]+)\)`)

			var currentIP, currentHostname string
			for _, line := range lines {
				// Check for IP/hostname
				ipMatches := ipRegex.FindStringSubmatch(line)
				if len(ipMatches) >= 3 {
					currentIP = ipMatches[2]
					currentHostname = ipMatches[1]

					// If hostname is the same as IP, set it to empty
					if currentHostname == currentIP {
						currentHostname = ""
					}

					// Create device
					device := map[string]interface{}{
						"ip": currentIP,
					}

					if currentHostname != "" {
						device["hostname"] = currentHostname
					}

					devices = append(devices, device)
					continue
				}

				// Check for MAC address
				macMatches := macRegex.FindStringSubmatch(line)
				if len(macMatches) >= 3 && currentIP != "" {
					// Find the device we just added
					for i := len(devices) - 1; i >= 0; i-- {
						if devices[i]["ip"] == currentIP {
							devices[i]["mac_address"] = macMatches[1]
							devices[i]["vendor"] = macMatches[2]
							break
						}
					}
				}
			}

			return devices, nil
		}
	}

	// If all else fails, just use the ARP table
	arpTable, err := getARPTable()
	if err != nil {
		return nil, fmt.Errorf("fallback methods failed, could not get ARP table: %w", err)
	}

	for ip, mac := range arpTable {
		device := map[string]interface{}{
			"ip":          ip,
			"mac_address": mac,
		}

		// Try to get hostname
		if hostname, err := resolveHostname(ip); err == nil && hostname != "" {
			device["hostname"] = hostname
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// getAllIPsInRange gets all IP addresses in a CIDR range
func getAllIPsInRange(ipnet *net.IPNet) ([]string, error) {
	var ips []string

	// Convert IPNet to a range
	ip := ipnet.IP.Mask(ipnet.Mask)

	// Skip network and broadcast addresses for /24 and larger
	skipFirst := false
	skipLast := false

	ones, bits := ipnet.Mask.Size()
	if ones <= 24 {
		skipFirst = true
		skipLast = true
	}

	// Increment IP until we reach the end of the range
	for {
		if !ipnet.Contains(ip) {
			break
		}

		// Skip the first address (network address)
		if skipFirst && isAllZeros(ip, ipnet.Mask) {
			ip = incrementIP(ip)
			continue
		}

		// Skip the last address (broadcast address)
		if skipLast && isAllOnes(ip, ipnet.Mask) {
			break
		}

		ips = append(ips, ip.String())

		// Increment IP for next iteration
		ip = incrementIP(ip)

		// Check if we've wrapped around (for small networks this is unnecessary but safeguard)
		if len(ips) > (1 << uint(bits-ones)) {
			break
		}
	}

	return ips, nil
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) net.IP {
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++
		if newIP[i] > 0 {
			break
		}
	}

	return newIP
}

// isAllZeros checks if the host part of the IP is all zeros (network address)
func isAllZeros(ip net.IP, mask net.IPMask) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i]&^mask[i] != 0 {
			return false
		}
	}
	return true
}

// isAllOnes checks if the host part of the IP is all ones (broadcast address)
func isAllOnes(ip net.IP, mask net.IPMask) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i]|mask[i] != 0xFF {
			return false
		}
	}
	return true
}

// getARPTable gets the current ARP table
func getARPTable() (map[string]string, error) {
	arpTable := make(map[string]string)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute arp command
	cmd := exec.CommandContext(ctx, "arp", "-n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return arpTable, fmt.Errorf("failed to get ARP table: %w", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	ipMacRegex := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+).*?([0-9a-f:]{17})`)

	for _, line := range lines {
		matches := ipMacRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			arpTable[matches[1]] = matches[2]
		}
	}

	return arpTable, nil
}

// resolveHostname attempts to resolve the hostname for an IP address
func resolveHostname(ip string) (string, error) {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "", err
	}

	// Remove trailing dot if present
	hostname := names[0]
	if hostname[len(hostname)-1] == '.' {
		hostname = hostname[:len(hostname)-1]
	}

	return hostname, nil
}

// identifyDeviceType attempts to identify the device type based on open ports
func identifyDeviceType(ip string) (string, error) {
	// Common ports and their associated device types
	portDeviceMap := map[int]string{
		22:   "Linux/SSH Server",
		23:   "Telnet Device",
		80:   "Web Server/IoT Device",
		443:  "Web Server/IoT Device",
		445:  "Windows/SMB Server",
		8080: "Web Server/Proxy",
		8443: "Web Server (HTTPS)",
		5000: "IoT Device",
		9999: "IoT Device",
		1883: "MQTT Broker",
		5353: "mDNS Device",
		5900: "VNC Server",
		3389: "Windows/RDP Server",
		53:   "DNS Server",
		67:   "DHCP Server",
		68:   "DHCP Client",
		161:  "SNMP Device",
		21:   "FTP Server",
		25:   "SMTP Server",
		110:  "POP3 Server",
		143:  "IMAP Server",
		993:  "IMAP SSL Server",
		995:  "POP3 SSL Server",
		631:  "Printer/CUPS Server",
		9100: "Printer (Raw)",
		139:  "NetBIOS",
		1900: "UPnP Device",
	}

	// Check a few key ports to identify device type
	portsToCheck := []int{80, 443, 22, 23, 445, 8080, 5000, 9999, 1883, 5353, 161, 139, 1900, 631}

	var openPorts []int

	// Limit the scan time to 3 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check each port
	for _, port := range portsToCheck {
		address := fmt.Sprintf("%s:%d", ip, port)
		dialer := net.Dialer{Timeout: 500 * time.Millisecond}

		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			conn.Close()
			openPorts = append(openPorts, port)
		}
	}

	// Identify device type based on open ports
	if len(openPorts) == 0 {
		return "Unknown", nil
	}

	// Try to determine device type based on open ports
	if contains(openPorts, 80) || contains(openPorts, 443) {
		if contains(openPorts, 8080) || contains(openPorts, 8443) {
			return "Web Server", nil
		} else if contains(openPorts, 1883) {
			return "IoT Gateway", nil
		} else if contains(openPorts, 631) || contains(openPorts, 9100) {
			return "Printer", nil
		} else if contains(openPorts, 5000) || contains(openPorts, 9999) {
			return "IoT Device", nil
		}
		return "Web Server/IoT Device", nil
	} else if contains(openPorts, 22) {
		return "Linux/SSH Server", nil
	} else if contains(openPorts, 445) || contains(openPorts, 139) {
		if contains(openPorts, 3389) {
			return "Windows Server", nil
		}
		return "Windows/SMB Device", nil
	} else if contains(openPorts, 23) {
		return "Network Equipment", nil
	} else if contains(openPorts, 161) {
		return "Network Equipment", nil
	} else if contains(openPorts, 5353) || contains(openPorts, 1900) {
		return "IoT/Media Device", nil
	} else if contains(openPorts, 67) || contains(openPorts, 53) {
		return "Router/Network Infrastructure", nil
	}

	// If we can't determine, return the first known device type
	for _, port := range openPorts {
		if deviceType, ok := portDeviceMap[port]; ok {
			return deviceType, nil
		}
	}

	return "Unknown Device", nil
}

// lookupVendor attempts to identify the vendor from a MAC address
func lookupVendor(mac string) string {
	// Extract the OUI (first 3 bytes) of the MAC address
	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return ""
	}

	// Convert to bytes
	var oui [3]byte
	for i := 0; i < 3; i++ {
		b, err := strconv.ParseInt(parts[i], 16, 8)
		if err != nil {
			return ""
		}
		oui[i] = byte(b)
	}

	// Common OUIs and their vendors (sample list)
	ouiMap := map[uint32]string{
		binary.BigEndian.Uint32(append([]byte{0}, oui[:]...)): "Unknown",
		0x000C29: "VMware",
		0x0050C2: "IEEE Registration Authority",
		0x00005E: "IANA",
		0x0001C7: "Cisco",
		0x000D3A: "Microsoft",
		0x00037F: "Atheros",
		0x0024D7: "Intel",
		0x001A11: "Google",
		0x001801: "Hewlett-Packard",
		0x002608: "Apple",
		0x1060B0: "Huawei",
		0x001D33: "Ubiquiti",
		0x001CDF: "Belkin",
		0x0024F7: "Cisco Meraki",
		0x70E284: "Wistron",
		0x74D02B: "ASUSTek",
		0x9C8E99: "Hewlett Packard Enterprise",
		0xE06995: "Dell",
		0xF01FAF: "Dell",
		0x5C26C1: "Apple",
		0x1865F5: "Apple",
		0x6C4B90: "Liteon",
		0xC8B5AD: "Hewlett Packard Enterprise",
		0x1CE85D: "Cisco",
		0x1CE2CC: "Texas Instruments",
		0x3CEC0F: "Wistron",
		0x74AC5F: "Qiku",
		0x7CC709: "Shenzhen RF",
	}

	// Convert OUI to uint32 for lookup
	ouiInt := uint32(oui[0])<<16 | uint32(oui[1])<<8 | uint32(oui[2])

	if vendor, ok := ouiMap[ouiInt]; ok {
		return vendor
	}

	return ""
}

// contains checks if a slice contains a value
func contains(slice []int, value int) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

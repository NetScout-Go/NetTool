// Package core provides core networking information retrieval functionality.
package core

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

// NetworkInfo represents the complete network information for the device
// Field names match frontend expectations (camelCase)
type NetworkInfo struct {
	// IP Configuration
	Ipv4           string `json:"ipv4"`
	Ipv6           string `json:"ipv6"`
	Subnet         string `json:"subnet"`
	Gateway        string `json:"gateway"`
	DhcpEnabled    bool   `json:"dhcpEnabled"`
	DhcpServer     string `json:"dhcpServer,omitempty"`
	DhcpLeaseStart string `json:"dhcpLeaseStart,omitempty"`
	DhcpLeaseEnd   string `json:"dhcpLeaseEnd,omitempty"`

	// Gateway/Router Information
	GatewayMac     string  `json:"gatewayMac"`
	GatewayLatency float64 `json:"gatewayLatency"`
	GatewayVendor  string  `json:"gatewayVendor,omitempty"`
	HopsToInternet int     `json:"hopsToInternet"`

	// DNS
	DnsServers []string `json:"dnsServers"`

	// Interface Information
	InterfaceName string `json:"interfaceName"`
	MacAddress    string `json:"macAddress"`
	LinkSpeed     string `json:"linkSpeed"`
	Mtu           int    `json:"mtu"`
	Duplex        string `json:"duplex"`

	// Connection Status
	ConnectionType string  `json:"connectionType"` // "Ethernet" or "WiFi"
	Connected      bool    `json:"connected"`
	Uptime         int64   `json:"uptime"`
	Latency        float64 `json:"latency"`
	PacketLoss     float64 `json:"packetLoss"`

	// WiFi specific (if applicable)
	Ssid           string `json:"ssid,omitempty"`
	SignalStrength int    `json:"signalStrength,omitempty"`
	WifiChannel    int    `json:"wifiChannel,omitempty"`
	WifiFrequency  string `json:"wifiFrequency,omitempty"`
	WifiBssid      string `json:"wifiBssid,omitempty"`

	// Switch/Network Detection
	SwitchDetected bool   `json:"switchDetected"`
	SwitchPort     string `json:"switchPort,omitempty"`
	SwitchInfo     string `json:"switchInfo,omitempty"`

	// VLAN Information
	VlanEnabled bool   `json:"vlanEnabled"`
	VlanId      int    `json:"vlanId,omitempty"`
	VlanName    string `json:"vlanName,omitempty"`

	// Traffic Statistics (kept for compatibility but not shown on simplified dashboard)
	BytesReceived   int64 `json:"bytesReceived"`
	BytesSent       int64 `json:"bytesSent"`
	PacketsReceived int64 `json:"packetsReceived"`
	PacketsSent     int64 `json:"packetsSent"`

	// Additional Network Info
	PublicIp      string `json:"publicIp,omitempty"`
	NetworkPrefix string `json:"networkPrefix,omitempty"`
	BroadcastAddr string `json:"broadcastAddr,omitempty"`

	// NAT Detection
	NatType        string `json:"natType,omitempty"`        // "None", "Single NAT", "Double NAT", "CGNAT", "Unknown"
	NatLayers      int    `json:"natLayers"`                // Number of NAT layers detected
	BehindNat      bool   `json:"behindNat"`                // Whether behind NAT
	BehindCgnat    bool   `json:"behindCgnat"`              // Carrier-Grade NAT detected
	DoubleNat      bool   `json:"doubleNat"`                // Multiple NAT layers
	NatGatewayIp   string `json:"natGatewayIp,omitempty"`   // First NAT gateway IP
	ExternalRouter string `json:"externalRouter,omitempty"` // External router if double NAT

	Timestamp time.Time `json:"timestamp"`
}

// GetNetworkInfo retrieves comprehensive network information
func GetNetworkInfo() (*NetworkInfo, error) {
	info := &NetworkInfo{
		Timestamp:  time.Now(),
		DnsServers: make([]string, 0),
	}

	// Find and analyze primary interface
	iface, counter := findPrimaryInterface()
	if iface == nil {
		info.Connected = false
		info.ConnectionType = "Unknown"
		return info, nil
	}

	// Extract IP configuration
	extractIPConfig(iface, info)

	// Get gateway information
	info.Gateway = getDefaultGateway()

	// Determine connection type
	if isWirelessInterface(iface.Name) {
		info.ConnectionType = "WiFi"
		extractWifiInfo(iface.Name, info)
	} else {
		info.ConnectionType = "Ethernet"
	}

	// Interface details
	info.InterfaceName = iface.Name
	info.MacAddress = formatMACAddress(iface.HardwareAddr.String())
	info.Mtu = iface.MTU
	info.LinkSpeed = getLinkSpeed(iface.Name)
	info.Duplex = getLinkDuplex(iface.Name)

	// Connection status
	info.Connected = info.Ipv4 != "" || info.Ipv6 != ""
	info.Uptime = getSystemUptime()

	// DNS servers
	info.DnsServers = getDNSServers()

	// DHCP information
	extractDHCPInfo(iface.Name, info)

	// Gateway MAC and latency (do in parallel for speed)
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		if info.Gateway != "" && info.Gateway != "N/A" {
			info.GatewayMac = getGatewayMAC(info.Gateway)
		}
	}()

	go func() {
		defer wg.Done()
		if info.Gateway != "" && info.Gateway != "N/A" {
			info.GatewayLatency = measureGatewayLatency(info.Gateway)
		}
	}()

	go func() {
		defer wg.Done()
		info.Latency, info.PacketLoss = measureInternetConnectivity()
	}()

	go func() {
		defer wg.Done()
		info.HopsToInternet = countHopsToInternet()
	}()

	go func() {
		defer wg.Done()
		info.PublicIp = getPublicIP()
	}()

	wg.Wait()

	// NAT detection (needs public IP to be fetched first)
	detectNAT(info)

	// VLAN detection
	detectVLAN(iface.Name, info)

	// Switch/LLDP detection (if available)
	detectSwitch(iface.Name, info)

	// Traffic statistics
	if counter != nil {
		info.BytesReceived = int64(counter.BytesRecv)
		info.BytesSent = int64(counter.BytesSent)
		info.PacketsReceived = int64(counter.PacketsRecv)
		info.PacketsSent = int64(counter.PacketsSent)
	}

	// Calculate network prefix and broadcast
	if info.Ipv4 != "" && info.Subnet != "" {
		info.NetworkPrefix = calculateNetworkPrefix(info.Ipv4, info.Subnet)
		info.BroadcastAddr = calculateBroadcast(info.Ipv4, info.Subnet)
	}

	return info, nil
}

func findPrimaryInterface() (*net.Interface, *psnet.IOCountersStat) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}

	counters, _ := psnet.IOCounters(true)
	counterMap := make(map[string]psnet.IOCountersStat)
	for _, c := range counters {
		counterMap[c.Name] = c
	}

	// Priority: interfaces with default route, then by traffic
	gateway := getDefaultGateway()
	var gatewayIface *net.Interface

	for i := range ifaces {
		iface := &ifaces[i]

		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip virtual/container interfaces
		if isVirtualInterface(iface.Name) {
			continue
		}

		// Check if this interface has the gateway
		if gateway != "" && gateway != "N/A" {
			if hasRouteToGateway(iface.Name, gateway) {
				gatewayIface = iface
				break
			}
		}

		// Fallback: first interface with an IP
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				if gatewayIface == nil {
					gatewayIface = iface
				}
			}
		}
	}

	if gatewayIface != nil {
		if c, ok := counterMap[gatewayIface.Name]; ok {
			return gatewayIface, &c
		}
		return gatewayIface, nil
	}

	return nil, nil
}

func isVirtualInterface(name string) bool {
	virtPrefixes := []string{"docker", "br-", "veth", "virbr", "vnet", "tun", "tap", "lo"}
	for _, prefix := range virtPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func hasRouteToGateway(ifaceName, gateway string) bool {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == ifaceName && fields[1] == "00000000" {
			return true
		}
	}
	return false
}

func extractIPConfig(iface *net.Interface, info *NetworkInfo) {
	addrs, err := iface.Addrs()
	if err != nil {
		return
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ip := ipNet.IP.To4(); ip != nil && info.Ipv4 == "" {
				info.Ipv4 = ip.String()
				ones, _ := ipNet.Mask.Size()
				info.Subnet = cidrToSubnet(ones)
			} else if ip == nil && info.Ipv6 == "" {
				// Skip link-local IPv6 for display
				if !ipNet.IP.IsLinkLocalUnicast() {
					info.Ipv6 = ipNet.IP.String()
				}
			}
		}
	}

	// If no global IPv6, use link-local
	if info.Ipv6 == "" {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() == nil {
				info.Ipv6 = ipNet.IP.String()
				break
			}
		}
	}
}

func getDefaultGateway() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}

		value, err := strconv.ParseUint(fields[2], 16, 32)
		if err != nil {
			continue
		}

		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(value))
		ip := net.IP(b)
		if !ip.Equal(net.IPv4zero) {
			return ip.String()
		}
	}

	return ""
}

func getDNSServers() []string {
	var servers []string

	// Try systemd-resolved first
	if data, err := exec.Command("resolvectl", "status").Output(); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "DNS Servers:") || strings.Contains(line, "Current DNS Server:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					ip := strings.TrimSpace(parts[1])
					if ip != "" && net.ParseIP(ip) != nil {
						servers = append(servers, ip)
					}
				}
			}
		}
	}

	// Fallback to resolv.conf
	if len(servers) == 0 {
		data, err := os.ReadFile("/etc/resolv.conf")
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "nameserver") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						servers = append(servers, fields[1])
					}
				}
			}
		}
	}

	return servers
}

func extractDHCPInfo(ifaceName string, info *NetworkInfo) {
	// Check for DHCP lease files
	leaseFiles := []string{
		"/var/lib/dhcp/dhclient." + ifaceName + ".leases",
		"/var/lib/dhcp/dhclient.leases",
		"/var/lib/dhclient/dhclient." + ifaceName + ".leases",
		"/var/lib/NetworkManager/" + ifaceName + ".lease",
	}

	for _, leaseFile := range leaseFiles {
		data, err := os.ReadFile(leaseFile)
		if err != nil {
			continue
		}

		info.DhcpEnabled = true
		content := string(data)

		// Parse DHCP server
		if idx := strings.Index(content, "dhcp-server-identifier"); idx != -1 {
			line := content[idx:]
			if end := strings.Index(line, ";"); end != -1 {
				parts := strings.Fields(line[:end])
				if len(parts) >= 2 {
					info.DhcpServer = parts[1]
				}
			}
		}
		return
	}

	// Check NetworkManager connection
	cmd := exec.Command("nmcli", "-t", "-f", "IP4.ADDRESS,IP4.GATEWAY,DHCP4.OPTION", "device", "show", ifaceName)
	if output, err := cmd.Output(); err == nil {
		if strings.Contains(string(output), "dhcp_server_identifier") {
			info.DhcpEnabled = true
		}
	}

	// Assume DHCP if we have a gateway
	if info.Gateway != "" {
		info.DhcpEnabled = true
	}
}

func getGatewayMAC(gateway string) string {
	// First try ARP cache
	data, err := os.ReadFile("/proc/net/arp")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[0] == gateway {
				mac := fields[3]
				if mac != "00:00:00:00:00:00" {
					return formatMACAddress(mac)
				}
			}
		}
	}

	// Try ip neigh
	cmd := exec.Command("ip", "neigh", "show", gateway)
	output, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(output))
		for i, f := range fields {
			if f == "lladdr" && i+1 < len(fields) {
				return formatMACAddress(fields[i+1])
			}
		}
	}

	// Ping to populate ARP cache, then retry
	exec.Command("ping", "-c", "1", "-W", "1", gateway).Run()
	time.Sleep(100 * time.Millisecond)

	data, err = os.ReadFile("/proc/net/arp")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[0] == gateway {
				mac := fields[3]
				if mac != "00:00:00:00:00:00" {
					return formatMACAddress(mac)
				}
			}
		}
	}

	return ""
}

func formatMACAddress(mac string) string {
	// Ensure consistent uppercase format
	return strings.ToUpper(strings.ReplaceAll(mac, "-", ":"))
}

func measureGatewayLatency(gateway string) float64 {
	// Use ICMP ping for gateway latency
	cmd := exec.Command("ping", "-c", "3", "-W", "1", gateway)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to TCP
		return measureTCPLatency(gateway, "53")
	}

	// Parse ping output for avg latency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "rtt") || strings.Contains(line, "round-trip") {
			// Format: rtt min/avg/max/mdev = 0.123/0.456/0.789/0.012 ms
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				stats := strings.Split(strings.TrimSpace(parts[1]), "/")
				if len(stats) >= 2 {
					if avg, err := strconv.ParseFloat(stats[1], 64); err == nil {
						return avg
					}
				}
			}
		}
	}

	return 0
}

func measureTCPLatency(host, port string) float64 {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return 0
	}
	latency := float64(time.Since(start).Milliseconds())
	conn.Close()
	return latency
}

func measureInternetConnectivity() (float64, float64) {
	targets := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
	attempts := 4
	var totalLatency float64
	var successCount int

	for _, target := range targets {
		cmd := exec.Command("ping", "-c", strconv.Itoa(attempts), "-W", "1", target)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse results
		outputStr := string(output)

		// Get packet stats
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.Contains(line, "packets transmitted") {
				// Format: 4 packets transmitted, 4 received, 0% packet loss
				parts := strings.Split(line, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.Contains(part, "received") {
						fields := strings.Fields(part)
						if len(fields) >= 1 {
							if recv, err := strconv.Atoi(fields[0]); err == nil {
								successCount += recv
							}
						}
					}
				}
			}

			// Get avg latency
			if strings.Contains(line, "rtt") || strings.Contains(line, "round-trip") {
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					stats := strings.Split(strings.TrimSpace(parts[1]), "/")
					if len(stats) >= 2 {
						if avg, err := strconv.ParseFloat(stats[1], 64); err == nil {
							totalLatency += avg
						}
					}
				}
			}
		}
		break // Use first successful target
	}

	totalAttempts := attempts
	if successCount == 0 {
		return 0, 100.0
	}

	avgLatency := totalLatency
	packetLoss := float64(totalAttempts-successCount) / float64(totalAttempts) * 100

	return avgLatency, packetLoss
}

func countHopsToInternet() int {
	// Quick traceroute to 8.8.8.8
	cmd := exec.Command("traceroute", "-n", "-m", "15", "-w", "1", "-q", "1", "8.8.8.8")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative
		cmd = exec.Command("tracepath", "-n", "-m", "15", "8.8.8.8")
		output, err = cmd.Output()
		if err != nil {
			return 0
		}
	}

	lines := strings.Split(string(output), "\n")
	hops := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Check if first field is a hop number
			if _, err := strconv.Atoi(fields[0]); err == nil {
				hops++
				// Check if we reached 8.8.8.8
				if strings.Contains(line, "8.8.8.8") {
					return hops
				}
			}
		}
	}

	return hops
}

func getLinkSpeed(ifaceName string) string {
	// Try reading from sysfs
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", ifaceName)
	data, err := os.ReadFile(speedPath)
	if err == nil {
		speed := strings.TrimSpace(string(data))
		if speedInt, err := strconv.Atoi(speed); err == nil && speedInt > 0 {
			if speedInt >= 1000 {
				return fmt.Sprintf("%d Gbps", speedInt/1000)
			}
			return fmt.Sprintf("%d Mbps", speedInt)
		}
	}

	// Try ethtool
	cmd := exec.Command("ethtool", ifaceName)
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Speed:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return "Unknown"
}

func getLinkDuplex(ifaceName string) string {
	// Try reading from sysfs
	duplexPath := fmt.Sprintf("/sys/class/net/%s/duplex", ifaceName)
	data, err := os.ReadFile(duplexPath)
	if err == nil {
		duplex := strings.TrimSpace(string(data))
		if duplex != "" && duplex != "unknown" {
			return strings.ToUpper(duplex[:1]) + duplex[1:]
		}
	}

	// Try ethtool
	cmd := exec.Command("ethtool", ifaceName)
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Duplex:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return "Full"
}

func isWirelessInterface(ifaceName string) bool {
	// Check sysfs for wireless
	wirelessPath := fmt.Sprintf("/sys/class/net/%s/wireless", ifaceName)
	if _, err := os.Stat(wirelessPath); err == nil {
		return true
	}

	// Check phy80211
	phyPath := fmt.Sprintf("/sys/class/net/%s/phy80211", ifaceName)
	if _, err := os.Stat(phyPath); err == nil {
		return true
	}

	// Naming convention
	return strings.HasPrefix(ifaceName, "wlan") || strings.HasPrefix(ifaceName, "wlp") || strings.HasPrefix(ifaceName, "wl")
}

func extractWifiInfo(ifaceName string, info *NetworkInfo) {
	// Get SSID using iw
	cmd := exec.Command("iw", "dev", ifaceName, "link")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID:") {
				info.Ssid = strings.TrimPrefix(line, "SSID:")
				info.Ssid = strings.TrimSpace(info.Ssid)
			}
			if strings.HasPrefix(line, "signal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if sig, err := strconv.Atoi(parts[1]); err == nil {
						info.SignalStrength = sig
					}
				}
			}
			if strings.HasPrefix(line, "freq:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					info.WifiFrequency = parts[1] + " MHz"
					if freq, err := strconv.Atoi(parts[1]); err == nil {
						if freq >= 5000 {
							info.WifiFrequency = parts[1] + " MHz (5 GHz)"
						} else {
							info.WifiFrequency = parts[1] + " MHz (2.4 GHz)"
						}
					}
				}
			}
		}
	}

	// Fallback to iwconfig
	if info.Ssid == "" {
		cmd = exec.Command("iwconfig", ifaceName)
		output, err = cmd.Output()
		if err == nil {
			outputStr := string(output)

			// ESSID
			if idx := strings.Index(outputStr, "ESSID:\""); idx != -1 {
				start := idx + 7
				end := strings.Index(outputStr[start:], "\"")
				if end != -1 {
					info.Ssid = outputStr[start : start+end]
				}
			}

			// Signal level
			if idx := strings.Index(outputStr, "Signal level="); idx != -1 {
				start := idx + 13
				end := strings.IndexAny(outputStr[start:], " \n")
				if end != -1 {
					sigStr := strings.TrimSuffix(outputStr[start:start+end], "dBm")
					if sig, err := strconv.Atoi(sigStr); err == nil {
						info.SignalStrength = sig
					}
				}
			}
		}
	}
}

func detectVLAN(ifaceName string, info *NetworkInfo) {
	// Check interface name for VLAN suffix
	if idx := strings.LastIndex(ifaceName, "."); idx != -1 {
		vlanStr := ifaceName[idx+1:]
		if vlanID, err := strconv.Atoi(vlanStr); err == nil {
			info.VlanEnabled = true
			info.VlanId = vlanID
			info.VlanName = fmt.Sprintf("VLAN %d", vlanID)
		}
	}

	// Check for 802.1Q VLAN
	vlanPath := fmt.Sprintf("/proc/net/vlan/%s", ifaceName)
	if _, err := os.Stat(vlanPath); err == nil {
		info.VlanEnabled = true
	}
}

func detectSwitch(ifaceName string, info *NetworkInfo) {
	// Try LLDP (Link Layer Discovery Protocol)
	cmd := exec.Command("lldpctl", "-f", "keyvalue", ifaceName)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		info.SwitchDetected = true
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "port.descr=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					info.SwitchPort = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "chassis.name=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					info.SwitchInfo = strings.TrimSpace(parts[1])
				}
			}
		}
		return
	}

	// Try lldpcli
	cmd = exec.Command("lldpcli", "show", "neighbors", "ports", ifaceName, "summary")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 && !strings.Contains(string(output), "No neighbor") {
		info.SwitchDetected = true
		info.SwitchInfo = "Switch detected via LLDP"
	}
}

func getSystemUptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}

	uptimeFloat, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	return int64(uptimeFloat)
}

func calculateNetworkPrefix(ip, subnet string) string {
	ipAddr := net.ParseIP(ip)
	subnetMask := net.ParseIP(subnet)
	if ipAddr == nil || subnetMask == nil {
		return ""
	}

	ipAddr = ipAddr.To4()
	subnetMask = subnetMask.To4()
	if ipAddr == nil || subnetMask == nil {
		return ""
	}

	network := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		network[i] = ipAddr[i] & subnetMask[i]
	}

	ones, _ := net.IPMask(subnetMask).Size()
	return fmt.Sprintf("%s/%d", network.String(), ones)
}

func calculateBroadcast(ip, subnet string) string {
	ipAddr := net.ParseIP(ip)
	subnetMask := net.ParseIP(subnet)
	if ipAddr == nil || subnetMask == nil {
		return ""
	}

	ipAddr = ipAddr.To4()
	subnetMask = subnetMask.To4()
	if ipAddr == nil || subnetMask == nil {
		return ""
	}

	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ipAddr[i] | ^subnetMask[i]
	}

	return broadcast.String()
}

func cidrToSubnet(ones int) string {
	if ones < 0 || ones > 32 {
		return "255.255.255.0"
	}
	mask := net.CIDRMask(ones, 32)
	return net.IP(mask).String()
}

// getPublicIP fetches the public IP from external services
func getPublicIP() string {
	// List of public IP services (try multiple for reliability)
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ipinfo.io/ip",
		"https://checkip.amazonaws.com",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			ip := strings.TrimSpace(string(body))
			// Validate it's an IP
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	return ""
}

// detectNAT analyzes NAT configuration and detects double NAT / CGNAT
func detectNAT(info *NetworkInfo) {
	// No public IP means we couldn't determine NAT status
	if info.PublicIp == "" {
		info.NatType = "Unknown"
		return
	}

	// Check if local IP equals public IP (no NAT)
	if info.Ipv4 == info.PublicIp {
		info.NatType = "None"
		info.BehindNat = false
		info.NatLayers = 0
		return
	}

	// We're behind NAT
	info.BehindNat = true
	info.NatGatewayIp = info.Gateway

	// Check for CGNAT (Carrier-Grade NAT)
	// CGNAT uses 100.64.0.0/10 range (RFC 6598)
	if isCGNATRange(info.PublicIp) || isCGNATRange(info.Gateway) {
		info.BehindCgnat = true
		info.NatType = "CGNAT"
		info.NatLayers = 2 // At minimum, CGNAT implies ISP NAT + your router
		return
	}

	// Check if gateway is in private range
	gatewayPrivate := isPrivateIP(info.Gateway)

	// Analyze traceroute for NAT layers
	natLayers, externalRouter := analyzeNATLayers(info.Gateway)
	info.NatLayers = natLayers
	info.ExternalRouter = externalRouter

	// Determine NAT type
	if natLayers > 1 || (gatewayPrivate && externalRouter != "") {
		info.DoubleNat = true
		info.NatType = "Double NAT"
	} else if natLayers == 1 {
		info.NatType = "Single NAT"
	} else {
		info.NatType = "Single NAT"
		info.NatLayers = 1
	}
}

// isCGNATRange checks if IP is in CGNAT range (100.64.0.0/10)
func isCGNATRange(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	ip = ip.To4()
	if ip == nil {
		return false
	}

	// CGNAT range: 100.64.0.0 - 100.127.255.255
	return ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127
}

// isPrivateIP checks if IP is in private ranges
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	ip = ip.To4()
	if ip == nil {
		return false
	}

	// Private ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	if ip[0] == 10 {
		return true
	}
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}

	return false
}

// analyzeNATLayers uses traceroute to detect multiple NAT layers
func analyzeNATLayers(gateway string) (int, string) {
	// Run traceroute to a public IP
	cmd := exec.Command("traceroute", "-n", "-m", "10", "-w", "1", "-q", "1", "8.8.8.8")
	output, err := cmd.Output()
	if err != nil {
		// Try tracepath as fallback
		cmd = exec.Command("tracepath", "-n", "-m", "10", "8.8.8.8")
		output, err = cmd.Output()
		if err != nil {
			return 1, ""
		}
	}

	lines := strings.Split(string(output), "\n")
	var privateHops []string
	var firstPublicHop string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Extract IP from traceroute output
		var hopIP string
		for _, field := range fields {
			if net.ParseIP(field) != nil {
				hopIP = field
				break
			}
		}

		if hopIP == "" || hopIP == "*" {
			continue
		}

		// Skip if it's our gateway (first hop)
		if hopIP == gateway {
			continue
		}

		// Check if this hop is private
		if isPrivateIP(hopIP) || isCGNATRange(hopIP) {
			privateHops = append(privateHops, hopIP)
		} else if firstPublicHop == "" {
			firstPublicHop = hopIP
			break // Stop at first public IP
		}
	}

	// NAT layers = number of private hops before reaching public internet + 1
	natLayers := len(privateHops) + 1

	// External router is the last private hop before public internet
	var externalRouter string
	if len(privateHops) > 0 {
		externalRouter = privateHops[len(privateHops)-1]
	}

	return natLayers, externalRouter
}

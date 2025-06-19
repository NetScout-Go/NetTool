package core

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

// NetworkInfo represents the network information for the device
type NetworkInfo struct {
	IPv4Address    string         `json:"ipv4Address"`
	IPv6Address    string         `json:"ipv6Address"`
	SubnetMask     string         `json:"subnetMask"`
	Gateway        string         `json:"gateway"`
	SSID           string         `json:"ssid,omitempty"`
	EthernetInfo   EthernetInfo   `json:"ethernetInfo,omitempty"`
	DNSServers     []string       `json:"dnsServers"`
	DHCPInfo       DHCPInfo       `json:"dhcpInfo"`
	VLANInfo       VLANInfo       `json:"vlanInfo,omitempty"`
	Connection     Connection     `json:"connection"`
	Traffic        Traffic        `json:"traffic"`
	ARPEntries     []ARPEntry     `json:"arpEntries"`
	ServiceLatency ServiceLatency `json:"serviceLatency"`
	Timestamp      time.Time      `json:"timestamp"`
}

// EthernetInfo represents ethernet connection details
type EthernetInfo struct {
	InterfaceName string `json:"interfaceName"`
	MACAddress    string `json:"macAddress"`
	Speed         string `json:"speed"`
	Duplex        string `json:"duplex"`
}

// DHCPInfo represents DHCP configuration
type DHCPInfo struct {
	Enabled       bool      `json:"enabled"`
	LeaseObtained time.Time `json:"leaseObtained,omitempty"`
	LeaseExpires  time.Time `json:"leaseExpires,omitempty"`
	DHCPServer    string    `json:"dhcpServer,omitempty"`
}

// VLANInfo represents VLAN configuration if applicable
type VLANInfo struct {
	Enabled  bool   `json:"enabled"`
	VLANID   int    `json:"vlanId,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Name     string `json:"name,omitempty"`
}

// Connection represents connection status and metrics
type Connection struct {
	Status         string  `json:"status"` // "connected", "disconnected", "limited"
	Uptime         int64   `json:"uptime"` // in seconds
	LatencyMS      float64 `json:"latencyMs"`
	PacketLoss     float64 `json:"packetLoss"`               // percentage
	SignalStrength int     `json:"signalStrength,omitempty"` // for wireless, in dBm
}

// Traffic represents network traffic statistics
type Traffic struct {
	BytesReceived    int64   `json:"bytesReceived"`
	BytesSent        int64   `json:"bytesSent"`
	PacketsReceived  int64   `json:"packetsReceived"`
	PacketsSent      int64   `json:"packetsSent"`
	CurrentBandwidth float64 `json:"currentBandwidth"` // in Mbps
}

// ARPEntry represents a single entry in the ARP table (IP to MAC mapping)
type ARPEntry struct {
	IPAddress  string `json:"ipAddress"`
	MACAddress string `json:"macAddress"`
	Device     string `json:"device"`
	State      string `json:"state"`
}

// ServiceLatency represents latency measurements to various major services
type ServiceLatency struct {
	Google     float64 `json:"google"`     // Google latency in ms
	Amazon     float64 `json:"amazon"`     // Amazon latency in ms
	Cloudflare float64 `json:"cloudflare"` // Cloudflare latency in ms
	Microsoft  float64 `json:"microsoft"`  // Microsoft latency in ms
	DNS        float64 `json:"dns"`        // DNS latency in ms
	HTTP       float64 `json:"http"`       // HTTP latency in ms
}

// GetNetworkInfo retrieves the current network information
func GetNetworkInfo() (*NetworkInfo, error) {
	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Get IO counters for traffic statistics
	ioCounters, err := psnet.IOCounters(true)
	if err != nil {
		return nil, err
	}

	// For simplicity, we'll focus on the first active interface
	var activeIface net.Interface
	var activeIOCounter psnet.IOCountersStat

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			activeIface = iface

			// Find matching IO counter
			for _, counter := range ioCounters {
				if counter.Name == iface.Name {
					activeIOCounter = counter
					break
				}
			}
			break
		}
	}

	// Get IP addresses
	var ipv4, ipv6, subnet string
	addrs, err := activeIface.Addrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.To4() != nil {
					ipv4 = ipNet.IP.String()
					mask := ipNet.Mask
					ones, _ := mask.Size()
					subnet = cidrToSubnet(ones)
				} else {
					ipv6 = ipNet.IP.String()
				}
			}
		}
	}

	// Create network info object
	networkInfo := &NetworkInfo{
		IPv4Address: ipv4,
		IPv6Address: ipv6,
		SubnetMask:  subnet,
		Gateway:     getDefaultGateway(),
		DNSServers:  getDNSServers(),
		DHCPInfo: DHCPInfo{
			Enabled:    true, // Assumption for simplicity
			DHCPServer: getDHCPServer(),
		},
		EthernetInfo: EthernetInfo{
			InterfaceName: activeIface.Name,
			MACAddress:    activeIface.HardwareAddr.String(),
			Speed:         "1 Gbps", // Placeholder - would need specific system calls to get real values
			Duplex:        "Full",   // Placeholder
		},
		Connection: Connection{
			Status:     "connected", // Assumption
			Uptime:     getUptime(),
			LatencyMS:  getPingLatency(),
			PacketLoss: getPacketLoss(),
		},
		Traffic: Traffic{
			BytesReceived:    int64(activeIOCounter.BytesRecv),
			BytesSent:        int64(activeIOCounter.BytesSent),
			PacketsReceived:  int64(activeIOCounter.PacketsRecv),
			PacketsSent:      int64(activeIOCounter.PacketsSent),
			CurrentBandwidth: calculateBandwidth(activeIOCounter),
		},
		ServiceLatency: ServiceLatency{
			Google:     measureServiceLatency("google.com"),
			Amazon:     measureServiceLatency("amazon.com"),
			Cloudflare: measureServiceLatency("cloudflare.com"),
			Microsoft:  measureServiceLatency("microsoft.com"),
			DNS:        measureDNSLatency(),
			HTTP:       measureHTTPLatency(),
		},
		Timestamp: time.Now(),
	}

	// Check if it's a wireless connection
	if isWireless(activeIface.Name) {
		networkInfo.SSID = getWirelessSSID(activeIface.Name)
		networkInfo.Connection.SignalStrength = getSignalStrength(activeIface.Name)
	}

	// Check for VLAN info
	if strings.Contains(activeIface.Name, ".") {
		networkInfo.VLANInfo = VLANInfo{
			Enabled: true,
			VLANID:  getVLANID(activeIface.Name),
			Name:    "VLAN " + string(activeIface.Name[strings.LastIndex(activeIface.Name, ".")+1:]),
		}
	} else {
		networkInfo.VLANInfo = VLANInfo{
			Enabled: false,
		}
	}

	// Get ARP table entries using 'ip neigh show' instead of 'arp -a'
	arpEntries, err := GetARPTable()
	if err == nil {
		networkInfo.ARPEntries = arpEntries
	}

	// Measure service latencies
	networkInfo.ServiceLatency = ServiceLatency{
		Google:     measureServiceLatency("google.com"),
		Amazon:     measureServiceLatency("amazon.com"),
		Cloudflare: measureServiceLatency("cloudflare.com"),
		Microsoft:  measureServiceLatency("microsoft.com"),
		DNS:        measureDNSLatency(),
		HTTP:       measureHTTPLatency(),
	}

	return networkInfo, nil
}

// GetARPTable retrieves the current ARP table using the modern 'ip neigh show' command
// instead of the legacy 'arp -a' command
func GetARPTable() ([]ARPEntry, error) {
	// Use the modern 'ip neigh show' command
	cmd := exec.Command("ip", "neigh", "show")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// Parse the output
	var entries []ARPEntry
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		// Parse line format: 192.168.1.1 dev eth0 lladdr 00:11:22:33:44:55 REACHABLE
		fields := strings.Fields(line)

		if len(fields) < 4 {
			continue
		}

		entry := ARPEntry{
			IPAddress: fields[0],
		}

		for i := 1; i < len(fields); i++ {
			switch fields[i] {
			case "dev":
				if i+1 < len(fields) {
					entry.Device = fields[i+1]
					i++
				}
			case "lladdr":
				if i+1 < len(fields) {
					entry.MACAddress = fields[i+1]
					i++
				}
			case "REACHABLE", "STALE", "DELAY", "PERMANENT", "INCOMPLETE", "FAILED", "PROBE", "NOARP":
				entry.State = fields[i]
			}
		}

		// Only add entries that have at least IP and MAC
		if entry.IPAddress != "" && entry.MACAddress != "" {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// Helper functions to retrieve network information
func getDefaultGateway() string {
	// Run ip route command to get default gateway
	cmd := exec.Command("ip", "route", "show", "default")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "N/A"
	}

	output := out.String()
	if output == "" {
		return "N/A"
	}

	// Parse output like: "default via 192.168.1.1 dev wlan0 proto dhcp metric 600"
	fields := strings.Fields(output)
	if len(fields) >= 3 && fields[0] == "default" && fields[1] == "via" {
		return fields[2]
	}
	return "N/A"
}

func getDNSServers() []string {
	// Read DNS servers from /etc/resolv.conf
	cmd := exec.Command("cat", "/etc/resolv.conf")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return []string{"N/A"}
	}

	var servers []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				servers = append(servers, fields[1])
			}
		}
	}

	if len(servers) == 0 {
		return []string{"N/A"}
	}
	return servers
}

func getDHCPServer() string {
	// Try to get DHCP server from dhclient lease files
	cmd := exec.Command("grep", "-l", "dhcp-server-identifier", "/var/lib/dhcp/dhclient*.leases", "/var/lib/dhcp/*.leases", "/var/lib/NetworkManager/*.lease")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		// Try another common location
		cmd = exec.Command("grep", "DHCPSID=", "/var/lib/dhcpcd/*.info")
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			return getDefaultGateway() // Fallback to gateway address
		}
	}

	leaseFiles := strings.Fields(out.String())
	if len(leaseFiles) == 0 {
		return getDefaultGateway() // Fallback to gateway address
	}

	// Read the most recent lease file
	mostRecentFile := leaseFiles[len(leaseFiles)-1]
	cmd = exec.Command("grep", "dhcp-server-identifier", mostRecentFile)
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return getDefaultGateway()
	}

	// Parse output like: "option dhcp-server-identifier 192.168.1.1;"
	output := out.String()
	if output == "" {
		return getDefaultGateway()
	}

	fields := strings.Fields(output)
	if len(fields) >= 3 {
		// Remove trailing semicolon if present
		serverIP := fields[len(fields)-1]
		return strings.TrimSuffix(serverIP, ";")
	}

	return getDefaultGateway() // Fallback to gateway address
}

func getUptime() int64 {
	// Read uptime from /proc/uptime
	cmd := exec.Command("cat", "/proc/uptime")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0
	}

	// Parse the first value, which is uptime in seconds
	uptimeStr := strings.Fields(out.String())
	if len(uptimeStr) > 0 {
		uptimeFloat, err := strconv.ParseFloat(uptimeStr[0], 64)
		if err == nil {
			return int64(uptimeFloat)
		}
	}

	return 0
}

func getPingLatency() float64 {
	// Ping the default gateway once to get latency
	gateway := getDefaultGateway()
	if gateway == "N/A" {
		gateway = "8.8.8.8" // Fallback to Google DNS
	}

	cmd := exec.Command("ping", "-c", "3", "-W", "1", gateway)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0
	}

	// Parse the output for average latency
	output := out.String()
	avgIndex := strings.Index(output, "min/avg/max/mdev")
	if avgIndex == -1 {
		return 0
	}

	statsLine := output[avgIndex:]
	stats := strings.Split(statsLine, " ")
	for _, stat := range stats {
		if strings.Contains(stat, "/") {
			values := strings.Split(stat, "/")
			if len(values) >= 2 {
				avgLatency, err := strconv.ParseFloat(values[1], 64)
				if err == nil {
					return avgLatency
				}
			}
			break
		}
	}

	return 0
}

func getPacketLoss() float64 {
	// Ping the default gateway to get packet loss
	gateway := getDefaultGateway()
	if gateway == "N/A" {
		gateway = "8.8.8.8" // Fallback to Google DNS
	}

	cmd := exec.Command("ping", "-c", "5", "-W", "1", gateway)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 100 // Assume 100% packet loss if ping fails
	}

	// Parse the output for packet loss percentage
	output := out.String()
	packetLossIndex := strings.Index(output, "packet loss")
	if packetLossIndex == -1 {
		return 0
	}

	// Look for the percentage before "packet loss"
	beforeLoss := output[:packetLossIndex]
	fields := strings.Fields(beforeLoss)
	if len(fields) > 0 {
		lastField := fields[len(fields)-1]
		percentStr := strings.TrimSuffix(lastField, "%")
		packetLoss, err := strconv.ParseFloat(percentStr, 64)
		if err == nil {
			return packetLoss
		}
	}

	return 0
}

func isWireless(ifaceName string) bool {
	// Check if interface is wireless by checking if it appears in iwconfig output
	cmd := exec.Command("iwconfig", ifaceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil && !strings.Contains(out.String(), "no wireless extensions") {
		return true
	}

	// Fallback to naming convention if iwconfig is not available
	return strings.HasPrefix(ifaceName, "wlan") || strings.HasPrefix(ifaceName, "wlp")
}

func getWirelessSSID(ifaceName string) string {
	if !isWireless(ifaceName) {
		return ""
	}

	// Try using iwconfig
	cmd := exec.Command("iwconfig", ifaceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return ""
	}

	// Parse the SSID from the output
	output := out.String()
	essidIndex := strings.Index(output, "ESSID:")
	if essidIndex == -1 {
		return ""
	}

	// Extract the SSID value between quotes
	essidPart := output[essidIndex+7:]
	endQuoteIndex := strings.Index(essidPart, "\"")
	if endQuoteIndex == -1 {
		return ""
	}

	return essidPart[:endQuoteIndex]
}

func getSignalStrength(ifaceName string) int {
	if !isWireless(ifaceName) {
		return 0
	}

	// Try using iwconfig to get signal strength
	cmd := exec.Command("iwconfig", ifaceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0
	}

	// Parse the signal level from the output
	output := out.String()
	signalIndex := strings.Index(output, "Signal level=")
	if signalIndex == -1 {
		return 0
	}

	// Extract the signal level value
	signalPart := output[signalIndex+13:]
	endIndex := strings.Index(signalPart, " ")
	if endIndex == -1 {
		return 0
	}

	// Remove dBm suffix if present
	signalStr := strings.TrimSuffix(signalPart[:endIndex], "dBm")
	signalInt, err := strconv.Atoi(signalStr)
	if err != nil {
		return 0
	}

	return signalInt
}

func getVLANID(ifaceName string) int {
	// Extract VLAN ID from interface name (e.g., eth0.10 -> 10)
	if idx := strings.LastIndex(ifaceName, "."); idx != -1 {
		vlanStr := ifaceName[idx+1:]
		vlanID, err := strconv.Atoi(vlanStr)
		if err == nil {
			return vlanID
		}
	}

	// Try reading from sysfs
	cmd := exec.Command("cat", "/proc/net/vlan/"+ifaceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err == nil {
		output := out.String()
		vlanIDIndex := strings.Index(output, "VID: ")
		if vlanIDIndex != -1 {
			vlanStr := strings.TrimSpace(output[vlanIDIndex+5:])
			endIndex := strings.Index(vlanStr, " ")
			if endIndex != -1 {
				vlanStr = vlanStr[:endIndex]
			}
			vlanID, err := strconv.Atoi(vlanStr)
			if err == nil {
				return vlanID
			}
		}
	}

	return 0
}

// Stores the last measured network counter values for bandwidth calculation
var (
	lastMeasurementTime time.Time
	lastBytesRecv       uint64
	lastBytesSent       uint64
	currentBandwidth    float64
)

func calculateBandwidth(counter psnet.IOCountersStat) float64 {
	now := time.Now()

	// Initialize on first call
	if lastMeasurementTime.IsZero() {
		lastMeasurementTime = now
		lastBytesRecv = counter.BytesRecv
		lastBytesSent = counter.BytesSent
		return 0 // No history for calculation yet
	}

	// Calculate time difference in seconds
	timeDiffSecs := now.Sub(lastMeasurementTime).Seconds()

	// Avoid division by zero or negative time
	if timeDiffSecs <= 0 {
		return currentBandwidth // Return last known bandwidth
	}

	// Calculate bytes transferred since last measurement
	bytesDiff := (counter.BytesRecv - lastBytesRecv) + (counter.BytesSent - lastBytesSent)

	// Calculate bandwidth in Megabits per second (1 Byte = 8 bits)
	// bytes/second * 8 / 1024 / 1024 = Mbps
	bandwidth := float64(bytesDiff) * 8 / 1024 / 1024 / timeDiffSecs

	// Update last values for next calculation
	lastMeasurementTime = now
	lastBytesRecv = counter.BytesRecv
	lastBytesSent = counter.BytesSent
	currentBandwidth = bandwidth

	return bandwidth
}

func cidrToSubnet(ones int) string {
	// Convert CIDR notation to subnet mask
	// For example, /24 -> 255.255.255.0
	switch ones {
	case 8:
		return "255.0.0.0"
	case 16:
		return "255.255.0.0"
	case 24:
		return "255.255.255.0"
	case 32:
		return "255.255.255.255"
	default:
		return "255.255.255.0" // Default
	}
}

// MeasureDNSLatency pings a DNS server to measure latency
func MeasureDNSLatency() float64 {
	// For simplicity, we use the first DNS server from the list
	dnsServers := getDNSServers()
	if len(dnsServers) == 0 {
		return 0
	}

	dnsServer := dnsServers[0]
	cmd := exec.Command("ping", "-c", "3", "-W", "1", dnsServer)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0
	}

	// Parse the output for average latency
	output := out.String()
	avgIndex := strings.Index(output, "min/avg/max/mdev")
	if avgIndex == -1 {
		return 0
	}

	statsLine := output[avgIndex:]
	stats := strings.Split(statsLine, " ")
	for _, stat := range stats {
		if strings.Contains(stat, "/") {
			values := strings.Split(stat, "/")
			if len(values) >= 2 {
				avgLatency, err := strconv.ParseFloat(values[1], 64)
				if err == nil {
					return avgLatency
				}
			}
			break
		}
	}

	return 0
}

// MeasureHTTPLatency measures latency to a public HTTP server
func MeasureHTTPLatency() float64 {
	httpServer := "http://www.google.com"

	startTime := time.Now()
	resp, err := http.Get(httpServer)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	// Measure time taken to get response
	elapsed := time.Since(startTime).Seconds() * 1000 // Convert to ms
	return elapsed
}

// measureServiceLatency measures latency to a service by hostname
func measureServiceLatency(hostname string) float64 {
	// Use ping with a timeout to avoid hanging
	cmd := exec.Command("ping", "-c", "1", "-w", "2", hostname)
	var out bytes.Buffer
	cmd.Stdout = &out

	// Execute ping command
	err := cmd.Run()
	if err != nil {
		return 0
	}

	// Parse the output for average latency
	output := out.String()

	// Try to parse the time from the first successful ping
	timePattern := regexp.MustCompile(`time=(\d+\.?\d*) ms`)
	matches := timePattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		latency, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return latency
		}
	}

	// Fallback to traditional min/avg/max parsing
	avgIndex := strings.Index(output, "min/avg/max/mdev")
	if avgIndex == -1 {
		return 0
	}

	statsLine := output[avgIndex:]
	stats := strings.Split(statsLine, " ")
	for _, stat := range stats {
		if strings.Contains(stat, "/") {
			values := strings.Split(stat, "/")
			if len(values) >= 2 {
				avgLatency, err := strconv.ParseFloat(values[1], 64)
				if err == nil {
					return avgLatency
				}
			}
			break
		}
	}

	return 0
}

// measureDNSLatency measures DNS resolution latency
func measureDNSLatency() float64 {
	startTime := time.Now()
	_, err := net.LookupHost("www.google.com")
	if err != nil {
		return 0
	}
	elapsed := time.Since(startTime)
	return float64(elapsed.Milliseconds())
}

// measureHTTPLatency measures HTTP connection latency
func measureHTTPLatency() float64 {
	httpServer := "https://www.google.com"

	// Create a client with a timeout to avoid hanging
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Get(httpServer)
	if err != nil {
		// Try a fallback if the first one fails
		httpServer = "https://www.cloudflare.com"
		startTime = time.Now()
		resp, err = client.Get(httpServer)
		if err != nil {
			return 0
		}
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)
	return float64(elapsed.Milliseconds())
}

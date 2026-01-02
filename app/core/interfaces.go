// Package core provides core networking information retrieval functionality.
package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// InterfaceType represents the type of network interface
type InterfaceType string

const (
	InterfaceTypeEthernet InterfaceType = "ethernet"
	InterfaceTypeWiFi     InterfaceType = "wifi"
	InterfaceTypeLoopback InterfaceType = "loopback"
	InterfaceTypeVirtual  InterfaceType = "virtual"
	InterfaceTypeBridge   InterfaceType = "bridge"
	InterfaceTypeVPN      InterfaceType = "vpn"
	InterfaceTypeUnknown  InterfaceType = "unknown"
)

// NetworkInterface represents a detected network interface
type NetworkInterface struct {
	Name       string        `json:"name"`
	Type       InterfaceType `json:"type"`
	MacAddress string        `json:"macAddress"`
	IPv4       string        `json:"ipv4,omitempty"`
	IPv6       string        `json:"ipv6,omitempty"`
	Subnet     string        `json:"subnet,omitempty"`
	IsUp       bool          `json:"isUp"`
	IsRunning  bool          `json:"isRunning"`
	HasCarrier bool          `json:"hasCarrier"`
	MTU        int           `json:"mtu"`
	Speed      string        `json:"speed,omitempty"`
	Driver     string        `json:"driver,omitempty"`

	// WiFi specific
	SSID           string `json:"ssid,omitempty"`
	SignalStrength int    `json:"signalStrength,omitempty"`
	Frequency      string `json:"frequency,omitempty"`
}

// InterfaceList contains categorized network interfaces
type InterfaceList struct {
	All             []NetworkInterface `json:"all"`
	Ethernet        []NetworkInterface `json:"ethernet"`
	WiFi            []NetworkInterface `json:"wifi"`
	Virtual         []NetworkInterface `json:"virtual"`
	Primary         *NetworkInterface  `json:"primary,omitempty"`
	PrimaryWiFi     *NetworkInterface  `json:"primaryWifi,omitempty"`
	PrimaryEthernet *NetworkInterface  `json:"primaryEthernet,omitempty"`
}

// GetInterfaces returns all available network interfaces categorized by type
func GetInterfaces() (*InterfaceList, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	result := &InterfaceList{
		All:      make([]NetworkInterface, 0),
		Ethernet: make([]NetworkInterface, 0),
		WiFi:     make([]NetworkInterface, 0),
		Virtual:  make([]NetworkInterface, 0),
	}

	for _, iface := range ifaces {
		ni := NetworkInterface{
			Name:       iface.Name,
			MacAddress: formatMACAddress(iface.HardwareAddr.String()),
			IsUp:       iface.Flags&net.FlagUp != 0,
			IsRunning:  iface.Flags&net.FlagRunning != 0,
			MTU:        iface.MTU,
		}

		// Determine interface type
		ni.Type = detectInterfaceType(iface.Name)

		// Check carrier status
		ni.HasCarrier = checkCarrier(iface.Name)

		// Get IP addresses
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ip := ipNet.IP.To4(); ip != nil && ni.IPv4 == "" {
					ni.IPv4 = ip.String()
					ones, _ := ipNet.Mask.Size()
					ni.Subnet = cidrToSubnet(ones)
				} else if ip == nil && ni.IPv6 == "" && !ipNet.IP.IsLinkLocalUnicast() {
					ni.IPv6 = ipNet.IP.String()
				}
			}
		}

		// Get speed for physical interfaces
		if ni.Type == InterfaceTypeEthernet || ni.Type == InterfaceTypeWiFi {
			ni.Speed = getLinkSpeed(iface.Name)
			ni.Driver = getInterfaceDriver(iface.Name)
		}

		// Get WiFi specific info
		if ni.Type == InterfaceTypeWiFi {
			getWiFiDetails(iface.Name, &ni)
		}

		// Add to appropriate category
		result.All = append(result.All, ni)

		switch ni.Type {
		case InterfaceTypeEthernet:
			result.Ethernet = append(result.Ethernet, ni)
			if result.PrimaryEthernet == nil && ni.IsUp && ni.HasCarrier && ni.IPv4 != "" {
				niCopy := ni
				result.PrimaryEthernet = &niCopy
			}
		case InterfaceTypeWiFi:
			result.WiFi = append(result.WiFi, ni)
			if result.PrimaryWiFi == nil && ni.IsUp && ni.IPv4 != "" {
				niCopy := ni
				result.PrimaryWiFi = &niCopy
			}
		case InterfaceTypeVirtual, InterfaceTypeBridge, InterfaceTypeVPN:
			result.Virtual = append(result.Virtual, ni)
		}

		// Set primary interface (first up interface with IP)
		if result.Primary == nil && ni.IsUp && ni.IPv4 != "" && ni.Type != InterfaceTypeLoopback {
			niCopy := ni
			result.Primary = &niCopy
		}
	}

	return result, nil
}

// GetPrimaryInterface returns the primary network interface
func GetPrimaryInterface() (*NetworkInterface, error) {
	list, err := GetInterfaces()
	if err != nil {
		return nil, err
	}
	return list.Primary, nil
}

// GetPrimaryWiFiInterface returns the primary WiFi interface
func GetPrimaryWiFiInterface() (*NetworkInterface, error) {
	list, err := GetInterfaces()
	if err != nil {
		return nil, err
	}
	return list.PrimaryWiFi, nil
}

// GetPrimaryEthernetInterface returns the primary Ethernet interface
func GetPrimaryEthernetInterface() (*NetworkInterface, error) {
	list, err := GetInterfaces()
	if err != nil {
		return nil, err
	}
	return list.PrimaryEthernet, nil
}

// GetInterfaceByName returns a specific interface by name
func GetInterfaceByName(name string) (*NetworkInterface, error) {
	list, err := GetInterfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range list.All {
		if iface.Name == name {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("interface %s not found", name)
}

// GetInterfacesByType returns all interfaces of a specific type
func GetInterfacesByType(ifaceType InterfaceType) ([]NetworkInterface, error) {
	list, err := GetInterfaces()
	if err != nil {
		return nil, err
	}

	var result []NetworkInterface
	for _, iface := range list.All {
		if iface.Type == ifaceType {
			result = append(result, iface)
		}
	}

	return result, nil
}

// detectInterfaceType determines the type of network interface
func detectInterfaceType(name string) InterfaceType {
	// Loopback
	if name == "lo" || strings.HasPrefix(name, "lo") {
		return InterfaceTypeLoopback
	}

	// Check sysfs for wireless
	wirelessPath := fmt.Sprintf("/sys/class/net/%s/wireless", name)
	if _, err := os.Stat(wirelessPath); err == nil {
		return InterfaceTypeWiFi
	}

	// Check phy80211 for WiFi
	phyPath := fmt.Sprintf("/sys/class/net/%s/phy80211", name)
	if _, err := os.Stat(phyPath); err == nil {
		return InterfaceTypeWiFi
	}

	// WiFi naming conventions
	wifiPrefixes := []string{"wlan", "wlp", "wl", "ath", "ra", "wifi"}
	for _, prefix := range wifiPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeWiFi
		}
	}

	// Virtual interfaces
	virtualPrefixes := []string{"docker", "veth", "virbr", "vnet", "vmnet", "vboxnet"}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeVirtual
		}
	}

	// Bridge interfaces
	bridgePrefixes := []string{"br-", "br0", "bridge"}
	for _, prefix := range bridgePrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeBridge
		}
	}

	// VPN interfaces
	vpnPrefixes := []string{"tun", "tap", "wg", "vpn", "ppp"}
	for _, prefix := range vpnPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeVPN
		}
	}

	// Check if it's a physical ethernet interface via sysfs
	devicePath := fmt.Sprintf("/sys/class/net/%s/device", name)
	if _, err := os.Stat(devicePath); err == nil {
		// Has a device, likely physical
		return InterfaceTypeEthernet
	}

	// Ethernet naming conventions
	ethernetPrefixes := []string{"eth", "enp", "eno", "ens", "em", "lan"}
	for _, prefix := range ethernetPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeEthernet
		}
	}

	return InterfaceTypeUnknown
}

// checkCarrier checks if the interface has a carrier (cable connected)
func checkCarrier(name string) bool {
	carrierPath := fmt.Sprintf("/sys/class/net/%s/carrier", name)
	data, err := os.ReadFile(carrierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// getInterfaceDriver returns the driver name for the interface
func getInterfaceDriver(name string) string {
	driverPath := fmt.Sprintf("/sys/class/net/%s/device/driver", name)
	link, err := os.Readlink(driverPath)
	if err != nil {
		return ""
	}
	// Extract driver name from path
	parts := strings.Split(link, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// getWiFiDetails populates WiFi-specific information
func getWiFiDetails(name string, ni *NetworkInterface) {
	// Try iw first
	cmd := exec.Command("iw", "dev", name, "link")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID:") {
				ni.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
			}
			if strings.HasPrefix(line, "signal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if sig, err := strconv.Atoi(parts[1]); err == nil {
						ni.SignalStrength = sig
					}
				}
			}
			if strings.HasPrefix(line, "freq:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					freq := parts[1]
					if freqInt, err := strconv.Atoi(freq); err == nil {
						if freqInt >= 5000 {
							ni.Frequency = freq + " MHz (5 GHz)"
						} else {
							ni.Frequency = freq + " MHz (2.4 GHz)"
						}
					}
				}
			}
		}
		return
	}

	// Fallback to iwconfig
	cmd = exec.Command("iwconfig", name)
	output, err = cmd.Output()
	if err != nil {
		return
	}

	outputStr := string(output)

	// ESSID
	if idx := strings.Index(outputStr, "ESSID:\""); idx != -1 {
		start := idx + 7
		end := strings.Index(outputStr[start:], "\"")
		if end != -1 {
			ni.SSID = outputStr[start : start+end]
		}
	}

	// Signal level
	if idx := strings.Index(outputStr, "Signal level="); idx != -1 {
		start := idx + 13
		end := strings.IndexAny(outputStr[start:], " \n")
		if end != -1 {
			sigStr := strings.TrimSuffix(outputStr[start:start+end], "dBm")
			if sig, err := strconv.Atoi(sigStr); err == nil {
				ni.SignalStrength = sig
			}
		}
	}

	// Frequency
	if idx := strings.Index(outputStr, "Frequency:"); idx != -1 {
		start := idx + 10
		end := strings.IndexAny(outputStr[start:], " \n")
		if end != -1 {
			ni.Frequency = strings.TrimSpace(outputStr[start : start+end])
		}
	}
}

// Note: getLinkSpeed, formatMACAddress, and cidrToSubnet functions
// are defined in network_info.go and reused here

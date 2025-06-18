package wifi_device_locator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DeviceType represents the type of Wi-Fi device
type DeviceType string

const (
	DeviceTypeAP      DeviceType = "AP"
	DeviceTypeClient  DeviceType = "Client"
	DeviceTypeUnknown DeviceType = "Unknown"
)

// WifiDevice represents a detected Wi-Fi device
type WifiDevice struct {
	BSSID          string     `json:"bssid"`
	SSID           string     `json:"ssid,omitempty"`
	SignalStrength int        `json:"signal_strength"` // in dBm
	SignalQuality  int        `json:"signal_quality"`  // 0-100%
	Channel        int        `json:"channel,omitempty"`
	Frequency      int        `json:"frequency,omitempty"` // in MHz
	DeviceType     DeviceType `json:"device_type"`
	FirstSeen      time.Time  `json:"first_seen"`
	LastSeen       time.Time  `json:"last_seen"`
	SignalHistory  []int      `json:"signal_history,omitempty"`
}

// Execute handles the wifi device locator plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	iface, ok := params["interface"].(string)
	if !ok || iface == "" {
		iface = "wlan0" // Default interface
	}

	scanTimeFloat, ok := params["scan_time"].(float64)
	if !ok {
		scanTimeFloat = 10 // Default scan time in seconds
	}
	scanTime := int(scanTimeFloat)

	scanMode, ok := params["scan_mode"].(string)
	if !ok {
		scanMode = "all" // Default scan mode
	}

	minSignalStrengthFloat, ok := params["min_signal_strength"].(float64)
	if !ok {
		minSignalStrengthFloat = -80 // Default minimum signal strength
	}
	minSignalStrength := int(minSignalStrengthFloat)

	continuousScan, ok := params["continuous_scan"].(bool)
	if !ok {
		continuousScan = false // Default to single scan
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(scanTime+30)*time.Second)
	defer cancel()

	// Check if interface exists and is up
	checkIface := exec.CommandContext(ctx, "ip", "link", "show", iface)
	if err := checkIface.Run(); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Interface %s does not exist or is not accessible", iface),
		}, nil
	}

	// Check if required tools are installed
	if err := checkRequiredTools(ctx); err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Put interface in monitor mode
	if err := setMonitorMode(ctx, iface); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to set monitor mode: %s", err.Error()),
		}, nil
	}

	// Ensure interface is reset when done
	defer resetInterface(iface)

	// Scan for devices
	devices, err := scanForDevices(ctx, iface, scanTime, continuousScan, scanMode, minSignalStrength)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to scan for devices: %s", err.Error()),
		}, nil
	}

	// Sort devices by signal strength (strongest first)
	sort.Slice(devices, func(i, j int) bool {
		return devices[i]["signal_strength"].(int) > devices[j]["signal_strength"].(int)
	})

	// Build result
	result := map[string]interface{}{
		"interface":           iface,
		"scan_time":           scanTime,
		"scan_mode":           scanMode,
		"min_signal_strength": minSignalStrength,
		"continuous_scan":     continuousScan,
		"devices":             devices,
		"device_count":        len(devices),
		"ap_count":            countDevicesByType(devices, DeviceTypeAP),
		"client_count":        countDevicesByType(devices, DeviceTypeClient),
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	return result, nil
}

// checkRequiredTools checks if the required tools are installed
func checkRequiredTools(ctx context.Context) error {
	tools := []string{"airmon-ng", "airodump-ng", "tcpdump"}

	for _, tool := range tools {
		cmd := exec.CommandContext(ctx, "which", tool)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s is not installed. Please install aircrack-ng suite with 'sudo apt-get install aircrack-ng'", tool)
		}
	}

	return nil
}

// setMonitorMode puts the interface in monitor mode
func setMonitorMode(ctx context.Context, iface string) error {
	// Check if the interface is already in monitor mode
	cmd := exec.CommandContext(ctx, "iwconfig", iface)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	if strings.Contains(stdout.String(), "Mode:Monitor") {
		return nil // Already in monitor mode
	}

	// Kill processes that might interfere with monitor mode
	killCmd := exec.CommandContext(ctx, "sudo", "airmon-ng", "check", "kill")
	if err := killCmd.Run(); err != nil {
		return fmt.Errorf("failed to kill interfering processes: %w", err)
	}

	// Set monitor mode using airmon-ng
	monCmd := exec.CommandContext(ctx, "sudo", "airmon-ng", "start", iface)
	if err := monCmd.Run(); err != nil {
		// Try alternative method if airmon-ng fails
		setDown := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "down")
		_ = setDown.Run()
		setMon := exec.CommandContext(ctx, "sudo", "iw", iface, "set", "monitor", "none")
		_ = setMon.Run()
		setUp := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "up")
		return setUp.Run()
	}

	return nil
}

// resetInterface resets the interface to managed mode
func resetInterface(iface string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to stop monitor mode with airmon-ng first
	stopCmd := exec.CommandContext(ctx, "sudo", "airmon-ng", "stop", iface)
	_ = stopCmd.Run()

	// Also try the manual method to ensure it's reset
	setDown := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "down")
	_ = setDown.Run()
	setManaged := exec.CommandContext(ctx, "sudo", "iw", iface, "set", "type", "managed")
	_ = setManaged.Run()
	setUp := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "up")
	_ = setUp.Run()

	// Restart network services
	restartCmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", "NetworkManager")
	_ = restartCmd.Run()
}

// scanForDevices scans for Wi-Fi devices using airodump-ng or tcpdump
func scanForDevices(ctx context.Context, iface string, scanTime int, continuousScan bool, scanMode string, minSignalStrength int) ([]map[string]interface{}, error) {
	devices := make(map[string]*WifiDevice)

	// Determine the number of scan iterations
	iterations := 1
	if continuousScan {
		iterations = scanTime / 2 // Scan every 2 seconds
		if iterations < 1 {
			iterations = 1
		}
	}

	for i := 0; i < iterations; i++ {
		// Use tcpdump to capture wireless frames
		captureTime := 2 // Seconds per capture
		if !continuousScan {
			captureTime = scanTime
		}

		tempFile := fmt.Sprintf("/tmp/wifi_scan_%d.pcap", time.Now().UnixNano())
		cmd := exec.CommandContext(ctx, "sudo", "tcpdump", "-i", iface, "-w", tempFile, "type", "mgt", "-G", strconv.Itoa(captureTime), "-W", "1", "-Z", "root")

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start tcpdump: %w", err)
		}

		// Wait for capture to complete
		time.Sleep(time.Duration(captureTime+1) * time.Second)
		_ = cmd.Process.Kill()

		// Process the capture file
		readCmd := exec.CommandContext(ctx, "sudo", "tcpdump", "-r", tempFile, "-v")
		var stdout bytes.Buffer
		readCmd.Stdout = &stdout
		if err := readCmd.Run(); err != nil {
			// If tcpdump fails, try with airodump-ng
			return scanWithAirodump(ctx, iface, scanTime, scanMode, minSignalStrength)
		}

		// Process tcpdump output
		parseTcpdumpOutput(stdout.String(), devices, scanMode, minSignalStrength)

		// Clean up temp file
		cleanCmd := exec.CommandContext(ctx, "sudo", "rm", "-f", tempFile)
		_ = cleanCmd.Run()

		// Update progress if doing multiple scans
		if continuousScan && i < iterations-1 {
			time.Sleep(time.Second)
		}
	}

	// Convert devices map to slice for output
	var result []map[string]interface{}
	for _, device := range devices {
		if device.SignalStrength >= minSignalStrength {
			deviceMap := map[string]interface{}{
				"bssid":           device.BSSID,
				"signal_strength": device.SignalStrength,
				"signal_quality":  device.SignalQuality,
				"device_type":     device.DeviceType,
				"first_seen":      device.FirstSeen.Format(time.RFC3339),
				"last_seen":       device.LastSeen.Format(time.RFC3339),
			}

			if device.SSID != "" {
				deviceMap["ssid"] = device.SSID
			}

			if device.Channel > 0 {
				deviceMap["channel"] = device.Channel
			}

			if device.Frequency > 0 {
				deviceMap["frequency"] = device.Frequency
			}

			if continuousScan && len(device.SignalHistory) > 0 {
				deviceMap["signal_history"] = device.SignalHistory

				// Calculate signal trend (increasing/decreasing)
				if len(device.SignalHistory) >= 2 {
					start := device.SignalHistory[0]
					end := device.SignalHistory[len(device.SignalHistory)-1]
					if end > start {
						deviceMap["signal_trend"] = "increasing"
					} else if end < start {
						deviceMap["signal_trend"] = "decreasing"
					} else {
						deviceMap["signal_trend"] = "stable"
					}
				}
			}

			result = append(result, deviceMap)
		}
	}

	return result, nil
}

// scanWithAirodump scans for Wi-Fi devices using airodump-ng
func scanWithAirodump(ctx context.Context, iface string, scanTime int, scanMode string, minSignalStrength int) ([]map[string]interface{}, error) {
	tempPrefix := fmt.Sprintf("/tmp/airodump_%d", time.Now().UnixNano())

	args := []string{
		"airodump-ng",
		"--output-format", "csv",
		"--write", tempPrefix,
		iface,
	}

	cmd := exec.CommandContext(ctx, "sudo", args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start airodump-ng: %w", err)
	}

	// Let it run for the specified scan time
	time.Sleep(time.Duration(scanTime) * time.Second)

	// Kill the process
	_ = cmd.Process.Kill()

	// Process the CSV file
	csvFile := fmt.Sprintf("%s-01.csv", tempPrefix)
	readCmd := exec.CommandContext(ctx, "cat", csvFile)
	var stdout bytes.Buffer
	readCmd.Stdout = &stdout
	if err := readCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to read airodump csv: %w", err)
	}

	// Parse the output
	devices := make(map[string]*WifiDevice)
	parseAirodumpOutput(stdout.String(), devices, scanMode, minSignalStrength)

	// Clean up temp files
	cleanCmd := exec.CommandContext(ctx, "sudo", "rm", "-f", fmt.Sprintf("%s-*", tempPrefix))
	_ = cleanCmd.Run()

	// Convert devices map to slice
	var result []map[string]interface{}
	for _, device := range devices {
		if device.SignalStrength >= minSignalStrength {
			deviceMap := map[string]interface{}{
				"bssid":           device.BSSID,
				"signal_strength": device.SignalStrength,
				"signal_quality":  device.SignalQuality,
				"device_type":     device.DeviceType,
				"first_seen":      device.FirstSeen.Format(time.RFC3339),
				"last_seen":       device.LastSeen.Format(time.RFC3339),
			}

			if device.SSID != "" {
				deviceMap["ssid"] = device.SSID
			}

			if device.Channel > 0 {
				deviceMap["channel"] = device.Channel
			}

			result = append(result, deviceMap)
		}
	}

	return result, nil
}

// parseTcpdumpOutput parses tcpdump output and updates the devices map
func parseTcpdumpOutput(output string, devices map[string]*WifiDevice, scanMode string, minSignalStrength int) {
	lines := strings.Split(output, "\n")

	// Regular expressions to extract information
	bssidRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2})`)
	ssidRegex := regexp.MustCompile(`SSID=([^,]+)`)
	signalRegex := regexp.MustCompile(`signal: (-?\d+)dBm`)

	currentTime := time.Now()

	for _, line := range lines {
		// Extract BSSID
		bssidMatches := bssidRegex.FindStringSubmatch(line)
		if len(bssidMatches) < 2 {
			continue
		}

		bssid := strings.ToUpper(bssidMatches[1])

		// Determine device type based on frame type
		deviceType := DeviceTypeUnknown
		if strings.Contains(line, "Beacon") || strings.Contains(line, "Probe Response") {
			deviceType = DeviceTypeAP
		} else if strings.Contains(line, "Probe Request") {
			deviceType = DeviceTypeClient
		}

		// Skip device if it doesn't match the scan mode
		if (scanMode == "ap" && deviceType != DeviceTypeAP) ||
			(scanMode == "client" && deviceType != DeviceTypeClient) {
			continue
		}

		// Extract signal strength if available
		signalStrength := -100 // Default weak signal
		signalMatches := signalRegex.FindStringSubmatch(line)
		if len(signalMatches) >= 2 {
			if s, err := strconv.Atoi(signalMatches[1]); err == nil {
				signalStrength = s
			}
		}

		// Skip if signal too weak
		if signalStrength < minSignalStrength {
			continue
		}

		// Calculate signal quality (0-100%)
		// Formula: 2 * (dbm + 100) where dbm is typically between -30 (excellent) and -90 (poor)
		signalQuality := 0
		if signalStrength >= -100 {
			quality := 2 * (signalStrength + 100)
			if quality > 100 {
				quality = 100
			} else if quality < 0 {
				quality = 0
			}
			signalQuality = quality
		}

		// Extract SSID if available (only for APs)
		ssid := ""
		if deviceType == DeviceTypeAP {
			ssidMatches := ssidRegex.FindStringSubmatch(line)
			if len(ssidMatches) >= 2 {
				ssid = ssidMatches[1]
			}
		}

		// Update or create device
		device, exists := devices[bssid]
		if !exists {
			device = &WifiDevice{
				BSSID:          bssid,
				SSID:           ssid,
				SignalStrength: signalStrength,
				SignalQuality:  signalQuality,
				DeviceType:     deviceType,
				FirstSeen:      currentTime,
				LastSeen:       currentTime,
				SignalHistory:  []int{signalStrength},
			}
			devices[bssid] = device
		} else {
			// Update existing device
			device.LastSeen = currentTime

			// Update SSID if we found one and it was empty before
			if ssid != "" && device.SSID == "" {
				device.SSID = ssid
			}

			// Update signal if stronger
			if signalStrength > device.SignalStrength {
				device.SignalStrength = signalStrength
				device.SignalQuality = signalQuality
			}

			// Add to signal history
			device.SignalHistory = append(device.SignalHistory, signalStrength)
		}
	}
}

// parseAirodumpOutput parses airodump-ng CSV output and updates the devices map
func parseAirodumpOutput(output string, devices map[string]*WifiDevice, scanMode string, minSignalStrength int) {
	lines := strings.Split(output, "\n")

	currentTime := time.Now()
	parsingClients := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if we're starting the client section
		if strings.Contains(line, "Station MAC") {
			parsingClients = true
			continue
		}

		// Skip header lines
		if strings.Contains(line, "BSSID") || strings.Contains(line, "First time seen") {
			continue
		}

		// Parse line based on whether we're looking at APs or clients
		if !parsingClients && (scanMode == "all" || scanMode == "ap") {
			// Parsing access points
			parts := strings.Split(line, ",")
			if len(parts) < 11 {
				continue
			}

			bssid := strings.TrimSpace(parts[0])
			if bssid == "" {
				continue
			}

			// Parse signal strength
			powerStr := strings.TrimSpace(parts[3])
			signalStrength := -100
			if powerStr != "" {
				if s, err := strconv.Atoi(powerStr); err == nil {
					signalStrength = s
				}
			}

			// Skip if signal too weak
			if signalStrength < minSignalStrength {
				continue
			}

			// Parse channel
			channelStr := strings.TrimSpace(parts[3])
			channel := 0
			if channelStr != "" {
				if c, err := strconv.Atoi(channelStr); err == nil {
					channel = c
				}
			}

			// Parse SSID
			ssid := strings.TrimSpace(parts[13])
			if len(ssid) >= 2 && ssid[0] == '"' && ssid[len(ssid)-1] == '"' {
				ssid = ssid[1 : len(ssid)-1]
			}

			// Calculate signal quality
			signalQuality := calculateSignalQuality(signalStrength)

			// Create or update device
			device, exists := devices[bssid]
			if !exists {
				device = &WifiDevice{
					BSSID:          bssid,
					SSID:           ssid,
					SignalStrength: signalStrength,
					SignalQuality:  signalQuality,
					Channel:        channel,
					DeviceType:     DeviceTypeAP,
					FirstSeen:      currentTime,
					LastSeen:       currentTime,
					SignalHistory:  []int{signalStrength},
				}
				devices[bssid] = device
			} else {
				// Update existing device
				device.LastSeen = currentTime
				device.SignalStrength = signalStrength
				device.SignalQuality = signalQuality
				device.Channel = channel
				device.SignalHistory = append(device.SignalHistory, signalStrength)
			}
		} else if parsingClients && (scanMode == "all" || scanMode == "client") {
			// Parsing clients
			parts := strings.Split(line, ",")
			if len(parts) < 6 {
				continue
			}

			bssid := strings.TrimSpace(parts[0])
			if bssid == "" {
				continue
			}

			// Parse signal strength
			powerStr := strings.TrimSpace(parts[3])
			signalStrength := -100
			if powerStr != "" {
				if s, err := strconv.Atoi(powerStr); err == nil {
					signalStrength = s
				}
			}

			// Skip if signal too weak
			if signalStrength < minSignalStrength {
				continue
			}

			// Calculate signal quality
			signalQuality := calculateSignalQuality(signalStrength)

			// Create or update device
			device, exists := devices[bssid]
			if !exists {
				device = &WifiDevice{
					BSSID:          bssid,
					SignalStrength: signalStrength,
					SignalQuality:  signalQuality,
					DeviceType:     DeviceTypeClient,
					FirstSeen:      currentTime,
					LastSeen:       currentTime,
					SignalHistory:  []int{signalStrength},
				}
				devices[bssid] = device
			} else {
				// Update existing device
				device.LastSeen = currentTime
				device.SignalStrength = signalStrength
				device.SignalQuality = signalQuality
				device.SignalHistory = append(device.SignalHistory, signalStrength)
			}
		}
	}
}

// calculateSignalQuality converts dBm to quality percentage (0-100%)
func calculateSignalQuality(signalStrength int) int {
	if signalStrength >= -50 {
		return 100
	} else if signalStrength <= -100 {
		return 0
	}

	// Linear conversion from -100 dBm (0%) to -50 dBm (100%)
	quality := 2 * (signalStrength + 100)
	if quality > 100 {
		quality = 100
	} else if quality < 0 {
		quality = 0
	}

	return quality
}

// countDevicesByType counts devices by type
func countDevicesByType(devices []map[string]interface{}, deviceType DeviceType) int {
	count := 0
	for _, device := range devices {
		if dt, ok := device["device_type"].(string); ok && DeviceType(dt) == deviceType {
			count++
		}
	}
	return count
}

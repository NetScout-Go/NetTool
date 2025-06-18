package wifi_device_proximity

import (
	"bytes"
	"context"
	"fmt"
	"math"
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

// ProximityLevel represents how close a device is
type ProximityLevel string

const (
	ProximityVeryClose ProximityLevel = "very_close"
	ProximityClose     ProximityLevel = "close"
	ProximityMedium    ProximityLevel = "medium"
	ProximityFar       ProximityLevel = "far"
	ProximityVeryFar   ProximityLevel = "very_far"
)

// WifiDevice represents a detected Wi-Fi device with proximity information
type WifiDevice struct {
	BSSID            string         `json:"bssid"`
	SSID             string         `json:"ssid,omitempty"`
	SignalStrength   int            `json:"signal_strength"` // in dBm
	SignalQuality    int            `json:"signal_quality"`  // 0-100%
	Channel          int            `json:"channel,omitempty"`
	Frequency        int            `json:"frequency,omitempty"` // in MHz
	DeviceType       DeviceType     `json:"device_type"`
	FirstSeen        time.Time      `json:"first_seen"`
	LastSeen         time.Time      `json:"last_seen"`
	SignalHistory    []int          `json:"signal_history,omitempty"`
	ProximityLevel   ProximityLevel `json:"proximity_level"`
	Manufacturer     string         `json:"manufacturer,omitempty"`
	DistanceEstimate float64        `json:"distance_estimate,omitempty"` // in meters (approximate)
}

// Execute handles the wifi device proximity plugin execution
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
		continuousScan = true // Default to continuous scan for better proximity tracking
	}

	targetDevice, _ := params["target_device"].(string)
	targetDevice = strings.ToUpper(targetDevice)

	proximityAlerts, _ := params["proximity_alerts"].(bool)

	proximityThresholdFloat, ok := params["proximity_threshold"].(float64)
	if !ok {
		proximityThresholdFloat = -50 // Default proximity threshold
	}
	proximityThreshold := int(proximityThresholdFloat)

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

	// Load OUI database for manufacturer lookup if available
	ouiMap := loadOUIDatabase()

	// Scan for devices
	devices, alerts, err := scanForDevices(ctx, iface, scanTime, continuousScan, scanMode, minSignalStrength,
		targetDevice, proximityAlerts, proximityThreshold, ouiMap)
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
		"proximity_alerts":    proximityAlerts,
	}

	// Add alerts if there are any
	if len(alerts) > 0 {
		result["alerts"] = alerts
	}

	// Add specific target tracking information if a target was specified
	if targetDevice != "" {
		var targetInfo map[string]interface{}
		for _, dev := range devices {
			if dev["bssid"].(string) == targetDevice {
				targetInfo = dev
				break
			}
		}

		if targetInfo != nil {
			result["target_tracking"] = targetInfo
		} else {
			result["target_tracking"] = map[string]interface{}{
				"status":  "not_found",
				"message": fmt.Sprintf("Target device %s not found during scan", targetDevice),
			}
		}
	}

	return result, nil
}

// checkRequiredTools checks if the required tools are installed
func checkRequiredTools(ctx context.Context) error {
	tools := []string{"airmon-ng", "airodump-ng", "tcpdump", "iw"}

	for _, tool := range tools {
		cmd := exec.CommandContext(ctx, "which", tool)
		if err := cmd.Run(); err != nil {
			if tool == "airmon-ng" || tool == "airodump-ng" {
				return fmt.Errorf("%s is not installed. Please install aircrack-ng suite with 'sudo apt-get install aircrack-ng'", tool)
			} else if tool == "tcpdump" {
				return fmt.Errorf("%s is not installed. Please install with 'sudo apt-get install tcpdump'", tool)
			} else if tool == "iw" {
				return fmt.Errorf("%s is not installed. Please install with 'sudo apt-get install iw'", tool)
			}
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

// scanForDevices scans for Wi-Fi devices and provides proximity information
func scanForDevices(ctx context.Context, iface string, scanTime int, continuousScan bool,
	scanMode string, minSignalStrength int, targetDevice string,
	proximityAlerts bool, proximityThreshold int, ouiMap map[string]string) ([]map[string]interface{}, []map[string]interface{}, error) {

	devices := make(map[string]*WifiDevice)
	var alerts []map[string]interface{}

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

		tempFile := fmt.Sprintf("/tmp/wifi_proximity_scan_%d.pcap", time.Now().UnixNano())
		cmd := exec.CommandContext(ctx, "sudo", "tcpdump", "-i", iface, "-w", tempFile, "type", "mgt", "-G", strconv.Itoa(captureTime), "-W", "1", "-Z", "root")

		if err := cmd.Start(); err != nil {
			return nil, nil, fmt.Errorf("failed to start tcpdump: %w", err)
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
			devs, alertsList, err := scanWithAirodump(ctx, iface, scanTime, scanMode, minSignalStrength, targetDevice, proximityAlerts, proximityThreshold, ouiMap)
			return devs, alertsList, err
		}

		// Process tcpdump output
		parseTcpdumpOutput(stdout.String(), devices, scanMode, minSignalStrength, ouiMap)

		// Check for proximity alerts
		if proximityAlerts {
			for _, device := range devices {
				if device.SignalStrength >= proximityThreshold {
					alert := map[string]interface{}{
						"type":            "proximity_alert",
						"bssid":           device.BSSID,
						"signal_strength": device.SignalStrength,
						"proximity_level": string(device.ProximityLevel),
						"timestamp":       time.Now().Format(time.RFC3339),
					}

					if device.SSID != "" {
						alert["ssid"] = device.SSID
					}

					if device.Manufacturer != "" {
						alert["manufacturer"] = device.Manufacturer
					}

					alerts = append(alerts, alert)
				}
			}
		}

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
			// Skip if we're only tracking a specific device
			if targetDevice != "" && device.BSSID != targetDevice {
				continue
			}

			deviceMap := map[string]interface{}{
				"bssid":           device.BSSID,
				"signal_strength": device.SignalStrength,
				"signal_quality":  device.SignalQuality,
				"device_type":     device.DeviceType,
				"proximity_level": string(device.ProximityLevel),
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

			if device.Manufacturer != "" {
				deviceMap["manufacturer"] = device.Manufacturer
			}

			if device.DistanceEstimate > 0 {
				deviceMap["distance_estimate"] = device.DistanceEstimate
			}

			if continuousScan && len(device.SignalHistory) > 0 {
				deviceMap["signal_history"] = device.SignalHistory

				// Calculate signal trend (increasing/decreasing)
				if len(device.SignalHistory) >= 2 {
					start := device.SignalHistory[0]
					end := device.SignalHistory[len(device.SignalHistory)-1]
					if end > start {
						deviceMap["signal_trend"] = "increasing" // Getting stronger (moving closer)
					} else if end < start {
						deviceMap["signal_trend"] = "decreasing" // Getting weaker (moving away)
					} else {
						deviceMap["signal_trend"] = "stable" // Not moving
					}
				}
			}

			result = append(result, deviceMap)
		}
	}

	return result, alerts, nil
}

// scanWithAirodump scans for Wi-Fi devices using airodump-ng with proximity info
func scanWithAirodump(ctx context.Context, iface string, scanTime int, scanMode string,
	minSignalStrength int, targetDevice string, proximityAlerts bool,
	proximityThreshold int, ouiMap map[string]string) ([]map[string]interface{}, []map[string]interface{}, error) {

	tempPrefix := fmt.Sprintf("/tmp/proximity_airodump_%d", time.Now().UnixNano())

	args := []string{
		"airodump-ng",
		"--output-format", "csv",
		"--write", tempPrefix,
	}

	// If we're targeting a specific device, add a BSSID filter
	if targetDevice != "" {
		args = append(args, "--bssid", targetDevice)
	}

	args = append(args, iface)

	cmd := exec.CommandContext(ctx, "sudo", args...)
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start airodump-ng: %w", err)
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
		return nil, nil, fmt.Errorf("failed to read airodump csv: %w", err)
	}

	// Parse the output
	devices := make(map[string]*WifiDevice)
	parseAirodumpOutput(stdout.String(), devices, scanMode, minSignalStrength, ouiMap)

	// Clean up temp files
	cleanCmd := exec.CommandContext(ctx, "sudo", "rm", "-f", fmt.Sprintf("%s-*", tempPrefix))
	_ = cleanCmd.Run()

	// Check for proximity alerts
	var alerts []map[string]interface{}
	if proximityAlerts {
		for _, device := range devices {
			if device.SignalStrength >= proximityThreshold {
				alert := map[string]interface{}{
					"type":            "proximity_alert",
					"bssid":           device.BSSID,
					"signal_strength": device.SignalStrength,
					"proximity_level": string(device.ProximityLevel),
					"timestamp":       time.Now().Format(time.RFC3339),
				}

				if device.SSID != "" {
					alert["ssid"] = device.SSID
				}

				if device.Manufacturer != "" {
					alert["manufacturer"] = device.Manufacturer
				}

				alerts = append(alerts, alert)
			}
		}
	}

	// Convert devices map to slice
	var result []map[string]interface{}
	for _, device := range devices {
		if device.SignalStrength >= minSignalStrength {
			// Skip if we're only tracking a specific device
			if targetDevice != "" && device.BSSID != targetDevice {
				continue
			}

			deviceMap := map[string]interface{}{
				"bssid":           device.BSSID,
				"signal_strength": device.SignalStrength,
				"signal_quality":  device.SignalQuality,
				"device_type":     device.DeviceType,
				"proximity_level": string(device.ProximityLevel),
				"first_seen":      device.FirstSeen.Format(time.RFC3339),
				"last_seen":       device.LastSeen.Format(time.RFC3339),
			}

			if device.SSID != "" {
				deviceMap["ssid"] = device.SSID
			}

			if device.Channel > 0 {
				deviceMap["channel"] = device.Channel
			}

			if device.Manufacturer != "" {
				deviceMap["manufacturer"] = device.Manufacturer
			}

			if device.DistanceEstimate > 0 {
				deviceMap["distance_estimate"] = device.DistanceEstimate
			}

			result = append(result, deviceMap)
		}
	}

	return result, alerts, nil
}

// parseTcpdumpOutput parses tcpdump output and updates the devices map with proximity info
func parseTcpdumpOutput(output string, devices map[string]*WifiDevice, scanMode string, minSignalStrength int, ouiMap map[string]string) {
	lines := strings.Split(output, "\n")

	// Regular expressions to extract information
	bssidRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2})`)
	ssidRegex := regexp.MustCompile(`SSID=([^,]+)`)
	signalRegex := regexp.MustCompile(`signal: (-?\d+)dBm`)
	channelRegex := regexp.MustCompile(`CH: (\d+)`)

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
			if signalVal, err := strconv.Atoi(signalMatches[1]); err == nil {
				signalStrength = signalVal
			}
		}

		// Skip if below minimum signal strength
		if signalStrength < minSignalStrength {
			continue
		}

		// Get or create device
		device, exists := devices[bssid]
		if !exists {
			device = &WifiDevice{
				BSSID:          bssid,
				DeviceType:     deviceType,
				SignalStrength: signalStrength,
				FirstSeen:      currentTime,
				LastSeen:       currentTime,
			}

			// Look up manufacturer
			if ouiMap != nil {
				oui := strings.Replace(bssid[0:8], ":", "", -1)
				if manufacturer, ok := ouiMap[oui]; ok {
					device.Manufacturer = manufacturer
				}
			}

			devices[bssid] = device
		} else {
			// Update last seen time
			device.LastSeen = currentTime

			// Only update signal strength if stronger
			if signalStrength > device.SignalStrength {
				device.SignalStrength = signalStrength
			}
		}

		// Extract SSID if it's an AP
		if deviceType == DeviceTypeAP && device.SSID == "" {
			ssidMatches := ssidRegex.FindStringSubmatch(line)
			if len(ssidMatches) >= 2 {
				device.SSID = ssidMatches[1]
			}
		}

		// Extract channel if available
		channelMatches := channelRegex.FindStringSubmatch(line)
		if len(channelMatches) >= 2 {
			if channelVal, err := strconv.Atoi(channelMatches[1]); err == nil {
				device.Channel = channelVal
			}
		}

		// Calculate signal quality (0-100%)
		device.SignalQuality = calculateSignalQuality(signalStrength)

		// Add to signal history for trend analysis
		device.SignalHistory = append(device.SignalHistory, signalStrength)

		// Determine proximity level based on signal strength
		device.ProximityLevel = determineProximityLevel(signalStrength)

		// Estimate distance in meters (very approximate)
		device.DistanceEstimate = estimateDistance(signalStrength)
	}
}

// parseAirodumpOutput parses airodump-ng output and updates the devices map with proximity info
func parseAirodumpOutput(output string, devices map[string]*WifiDevice, scanMode string, minSignalStrength int, ouiMap map[string]string) {
	lines := strings.Split(output, "\n")

	processingAPs := true
	currentTime := time.Now()

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for the dividing line between APs and clients
		if line == "" {
			continue
		} else if strings.Contains(line, "Station MAC") {
			processingAPs = false
			continue
		} else if strings.Contains(line, "BSSID") || strings.Contains(line, "First time seen") {
			continue
		}

		parts := strings.Split(line, ",")

		if processingAPs && len(parts) >= 13 && scanMode != "client" {
			// Processing Access Points
			bssid := strings.TrimSpace(parts[0])
			if bssid == "" {
				continue
			}

			// Get signal strength
			powerStr := strings.TrimSpace(parts[8])
			signalStrength := -100
			if powerStr != "" {
				if power, err := strconv.Atoi(powerStr); err == nil {
					signalStrength = power
				}
			}

			// Skip if below minimum signal strength
			if signalStrength < minSignalStrength {
				continue
			}

			// Get channel
			channelStr := strings.TrimSpace(parts[3])
			channel := 0
			if channelStr != "" {
				if ch, err := strconv.Atoi(channelStr); err == nil {
					channel = ch
				}
			}

			// Get SSID
			ssid := strings.TrimSpace(parts[13])
			if ssid == "<length: 0>" {
				ssid = "(Hidden Network)"
			}

			// Create or update device
			device, exists := devices[bssid]
			if !exists {
				device = &WifiDevice{
					BSSID:          bssid,
					SSID:           ssid,
					SignalStrength: signalStrength,
					Channel:        channel,
					DeviceType:     DeviceTypeAP,
					FirstSeen:      currentTime,
					LastSeen:       currentTime,
				}

				// Look up manufacturer
				if ouiMap != nil {
					oui := strings.Replace(bssid[0:8], ":", "", -1)
					if manufacturer, ok := ouiMap[oui]; ok {
						device.Manufacturer = manufacturer
					}
				}

				devices[bssid] = device
			} else {
				// Update last seen time
				device.LastSeen = currentTime

				// Only update signal strength if stronger
				if signalStrength > device.SignalStrength {
					device.SignalStrength = signalStrength
				}

				// Update SSID if needed
				if device.SSID == "" && ssid != "" {
					device.SSID = ssid
				}

				// Update channel if needed
				if device.Channel == 0 && channel != 0 {
					device.Channel = channel
				}
			}

			// Calculate signal quality (0-100%)
			device.SignalQuality = calculateSignalQuality(signalStrength)

			// Determine proximity level based on signal strength
			device.ProximityLevel = determineProximityLevel(signalStrength)

			// Estimate distance in meters (very approximate)
			device.DistanceEstimate = estimateDistance(signalStrength)

		} else if !processingAPs && len(parts) >= 6 && scanMode != "ap" {
			// Processing Clients
			bssid := strings.TrimSpace(parts[0])
			if bssid == "" {
				continue
			}

			// Get signal strength
			powerStr := strings.TrimSpace(parts[3])
			signalStrength := -100
			if powerStr != "" {
				if power, err := strconv.Atoi(powerStr); err == nil {
					signalStrength = power
				}
			}

			// Skip if below minimum signal strength
			if signalStrength < minSignalStrength {
				continue
			}

			// Create or update device
			device, exists := devices[bssid]
			if !exists {
				device = &WifiDevice{
					BSSID:          bssid,
					SignalStrength: signalStrength,
					DeviceType:     DeviceTypeClient,
					FirstSeen:      currentTime,
					LastSeen:       currentTime,
				}

				// Look up manufacturer
				if ouiMap != nil {
					oui := strings.Replace(bssid[0:8], ":", "", -1)
					if manufacturer, ok := ouiMap[oui]; ok {
						device.Manufacturer = manufacturer
					}
				}

				devices[bssid] = device
			} else {
				// Update last seen time
				device.LastSeen = currentTime

				// Only update signal strength if stronger
				if signalStrength > device.SignalStrength {
					device.SignalStrength = signalStrength
				}
			}

			// Calculate signal quality (0-100%)
			device.SignalQuality = calculateSignalQuality(signalStrength)

			// Determine proximity level based on signal strength
			device.ProximityLevel = determineProximityLevel(signalStrength)

			// Estimate distance in meters (very approximate)
			device.DistanceEstimate = estimateDistance(signalStrength)
		}
	}
}

// calculateSignalQuality converts dBm signal strength to a quality percentage (0-100%)
func calculateSignalQuality(signalStrength int) int {
	// Convert dBm to quality percentage (typical range is -30 dBm to -90 dBm)
	if signalStrength >= -30 {
		return 100
	} else if signalStrength <= -90 {
		return 0
	}

	// Linear mapping from [-90, -30] to [0, 100]
	return int(float64(signalStrength+90) / 60.0 * 100.0)
}

// determineProximityLevel determines how close a device is based on signal strength
func determineProximityLevel(signalStrength int) ProximityLevel {
	if signalStrength >= -40 {
		return ProximityVeryClose // Excellent signal, very close
	} else if signalStrength >= -55 {
		return ProximityClose // Good signal, close
	} else if signalStrength >= -70 {
		return ProximityMedium // Fair signal, medium distance
	} else if signalStrength >= -85 {
		return ProximityFar // Weak signal, far
	} else {
		return ProximityVeryFar // Very weak signal, very far
	}
}

// estimateDistance provides a rough estimate of distance based on signal strength
// using a simplified path loss model (very approximate)
func estimateDistance(signalStrength int) float64 {
	// This is a very simplified calculation and should not be relied upon for accuracy
	// Assumes free space path loss model with a reference of -40dBm at 1 meter
	// Real-world distances will vary significantly based on environment and obstacles

	// Only provide estimates for reasonable signal strengths
	if signalStrength < -90 || signalStrength > -20 {
		return 0 // Cannot provide a reliable estimate
	}

	// Simplified calculation based on free space path loss model
	// Using -40dBm as reference at 1 meter
	referenceSignal := -40
	pathLossExponent := 2.7 // Value between 2 (free space) and 4 (indoors with obstacles)

	// Calculate distance in meters
	signalDiff := float64(referenceSignal - signalStrength)
	distance := math.Pow(10, signalDiff/(10*pathLossExponent))

	// Round to one decimal place
	return math.Round(distance*10) / 10
}

// countDevicesByType counts the number of devices of a specific type
func countDevicesByType(devices []map[string]interface{}, deviceType DeviceType) int {
	count := 0
	for _, device := range devices {
		if dt, ok := device["device_type"].(DeviceType); ok && dt == deviceType {
			count++
		}
	}
	return count
}

// loadOUIDatabase loads the OUI database for manufacturer lookup if available
func loadOUIDatabase() map[string]string {
	ouiMap := make(map[string]string)

	// Try to read the OUI database file
	ouiFiles := []string{
		"/usr/share/nmap/nmap-mac-prefixes",
		"/usr/share/ieee-data/oui.txt",
		"/usr/share/wireshark/manuf",
	}

	for _, ouiFile := range ouiFiles {
		data, err := exec.Command("cat", ouiFile).Output()
		if err == nil {
			// Parse the OUI database
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.SplitN(line, " ", 2)
				if len(parts) < 2 {
					continue
				}

				oui := strings.TrimSpace(parts[0])
				manufacturer := strings.TrimSpace(parts[1])

				// Clean up OUI format
				oui = strings.ReplaceAll(oui, ":", "")
				oui = strings.ReplaceAll(oui, "-", "")
				oui = strings.ToUpper(oui)

				if len(oui) >= 6 {
					ouiMap[oui[0:6]] = manufacturer
				}
			}

			// Found and parsed a database, no need to check others
			if len(ouiMap) > 0 {
				break
			}
		}
	}

	return ouiMap
}

package wifi_scanner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Execute handles the wifi scanner plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	iface, ok := params["interface"].(string)
	if !ok || iface == "" {
		iface = "wlan0" // Default interface
	}

	scanTimeFloat, ok := params["scan_time"].(float64)
	if !ok {
		scanTimeFloat = 5 // Default scan time in seconds
	}
	scanTime := int(scanTimeFloat)

	showHidden, ok := params["show_hidden"].(bool)
	if !ok {
		showHidden = false // Default to not show hidden networks
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(scanTime+10)*time.Second)
	defer cancel()

	// Check if interface exists and is up
	checkIface := exec.CommandContext(ctx, "ip", "link", "show", iface)
	if err := checkIface.Run(); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Interface %s does not exist or is not accessible", iface),
		}, nil
	}

	// Check if iw is installed
	checkIw := exec.CommandContext(ctx, "which", "iw")
	if err := checkIw.Run(); err != nil {
		return map[string]interface{}{
			"error": "iw is not installed. Please install it with 'sudo apt-get install iw'",
		}, nil
	}

	// Put interface in monitor mode if possible
	setMonitorMode := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "down")
	_ = setMonitorMode.Run()
	setMonitorMode = exec.CommandContext(ctx, "sudo", "iw", iface, "set", "monitor", "none")
	_ = setMonitorMode.Run()
	setMonitorMode = exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "up")
	_ = setMonitorMode.Run()

	// Ensure interface is back to managed mode when done
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resetIface := exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "down")
		_ = resetIface.Run()
		resetIface = exec.CommandContext(ctx, "sudo", "iw", iface, "set", "type", "managed")
		_ = resetIface.Run()
		resetIface = exec.CommandContext(ctx, "sudo", "ip", "link", "set", iface, "up")
		_ = resetIface.Run()
	}()

	// Scan for networks
	var networks []map[string]interface{}

	// First try using iw scan
	networks, err := scanWithIw(ctx, iface, showHidden)
	if err != nil || len(networks) == 0 {
		// Fallback to iwlist
		networks, err = scanWithIwlist(ctx, iface, showHidden)
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to scan networks: %s", err.Error()),
			}, nil
		}
	}

	// Build result
	result := map[string]interface{}{
		"interface":     iface,
		"scan_time":     scanTime,
		"show_hidden":   showHidden,
		"networks":      networks,
		"network_count": len(networks),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	return result, nil
}

// scanWithIw uses the iw tool to scan for networks
func scanWithIw(ctx context.Context, iface string, showHidden bool) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "sudo", "iw", "dev", iface, "scan")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("iw scan failed: %w: %s", err, stderr.String())
	}

	output := stdout.String()
	return parseIwScanOutput(output, showHidden), nil
}

// scanWithIwlist uses the iwlist tool to scan for networks
func scanWithIwlist(ctx context.Context, iface string, showHidden bool) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "sudo", "iwlist", iface, "scanning")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("iwlist scan failed: %w: %s", err, stderr.String())
	}

	output := stdout.String()
	return parseIwlistScanOutput(output, showHidden), nil
}

// parseIwScanOutput parses the output of the iw scan command
func parseIwScanOutput(output string, showHidden bool) []map[string]interface{} {
	var networks []map[string]interface{}
	var currentNetwork map[string]interface{}

	lines := strings.Split(output, "\n")
	bssidRegex := regexp.MustCompile(`BSS ([0-9a-f:]{17})`)
	ssidRegex := regexp.MustCompile(`SSID: (.+)`)
	signalRegex := regexp.MustCompile(`signal: (-?\d+\.\d+) dBm`)
	channelRegex := regexp.MustCompile(`DS Parameter set: channel (\d+)`)
	freqRegex := regexp.MustCompile(`freq: (\d+)`)
	encryptionRegex := regexp.MustCompile(`capability:.*?(Privacy|IBSS)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for new BSS (new network)
		if matches := bssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous network if exists
			if currentNetwork != nil && len(currentNetwork) > 0 {
				// Only add if it has an SSID or if showing hidden networks
				if ssid, ok := currentNetwork["ssid"].(string); ok && (ssid != "" || showHidden) {
					networks = append(networks, currentNetwork)
				}
			}

			// Start new network
			currentNetwork = map[string]interface{}{
				"bssid": matches[1],
			}
			continue
		}

		// Skip if no current network
		if currentNetwork == nil {
			continue
		}

		// SSID
		if matches := ssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentNetwork["ssid"] = matches[1]
			continue
		}

		// Signal strength
		if matches := signalRegex.FindStringSubmatch(line); len(matches) > 1 {
			if signalStr, err := strconv.ParseFloat(matches[1], 64); err == nil {
				currentNetwork["signal_dbm"] = signalStr

				// Calculate signal quality (0-100%)
				// Formula: 2 * (dbm + 100) where dbm is typically between -30 (excellent) and -90 (poor)
				signalQuality := 2 * (signalStr + 100)
				if signalQuality > 100 {
					signalQuality = 100
				} else if signalQuality < 0 {
					signalQuality = 0
				}
				currentNetwork["signal_quality"] = signalQuality
			}
			continue
		}

		// Channel
		if matches := channelRegex.FindStringSubmatch(line); len(matches) > 1 {
			if channel, err := strconv.Atoi(matches[1]); err == nil {
				currentNetwork["channel"] = channel
			}
			continue
		}

		// Frequency
		if matches := freqRegex.FindStringSubmatch(line); len(matches) > 1 {
			if freq, err := strconv.Atoi(matches[1]); err == nil {
				currentNetwork["frequency"] = freq
			}
			continue
		}

		// Encryption
		if matches := encryptionRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentNetwork["encrypted"] = true
			continue
		}

		// WPA/WPA2
		if strings.Contains(line, "WPA") || strings.Contains(line, "RSN") {
			currentNetwork["security"] = "WPA/WPA2"
			continue
		}
	}

	// Add the last network if exists
	if currentNetwork != nil && len(currentNetwork) > 0 {
		// Only add if it has an SSID or if showing hidden networks
		if ssid, ok := currentNetwork["ssid"].(string); ok && (ssid != "" || showHidden) {
			networks = append(networks, currentNetwork)
		}
	}

	return networks
}

// parseIwlistScanOutput parses the output of the iwlist scan command
func parseIwlistScanOutput(output string, showHidden bool) []map[string]interface{} {
	var networks []map[string]interface{}
	var currentNetwork map[string]interface{}

	lines := strings.Split(output, "\n")
	cellRegex := regexp.MustCompile(`Cell \d+ - Address: ([0-9A-F:]{17})`)
	ssidRegex := regexp.MustCompile(`ESSID:"(.*)"`)
	qualityRegex := regexp.MustCompile(`Quality=(\d+)/(\d+)`)
	signalRegex := regexp.MustCompile(`Signal level=(-?\d+) dBm`)
	channelRegex := regexp.MustCompile(`Channel:(\d+)`)
	freqRegex := regexp.MustCompile(`Frequency:(\d+\.\d+) GHz`)
	encryptionRegex := regexp.MustCompile(`Encryption key:(on|off)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for new Cell (new network)
		if matches := cellRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous network if exists
			if currentNetwork != nil && len(currentNetwork) > 0 {
				// Only add if it has an SSID or if showing hidden networks
				if ssid, ok := currentNetwork["ssid"].(string); ok && (ssid != "" || showHidden) {
					networks = append(networks, currentNetwork)
				}
			}

			// Start new network
			currentNetwork = map[string]interface{}{
				"bssid": matches[1],
			}
			continue
		}

		// Skip if no current network
		if currentNetwork == nil {
			continue
		}

		// SSID
		if matches := ssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentNetwork["ssid"] = matches[1]
			continue
		}

		// Quality
		if matches := qualityRegex.FindStringSubmatch(line); len(matches) > 2 {
			if quality, err := strconv.Atoi(matches[1]); err == nil {
				if maxQuality, err := strconv.Atoi(matches[2]); err == nil && maxQuality > 0 {
					currentNetwork["signal_quality"] = float64(quality) * 100 / float64(maxQuality)
				}
			}
			continue
		}

		// Signal strength
		if matches := signalRegex.FindStringSubmatch(line); len(matches) > 1 {
			if signalStr, err := strconv.ParseFloat(matches[1], 64); err == nil {
				currentNetwork["signal_dbm"] = signalStr
			}
			continue
		}

		// Channel
		if matches := channelRegex.FindStringSubmatch(line); len(matches) > 1 {
			if channel, err := strconv.Atoi(matches[1]); err == nil {
				currentNetwork["channel"] = channel
			}
			continue
		}

		// Frequency
		if matches := freqRegex.FindStringSubmatch(line); len(matches) > 1 {
			if freq, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// Convert GHz to MHz
				currentNetwork["frequency"] = int(freq * 1000)
			}
			continue
		}

		// Encryption
		if matches := encryptionRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentNetwork["encrypted"] = matches[1] == "on"
			continue
		}

		// Security type
		if strings.Contains(line, "WPA2") {
			currentNetwork["security"] = "WPA2"
		} else if strings.Contains(line, "WPA") {
			currentNetwork["security"] = "WPA"
		} else if strings.Contains(line, "WEP") {
			currentNetwork["security"] = "WEP"
		}
	}

	// Add the last network if exists
	if currentNetwork != nil && len(currentNetwork) > 0 {
		// Only add if it has an SSID or if showing hidden networks
		if ssid, ok := currentNetwork["ssid"].(string); ok && (ssid != "" || showHidden) {
			networks = append(networks, currentNetwork)
		}
	}

	return networks
}

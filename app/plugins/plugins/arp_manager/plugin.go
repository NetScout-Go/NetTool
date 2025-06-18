package arp_manager

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ARPEntry represents an entry in the ARP table
type ARPEntry struct {
	IPAddress    string `json:"ip_address"`
	HWType       string `json:"hw_type,omitempty"`
	HWAddress    string `json:"hw_address"`
	Flags        string `json:"flags,omitempty"`
	Mask         string `json:"mask,omitempty"`
	Device       string `json:"device"`
	IsComplete   bool   `json:"is_complete"`
	IsStatic     bool   `json:"is_static"`
	LastSeen     string `json:"last_seen,omitempty"`
	IsReachable  bool   `json:"is_reachable"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

// Execute handles the ARP manager plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	action, ok := params["action"].(string)
	if !ok {
		action = "show" // Default action
	}

	iface, _ := params["interface"].(string)
	ipAddress, _ := params["ip_address"].(string)
	macAddress, _ := params["mac_address"].(string)
	entryType, _ := params["entry_type"].(string)
	verbose, _ := params["verbose"].(bool)
	numeric, _ := params["numeric"].(bool)

	// Create context with timeout for commands
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load OUI database for manufacturer lookup
	ouiMap := loadOUIDatabase()

	// Execute the requested ARP action
	switch action {
	case "show":
		return showARPTable(ctx, iface, verbose, numeric, ouiMap)
	case "get":
		return getARPEntry(ctx, ipAddress, iface, verbose, numeric, ouiMap)
	case "add":
		return addARPEntry(ctx, ipAddress, macAddress, iface, entryType)
	case "delete":
		return deleteARPEntry(ctx, ipAddress, iface)
	case "flush":
		return flushARPCache(ctx, iface)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// showARPTable retrieves and formats the ARP table
func showARPTable(ctx context.Context, iface string, verbose, numeric bool, ouiMap map[string]string) (interface{}, error) {
	// Build the ARP command
	args := []string{"-n"}
	if verbose {
		args = append(args, "-v")
	}
	if !numeric {
		args = []string{} // Default is to show hostnames
	}
	if iface != "" {
		args = append(args, "-i", iface)
	}

	// Run the ARP command
	cmd := exec.CommandContext(ctx, "arp", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return map[string]interface{}{
			"error":   fmt.Sprintf("Failed to execute ARP command: %s", err.Error()),
			"stderr":  stderr.String(),
			"command": fmt.Sprintf("arp %s", strings.Join(args, " ")),
		}, nil
	}

	// Parse the output
	entries, err := parseARPOutput(stdout.String(), ouiMap)
	if err != nil {
		return map[string]interface{}{
			"error":  fmt.Sprintf("Failed to parse ARP output: %s", err.Error()),
			"output": stdout.String(),
		}, nil
	}

	// Get additional details if verbose is enabled
	if verbose {
		for i, entry := range entries {
			if entry.IPAddress != "" {
				// Check if host is reachable
				pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", entry.IPAddress)
				entries[i].IsReachable = pingCmd.Run() == nil

				// Get last seen time if possible
				entries[i].LastSeen = time.Now().Format(time.RFC3339)
			}
		}
	}

	// Build the result
	result := map[string]interface{}{
		"action":       "show",
		"interface":    iface,
		"entries":      entries,
		"entry_count":  len(entries),
		"timestamp":    time.Now().Format(time.RFC3339),
		"command_used": fmt.Sprintf("arp %s", strings.Join(args, " ")),
	}

	return result, nil
}

// getARPEntry retrieves a specific ARP entry
func getARPEntry(ctx context.Context, ipAddress, iface string, verbose, numeric bool, ouiMap map[string]string) (interface{}, error) {
	if ipAddress == "" {
		return nil, fmt.Errorf("IP address is required for get action")
	}

	// Validate IP address
	if net.ParseIP(ipAddress) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	// Build the ARP command
	args := []string{"-n"}
	if verbose {
		args = append(args, "-v")
	}
	if !numeric {
		args = []string{} // Default is to show hostnames
	}
	if iface != "" {
		args = append(args, "-i", iface)
	}

	// Run the ARP command to get all entries
	cmd := exec.CommandContext(ctx, "arp", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return map[string]interface{}{
			"error":   fmt.Sprintf("Failed to execute ARP command: %s", err.Error()),
			"command": fmt.Sprintf("arp %s", strings.Join(args, " ")),
		}, nil
	}

	// Parse the output
	entries, err := parseARPOutput(stdout.String(), ouiMap)
	if err != nil {
		return map[string]interface{}{
			"error":  fmt.Sprintf("Failed to parse ARP output: %s", err.Error()),
			"output": stdout.String(),
		}, nil
	}

	// Find the requested entry
	var targetEntry *ARPEntry
	for _, entry := range entries {
		if entry.IPAddress == ipAddress {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		return map[string]interface{}{
			"action":     "get",
			"ip_address": ipAddress,
			"interface":  iface,
			"found":      false,
			"message":    fmt.Sprintf("No ARP entry found for IP address %s", ipAddress),
			"timestamp":  time.Now().Format(time.RFC3339),
		}, nil
	}

	// Check if host is reachable
	if verbose {
		pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ipAddress)
		targetEntry.IsReachable = pingCmd.Run() == nil
		targetEntry.LastSeen = time.Now().Format(time.RFC3339)
	}

	// Build the result
	result := map[string]interface{}{
		"action":       "get",
		"ip_address":   ipAddress,
		"interface":    iface,
		"found":        true,
		"entry":        targetEntry,
		"timestamp":    time.Now().Format(time.RFC3339),
		"command_used": fmt.Sprintf("arp %s", strings.Join(args, " ")),
	}

	return result, nil
}

// addARPEntry adds a new entry to the ARP table
func addARPEntry(ctx context.Context, ipAddress, macAddress, iface, entryType string) (interface{}, error) {
	if ipAddress == "" || macAddress == "" {
		return nil, fmt.Errorf("IP address and MAC address are required for add action")
	}

	// Validate IP address
	if net.ParseIP(ipAddress) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	// Validate MAC address
	macRegex := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	if !macRegex.MatchString(macAddress) {
		return nil, fmt.Errorf("invalid MAC address: %s (expected format: 00:11:22:33:44:55)", macAddress)
	}

	// Build the ARP command
	args := []string{"-s", ipAddress, macAddress}

	// Add entry type if specified
	switch entryType {
	case "temp":
		args = append(args, "temp")
	case "pub":
		args = append(args, "pub")
		// "permanent" is the default behavior, no need to specify
	}

	// Add interface if specified
	if iface != "" {
		args = append(args, "dev", iface)
	}

	// Run the ARP command
	cmd := exec.CommandContext(ctx, "arp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return map[string]interface{}{
			"action":      "add",
			"success":     false,
			"error":       fmt.Sprintf("Failed to add ARP entry: %s", err.Error()),
			"stderr":      stderr.String(),
			"ip_address":  ipAddress,
			"mac_address": macAddress,
			"interface":   iface,
			"entry_type":  entryType,
			"command":     fmt.Sprintf("arp %s", strings.Join(args, " ")),
			"timestamp":   time.Now().Format(time.RFC3339),
		}, nil
	}

	// Verify the entry was added
	verifyCmd := exec.CommandContext(ctx, "arp", "-n", ipAddress)
	var stdout bytes.Buffer
	verifyCmd.Stdout = &stdout
	err := verifyCmd.Run()

	success := err == nil && strings.Contains(stdout.String(), macAddress)

	// Build the result
	result := map[string]interface{}{
		"action":      "add",
		"success":     success,
		"ip_address":  ipAddress,
		"mac_address": macAddress,
		"interface":   iface,
		"entry_type":  entryType,
		"command":     fmt.Sprintf("arp %s", strings.Join(args, " ")),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	if !success {
		result["warning"] = "Entry might not have been added successfully. Check the ARP table to verify."
		result["verify_output"] = stdout.String()
	}

	return result, nil
}

// deleteARPEntry deletes an entry from the ARP table
func deleteARPEntry(ctx context.Context, ipAddress, iface string) (interface{}, error) {
	if ipAddress == "" {
		return nil, fmt.Errorf("IP address is required for delete action")
	}

	// Validate IP address
	if net.ParseIP(ipAddress) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	// Build the ARP command
	args := []string{"-d", ipAddress}
	if iface != "" {
		args = append(args, "dev", iface)
	}

	// Run the ARP command
	cmd := exec.CommandContext(ctx, "arp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return map[string]interface{}{
			"action":     "delete",
			"success":    false,
			"error":      fmt.Sprintf("Failed to delete ARP entry: %s", err.Error()),
			"stderr":     stderr.String(),
			"ip_address": ipAddress,
			"interface":  iface,
			"command":    fmt.Sprintf("arp %s", strings.Join(args, " ")),
			"timestamp":  time.Now().Format(time.RFC3339),
		}, nil
	}

	// Verify the entry was deleted
	verifyCmd := exec.CommandContext(ctx, "arp", "-n", ipAddress)
	var stdout bytes.Buffer
	verifyCmd.Stdout = &stdout
	err := verifyCmd.Run()

	success := err != nil || !strings.Contains(stdout.String(), ipAddress)

	// Build the result
	result := map[string]interface{}{
		"action":     "delete",
		"success":    success,
		"ip_address": ipAddress,
		"interface":  iface,
		"command":    fmt.Sprintf("arp %s", strings.Join(args, " ")),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if !success {
		result["warning"] = "Entry might not have been deleted successfully. Check the ARP table to verify."
		result["verify_output"] = stdout.String()
	}

	return result, nil
}

// flushARPCache flushes the ARP cache
func flushARPCache(ctx context.Context, iface string) (interface{}, error) {
	// Get the current ARP table before flushing
	var beforeCmd *exec.Cmd
	if iface != "" {
		beforeCmd = exec.CommandContext(ctx, "arp", "-n", "-i", iface)
	} else {
		beforeCmd = exec.CommandContext(ctx, "arp", "-n")
	}
	var beforeOutput bytes.Buffer
	beforeCmd.Stdout = &beforeOutput
	_ = beforeCmd.Run()
	beforeEntries := strings.Split(strings.TrimSpace(beforeOutput.String()), "\n")
	beforeCount := len(beforeEntries) - 1 // Subtract header line
	if beforeCount < 0 {
		beforeCount = 0
	}

	// There's no direct "flush" command for ARP in Linux, so we use ip neigh flush
	var cmd *exec.Cmd
	if iface != "" {
		cmd = exec.CommandContext(ctx, "ip", "neigh", "flush", "dev", iface)
	} else {
		cmd = exec.CommandContext(ctx, "ip", "neigh", "flush", "all")
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return map[string]interface{}{
			"action":    "flush",
			"success":   false,
			"error":     fmt.Sprintf("Failed to flush ARP cache: %s", err.Error()),
			"stderr":    stderr.String(),
			"interface": iface,
			"command":   fmt.Sprintf("ip neigh flush %s", iface),
			"timestamp": time.Now().Format(time.RFC3339),
		}, nil
	}

	// Get the ARP table after flushing
	var afterCmd *exec.Cmd
	if iface != "" {
		afterCmd = exec.CommandContext(ctx, "arp", "-n", "-i", iface)
	} else {
		afterCmd = exec.CommandContext(ctx, "arp", "-n")
	}
	var afterOutput bytes.Buffer
	afterCmd.Stdout = &afterOutput
	_ = afterCmd.Run()
	afterEntries := strings.Split(strings.TrimSpace(afterOutput.String()), "\n")
	afterCount := len(afterEntries) - 1 // Subtract header line
	if afterCount < 0 {
		afterCount = 0
	}

	// Build the result
	result := map[string]interface{}{
		"action":          "flush",
		"success":         true,
		"interface":       iface,
		"entries_before":  beforeCount,
		"entries_after":   afterCount,
		"entries_removed": beforeCount - afterCount,
		"command":         fmt.Sprintf("ip neigh flush %s", iface),
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	return result, nil
}

// parseARPOutput parses the output of the ARP command
func parseARPOutput(output string, ouiMap map[string]string) ([]ARPEntry, error) {
	lines := strings.Split(output, "\n")
	var entries []ARPEntry

	// Skip the header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse fields (format depends on the arp command output format)
		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(fields) < 3 {
			continue
		}

		// Typical Linux arp output: IP_address HW_type HW_address Flags Mask Iface
		var entry ARPEntry
		entry.IPAddress = fields[0]

		// Handle different field positions depending on output format
		if len(fields) >= 6 {
			entry.HWType = fields[1]
			entry.HWAddress = fields[2]
			entry.Flags = fields[3]
			entry.Mask = fields[4]
			entry.Device = fields[5]
		} else if len(fields) >= 5 {
			entry.HWAddress = fields[2]
			entry.Device = fields[4]
		} else if len(fields) >= 3 {
			entry.HWAddress = fields[2]
			if len(fields) >= 4 {
				entry.Device = fields[3]
			}
		}

		// Clean up MAC address
		entry.HWAddress = strings.TrimSpace(entry.HWAddress)
		if entry.HWAddress == "(incomplete)" {
			entry.HWAddress = ""
			entry.IsComplete = false
		} else {
			entry.IsComplete = true
		}

		// Set static flag
		entry.IsStatic = strings.Contains(strings.ToLower(entry.Flags), "perm") ||
			strings.Contains(strings.ToLower(entry.Flags), "static")

		// Look up manufacturer
		if entry.HWAddress != "" && ouiMap != nil {
			macPrefix := strings.ReplaceAll(entry.HWAddress, ":", "")
			macPrefix = strings.ReplaceAll(macPrefix, "-", "")
			macPrefix = strings.ToUpper(macPrefix)
			if len(macPrefix) >= 6 {
				if manufacturer, ok := ouiMap[macPrefix[0:6]]; ok {
					entry.Manufacturer = manufacturer
				}
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// loadOUIDatabase loads the OUI database for manufacturer lookup
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

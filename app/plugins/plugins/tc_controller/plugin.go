package tc_controller

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Execute handles the tc controller plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	iface, _ := params["interface"].(string)
	mode, _ := params["mode"].(string)
	bandwidthFloat, _ := params["bandwidth"].(float64)
	latencyFloat, _ := params["latency"].(float64)
	packetLossFloat, _ := params["packet_loss"].(float64)
	jitterFloat, _ := params["jitter"].(float64)
	durationFloat, _ := params["duration"].(float64)

	// Convert float values to integers
	bandwidth := int(bandwidthFloat)
	latency := int(latencyFloat)
	packetLoss := packetLossFloat // Keep as float for percentage
	jitter := int(jitterFloat)
	duration := int(durationFloat)

	// Validate required parameters
	if iface == "" {
		return nil, fmt.Errorf("interface parameter is required")
	}
	if mode == "" {
		return nil, fmt.Errorf("mode parameter is required")
	}

	// Set defaults if not provided
	if bandwidth == 0 {
		bandwidth = 1000 // 1 Mbps
	}
	if latency == 0 {
		latency = 100 // 100ms
	}
	if packetLoss == 0 && (mode == "packet_loss" || mode == "combination") {
		packetLoss = 1.0 // 1%
	}
	if jitter == 0 {
		jitter = 20 // 20ms
	}
	if duration == 0 && mode != "reset" {
		duration = 60 // 60 seconds
	}

	// Check if tc is installed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "which", "tc")
	err := cmd.Run()

	if err != nil {
		return map[string]interface{}{
			"success":  false,
			"error":    "Traffic Control (tc) is not installed on the system. Please install it using 'sudo apt-get install iproute2' or the appropriate package manager for your system.",
			"commands": getCommandsForMode(mode, iface, bandwidth, latency, packetLoss, jitter),
		}, nil
	}

	// Build and execute tc commands based on mode
	commands, err := executeTcCommands(mode, iface, bandwidth, latency, packetLoss, jitter)
	if err != nil {
		return map[string]interface{}{
			"success":  false,
			"error":    fmt.Sprintf("Failed to execute tc commands: %v", err),
			"commands": commands,
		}, nil
	}

	// Set up automatic reset if duration is specified
	resetMessage := ""
	if duration > 0 && mode != "reset" {
		resetMessage = fmt.Sprintf("Rules will be automatically reset after %d seconds", duration)
		go func() {
			time.Sleep(time.Duration(duration) * time.Second)
			resetTcRules(iface)
		}()
	}

	// Return response
	return map[string]interface{}{
		"success":   true,
		"mode":      mode,
		"interface": iface,
		"commands":  commands,
		"parameters": map[string]interface{}{
			"bandwidth":   bandwidth,
			"latency":     latency,
			"packet_loss": packetLoss,
			"jitter":      jitter,
			"duration":    duration,
		},
		"timestamp":    time.Now().Format(time.RFC3339),
		"resetMessage": resetMessage,
	}, nil
}

// executeTcCommands executes the tc commands based on the selected mode
func executeTcCommands(mode, iface string, bandwidth, latency int, packetLoss float64, jitter int) ([]string, error) {
	// Always reset existing rules first
	resetTcRules(iface)

	// If mode is reset, we're done
	if mode == "reset" {
		return []string{fmt.Sprintf("tc qdisc del dev %s root", iface)}, nil
	}

	var commands []string
	var err error

	switch mode {
	case "bandwidth":
		commands, err = applyBandwidthLimit(iface, bandwidth)
	case "latency":
		commands, err = applyLatency(iface, latency)
	case "packet_loss":
		commands, err = applyPacketLoss(iface, packetLoss)
	case "jitter":
		commands, err = applyJitter(iface, latency, jitter)
	case "combination":
		commands, err = applyCombination(iface, bandwidth, latency, packetLoss, jitter)
	default:
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	return commands, err
}

// resetTcRules removes all tc rules from the interface
func resetTcRules(iface string) error {
	cmd := exec.Command("tc", "qdisc", "del", "dev", iface, "root")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Ignore "No such file or directory" errors which occur when no rules exist
	if err != nil && !strings.Contains(stderr.String(), "No such file or directory") {
		return fmt.Errorf("failed to reset tc rules: %v", err)
	}

	return nil
}

// applyBandwidthLimit applies bandwidth limitation using tc
func applyBandwidthLimit(iface string, bandwidth int) ([]string, error) {
	command := fmt.Sprintf("tc qdisc add dev %s root handle 1: tbf rate %dkbit burst 32kB latency 400ms",
		iface, bandwidth)

	cmd := exec.Command("sh", "-c", command)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return []string{command}, fmt.Errorf("failed to apply bandwidth limit: %v - %s", err, stderr.String())
	}

	return []string{command}, nil
}

// applyLatency applies fixed latency using tc
func applyLatency(iface string, latency int) ([]string, error) {
	command := fmt.Sprintf("tc qdisc add dev %s root netem delay %dms",
		iface, latency)

	cmd := exec.Command("sh", "-c", command)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return []string{command}, fmt.Errorf("failed to apply latency: %v - %s", err, stderr.String())
	}

	return []string{command}, nil
}

// applyPacketLoss applies packet loss using tc
func applyPacketLoss(iface string, packetLoss float64) ([]string, error) {
	command := fmt.Sprintf("tc qdisc add dev %s root netem loss %.2f%%",
		iface, packetLoss)

	cmd := exec.Command("sh", "-c", command)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return []string{command}, fmt.Errorf("failed to apply packet loss: %v - %s", err, stderr.String())
	}

	return []string{command}, nil
}

// applyJitter applies variable latency (jitter) using tc
func applyJitter(iface string, latency, jitter int) ([]string, error) {
	command := fmt.Sprintf("tc qdisc add dev %s root netem delay %dms %dms distribution normal",
		iface, latency, jitter)

	cmd := exec.Command("sh", "-c", command)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return []string{command}, fmt.Errorf("failed to apply jitter: %v - %s", err, stderr.String())
	}

	return []string{command}, nil
}

// applyCombination applies a combination of network conditions
func applyCombination(iface string, bandwidth, latency int, packetLoss float64, jitter int) ([]string, error) {
	// First apply netem qdisc for latency, jitter, and packet loss
	netemCmd := fmt.Sprintf("tc qdisc add dev %s root handle 1: netem delay %dms %dms distribution normal loss %.2f%%",
		iface, latency, jitter, packetLoss)

	cmd1 := exec.Command("sh", "-c", netemCmd)
	var stderr1 bytes.Buffer
	cmd1.Stderr = &stderr1
	err1 := cmd1.Run()

	if err1 != nil {
		return []string{netemCmd}, fmt.Errorf("failed to apply netem parameters: %v - %s", err1, stderr1.String())
	}

	// Then apply bandwidth limitation using tbf
	if bandwidth > 0 {
		tbfCmd := fmt.Sprintf("tc qdisc add dev %s parent 1:1 handle 10: tbf rate %dkbit burst 32kB latency 400ms",
			iface, bandwidth)

		cmd2 := exec.Command("sh", "-c", tbfCmd)
		var stderr2 bytes.Buffer
		cmd2.Stderr = &stderr2
		err2 := cmd2.Run()

		if err2 != nil {
			// Try to reset since we're in an inconsistent state
			resetTcRules(iface)
			return []string{netemCmd, tbfCmd}, fmt.Errorf("failed to apply bandwidth limit: %v - %s", err2, stderr2.String())
		}

		return []string{netemCmd, tbfCmd}, nil
	}

	return []string{netemCmd}, nil
}

// getCommandsForMode returns the tc commands that would be executed for the given mode
// This is used when tc is not installed to show the user what commands would be run
func getCommandsForMode(mode, iface string, bandwidth, latency int, packetLoss float64, jitter int) []string {
	var commands []string

	// Reset command
	commands = append(commands, fmt.Sprintf("tc qdisc del dev %s root", iface))

	// Mode-specific commands
	switch mode {
	case "reset":
		// Already added reset command
	case "bandwidth":
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s root handle 1: tbf rate %dkbit burst 32kB latency 400ms",
			iface, bandwidth))
	case "latency":
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s root netem delay %dms",
			iface, latency))
	case "packet_loss":
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s root netem loss %.2f%%",
			iface, packetLoss))
	case "jitter":
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s root netem delay %dms %dms distribution normal",
			iface, latency, jitter))
	case "combination":
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s root handle 1: netem delay %dms %dms distribution normal loss %.2f%%",
			iface, latency, jitter, packetLoss))
		commands = append(commands, fmt.Sprintf("tc qdisc add dev %s parent 1:1 handle 10: tbf rate %dkbit burst 32kB latency 400ms",
			iface, bandwidth))
	}

	return commands
}

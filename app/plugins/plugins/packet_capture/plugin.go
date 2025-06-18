package packet_capture

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Execute handles the packet capture plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	iface, ok := params["interface"].(string)
	if !ok || iface == "" {
		iface = "eth0" // Default interface
	}

	captureFilter, _ := params["capture_filter"].(string)

	packetCountFloat, ok := params["packet_count"].(float64)
	if !ok {
		packetCountFloat = 100 // Default packet count
	}
	packetCount := int(packetCountFloat)

	timeoutFloat, ok := params["capture_timeout"].(float64)
	if !ok {
		timeoutFloat = 10 // Default timeout in seconds
	}
	timeout := int(timeoutFloat)

	includeHeaders, ok := params["include_headers"].(bool)
	if !ok {
		includeHeaders = true // Default to include headers
	}

	// Build tcpdump command
	args := []string{"-i", iface}

	// Add packet count if specified
	if packetCount > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", packetCount))
	}

	// Add verbose output if headers are requested
	if includeHeaders {
		args = append(args, "-v")
	}

	// Add filter if specified
	if captureFilter != "" {
		args = append(args, captureFilter)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout+5)*time.Second)
	defer cancel()

	// Check if tcpdump is installed
	checkCmd := exec.CommandContext(ctx, "which", "tcpdump")
	if err := checkCmd.Run(); err != nil {
		return map[string]interface{}{
			"error": "tcpdump is not installed. Please install it with 'sudo apt-get install tcpdump'",
		}, nil
	}

	// Execute tcpdump
	cmd := exec.CommandContext(ctx, "tcpdump", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start capture
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start packet capture: %w", err)
	}

	// Create a timer to stop capture after the specified timeout
	var timedOut bool
	if timeout > 0 {
		timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
			timedOut = true
			cmd.Process.Kill()
		})
		defer timer.Stop()
	}

	// Wait for command to complete
	err = cmd.Wait()

	// Format output
	var output string
	if stdout.Len() > 0 {
		output = stdout.String()
	} else if stderr.Len() > 0 {
		output = stderr.String()
	}

	// Clean up output for display
	output = strings.TrimSpace(output)
	outputLines := strings.Split(output, "\n")

	// Limit output lines if there are too many
	maxLines := 1000
	if len(outputLines) > maxLines {
		outputLines = outputLines[:maxLines]
		outputLines = append(outputLines, fmt.Sprintf("... output truncated (showing %d of %d lines)", maxLines, len(outputLines)))
	}

	// Build result
	result := map[string]interface{}{
		"interface":       iface,
		"capture_filter":  captureFilter,
		"packet_count":    packetCount,
		"timeout":         timeout,
		"include_headers": includeHeaders,
		"packets":         outputLines,
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	if timedOut {
		result["note"] = "Capture stopped due to timeout"
	}

	if err != nil && !timedOut {
		result["error"] = fmt.Sprintf("capture error: %s", err.Error())
	}

	return result, nil
}

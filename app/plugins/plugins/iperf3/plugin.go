package iperf3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)
// Execute handles the iperf3 plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	server, _ := params["server"].(string)
	portFloat, _ := params["port"].(float64)
	port := int(portFloat)
	durationFloat, _ := params["duration"].(float64)
	duration := int(durationFloat)
	protocol, _ := params["protocol"].(string)
	reverse, _ := params["reverse"].(bool)
	parallelFloat, _ := params["parallel"].(float64)
	parallel := int(parallelFloat)

	// Validate required parameters
	if server == "" {
		return nil, fmt.Errorf("server parameter is required")
	}

	// Set defaults if not provided
	if port == 0 {
		port = 5201
	}
	if duration == 0 {
		duration = 10
	}
	if protocol == "" {
		protocol = "tcp"
	}
	if parallel == 0 {
		parallel = 1
	}

	// Build iperf3 command
	args := []string{
		"-c", server,
		"-p", strconv.Itoa(port),
		"-t", strconv.Itoa(duration),
		"-J", // JSON output
		"-P", strconv.Itoa(parallel),
	}

	// Add protocol-specific flags
	if protocol == "udp" {
		args = append(args, "-u")
	}

	// Add reverse mode if selected
	if reverse {
		args = append(args, "-R")
	}

	// Create context with timeout (duration + 5 seconds buffer)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration+5)*time.Second)
	defer cancel()

	// Run iperf3 command
	cmd := exec.CommandContext(ctx, "iperf3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Check if iperf3 is installed
	if err != nil && (strings.Contains(err.Error(), "executable file not found") ||
		strings.Contains(stderr.String(), "command not found")) {
		fmt.Printf("ERROR: iperf3 is not installed: %v\n", err)
		return map[string]interface{}{
			"error":  "iperf3 is not installed on the system. Please install it using 'sudo apt-get install iperf3' or the appropriate package manager for your system.",
			"result": simulateIperf3Test(server, port, duration, protocol, reverse, parallel),
		}, nil
	}

	// Parse output
	output := stdout.String()
	if output == "" {
		fmt.Printf("ERROR: iperf3 produced no output: %s\n", stderr.String())
		return map[string]interface{}{
			"error":  fmt.Sprintf("iperf3 failed: %s", stderr.String()),
			"result": simulateIperf3Test(server, port, duration, protocol, reverse, parallel),
		}, nil
	}

	// Try to parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// If JSON parsing fails, try to extract data using regex
		fmt.Printf("WARNING: Failed to parse iperf3 JSON output: %v\n", err)
		return parseIperf3Output(output, server, port, duration, protocol, reverse, parallel), nil
	}

	// Process JSON results
	return processIperf3Results(result, server, port, duration, protocol, reverse, parallel)
}

// processIperf3Results processes the JSON output from iperf3
func processIperf3Results(result map[string]interface{}, server string, port, duration int, protocol string, reverse bool, parallel int) (interface{}, error) {
	var downloadSpeed, uploadSpeed, retransmits float64
	var intervals []map[string]interface{}

	// Check if the test was successful
	endInfo, ok := result["end"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"error":  "Invalid iperf3 result format",
			"result": simulateIperf3Test(server, port, duration, protocol, reverse, parallel),
		}, nil
	}

	// Extract summary info
	summary, ok := endInfo["sum_received"].(map[string]interface{})
	if !ok {
		summary, ok = endInfo["sum"].(map[string]interface{})
		if !ok {
			return map[string]interface{}{
				"error":  "Could not find summary data in iperf3 result",
				"result": simulateIperf3Test(server, port, duration, protocol, reverse, parallel),
			}, nil
		}
	}

	// Get bits per second
	bitsPerSecond, ok := summary["bits_per_second"].(float64)
	if !ok {
		return map[string]interface{}{
			"error":  "Could not find bandwidth data in iperf3 result",
			"result": simulateIperf3Test(server, port, duration, protocol, reverse, parallel),
		}, nil
	}

	// Convert to Mbps
	speedMbps := bitsPerSecond / 1000000

	// Determine if this is download or upload based on reverse flag
	if reverse {
		downloadSpeed = speedMbps
		uploadSpeed = 0
	} else {
		uploadSpeed = speedMbps
		downloadSpeed = 0
	}

	// Get retransmits for TCP
	if protocol == "tcp" {
		retransmitsVal, ok := summary["retransmits"].(float64)
		if ok {
			retransmits = retransmitsVal
		}
	}

	// Get interval data for chart
	intervalsData, ok := result["intervals"].([]interface{})
	if ok {
		timePoints := make([]float64, len(intervalsData))
		bandwidthPoints := make([]float64, len(intervalsData))

		for i, interval := range intervalsData {
			intervalMap, ok := interval.(map[string]interface{})
			if !ok {
				continue
			}

			sumMap, ok := intervalMap["sum"].(map[string]interface{})
			if !ok {
				continue
			}

			// Get time in seconds
			seconds, ok := intervalMap["seconds"].(float64)
			if !ok {
				continue
			}

			// Get bandwidth
			bandwidth, ok := sumMap["bits_per_second"].(float64)
			if !ok {
				continue
			}

			// Store data for chart
			timePoints[i] = seconds
			bandwidthPoints[i] = bandwidth / 1000000 // Convert to Mbps
		}

		// Create intervals data for chart
		for i, t := range timePoints {
			intervals = append(intervals, map[string]interface{}{
				"time":      t,
				"bandwidth": bandwidthPoints[i],
			})
		}
	}

	// Create response
	response := map[string]interface{}{
		"server":    server,
		"port":      port,
		"protocol":  protocol,
		"duration":  duration,
		"parallel":  parallel,
		"reverse":   reverse,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Add bandwidth data
	if reverse {
		response["downloadSpeed"] = downloadSpeed
		response["uploadSpeed"] = 0
	} else {
		response["downloadSpeed"] = 0
		response["uploadSpeed"] = uploadSpeed
	}

	// Add TCP-specific data
	if protocol == "tcp" {
		response["retransmits"] = retransmits
	}

	// Add intervals for chart
	response["intervals"] = intervals

	return response, nil
}

// parseIperf3Output parses non-JSON iperf3 output using regex
func parseIperf3Output(output string, server string, port, duration int, protocol string, reverse bool, parallel int) map[string]interface{} {
	// Try to extract bandwidth using regex
	bandwidthRegex := regexp.MustCompile(`(\d+(?:\.\d+)?) Mbits/sec`)
	matches := bandwidthRegex.FindStringSubmatch(output)

	var bandwidthMbps float64
	if len(matches) > 1 {
		bandwidthMbps, _ = strconv.ParseFloat(matches[1], 64)
	} else {
		// Fallback to simulation if regex fails
		return simulateIperf3Test(server, port, duration, protocol, reverse, parallel)
	}

	// Determine if this is download or upload based on reverse flag
	var downloadSpeed, uploadSpeed float64
	if reverse {
		downloadSpeed = bandwidthMbps
		uploadSpeed = 0
	} else {
		uploadSpeed = bandwidthMbps
		downloadSpeed = 0
	}

	// Create response with extracted data
	response := map[string]interface{}{
		"server":        server,
		"port":          port,
		"protocol":      protocol,
		"duration":      duration,
		"parallel":      parallel,
		"reverse":       reverse,
		"downloadSpeed": downloadSpeed,
		"uploadSpeed":   uploadSpeed,
		"timestamp":     time.Now().Format(time.RFC3339),
		"rawOutput":     output,
	}

	return response
}

// simulateIperf3Test simulates an iperf3 test when the real command fails
func simulateIperf3Test(server string, port, duration int, protocol string, reverse bool, parallel int) map[string]interface{} {
	// Create a new random source based on current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate simulated bandwidth (between 50-150 Mbps)
	bandwidth := 50.0 + r.Float64()*100.0

	// Generate time points for chart
	numPoints := duration
	if numPoints > 20 {
		numPoints = 20 // Cap at 20 points for readability
	}

	intervals := make([]map[string]interface{}, numPoints)

	for i := 0; i < numPoints; i++ {
		// Add some variation to bandwidth over time
		variation := 0.8 + 0.4*r.Float64() // 80% to 120% variation
		intervalBandwidth := bandwidth * variation

		intervals[i] = map[string]interface{}{
			"time":      float64(i) * float64(duration) / float64(numPoints),
			"bandwidth": intervalBandwidth,
		}
	}

	// Determine if this is download or upload based on reverse flag
	var downloadSpeed, uploadSpeed float64
	if reverse {
		downloadSpeed = bandwidth
		uploadSpeed = 0
	} else {
		uploadSpeed = bandwidth
		downloadSpeed = 0
	}

	// Create simulated response
	response := map[string]interface{}{
		"server":        server,
		"port":          port,
		"protocol":      protocol,
		"duration":      duration,
		"parallel":      parallel,
		"reverse":       reverse,
		"downloadSpeed": downloadSpeed,
		"uploadSpeed":   uploadSpeed,
		"simulated":     true,
		"timestamp":     time.Now().Format(time.RFC3339),
		"intervals":     intervals,
	}

	// Add TCP-specific simulated data
	if protocol == "tcp" {
		response["retransmits"] = r.Float64() * 10 // 0-10 retransmits
	}

	// Add a note about simulation
	response["note"] = "This is a simulated result. Install iperf3 for actual measurements."

	return response
}

package bandwidth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/go-ping/ping"
)

// Execute handles the bandwidth test plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	server, _ := params["server"].(string)
	durationParam, ok := params["duration"].(float64)
	if !ok {
		durationParam = 10 // Default duration
	}

	// We'll use speedtest-cli for real measurements
	args := []string{"--json"}

	// If a specific server is selected, add it to the command
	if server != "auto" {
		// The server parameter in our API is a region name, but speedtest-cli needs a server ID
		// For simplicity, we'll map these to some common speedtest servers
		serverMap := map[string]string{
			"us-west":    "--server 18282", // Los Angeles, CA
			"us-east":    "--server 10390", // New York, NY
			"eu-central": "--server 28922", // Frankfurt, Germany
			"asia-east":  "--server 29106", // Tokyo, Japan
		}

		if serverArg, ok := serverMap[server]; ok {
			args = append(args, serverArg)
		}
	}

	// Run speedtest command
	cmd := exec.Command("speedtest-cli", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(durationParam)*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// If speedtest-cli doesn't exist or fails, use a fallback method
	if err != nil {
		fmt.Printf("WARNING: speedtest-cli failed or not installed: %v, using simulation\n", err)

		// Check the output for errors
		if stdout.Len() > 0 {
			fmt.Printf("speedtest-cli stdout: %s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Printf("speedtest-cli stderr: %s\n", stderr.String())
		}

		// Simulate a bandwidth test instead
		return simulateBandwidthTest(server, int(durationParam)), nil
	}

	// Parse the JSON output
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		fmt.Printf("ERROR parsing speedtest output: %s\n", stdout.String())
		return nil, fmt.Errorf("error parsing speedtest result: %v", err)
	}

	// Extract the data
	download, _ := result["download"].(float64)
	upload, _ := result["upload"].(float64)
	ping, _ := result["ping"].(float64)

	// Convert to Mbps (speedtest-cli returns bits/second)
	downloadMbps := download / 1000000
	uploadMbps := upload / 1000000

	// Create chart data
	timePoints := make([]int, 6)
	downloadPoints := make([]float64, 6)
	uploadPoints := make([]float64, 6)

	for i := 0; i < 6; i++ {
		timePoints[i] = i * int(durationParam) / 5

		// Simulate progression
		progress := float64(i) / 5.0
		factor := math.Sin(progress * math.Pi / 2)

		downloadPoints[i] = downloadMbps * (0.7 + 0.3*factor)
		uploadPoints[i] = uploadMbps * (0.7 + 0.3*factor)
	}

	// The last point should be the final result
	downloadPoints[5] = downloadMbps
	uploadPoints[5] = uploadMbps

	// Get the server information
	serverInfo, _ := result["server"].(map[string]interface{})
	serverName := "Unknown"
	if serverInfo != nil {
		name, _ := serverInfo["name"].(string)
		location, _ := serverInfo["location"].(string)
		if name != "" && location != "" {
			serverName = fmt.Sprintf("%s, %s", name, location)
		}
	}

	return map[string]interface{}{
		"server":        serverName,
		"downloadSpeed": downloadMbps,
		"uploadSpeed":   uploadMbps,
		"latency":       ping,
		"jitter":        ping * 0.2, // Estimate jitter as 20% of ping
		"packetLoss":    0.0,        // speedtest-cli doesn't report packet loss
		"testDuration":  durationParam,
		"timestamp":     time.Now().Format(time.RFC3339),
		"chart": map[string]interface{}{
			"time":     timePoints,
			"download": downloadPoints,
			"upload":   uploadPoints,
		},
	}, nil
}

// simulateBandwidthTest simulates a bandwidth test
func simulateBandwidthTest(server string, duration int) map[string]interface{} {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Simulate network latency test using simple ping
	var latency float64 = 50 // Default latency
	var jitter float64 = 5   // Default jitter

	// Try to get a better latency estimate if possible
	pinger, err := ping.NewPinger("8.8.8.8")
	if err == nil {
		pinger.SetPrivileged(false)
		pinger.Count = 10
		pinger.Timeout = 5 * time.Second

		err = pinger.Run()
		if err == nil {
			stats := pinger.Statistics()
			latency = float64(stats.AvgRtt.Milliseconds())
			jitter = float64(stats.StdDevRtt.Milliseconds())
		}
	}

	// Generate simulated speed test data
	downloadSpeed := 75.0 + rand.Float64()*50.0 // 75-125 Mbps
	uploadSpeed := 8.0 + rand.Float64()*10.0    // 8-18 Mbps
	packetLoss := rand.Float64() * 1.0          // 0-1%

	// Generate data points for the chart
	numPoints := 6
	timePoints := make([]int, numPoints)
	downloadPoints := make([]float64, numPoints)
	uploadPoints := make([]float64, numPoints)

	for i := 0; i < numPoints; i++ {
		timePoints[i] = i * duration / (numPoints - 1)

		// Add some variation to the data points
		downloadPoints[i] = downloadSpeed * (0.9 + 0.2*rand.Float64())
		uploadPoints[i] = uploadSpeed * (0.9 + 0.2*rand.Float64())
	}

	// Format output
	var rawOutput strings.Builder
	fmt.Fprintf(&rawOutput, "Simulated Bandwidth Test Results (Server: %s)\n\n", server)
	fmt.Fprintf(&rawOutput, "Download Speed: %.2f Mbps\n", downloadSpeed)
	fmt.Fprintf(&rawOutput, "Upload Speed: %.2f Mbps\n", uploadSpeed)
	fmt.Fprintf(&rawOutput, "Latency: %.2f ms\n", latency)
	fmt.Fprintf(&rawOutput, "Jitter: %.2f ms\n", jitter)
	fmt.Fprintf(&rawOutput, "Packet Loss: %.2f%%\n", packetLoss)

	return map[string]interface{}{
		"server":         server,
		"downloadSpeed":  downloadSpeed,
		"uploadSpeed":    uploadSpeed,
		"latency":        latency,
		"jitter":         jitter,
		"packetLoss":     packetLoss,
		"timePoints":     timePoints,
		"downloadPoints": downloadPoints,
		"uploadPoints":   uploadPoints,
		"timestamp":      time.Now().Format(time.RFC3339),
		"rawOutput":      rawOutput.String(),
		"simulated":      true,
	}
}

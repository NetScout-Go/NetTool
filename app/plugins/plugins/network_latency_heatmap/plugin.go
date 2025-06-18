package network_latency_heatmap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

// Execute runs the network latency heatmap plugin
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters with validation and defaults
	targetsStr, ok := params["targets"].(string)
	if !ok || targetsStr == "" {
		return nil, fmt.Errorf("target hosts parameter is required")
	}

	// Split the targets string into individual hosts
	targets := strings.Split(targetsStr, ",")
	for i, target := range targets {
		targets[i] = strings.TrimSpace(target)
	}

	// Get interval parameter (default: 1 second)
	interval := 1.0
	if intervalParam, ok := params["interval"].(float64); ok && intervalParam > 0 {
		interval = intervalParam
	}

	// Get samples parameter (default: 30)
	samples := 30
	if samplesParam, ok := params["samples"].(float64); ok && samplesParam > 0 {
		samples = int(samplesParam)
	}

	// Get timeout parameter (default: 2 seconds)
	timeout := 2.0
	if timeoutParam, ok := params["timeout"].(float64); ok && timeoutParam > 0 {
		timeout = timeoutParam
	}

	// Get packet size parameter (default: 56 bytes)
	packetSize := 56
	if packetSizeParam, ok := params["packetSize"].(float64); ok && packetSizeParam > 0 {
		packetSize = int(packetSizeParam)
	}

	// Get showGraph parameter (default: true)
	showGraph := true
	if showGraphParam, ok := params["showGraph"].(bool); ok {
		showGraph = showGraphParam
	}

	// Define the results structure
	type pingResult struct {
		Target    string    `json:"target"`
		Timestamp time.Time `json:"timestamp"`
		RTT       float64   `json:"rtt"`     // in milliseconds
		Success   bool      `json:"success"` // true if ping succeeded
	}

	// Create results channel and wait group for goroutines
	resultsChan := make(chan pingResult, len(targets)*samples)
	var wg sync.WaitGroup

	// Define context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// For each target, create a goroutine to ping it repeatedly
	for _, target := range targets {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			for i := 0; i < samples; i++ {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
					// Continue with the ping
				}

				// Create a new pinger
				pinger, err := ping.NewPinger(host)
				if err != nil {
					// Record a failed ping
					resultsChan <- pingResult{
						Target:    host,
						Timestamp: time.Now(),
						RTT:       -1,
						Success:   false,
					}
					// Sleep before next sample
					time.Sleep(time.Duration(interval * float64(time.Second)))
					continue
				}

				// Configure the pinger
				pinger.Count = 1
				pinger.Size = packetSize
				pinger.Timeout = time.Duration(timeout * float64(time.Second))
				pinger.SetPrivileged(true) // May require sudo on some systems

				// Run the ping
				err = pinger.Run()
				if err != nil {
					// Record a failed ping
					resultsChan <- pingResult{
						Target:    host,
						Timestamp: time.Now(),
						RTT:       -1,
						Success:   false,
					}
				} else {
					// Get statistics
					stats := pinger.Statistics()
					if stats.PacketsRecv > 0 {
						// Record a successful ping
						resultsChan <- pingResult{
							Target:    host,
							Timestamp: time.Now(),
							RTT:       float64(stats.AvgRtt.Microseconds()) / 1000.0, // Convert to milliseconds
							Success:   true,
						}
					} else {
						// Record a failed ping (timeout)
						resultsChan <- pingResult{
							Target:    host,
							Timestamp: time.Now(),
							RTT:       -1,
							Success:   false,
						}
					}
				}

				// Sleep before next sample
				time.Sleep(time.Duration(interval * float64(time.Second)))
			}
		}(target)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect all results
	var results []pingResult
	for result := range resultsChan {
		results = append(results, result)
	}

	// Organize results by target for easier processing
	targetResults := make(map[string][]pingResult)
	for _, result := range results {
		targetResults[result.Target] = append(targetResults[result.Target], result)
	}

	// For each target, sort results by timestamp
	for target, res := range targetResults {
		sort.Slice(res, func(i, j int) bool {
			return res[i].Timestamp.Before(res[j].Timestamp)
		})
		targetResults[target] = res
	}

	// Calculate statistics for each target
	type targetStats struct {
		Target     string    `json:"target"`
		MinRTT     float64   `json:"minRtt"`     // in milliseconds
		AvgRTT     float64   `json:"avgRtt"`     // in milliseconds
		MaxRTT     float64   `json:"maxRtt"`     // in milliseconds
		MedianRTT  float64   `json:"medianRtt"`  // in milliseconds
		Jitter     float64   `json:"jitter"`     // in milliseconds
		PacketLoss float64   `json:"packetLoss"` // percentage
		RTTs       []float64 `json:"rtts"`       // all RTTs for visualization
		Timestamps []string  `json:"timestamps"` // all timestamps for visualization
	}

	var allStats []targetStats
	for target, res := range targetResults {
		// Calculate statistics
		var successCount int
		var totalRTT float64
		var rtts []float64
		var timestamps []string

		// Initialize with impossible values
		minRTT := float64(9999999)
		maxRTT := float64(-1)

		for _, r := range res {
			timestamps = append(timestamps, r.Timestamp.Format(time.RFC3339))

			if r.Success {
				successCount++
				totalRTT += r.RTT
				rtts = append(rtts, r.RTT)

				if r.RTT < minRTT {
					minRTT = r.RTT
				}
				if r.RTT > maxRTT {
					maxRTT = r.RTT
				}
			} else {
				// For visualization, use -1 to indicate failed pings
				rtts = append(rtts, -1)
			}
		}

		// Calculate average, median, and jitter
		avgRTT := 0.0
		medianRTT := 0.0
		jitter := 0.0

		if successCount > 0 {
			avgRTT = totalRTT / float64(successCount)

			// Calculate median
			successRTTs := make([]float64, 0, successCount)
			for _, rtt := range rtts {
				if rtt >= 0 {
					successRTTs = append(successRTTs, rtt)
				}
			}

			if len(successRTTs) > 0 {
				sort.Float64s(successRTTs)
				if len(successRTTs)%2 == 0 {
					medianRTT = (successRTTs[len(successRTTs)/2-1] + successRTTs[len(successRTTs)/2]) / 2
				} else {
					medianRTT = successRTTs[len(successRTTs)/2]
				}

				// Calculate jitter (average deviation from mean)
				var totalDev float64
				for _, rtt := range successRTTs {
					totalDev += absFloat(rtt - avgRTT)
				}
				jitter = totalDev / float64(len(successRTTs))
			}
		}

		// If no successful pings, set min/max to 0
		if minRTT == 9999999 {
			minRTT = 0
		}
		if maxRTT == -1 {
			maxRTT = 0
		}

		// Calculate packet loss
		packetLoss := float64(len(res)-successCount) / float64(len(res)) * 100

		// Add to all stats
		allStats = append(allStats, targetStats{
			Target:     target,
			MinRTT:     roundFloat(minRTT, 2),
			AvgRTT:     roundFloat(avgRTT, 2),
			MaxRTT:     roundFloat(maxRTT, 2),
			MedianRTT:  roundFloat(medianRTT, 2),
			Jitter:     roundFloat(jitter, 2),
			PacketLoss: roundFloat(packetLoss, 2),
			RTTs:       rtts,
			Timestamps: timestamps,
		})
	}

	// Sort stats by target name for consistent output
	sort.Slice(allStats, func(i, j int) bool {
		return allStats[i].Target < allStats[j].Target
	})

	// Generate heatmap data in a format suitable for visualization
	type heatmapData struct {
		Targets     []string    `json:"targets"`     // List of all targets
		Timestamps  []string    `json:"timestamps"`  // List of all timestamps
		LatencyData [][]float64 `json:"latencyData"` // 2D array of latency values
		MinLatency  float64     `json:"minLatency"`  // Minimum latency value for color scaling
		MaxLatency  float64     `json:"maxLatency"`  // Maximum latency value for color scaling
	}

	// Prepare heatmap data
	heatmap := heatmapData{
		Targets:    make([]string, 0, len(allStats)),
		Timestamps: make([]string, 0),
		MinLatency: 9999999,
		MaxLatency: 0,
	}

	// Add targets
	for _, stat := range allStats {
		heatmap.Targets = append(heatmap.Targets, stat.Target)
	}

	// Find common timestamps (use the first target's timestamps as reference)
	if len(allStats) > 0 {
		heatmap.Timestamps = allStats[0].Timestamps
	}

	// Initialize latency data array
	heatmap.LatencyData = make([][]float64, len(heatmap.Targets))
	for i := range heatmap.LatencyData {
		heatmap.LatencyData[i] = make([]float64, len(heatmap.Timestamps))
	}

	// Fill latency data and find min/max values
	for i, stat := range allStats {
		for j, rtt := range stat.RTTs {
			if j < len(heatmap.LatencyData[i]) {
				heatmap.LatencyData[i][j] = rtt

				// Update min/max (only consider successful pings)
				if rtt > 0 {
					if rtt < heatmap.MinLatency {
						heatmap.MinLatency = rtt
					}
					if rtt > heatmap.MaxLatency {
						heatmap.MaxLatency = rtt
					}
				}
			}
		}
	}

	// If no successful pings, set default range
	if heatmap.MinLatency == 9999999 || heatmap.MaxLatency == 0 {
		heatmap.MinLatency = 0
		heatmap.MaxLatency = 100
	}

	// Prepare final result structure
	result := map[string]interface{}{
		"targets":     targets,
		"interval":    interval,
		"samples":     samples,
		"timeout":     timeout,
		"packetSize":  packetSize,
		"statistics":  allStats,
		"heatmapData": heatmap,
		"showGraph":   showGraph,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	return result, nil
}

// Helper function for absolute value of float64
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Helper function to round float to specified decimal places
func roundFloat(x float64, decimals int) float64 {
	// Quick implementation for rounding
	if x == 0 {
		return 0
	}

	// For simplicity in this plugin
	// This is not a precise implementation but works for our visualization needs
	factor := float64(1)
	for i := 0; i < decimals; i++ {
		factor *= 10
	}

	return float64(int(x*factor+0.5)) / factor
}

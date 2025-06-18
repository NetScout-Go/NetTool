package network_quality

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

// Execute handles the network quality monitor plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	target, ok := params["target"].(string)
	if !ok || target == "" {
		target = "8.8.8.8" // Default target
	}

	durationFloat, ok := params["duration"].(float64)
	if !ok {
		durationFloat = 30 // Default duration in seconds
	}
	duration := time.Duration(durationFloat) * time.Second

	intervalFloat, ok := params["interval"].(float64)
	if !ok {
		intervalFloat = 1 // Default interval in seconds
	}
	interval := time.Duration(intervalFloat*1000) * time.Millisecond

	packetSizeFloat, ok := params["packet_size"].(float64)
	if !ok {
		packetSizeFloat = 56 // Default packet size in bytes
	}
	packetSize := int(packetSizeFloat)

	// Initialize result structure
	result := map[string]interface{}{
		"target":      target,
		"duration":    durationFloat,
		"interval":    intervalFloat,
		"packet_size": packetSize,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// Run the network quality test
	metrics, err := monitorNetworkQuality(target, duration, interval, packetSize)
	if err != nil {
		result["error"] = err.Error()
		return result, nil
	}

	// Add metrics to result
	result["metrics"] = metrics

	// Calculate summary statistics
	if len(metrics) > 0 {
		// Extract latency values
		var latencies []float64
		var packetLossValues []float64
		var jitterValues []float64

		for _, metric := range metrics {
			if latency, ok := metric["latency"].(float64); ok {
				latencies = append(latencies, latency)
			}
			if packetLoss, ok := metric["packet_loss"].(float64); ok {
				packetLossValues = append(packetLossValues, packetLoss)
			}
			if jitter, ok := metric["jitter"].(float64); ok {
				jitterValues = append(jitterValues, jitter)
			}
		}

		// Calculate latency statistics
		if len(latencies) > 0 {
			latencyStats := calculateStats(latencies)
			result["latency_stats"] = latencyStats
		}

		// Calculate packet loss statistics
		if len(packetLossValues) > 0 {
			packetLossStats := calculateStats(packetLossValues)
			result["packet_loss_stats"] = packetLossStats
		}

		// Calculate jitter statistics
		if len(jitterValues) > 0 {
			jitterStats := calculateStats(jitterValues)
			result["jitter_stats"] = jitterStats
		}

		// Determine overall network quality
		networkQuality := "excellent"
		avgLatency := 0.0
		avgJitter := 0.0
		avgPacketLoss := 0.0

		if len(latencies) > 0 {
			avgLatency = sum(latencies) / float64(len(latencies))
		}

		if len(jitterValues) > 0 {
			avgJitter = sum(jitterValues) / float64(len(jitterValues))
		}

		if len(packetLossValues) > 0 {
			avgPacketLoss = sum(packetLossValues) / float64(len(packetLossValues))
		}

		// Determine network quality based on metrics
		if avgPacketLoss > 5 || avgLatency > 300 || avgJitter > 50 {
			networkQuality = "poor"
		} else if avgPacketLoss > 1 || avgLatency > 100 || avgJitter > 20 {
			networkQuality = "fair"
		} else if avgPacketLoss > 0.1 || avgLatency > 50 || avgJitter > 10 {
			networkQuality = "good"
		}

		result["network_quality"] = networkQuality
		result["avg_latency"] = avgLatency
		result["avg_jitter"] = avgJitter
		result["avg_packet_loss"] = avgPacketLoss
	}

	return result, nil
}

// monitorNetworkQuality performs the actual network quality monitoring
func monitorNetworkQuality(target string, duration time.Duration, interval time.Duration, packetSize int) ([]map[string]interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), duration+5*time.Second)
	defer cancel()

	// Initialize pinger
	pinger, err := ping.NewPinger(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create pinger: %w", err)
	}

	pinger.SetPrivileged(false) // Try unprivileged mode first
	pinger.Size = packetSize
	pinger.Count = 3 // Send 3 pings per interval
	pinger.Timeout = interval

	// Initialize metrics storage
	var metrics []map[string]interface{}
	var metricsMutex sync.Mutex
	var prevLatency float64

	// Start time
	startTime := time.Now()
	endTime := startTime.Add(duration)

	// Run the test until the duration is reached
	for time.Now().Before(endTime) {
		// Run a ping test
		err := pinger.Run()
		if err != nil {
			// Try privileged mode if unprivileged fails
			pinger.SetPrivileged(true)
			err = pinger.Run()
			if err != nil {
				return nil, fmt.Errorf("ping failed: %w", err)
			}
		}

		// Get statistics
		stats := pinger.Statistics()

		// Calculate metrics
		packetLoss := stats.PacketLoss
		avgLatency := float64(stats.AvgRtt) / float64(time.Millisecond)

		// Calculate jitter (variation in latency)
		jitter := 0.0
		if prevLatency > 0 {
			jitter = math.Abs(avgLatency - prevLatency)
		}
		prevLatency = avgLatency

		// Record metrics
		metricsMutex.Lock()
		metrics = append(metrics, map[string]interface{}{
			"timestamp":   time.Now().Format(time.RFC3339),
			"latency":     avgLatency,
			"packet_loss": packetLoss,
			"jitter":      jitter,
		})
		metricsMutex.Unlock()

		// Check if we've reached the end time
		if time.Now().Add(interval).After(endTime) {
			break
		}

		// Wait for the next interval
		select {
		case <-ctx.Done():
			return metrics, nil
		case <-time.After(interval):
			// Continue to next sample
		}
	}

	return metrics, nil
}

// calculateStats calculates statistical measures for a slice of float64 values
func calculateStats(values []float64) map[string]interface{} {
	if len(values) == 0 {
		return map[string]interface{}{}
	}

	// Make a copy to avoid modifying the original slice
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate statistics
	min := sorted[0]
	max := sorted[len(sorted)-1]
	mean := sum(sorted) / float64(len(sorted))

	// Calculate median
	median := 0.0
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	// Calculate standard deviation
	variance := 0.0
	for _, v := range sorted {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(sorted))
	stdDev := math.Sqrt(variance)

	return map[string]interface{}{
		"min":    min,
		"max":    max,
		"mean":   mean,
		"median": median,
		"stddev": stdDev,
	}
}

// sum calculates the sum of a slice of float64 values
func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

package mtu_tester

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Execute handles the MTU size tester plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	target, ok := params["target"].(string)
	if !ok || target == "" {
		target = "8.8.8.8" // Default target
	}

	minMTUFloat, ok := params["min_mtu"].(float64)
	if !ok {
		minMTUFloat = 576 // Default min MTU
	}
	minMTU := int(minMTUFloat)

	maxMTUFloat, ok := params["max_mtu"].(float64)
	if !ok {
		maxMTUFloat = 1500 // Default max MTU
	}
	maxMTU := int(maxMTUFloat)

	stepSizeFloat, ok := params["step_size"].(float64)
	if !ok {
		stepSizeFloat = 8 // Default step size
	}
	stepSize := int(stepSizeFloat)

	// Initialize result structure
	result := map[string]interface{}{
		"target":    target,
		"min_mtu":   minMTU,
		"max_mtu":   maxMTU,
		"step_size": stepSize,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check if ping command is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checkCmd := exec.CommandContext(ctx, "which", "ping")
	if err := checkCmd.Run(); err != nil {
		result["error"] = "ping command not found"
		return result, nil
	}

	// First, verify that we can reach the target at all
	if err := checkTargetReachable(target); err != nil {
		result["error"] = fmt.Sprintf("Target %s is not reachable: %s", target, err.Error())
		return result, nil
	}

	// Test MTU using binary search for efficiency
	optimumMTU, testResults, err := findOptimalMTU(target, minMTU, maxMTU, stepSize)
	if err != nil {
		result["error"] = fmt.Sprintf("MTU test failed: %s", err.Error())
		return result, nil
	}

	// Add test results to the result
	result["optimum_mtu"] = optimumMTU
	result["test_results"] = testResults

	// Add recommendations based on the results
	recommendations := generateRecommendations(optimumMTU, maxMTU)
	result["recommendations"] = recommendations

	return result, nil
}

// checkTargetReachable checks if the target is reachable
func checkTargetReachable(target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "5", target)
	return cmd.Run()
}

// findOptimalMTU finds the optimal MTU size using binary search
func findOptimalMTU(target string, minMTU, maxMTU, stepSize int) (int, []map[string]interface{}, error) {
	var testResults []map[string]interface{}

	// Start with binary search to quickly narrow down the range
	low := minMTU
	high := maxMTU
	var optimalMTU int

	for low <= high {
		mid := (low + high) / 2
		// Round to the nearest step size
		mid = (mid / stepSize) * stepSize

		success, err := testMTU(target, mid)
		if err != nil {
			return 0, nil, err
		}

		// Record test result
		testResults = append(testResults, map[string]interface{}{
			"mtu":     mid,
			"success": success,
		})

		if success {
			optimalMTU = mid
			low = mid + stepSize // Try larger MTU
		} else {
			high = mid - stepSize // Try smaller MTU
		}
	}

	// If we didn't find a working MTU, use the minimum
	if optimalMTU == 0 {
		optimalMTU = minMTU
	}

	// Fine-tune the result with linear search
	fineTunedMTU := optimalMTU
	for i := 1; i < stepSize; i++ {
		mtuToTest := optimalMTU + i
		if mtuToTest > maxMTU {
			break
		}

		success, err := testMTU(target, mtuToTest)
		if err != nil {
			return optimalMTU, testResults, nil // Return what we have so far
		}

		// Record test result
		testResults = append(testResults, map[string]interface{}{
			"mtu":     mtuToTest,
			"success": success,
		})

		if success {
			fineTunedMTU = mtuToTest
		} else {
			break
		}
	}

	return fineTunedMTU, testResults, nil
}

// testMTU tests if a specific MTU size works
func testMTU(target string, mtuSize int) (bool, error) {
	// Calculate the packet size: MTU - IP header (20) - ICMP header (8)
	dataSize := mtuSize - 28

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run ping with don't fragment flag and specified packet size
	cmd := exec.CommandContext(ctx, "ping", "-c", "3", "-M", "do", "-s", strconv.Itoa(dataSize), "-W", "2", target)
	output, err := cmd.CombinedOutput()

	// Check if the ping was successful
	if err != nil {
		// If the error is due to packet fragmentation, it's not a failure of the test
		if strings.Contains(string(output), "Message too long") ||
			strings.Contains(string(output), "fragmentation needed") {
			return false, nil
		}

		// For other errors, just return the result
		return false, nil
	}

	// Check if there were any successful pings
	if strings.Contains(string(output), " 0% packet loss") {
		return true, nil
	} else if strings.Contains(string(output), "100% packet loss") {
		return false, nil
	}

	// If we received at least some responses, consider it successful
	return true, nil
}

// generateRecommendations generates recommendations based on the optimal MTU
func generateRecommendations(optimalMTU, maxMTU int) []string {
	var recommendations []string

	if optimalMTU >= maxMTU {
		recommendations = append(recommendations,
			"Your connection supports the standard MTU size. No changes are needed.")
	} else if optimalMTU >= 1400 {
		recommendations = append(recommendations,
			fmt.Sprintf("Your optimal MTU is %d, which is slightly below the standard 1500 bytes.", optimalMTU),
			"This is common with some VPN or PPPoE connections.")
	} else if optimalMTU >= 1000 {
		recommendations = append(recommendations,
			fmt.Sprintf("Your optimal MTU is %d, which is significantly below the standard.", optimalMTU),
			"You may be experiencing packet fragmentation issues.",
			fmt.Sprintf("Consider setting your network interface MTU to %d.", optimalMTU))
	} else {
		recommendations = append(recommendations,
			fmt.Sprintf("Your optimal MTU is %d, which is very low.", optimalMTU),
			"This might indicate serious network issues or router misconfiguration.",
			"Check your network equipment and connection type.",
			fmt.Sprintf("For now, set your network interface MTU to %d to avoid fragmentation.", optimalMTU))
	}

	// Add command example
	recommendations = append(recommendations,
		fmt.Sprintf("To set MTU on Linux: sudo ip link set dev <interface> mtu %d", optimalMTU))

	return recommendations
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TraceroutePlugin is the main plugin struct
type TraceroutePlugin struct {
	Results        []interface{}
	StartTime      time.Time
	IterationCount int
}

// NewPlugin creates a new plugin instance
func NewPlugin() *TraceroutePlugin {
	return &TraceroutePlugin{
		StartTime: time.Now(),
		Results:   []interface{}{},
	}
}

// Execute handles the traceroute plugin execution
func (p *TraceroutePlugin) Execute(params map[string]interface{}) (interface{}, error) {
	// Check if we should use iteration
	continueToIterate, _ := params["continueToIterate"].(bool)
	if continueToIterate {
		return p.executeWithIteration(params)
	}

	// Run a single execution
	return p.performTraceroute(params)
}

// executeWithIteration handles running the plugin in iteration mode
func (p *TraceroutePlugin) executeWithIteration(params map[string]interface{}) (interface{}, error) {
	// Run the traceroute operation
	result, err := p.performTraceroute(params)
	if err != nil {
		return nil, err
	}

	// Update state
	p.IterationCount++
	if resultMap, ok := result.(map[string]interface{}); ok {
		// Create a copy of the result for history to avoid reference issues
		historyCopy := make(map[string]interface{})
		for k, v := range resultMap {
			historyCopy[k] = v
		}
		p.Results = append(p.Results, historyCopy)

		// Add iteration metadata to the result
		resultMap["iterationCount"] = p.IterationCount
		resultMap["elapsedTime"] = time.Since(p.StartTime).String()

		// Create a summary for the UI
		host := resultMap["host"].(string)
		if hops, ok := resultMap["hops"].([]map[string]interface{}); ok {
			hopCount := len(hops)
			var lastHopIP string
			if hopCount > 0 {
				if ip, ok := hops[hopCount-1]["ip"].(string); ok {
					lastHopIP = ip
				} else if host, ok := hops[hopCount-1]["host"].(string); ok {
					lastHopIP = host
				}
			}

			// Add iteration_data for UI display
			resultMap["iteration_data"] = map[string]interface{}{
				"can_iterate":        true,
				"supports_iteration": true,
				"iteration_summary": fmt.Sprintf(
					"Iteration %d: %s - %d hops, final: %s",
					p.IterationCount,
					host,
					hopCount,
					lastHopIP,
				),
			}
		}

		// Add history summary
		if len(p.Results) > 1 {
			history := make([]map[string]interface{}, 0)
			for i, res := range p.Results {
				if resMap, ok := res.(map[string]interface{}); ok {
					// Create a simplified history entry
					host := resMap["host"].(string)
					timestamp := resMap["timestamp"].(string)

					var hopCount int
					var lastHopIP string
					if hops, ok := resMap["hops"].([]map[string]interface{}); ok {
						hopCount = len(hops)
						if hopCount > 0 {
							if ip, ok := hops[hopCount-1]["ip"].(string); ok {
								lastHopIP = ip
							} else if host, ok := hops[hopCount-1]["host"].(string); ok {
								lastHopIP = host
							}
						}
					}

					historyEntry := map[string]interface{}{
						"iteration": i + 1,
						"timestamp": timestamp,
						"host":      host,
						"hopCount":  hopCount,
						"lastHop":   lastHopIP,
					}
					history = append(history, historyEntry)
				}
			}
			resultMap["history"] = history
		}
	}

	return result, nil
}

// performTraceroute handles the actual traceroute logic
func (p *TraceroutePlugin) performTraceroute(params map[string]interface{}) (interface{}, error) {
	host, _ := params["host"].(string)
	maxHopsParam, ok := params["maxHops"].(float64)
	if !ok {
		maxHopsParam = 30 // Default max hops
	}
	maxHops := int(maxHopsParam)

	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	// Build the traceroute command
	cmd := exec.Command("traceroute", "-n", "-m", fmt.Sprintf("%d", maxHops), host)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil && stderr.Len() > 0 {
		return nil, fmt.Errorf("traceroute failed: %v: %s", err, stderr.String())
	}

	output := stdout.String()

	// Parse the output
	lines := strings.Split(output, "\n")
	hops := []map[string]interface{}{}

	for i, line := range lines {
		if i == 0 || len(line) == 0 {
			continue // Skip the header line and empty lines
		}

		// Extract hop information
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hopNumber, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		var hopIP, hopName string
		var rtt float64

		// Get the hop IP address and RTT
		if len(parts) >= 4 && parts[1] != "*" {
			hopIP = parts[1]

			// Try to get hostname
			addr, err := net.LookupAddr(hopIP)
			if err == nil && len(addr) > 0 {
				hopName = strings.TrimSuffix(addr[0], ".")
			} else {
				hopName = hopIP
			}

			// Get RTT
			rttStr := strings.TrimSuffix(parts[2], "ms")
			rtt, _ = strconv.ParseFloat(rttStr, 64)
		} else {
			hopIP = "*"
			hopName = "*"
			rtt = 0
		}

		hop := map[string]interface{}{
			"hop":  hopNumber,
			"host": hopIP,
			"name": hopName,
			"rtt":  rtt,
			"status": func() string {
				if hopIP != "*" {
					return "OK"
				}
				return "NO RESPONSE"
			}(),
		}

		hops = append(hops, hop)
	}

	return map[string]interface{}{
		"host":      host,
		"hops":      hops,
		"timestamp": time.Now().Format(time.RFC3339),
		"rawOutput": output,
	}, nil
}

// Main function
func main() {
	// Create plugin instance
	plugin := NewPlugin()

	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: plugin.go --definition|--execute='{\"params\":...}'")
		os.Exit(1)
	}

	// Handle --definition argument
	if os.Args[1] == "--definition" {
		// Read plugin.json for definition
		definitionBytes, err := os.ReadFile("plugin.json")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(definitionBytes))
		return
	}

	// Handle --execute argument
	if strings.HasPrefix(os.Args[1], "--execute=") {
		// Extract parameters JSON
		paramsJSON := strings.TrimPrefix(os.Args[1], "--execute=")

		// Parse parameters
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Execute plugin
		result, err := plugin.Execute(params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			os.Exit(1)
		}

		// Output result as JSON
		resultJSON, err := json.Marshal(result)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(resultJSON))
		return
	}

	fmt.Println("Unknown command")
	os.Exit(1)
}

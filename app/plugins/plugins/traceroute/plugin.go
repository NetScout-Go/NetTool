package traceroute

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Execute handles the traceroute plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	host, _ := params["host"].(string)
	maxHopsParam, ok := params["maxHops"].(float64)
	if !ok {
		maxHopsParam = 30 // Default max hops
	}
	maxHops := int(maxHopsParam)

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

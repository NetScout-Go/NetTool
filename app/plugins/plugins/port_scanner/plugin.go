package port_scanner

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Execute handles the port scanner plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	host, _ := params["host"].(string)
	portRangeStr, _ := params["portRange"].(string)
	timeoutParam, ok := params["timeout"].(float64)
	if !ok {
		timeoutParam = 1.0 // Default timeout
	}
	timeout := time.Duration(timeoutParam * float64(time.Second))

	// Parse port range
	portRangeParts := strings.Split(portRangeStr, "-")
	if len(portRangeParts) != 2 {
		return nil, fmt.Errorf("invalid port range format, expected start-end, got %s", portRangeStr)
	}

	startPort, err := strconv.Atoi(strings.TrimSpace(portRangeParts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start port: %v", err)
	}

	endPort, err := strconv.Atoi(strings.TrimSpace(portRangeParts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end port: %v", err)
	}

	if startPort > endPort {
		return nil, fmt.Errorf("start port must be less than or equal to end port")
	}

	if startPort < 1 || endPort > 65535 {
		return nil, fmt.Errorf("ports must be between 1 and 65535")
	}

	// Scan ports
	var openPorts []map[string]interface{}
	closedPorts := 0

	startTime := time.Now()

	// Limit concurrent scans
	maxConcurrent := 50
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	// Create a lookup for common service names
	commonPorts := map[int]string{
		21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP", 53: "DNS",
		80: "HTTP", 110: "POP3", 143: "IMAP", 443: "HTTPS", 587: "SMTP",
		993: "IMAPS", 995: "POP3S", 3306: "MySQL", 5432: "PostgreSQL",
		8080: "HTTP-Alt", 8443: "HTTPS-Alt",
	}

	for port := startPort; port <= endPort; port++ {
		sem <- struct{}{} // Acquire a token
		wg.Add(1)

		go func(p int) {
			defer func() {
				<-sem // Release token
				wg.Done()
			}()

			// Use JoinHostPort to handle IPv6 addresses correctly
			address := net.JoinHostPort(host, strconv.Itoa(p))
			conn, err := net.DialTimeout("tcp", address, timeout)

			if err == nil {
				conn.Close()

				// Determine service name
				serviceName, found := commonPorts[p]
				if !found {
					serviceName = "Unknown"
				}

				portInfo := map[string]interface{}{
					"port":    p,
					"service": serviceName,
					"status":  "open",
				}

				mutex.Lock()
				openPorts = append(openPorts, portInfo)
				mutex.Unlock()
			} else {
				mutex.Lock()
				closedPorts++
				mutex.Unlock()
			}
		}(port)
	}

	wg.Wait()
	scanTime := time.Since(startTime).Seconds()

	// Sort open ports by port number
	sort.Slice(openPorts, func(i, j int) bool {
		return openPorts[i]["port"].(int) < openPorts[j]["port"].(int)
	})

	return map[string]interface{}{
		"host":        host,
		"portRange":   portRangeStr,
		"openPorts":   openPorts,
		"closedPorts": closedPorts,
		"scanTime":    scanTime,
		"timestamp":   time.Now().Format(time.RFC3339),
	}, nil
}

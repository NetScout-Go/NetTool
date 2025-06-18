package iperf3_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// Track the running server process
	serverProcess *exec.Cmd
	// Store server info
	serverInfo map[string]interface{}
	// Mutex for thread safety
	serverMutex sync.Mutex
	// Server status file
	statusFilePath = filepath.Join(os.TempDir(), "netscout_iperf3_server.json")
	// Server log file
	logFilePath = filepath.Join(os.TempDir(), "netscout_iperf3_server.log")
)

// Execute handles the iperf3 server plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	portFloat, _ := params["port"].(float64)
	port := int(portFloat)
	bindAddress, _ := params["bind_address"].(string)
	protocol, _ := params["protocol"].(string)
	durationFloat, _ := params["duration"].(float64)
	duration := int(durationFloat)
	action, _ := params["action"].(string)

	// Set defaults if not provided
	if port == 0 {
		port = 5201
	}
	if protocol == "" {
		protocol = "both"
	}
	if duration == 0 {
		duration = 30 // 30 minutes by default
	}
	if action == "" {
		action = "start"
	}

	// Lock for thread safety
	serverMutex.Lock()
	defer serverMutex.Unlock()

	// Try to load server info from disk
	loadServerInfoFromDisk()

	// Handle different actions
	switch action {
	case "start":
		return startServer(port, bindAddress, protocol, duration)
	case "stop":
		return stopServer()
	case "status":
		return getServerStatus()
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}
}

// startServer starts the iperf3 server
func startServer(port int, bindAddress, protocol string, duration int) (interface{}, error) {
	// Check if iperf3 is installed
	_, err := exec.LookPath("iperf3")
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": "iPerf3 is not installed. Please install it using 'sudo apt-get install iperf3' or the appropriate package manager for your system.",
		}, nil
	}

	// Check if a server is already running
	if serverProcess != nil {
		// If we have a running process, check if it's still actually running
		if serverProcess.ProcessState == nil || !serverProcess.ProcessState.Exited() {
			return map[string]interface{}{
				"status":      "running",
				"message":     "iPerf3 server is already running",
				"server_info": serverInfo,
			}, nil
		}
	}

	// Build iperf3 server command
	args := []string{"-s"}

	// Add port if specified
	if port > 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}

	// Add bind address if specified
	if bindAddress != "" {
		args = append(args, "-B", bindAddress)
	}

	// Configure protocol
	if protocol == "udp" {
		args = append(args, "-u")
	}

	// Setup for running in the background
	cmd := exec.Command("iperf3", args...)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to open log file: %v", err),
		}, nil
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the server
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to start iPerf3 server: %v", err),
		}, nil
	}

	// Get server's IP addresses
	ipAddresses, _ := getLocalIPAddresses()

	// Store server info
	serverProcess = cmd
	serverInfo = map[string]interface{}{
		"pid":          cmd.Process.Pid,
		"port":         port,
		"protocol":     protocol,
		"bind_address": bindAddress,
		"ip_addresses": ipAddresses,
		"started_at":   time.Now().Format(time.RFC3339),
		"log_file":     logFilePath,
		"duration":     duration,
		"status":       "running",
	}

	// Save server info to disk
	saveServerInfoToDisk()

	// If duration is specified, schedule server stop
	if duration > 0 {
		go func() {
			time.Sleep(time.Duration(duration) * time.Minute)
			stopServerAsync()
		}()
	}

	// Let the OS process run independently
	go func() {
		cmd.Wait()
		serverMutex.Lock()
		defer serverMutex.Unlock()

		// Update status when server exits
		if serverInfo != nil {
			serverInfo["status"] = "stopped"
			serverInfo["stopped_at"] = time.Now().Format(time.RFC3339)
			saveServerInfoToDisk()
		}
	}()

	// Return success response
	connectionInstructions := generateConnectionInstructions(port, ipAddresses, protocol)

	return map[string]interface{}{
		"status":                  "started",
		"message":                 "iPerf3 server started successfully",
		"server_info":             serverInfo,
		"connection_instructions": connectionInstructions,
	}, nil
}

// stopServer stops the running iperf3 server
func stopServer() (interface{}, error) {
	// If no server is running
	if serverProcess == nil || (serverProcess.ProcessState != nil && serverProcess.ProcessState.Exited()) {
		return map[string]interface{}{
			"status":  "not_running",
			"message": "No iPerf3 server is currently running",
		}, nil
	}

	// Stop the server process
	if err := serverProcess.Process.Kill(); err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to stop iPerf3 server: %v", err),
		}, nil
	}

	// Update server info
	if serverInfo != nil {
		serverInfo["status"] = "stopped"
		serverInfo["stopped_at"] = time.Now().Format(time.RFC3339)
		saveServerInfoToDisk()
	}

	return map[string]interface{}{
		"status":  "stopped",
		"message": "iPerf3 server stopped successfully",
	}, nil
}

// stopServerAsync stops the server asynchronously
func stopServerAsync() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverProcess != nil && (serverProcess.ProcessState == nil || !serverProcess.ProcessState.Exited()) {
		serverProcess.Process.Kill()

		if serverInfo != nil {
			serverInfo["status"] = "stopped"
			serverInfo["stopped_at"] = time.Now().Format(time.RFC3339)
			saveServerInfoToDisk()
		}
	}
}

// getServerStatus returns the current status of the iperf3 server
func getServerStatus() (interface{}, error) {
	// If server info is not available, try loading from disk
	if serverInfo == nil {
		loadServerInfoFromDisk()
	}

	// If still no server info, the server has never been started
	if serverInfo == nil {
		return map[string]interface{}{
			"status":  "not_running",
			"message": "iPerf3 server has not been started",
		}, nil
	}

	// If we have server info but no process or the process has exited
	if serverProcess == nil || (serverProcess.ProcessState != nil && serverProcess.ProcessState.Exited()) {
		// Check if we need to update the status
		if serverInfo["status"] == "running" {
			serverInfo["status"] = "stopped"
			serverInfo["stopped_at"] = time.Now().Format(time.RFC3339)
			saveServerInfoToDisk()
		}

		return map[string]interface{}{
			"status":      "not_running",
			"message":     "iPerf3 server is not running",
			"server_info": serverInfo,
		}, nil
	}

	// Server is running, get IP addresses again in case they've changed
	ipAddresses, _ := getLocalIPAddresses()
	if serverInfo != nil {
		serverInfo["ip_addresses"] = ipAddresses
	}

	// Generate connection instructions
	port := 5201
	if portVal, ok := serverInfo["port"].(float64); ok {
		port = int(portVal)
	}
	protocol := "both"
	if protocolVal, ok := serverInfo["protocol"].(string); ok {
		protocol = protocolVal
	}

	connectionInstructions := generateConnectionInstructions(port, ipAddresses, protocol)

	return map[string]interface{}{
		"status":                  "running",
		"message":                 "iPerf3 server is running",
		"server_info":             serverInfo,
		"connection_instructions": connectionInstructions,
	}, nil
}

// getLocalIPAddresses returns a list of IP addresses for the local interfaces
func getLocalIPAddresses() ([]string, error) {
	var ipAddresses []string

	// Run 'hostname -I' to get IP addresses
	cmd := exec.Command("hostname", "-I")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()

	if err != nil {
		// Fall back to 'ip addr' if hostname -I fails
		cmd = exec.Command("ip", "addr")
		stdout.Reset()
		cmd.Stdout = &stdout
		err = cmd.Run()

		if err != nil {
			return []string{"127.0.0.1"}, err
		}

		// Parse ip addr output
		output := stdout.String()
		lines := strings.Split(output, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "inet ") && !strings.Contains(line, "127.0.0.1") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					// Extract IP address without subnet
					ip := strings.Split(parts[1], "/")[0]
					ipAddresses = append(ipAddresses, ip)
				}
			}
		}
	} else {
		// Parse hostname -I output
		ips := strings.Fields(stdout.String())
		ipAddresses = append(ipAddresses, ips...)
	}

	// Add localhost if no other IPs found
	if len(ipAddresses) == 0 {
		ipAddresses = append(ipAddresses, "127.0.0.1")
	}

	return ipAddresses, nil
}

// generateConnectionInstructions generates instructions for connecting to the server
func generateConnectionInstructions(port int, ipAddresses []string, protocol string) string {
	var sb strings.Builder
	sb.WriteString("To connect to this iPerf3 server from another device:\n\n")

	protocolFlag := ""
	if protocol == "udp" {
		protocolFlag = " -u"
	}

	sb.WriteString("Run one of these commands on the client device:\n\n")

	for _, ip := range ipAddresses {
		sb.WriteString(fmt.Sprintf("iperf3 -c %s -p %d%s\n", ip, port, protocolFlag))
	}

	sb.WriteString("\nFor more options, run: iperf3 --help")

	return sb.String()
}

// saveServerInfoToDisk saves the server info to a temporary file
func saveServerInfoToDisk() {
	if serverInfo == nil {
		return
	}

	data, err := json.Marshal(serverInfo)
	if err != nil {
		return
	}

	os.WriteFile(statusFilePath, data, 0644)
}

// loadServerInfoFromDisk loads the server info from a temporary file
func loadServerInfoFromDisk() {
	data, err := os.ReadFile(statusFilePath)
	if err != nil {
		return
	}

	var info map[string]interface{}
	if err := json.Unmarshal(data, &info); err != nil {
		return
	}

	serverInfo = info

	// Check if the server is still running based on PID
	if pidFloat, ok := info["pid"].(float64); ok {
		pid := int(pidFloat)

		// Try to find the process
		process, err := os.FindProcess(pid)
		if err != nil {
			serverInfo["status"] = "stopped"
			return
		}

		// On Unix, FindProcess always succeeds, so we need to send signal 0 to check existence
		if err := process.Signal(os.Signal(nil)); err != nil {
			serverInfo["status"] = "stopped"
			return
		}

		// Process exists, recreate the serverProcess
		serverProcess = &exec.Cmd{
			Process: process,
		}
	}
}

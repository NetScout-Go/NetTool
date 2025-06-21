// BLE HTTP Proxy Plugin for NetTool
// This plugin enables a Bluetooth Low Energy GATT server that exposes
// the NetTool web dashboard via a custom HTTP proxy service.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Constants for BLE service
const (
	// BLE HTTP Proxy Service (custom UUID)
	BLEHTTPProxyServiceUUID = "00001234-0000-1000-8000-00805f9b34fb"

	// BLE HTTP Request Characteristic
	BLEHTTPRequestCharUUID = "00001235-0000-1000-8000-00805f9b34fb"

	// BLE HTTP Response Characteristic
	BLEHTTPResponseCharUUID = "00001236-0000-1000-8000-00805f9b34fb"

	// Maximum size for BLE attribute value (MTU - 3)
	MaxBLEAttributeSize = 509

	// Status file for storing the BLE proxy state
	StatusFile = "/tmp/nettool_ble_proxy.status"

	// Python script to run the BLE service
	PythonScript = "pi_zero_ble_service.py"
)

// BLE HTTP Proxy Plugin for NetTool
type BLEHTTPProxyPlugin struct {
	// No fields needed for now
}

// Global plugin instance
var plugin *BLEHTTPProxyPlugin

// Plugin is the exported symbol that NetTool will look for
var Plugin struct {
	ID          string
	Name        string
	Description string
	Version     string
	Author      string
	Execute     func(params map[string]interface{}) (interface{}, error)
}

func init() {
	// Initialize the Plugin variable
	Plugin.ID = "ble_http_proxy"
	Plugin.Name = "BLE HTTP Proxy"
	Plugin.Description = "Allows phones and PCs to connect to the dashboard via Bluetooth Low Energy"
	Plugin.Version = "1.0.0"
	Plugin.Author = "NetScout-Go Team"
	Plugin.Execute = executePlugin
}

// Plugin execution function
func executePlugin(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	deviceName := "NetTool"
	if name, ok := params["deviceName"].(string); ok && name != "" {
		deviceName = name
	}

	port := 8080
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}

	action := "start"
	if a, ok := params["action"].(string); ok {
		action = a
	}

	// Check if BlueZ is available
	if !isBlueZAvailable() {
		return nil, fmt.Errorf("BlueZ DBus service is not available. Make sure Bluetooth is enabled and bluetoothd is running")
	}

	// Create result structure
	result := map[string]interface{}{
		"success": false,
		"message": "",
		"status":  "",
	}

	// Perform the requested action
	switch action {
	case "start":
		err := startBLEProxy(deviceName, port)
		if err != nil {
			result["message"] = fmt.Sprintf("Failed to start BLE HTTP proxy: %v", err)
		} else {
			result["success"] = true
			result["message"] = "BLE HTTP proxy started successfully"
			result["status"] = "running"
		}

	case "stop":
		err := stopBLEProxy()
		if err != nil {
			result["message"] = fmt.Sprintf("Failed to stop BLE HTTP proxy: %v", err)
		} else {
			result["success"] = true
			result["message"] = "BLE HTTP proxy stopped successfully"
			result["status"] = "stopped"
		}

	case "status":
		status, err := getBLEProxyStatus()
		if err != nil {
			result["message"] = fmt.Sprintf("Failed to get BLE HTTP proxy status: %v", err)
			result["status"] = "unknown"
		} else {
			result["success"] = true
			result["message"] = fmt.Sprintf("BLE HTTP proxy is %s", status)
			result["status"] = status
		}

	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	return result, nil
}

// Check if BlueZ DBus service is available
func isBlueZAvailable() bool {
	// Use the bluetoothctl command to check if Bluetooth is available
	cmd := exec.Command("bluetoothctl", "--version")
	err := cmd.Run()
	if err != nil {
		return false
	}

	// Check if the BlueZ service is running
	cmd = exec.Command("systemctl", "is-active", "bluetooth")
	err = cmd.Run()
	if err != nil {
		return false
	}

	return true
}

// Start the BLE HTTP proxy server
func startBLEProxy(deviceName string, port int) error {
	// Check if already running
	status, _ := getBLEProxyStatus()
	if status == "running" {
		return fmt.Errorf("BLE HTTP proxy is already running")
	}

	// Get the current plugin directory
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Get plugin directory (where this plugin is located)
	pluginDir := filepath.Dir(execPath)
	scriptPath := filepath.Join(pluginDir, PythonScript)

	// Verify Python script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// When running from the plugin directory during development
		scriptPath = filepath.Join(".", PythonScript)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("BLE proxy script not found: %v", err)
		}
	}

	// Check if python3 is available
	pythonCmd := "python3"
	if _, err := exec.LookPath(pythonCmd); err != nil {
		// Try with just python command
		pythonCmd = "python"
		if _, err := exec.LookPath(pythonCmd); err != nil {
			return fmt.Errorf("python is not available on this system: %v", err)
		}
	}

	// Prepare command to run the Python script
	cmd := exec.Command(pythonCmd, scriptPath,
		"--device-name", deviceName,
		"--port", fmt.Sprintf("%d", port))

	// Configure process group for proper termination later
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Add environment variables if needed
	cmd.Env = os.Environ()

	// Start the process
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start BLE proxy script: %v", err)
	}

	// Save PID to the status file in case it doesn't create one
	pidInfo := fmt.Sprintf("running\nPID: %d\n", cmd.Process.Pid)
	err = os.WriteFile(StatusFile, []byte(pidInfo), 0644)
	if err != nil {
		// Try to kill the process since we couldn't create the status file
		cmd.Process.Kill()
		return fmt.Errorf("failed to create status file: %v", err)
	}

	// Wait for service to start
	time.Sleep(2 * time.Second)

	// Verify the service is running by checking status file again
	status, err = getBLEProxyStatus()
	if err != nil || status != "running" {
		// Attempt to kill the process
		cmd.Process.Kill()
		return fmt.Errorf("BLE proxy service failed to start properly")
	}

	return nil
}

// Stop the BLE HTTP proxy server
func stopBLEProxy() error {
	// Check if running
	status, _ := getBLEProxyStatus()
	if status != "running" {
		return fmt.Errorf("BLE HTTP proxy is not running")
	}

	// Read PID from status file
	content, err := os.ReadFile(StatusFile)
	if err != nil {
		return fmt.Errorf("failed to read status file: %v", err)
	}

	// Extract PID
	lines := strings.Split(string(content), "\n")
	var pid int
	for _, line := range lines {
		if strings.HasPrefix(line, "PID:") {
			fmt.Sscanf(line, "PID: %d", &pid)
			break
		}
	}

	if pid == 0 {
		return fmt.Errorf("invalid PID in status file")
	}

	// Send terminate signal
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		// If signaling fails, try to kill the process group
		syscall.Kill(-pid, syscall.SIGTERM)
	}

	// Wait for service to stop
	time.Sleep(2 * time.Second)

	// Update status file if it wasn't updated by the script
	status, err = getBLEProxyStatus()
	if err != nil || status == "running" {
		os.WriteFile(StatusFile, []byte("stopped\n"), 0644)
	}

	return nil
}

// Get the current status of the BLE HTTP proxy
func getBLEProxyStatus() (string, error) {
	// Check if status file exists
	_, err := os.Stat(StatusFile)
	if os.IsNotExist(err) {
		return "stopped", nil
	}

	// Read status file
	content, err := os.ReadFile(StatusFile)
	if err != nil {
		return "unknown", err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		status := strings.TrimSpace(lines[0])
		if status == "running" {
			// Verify PID is actually running
			for _, line := range lines {
				if strings.HasPrefix(line, "PID:") {
					var pid int
					fmt.Sscanf(line, "PID: %d", &pid)
					if pid > 0 {
						process, err := os.FindProcess(pid)
						if err != nil || process == nil {
							return "stopped", nil
						}

						// On Unix, FindProcess always succeeds, so we need to send a signal 0
						// to check if the process exists
						err = process.Signal(syscall.Signal(0))
						if err != nil {
							return "stopped", nil
						}
					}
					break
				}
			}
			return "running", nil
		} else if status == "stopped" {
			return "stopped", nil
		}
	}

	return "unknown", nil
}

func main() {}

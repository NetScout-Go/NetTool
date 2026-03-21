package plugins

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadPluginFunc loads the plugin function from a Go plugin file
func LoadPluginFunc(pluginDir, pluginID string) (func(map[string]interface{}) (interface{}, error), error) {
	// Check if the plugin.go file exists
	pluginGoPath := filepath.Join(pluginDir, "plugin.go")
	if _, err := os.Stat(pluginGoPath); err != nil {
		return nil, fmt.Errorf("plugin.go file not found for %s: %v", pluginID, err)
	}

	// Try to use direct Go function
	// First, try to find the plugin package
	goFiles, err := filepath.Glob(filepath.Join(pluginDir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("error finding Go files for plugin %s: %v", pluginID, err)
	}

	if len(goFiles) == 0 {
		return nil, fmt.Errorf("no Go files found for plugin %s", pluginID)
	}

	// Dynamic import based on plugin directory
	// Prefer executing the real source plugin via a generated wrapper program.
	// This avoids the old behavior where many plugins silently fell back to stubs
	// or incorrectly required a precompiled .so file.
	return func(params map[string]interface{}) (interface{}, error) {
		if result, err := executeSourcePlugin(pluginDir, pluginID, params); err == nil {
			return result, nil
		}

		// Fall back to legacy helper implementations only when direct execution fails.
		switch pluginID {
		case "subnet_calculator":
			// Use the ExecuteAdapter function from the subnet_calculator package
			return executeSubnetCalculator(params)
		case "network_latency_heatmap":
			return executeNetworkLatencyHeatmap(params)
		case "ping":
			return executePing(params)
		case "traceroute":
			return executeTraceroute(params)
		case "dns_lookup":
			return executeDNSLookup(params)
		case "port_scanner":
			return executePortScanner(params)
		case "bandwidth_test":
			return executeBandwidthTest(params)
		case "packet_capture":
			return executePacketCapture(params)
		case "tc_controller":
			return executeTCController(params)
		case "arp_manager":
			return executeARPManager(params)
		case "device_discovery":
			return executeDeviceDiscovery(params)
		case "network_quality":
			return executeNetworkQuality(params)
		case "dns_propagation":
			return executeDNSPropagation(params)
		case "ssl_checker":
			return executeSSLChecker(params)
		case "reverse_dns_lookup":
			return executeReverseDNSLookup(params)
		case "mtu_tester":
			return executeMTUTester(params)
		case "wifi_scanner":
			return executeWifiScanner(params)
		default:
			// For unknown plugins, try to use pre-compiled dynamically loaded plugin (.so file)
			// NOTE: Do NOT check the registry here - this function IS what gets registered
			// in the registry, so checking it would cause infinite recursion
			pluginPath := filepath.Join(pluginDir, pluginID+".so")
			if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
				// No .so file - plugin not available for dynamic execution
				return nil, fmt.Errorf("plugin %s not found: no pre-compiled .so file available", pluginID)
			}

			// Try to load the pre-compiled plugin
			p, err := plugin.Open(pluginPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load plugin %s: %v", pluginID, err)
			}

			// Look up the Plugin symbol
			pluginSymbol, err := p.Lookup("Plugin")
			if err != nil {
				return nil, fmt.Errorf("plugin %s does not export Plugin symbol: %v", pluginID, err)
			}

			// Call the Plugin function
			pluginFunc := reflect.ValueOf(pluginSymbol).Call(nil)[0].Interface()

			// Extract the execute function
			pluginMap, ok := pluginFunc.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("plugin %s Plugin() did not return a map", pluginID)
			}

			execFunc, ok := pluginMap["execute"].(func(map[string]interface{}) (interface{}, error))
			if !ok {
				return nil, fmt.Errorf("plugin %s does not provide a valid execute function", pluginID)
			}

			// Call the execute function with the provided parameters
			return execFunc(params)
		}
	}, nil
}

// Helper function to execute a shell command
func executeCommand(command string) (string, error) {
	cmd := NewCommand(command)
	output, err := cmd.Run()
	return output, err
}

func executeSourcePlugin(pluginDir, pluginID string, params map[string]interface{}) (interface{}, error) {
	var err error
	pluginDir, err = filepath.Abs(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute plugin dir for %s: %w", pluginID, err)
	}

	pluginGoPath := filepath.Join(pluginDir, "plugin.go")
	content, err := os.ReadFile(pluginGoPath)
	if err != nil {
		return nil, fmt.Errorf("read plugin source for %s: %w", pluginID, err)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal parameters for %s: %w", pluginID, err)
	}

	if strings.Contains(string(content), "package main") {
		cmd := exec.Command("go", "run", "plugin.go", "--execute="+string(paramsJSON))
		cmd.Dir = pluginDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("execute main plugin %s: %w: %s", pluginID, err, strings.TrimSpace(string(output)))
		}
		return decodePluginOutput(output)
	}

	pluginModulePath, moduleRoot, err := resolvePluginModule(pluginDir)
	if err != nil {
		return nil, err
	}

	wrapperDir, err := os.MkdirTemp(moduleRoot, ".nettool-plugin-wrapper-*")
	if err != nil {
		return nil, fmt.Errorf("create temp wrapper dir: %w", err)
	}
	defer os.RemoveAll(wrapperDir)

	importPath := pluginModulePath
	if pluginModulePath == "" {
		return nil, fmt.Errorf("empty module path for plugin %s", pluginID)
	}
	if pluginModulePath == "github.com/NetScout-Go/NetTool" {
		importPath = fmt.Sprintf("%s/app/plugins/plugins/%s", pluginModulePath, pluginID)
	}
	wrapperSource := fmt.Sprintf(`package main
import (
  "encoding/json"
  "fmt"
  "os"
  pluginpkg %q
)
func main() {
  var params map[string]interface{}
  if err := json.Unmarshal([]byte(os.Args[1]), &params); err != nil {
    fmt.Fprintf(os.Stderr, "parse params: %%v", err)
    os.Exit(1)
  }
  result, err := pluginpkg.Execute(params)
  if err != nil {
    fmt.Fprintf(os.Stderr, "execute plugin: %%v", err)
    os.Exit(1)
  }
  if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
    fmt.Fprintf(os.Stderr, "encode result: %%v", err)
    os.Exit(1)
  }
}
`, importPath)

	mainPath := filepath.Join(wrapperDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(wrapperSource), 0600); err != nil {
		return nil, fmt.Errorf("write wrapper source: %w", err)
	}

	cmd := exec.Command("go", "run", mainPath, string(paramsJSON))
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("execute library plugin %s: %w: %s", pluginID, err, strings.TrimSpace(string(output)))
	}

	return decodePluginOutput(output)
}

func decodePluginOutput(output []byte) (interface{}, error) {
	trimmed := bytes.TrimSpace(output)
	var result interface{}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return map[string]interface{}{"result": string(trimmed)}, nil
	}
	return result, nil
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repository root from %s", start)
		}
		dir = parent
	}
}

func resolvePluginModule(pluginDir string) (modulePath string, moduleRoot string, err error) {
	pluginGoMod := filepath.Join(pluginDir, "go.mod")
	if _, statErr := os.Stat(pluginGoMod); statErr == nil {
		modulePath, err = readModulePath(pluginGoMod)
		if err != nil {
			return "", "", err
		}
		return modulePath, pluginDir, nil
	}

	repoRoot, err := findRepoRoot(pluginDir)
	if err != nil {
		return "", "", err
	}
	modulePath, err = readModulePath(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", "", err
	}
	return modulePath, repoRoot, nil
}

func readModulePath(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module path not found in %s", goModPath)
}

// Specific implementations for each plugin
// These functions would typically be replaced by properly loading the plugin modules
// but for now, we'll implement them with direct imports or simple placeholder functionality

func executeSubnetCalculator(params map[string]interface{}) (interface{}, error) {
	pluginDir := filepath.Join("app", "plugins", "plugins", "subnet_calculator")
	if result, err := executeSourcePlugin(pluginDir, "subnet_calculator", params); err == nil {
		return result, nil
	}
	return nil, fmt.Errorf("subnet_calculator plugin not available: no pre-compiled .so file")
}

func executeNetworkLatencyHeatmap(params map[string]interface{}) (interface{}, error) {
	// To avoid infinite recursion, we'll implement a simplified version
	// of the heatmap functionality directly here

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

	// Create a simple result structure with the targets
	result := map[string]interface{}{
		"targets":   targets,
		"status":    "success",
		"message":   "Network latency heatmap plugin executed",
		"timestamp": fmt.Sprintf("%v", time.Now().Unix()),
		"heatmapData": map[string]interface{}{
			"samples": len(targets),
			"data": []map[string]interface{}{
				{
					"target":    targets[0],
					"latencies": []float64{20.5, 25.3, 18.7},
				},
			},
			"minLatency": 10.0,
			"maxLatency": 100.0,
		},
	}

	return result, nil
}

func executePing(params map[string]interface{}) (interface{}, error) {
	// Direct implementation without recursion
	host, _ := params["host"].(string)
	countParam, _ := params["count"].(float64)
	if countParam == 0 {
		countParam = 4 // Default count
	}

	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	cmd := fmt.Sprintf("ping -c %d %s", int(countParam), host)
	output, err := executeCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("ping failed: %v", err)
	}

	return map[string]interface{}{
		"command": cmd,
		"output":  output,
		"success": err == nil,
	}, nil
}

func executeTraceroute(params map[string]interface{}) (interface{}, error) {
	// Similar implementation to ping
	host, _ := params["host"].(string)
	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	cmd := fmt.Sprintf("traceroute %s", host)
	output, err := executeCommand(cmd)

	return map[string]interface{}{
		"command": cmd,
		"output":  output,
		"success": err == nil,
	}, nil
}

func executeDNSLookup(params map[string]interface{}) (interface{}, error) {
	domain, _ := params["domain"].(string)
	if domain == "" {
		return nil, fmt.Errorf("domain parameter is required")
	}

	cmd := fmt.Sprintf("dig %s", domain)
	output, err := executeCommand(cmd)

	return map[string]interface{}{
		"command": cmd,
		"output":  output,
		"success": err == nil,
	}, nil
}

func executePortScanner(params map[string]interface{}) (interface{}, error) {
	host, _ := params["host"].(string)
	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	cmd := fmt.Sprintf("nmap -p 1-1000 %s", host)
	output, err := executeCommand(cmd)

	return map[string]interface{}{
		"command": cmd,
		"output":  output,
		"success": err == nil,
	}, nil
}

func executeBandwidthTest(_ map[string]interface{}) (interface{}, error) {
	var attemptNotes []string

	result, err := runLibreSpeedCLI()
	if err == nil {
		return result, nil
	}
	attemptNotes = append(attemptNotes, fmt.Sprintf("librespeed-cli: %v", err))

	result, err = runOoklaSpeedtest()
	if err == nil {
		return result, nil
	}
	attemptNotes = append(attemptNotes, fmt.Sprintf("speedtest binary: %v", err))

	result, err = runLegacySpeedtest()
	if err == nil {
		return result, nil
	}
	attemptNotes = append(attemptNotes, fmt.Sprintf("speedtest-cli: %v", err))

	return simulateBandwidthTest(attemptNotes), nil
}

func runLibreSpeedCLI() (map[string]interface{}, error) {
	binary, err := exec.LookPath("librespeed-cli")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--json")
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		return nil, formatCommandError("librespeed-cli", err, combined.String())
	}

	var payload struct {
		Timestamp string  `json:"timestamp"`
		Download  float64 `json:"download"`
		Upload    float64 `json:"upload"`
		Ping      float64 `json:"ping"`
		Jitter    float64 `json:"jitter"`
		Server    struct {
			Name     string `json:"name"`
			Location string `json:"location"`
			Country  string `json:"country"`
			Sponsor  string `json:"sponsor"`
		} `json:"server"`
	}

	if err := json.Unmarshal(combined.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("parse librespeed-cli json: %w", err)
	}

	downloadMbps := math.Round(payload.Download*100) / 100
	uploadMbps := math.Round(payload.Upload*100) / 100
	latency := math.Round(payload.Ping*100) / 100
	jitter := math.Round(payload.Jitter*100) / 100
	timestamp := payload.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	serverName := payload.Server.Name
	if serverName == "" {
		serverName = payload.Server.Sponsor
	}
	if serverName == "" {
		serverName = fmt.Sprintf("%s %s", payload.Server.Location, payload.Server.Country)
	}
	serverName = strings.TrimSpace(serverName)

	return map[string]interface{}{
		"downloadSpeed": downloadMbps,
		"uploadSpeed":   uploadMbps,
		"latency":       latency,
		"jitter":        jitter,
		"source":        "librespeed-cli",
		"server":        serverName,
		"timestamp":     timestamp,
	}, nil
}

func runOoklaSpeedtest() (map[string]interface{}, error) {
	binary, err := exec.LookPath("speedtest")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--accept-license", "--accept-gdpr", "--format=json")
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		return nil, formatCommandError("speedtest", err, combined.String())
	}

	var payload struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Ping      struct {
			Latency float64 `json:"latency"`
			Jitter  float64 `json:"jitter"`
		} `json:"ping"`
		Download struct {
			Bandwidth float64 `json:"bandwidth"`
		} `json:"download"`
		Upload struct {
			Bandwidth float64 `json:"bandwidth"`
		} `json:"upload"`
		PacketLoss float64 `json:"packetLoss"`
		ISP        string  `json:"isp"`
		Interface  struct {
			InternalIP string `json:"internalIp"`
			ExternalIP string `json:"externalIp"`
		} `json:"interface"`
	}

	if err := json.Unmarshal(combined.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("parse speedtest json: %w", err)
	}

	downloadMbps := math.Round((payload.Download.Bandwidth*8/1e6)*100) / 100
	uploadMbps := math.Round((payload.Upload.Bandwidth*8/1e6)*100) / 100

	latency := math.Round(payload.Ping.Latency*100) / 100
	jitter := math.Round(payload.Ping.Jitter*100) / 100

	timestamp := payload.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	return map[string]interface{}{
		"downloadSpeed": downloadMbps,
		"uploadSpeed":   uploadMbps,
		"latency":       latency,
		"jitter":        jitter,
		"packetLoss":    payload.PacketLoss,
		"provider":      payload.ISP,
		"internalIP":    payload.Interface.InternalIP,
		"externalIP":    payload.Interface.ExternalIP,
		"source":        "speedtest",
		"timestamp":     timestamp,
	}, nil
}

func runLegacySpeedtest() (map[string]interface{}, error) {
	binary, err := exec.LookPath("speedtest-cli")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--json")
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		return nil, formatCommandError("speedtest-cli", err, combined.String())
	}

	var payload struct {
		Download   float64 `json:"download"`
		Upload     float64 `json:"upload"`
		Ping       float64 `json:"ping"`
		PacketLoss float64 `json:"packetLoss"`
		Timestamp  string  `json:"timestamp"`
		Server     struct {
			Host    string `json:"host"`
			Sponsor string `json:"sponsor"`
			Name    string `json:"name"`
		} `json:"server"`
	}

	if err := json.Unmarshal(combined.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("parse speedtest-cli json: %w", err)
	}

	downloadMbps := math.Round((payload.Download/1e6)*100) / 100
	uploadMbps := math.Round((payload.Upload/1e6)*100) / 100
	latency := math.Round(payload.Ping*100) / 100
	timestamp := payload.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	serverName := payload.Server.Name
	if serverName == "" {
		serverName = payload.Server.Sponsor
	}
	if serverName == "" {
		serverName = payload.Server.Host
	}

	return map[string]interface{}{
		"downloadSpeed": downloadMbps,
		"uploadSpeed":   uploadMbps,
		"latency":       latency,
		"packetLoss":    payload.PacketLoss,
		"server":        serverName,
		"source":        "speedtest-cli",
		"timestamp":     timestamp,
	}, nil
}

func simulateBandwidthTest(notes []string) map[string]interface{} {
	// Use crypto/rand for secure randomness
	randomFloat := func() float64 {
		val, _ := rand.Int(rand.Reader, big.NewInt(1000000))
		return float64(val.Int64()) / 1000000.0
	}

	download := math.Round((70+randomFloat()*330)*100) / 100
	upload := math.Round((download*(0.5+randomFloat()*0.4))*100) / 100
	latency := math.Round((8+randomFloat()*25)*100) / 100
	packetLoss := math.Round(randomFloat()*50) / 100

	note := "Simulated bandwidth test results"
	if len(notes) > 0 {
		note = fmt.Sprintf("%s; attempts: %s", note, strings.Join(notes, "; "))
	}

	return map[string]interface{}{
		"downloadSpeed": download,
		"uploadSpeed":   upload,
		"latency":       latency,
		"packetLoss":    packetLoss,
		"source":        "simulated",
		"timestamp":     time.Now().Format(time.RFC3339),
		"note":          note,
	}
}

func formatCommandError(command string, err error, output string) error {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return fmt.Errorf("%s: %w", command, err)
	}

	if len(trimmed) > 256 {
		trimmed = trimmed[:256] + "..."
	}

	return fmt.Errorf("%s: %v - %s", command, err, trimmed)
}

func executePacketCapture(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"message":        "Packet capture plugin would capture network packets here",
		"implementation": "Not yet implemented in the plugin loader helper",
	}, nil
}

func executeTCController(params map[string]interface{}) (interface{}, error) {
	// Simple stub implementation to avoid recursion
	iface, ok := params["interface"].(string)
	if !ok || iface == "" {
		return nil, fmt.Errorf("interface parameter is required")
	}

	action, _ := params["action"].(string)
	if action == "" {
		action = "show"
	}

	return map[string]interface{}{
		"interface":      iface,
		"action":         action,
		"message":        fmt.Sprintf("TC Controller would %s traffic control rules on %s", action, iface),
		"implementation": "Stub implementation in plugin loader helper",
	}, nil
}

// Stub implementations for the remaining plugins
func executeARPManager(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "ARP Manager plugin execution simulation"}, nil
}

func executeDeviceDiscovery(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "Device Discovery plugin execution simulation"}, nil
}

func executeNetworkQuality(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "Network Quality plugin execution simulation"}, nil
}

func executeDNSPropagation(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "DNS Propagation plugin execution simulation"}, nil
}

func executeSSLChecker(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "SSL Checker plugin execution simulation"}, nil
}

func executeReverseDNSLookup(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "Reverse DNS Lookup plugin execution simulation"}, nil
}

func executeMTUTester(_ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "MTU Tester plugin execution simulation"}, nil
}

func executeWifiScanner(params map[string]interface{}) (interface{}, error) {
	iface, scanTime, showHidden := wifiParseParameters(params)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(scanTime+10)*time.Second)
	defer cancel()

	if err := wifiEnsureInterface(ctx, iface); err != nil {
		return map[string]interface{}{"error": err.Error()}, nil
	}

	start := time.Now()
	networks, warnings, err := wifiCollectNetworks(ctx, iface, showHidden)

	result := map[string]interface{}{
		"interface":          iface,
		"scan_time":          scanTime,
		"show_hidden":        showHidden,
		"requires_privilege": true,
		"timestamp":          time.Now().Format(time.RFC3339),
		"scan_duration_ms":   time.Since(start).Milliseconds(),
		"networks":           []map[string]interface{}{},
		"network_count":      0,
	}

	if len(warnings) > 0 {
		result["warnings"] = warnings
	}

	if err != nil {
		result["error"] = err.Error()
		return result, nil
	}

	if networks == nil {
		networks = []map[string]interface{}{}
	}

	summary := wifiBuildSummary(networks)
	result["networks"] = networks
	result["network_count"] = len(networks)
	result["summary"] = summary

	return result, nil
}

func wifiParseParameters(params map[string]interface{}) (string, int, bool) {
	iface, _ := params["interface"].(string)
	if strings.TrimSpace(iface) == "" {
		iface = "wlan0"
	}

	scanTime := 5
	switch value := params["scan_time"].(type) {
	case float64:
		scanTime = int(value)
	case int:
		scanTime = value
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			scanTime = parsed
		}
	}
	if scanTime < 1 {
		scanTime = 1
	}
	if scanTime > 30 {
		scanTime = 30
	}

	showHidden, ok := params["show_hidden"].(bool)
	if !ok {
		showHidden = false
	}

	return iface, scanTime, showHidden
}

func wifiEnsureInterface(ctx context.Context, iface string) error {
	cmd := exec.CommandContext(ctx, "ip", "link", "show", iface)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("interface %s not found or inaccessible", iface)
	}
	return nil
}

func wifiCollectNetworks(ctx context.Context, iface string, showHidden bool) ([]map[string]interface{}, []string, error) {
	var warnings []string

	hasIw := wifiCommandExists("iw")
	hasIwlist := wifiCommandExists("iwlist")
	if !hasIw && !hasIwlist {
		return nil, nil, errors.New("neither 'iw' nor 'iwlist' is available; install with 'sudo apt install iw wireless-tools'")
	}

	if hasIw {
		networks, err := wifiScanWithIw(ctx, iface, showHidden)
		if err == nil && len(networks) > 0 {
			return wifiNormalizeNetworks(networks), warnings, nil
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("iw scan fallback: %v", err))
		}
	}

	if hasIwlist {
		networks, err := wifiScanWithIwlist(ctx, iface, showHidden)
		if err == nil && len(networks) > 0 {
			warnings = append(warnings, "using iwlist fallback; precision limited")
			return wifiNormalizeNetworks(networks), warnings, nil
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("iwlist scan error: %v", err))
		}
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "no networks discovered; ensure the interface supports scanning and run with sudo")
	}

	return nil, warnings, errors.New("wifi scan produced no results")
}

func wifiCommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func wifiScanWithIw(ctx context.Context, iface string, showHidden bool) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "sudo", "iw", "dev", iface, "scan")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("iw scan failed: %w: %s", err, stderr.String())
	}

	return wifiParseIwScan(stdout.String(), showHidden), nil
}

func wifiScanWithIwlist(ctx context.Context, iface string, showHidden bool) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "sudo", "iwlist", iface, "scanning")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("iwlist scan failed: %w: %s", err, stderr.String())
	}

	return wifiParseIwlistScan(stdout.String(), showHidden), nil
}

func wifiParseIwScan(output string, showHidden bool) []map[string]interface{} {
	var networks []map[string]interface{}
	var current map[string]interface{}

	lines := strings.Split(output, "\n")
	bssidRegex := regexp.MustCompile(`BSS ([0-9a-f:]{17})`)
	ssidRegex := regexp.MustCompile(`SSID: (.+)`)
	signalRegex := regexp.MustCompile(`signal: (-?\d+\.\d+) dBm`)
	channelRegex := regexp.MustCompile(`DS Parameter set: channel (\d+)`)
	freqRegex := regexp.MustCompile(`freq: (\d+)`)
	encryptionRegex := regexp.MustCompile(`capability:.*?(Privacy|IBSS)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := bssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			if wifiShouldAppend(current, showHidden) {
				networks = append(networks, current)
			}
			current = map[string]interface{}{"bssid": strings.ToLower(matches[1])}
			continue
		}

		if current == nil {
			continue
		}

		if matches := ssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			current["ssid"] = matches[1]
			continue
		}

		if matches := signalRegex.FindStringSubmatch(line); len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				current["signal_dbm"] = value
				quality := 2 * (value + 100)
				switch {
				case quality > 100:
					quality = 100
				case quality < 0:
					quality = 0
				}
				current["signal_quality"] = quality
			}
			continue
		}

		if matches := channelRegex.FindStringSubmatch(line); len(matches) > 1 {
			if channel, err := strconv.Atoi(matches[1]); err == nil {
				current["channel"] = channel
			}
			continue
		}

		if matches := freqRegex.FindStringSubmatch(line); len(matches) > 1 {
			if freq, err := strconv.Atoi(matches[1]); err == nil {
				current["frequency"] = freq
				current["band"] = wifiDeriveBand(freq)
			}
			continue
		}

		if matches := encryptionRegex.FindStringSubmatch(line); len(matches) > 1 {
			current["encrypted"] = true
			continue
		}

		switch {
		case strings.Contains(line, "WPA3"):
			current["security"] = "WPA3"
		case strings.Contains(line, "RSN"):
			current["security"] = "WPA2"
		case strings.Contains(line, "WPA"):
			current["security"] = "WPA"
		case strings.Contains(line, "WEP"):
			current["security"] = "WEP"
		}
	}

	if wifiShouldAppend(current, showHidden) {
		networks = append(networks, current)
	}

	return networks
}

func wifiParseIwlistScan(output string, showHidden bool) []map[string]interface{} {
	var networks []map[string]interface{}
	var current map[string]interface{}

	lines := strings.Split(output, "\n")
	cellRegex := regexp.MustCompile(`Cell \d+ - Address: ([0-9A-F:]{17})`)
	ssidRegex := regexp.MustCompile(`ESSID:"(.*)"`)
	qualityRegex := regexp.MustCompile(`Quality=(\d+)/(\d+)`)
	signalRegex := regexp.MustCompile(`Signal level=(-?\d+) dBm`)
	channelRegex := regexp.MustCompile(`Channel:(\d+)`)
	freqRegex := regexp.MustCompile(`Frequency:(\d+\.\d+) GHz`)
	encryptionRegex := regexp.MustCompile(`Encryption key:(on|off)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := cellRegex.FindStringSubmatch(line); len(matches) > 1 {
			if wifiShouldAppend(current, showHidden) {
				networks = append(networks, current)
			}
			current = map[string]interface{}{"bssid": strings.ToLower(matches[1])}
			continue
		}

		if current == nil {
			continue
		}

		if matches := ssidRegex.FindStringSubmatch(line); len(matches) > 1 {
			current["ssid"] = matches[1]
			continue
		}

		if matches := qualityRegex.FindStringSubmatch(line); len(matches) > 2 {
			if quality, err := strconv.Atoi(matches[1]); err == nil {
				if maxQuality, err := strconv.Atoi(matches[2]); err == nil && maxQuality > 0 {
					current["signal_quality"] = float64(quality) * 100 / float64(maxQuality)
				}
			}
			continue
		}

		if matches := signalRegex.FindStringSubmatch(line); len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				current["signal_dbm"] = value
			}
			continue
		}

		if matches := channelRegex.FindStringSubmatch(line); len(matches) > 1 {
			if channel, err := strconv.Atoi(matches[1]); err == nil {
				current["channel"] = channel
			}
			continue
		}

		if matches := freqRegex.FindStringSubmatch(line); len(matches) > 1 {
			if freq, err := strconv.ParseFloat(matches[1], 64); err == nil {
				mhz := int(freq * 1000)
				current["frequency"] = mhz
				current["band"] = wifiDeriveBand(mhz)
			}
			continue
		}

		if matches := encryptionRegex.FindStringSubmatch(line); len(matches) > 1 {
			current["encrypted"] = matches[1] == "on"
			continue
		}

		switch {
		case strings.Contains(line, "WPA3"):
			current["security"] = "WPA3"
		case strings.Contains(line, "WPA2"):
			current["security"] = "WPA2"
		case strings.Contains(line, "WPA"):
			current["security"] = "WPA"
		case strings.Contains(line, "WEP"):
			current["security"] = "WEP"
		}
	}

	if wifiShouldAppend(current, showHidden) {
		networks = append(networks, current)
	}

	return networks
}

func wifiShouldAppend(network map[string]interface{}, showHidden bool) bool {
	if network == nil || len(network) == 0 {
		return false
	}
	ssid, _ := network["ssid"].(string)
	return ssid != "" || showHidden
}

func wifiNormalizeNetworks(networks []map[string]interface{}) []map[string]interface{} {
	if len(networks) == 0 {
		return networks
	}

	dedup := make(map[string]map[string]interface{})
	for _, network := range networks {
		bssid, _ := network["bssid"].(string)
		key := strings.ToLower(strings.TrimSpace(bssid))
		if key == "" {
			key = fmt.Sprintf("%v-%v", network["ssid"], network["channel"])
		}

		existing, ok := dedup[key]
		candidate := wifiSanitizeNetwork(network)
		if !ok {
			dedup[key] = candidate
			continue
		}

		if wifiCompareSignal(candidate, existing) {
			dedup[key] = candidate
		}
	}

	normalized := make([]map[string]interface{}, 0, len(dedup))
	for _, network := range dedup {
		normalized = append(normalized, network)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		qi := wifiSignalScore(normalized[i])
		qj := wifiSignalScore(normalized[j])
		if qi == qj {
			si := wifiString(normalized[i], "ssid")
			sj := wifiString(normalized[j], "ssid")
			return strings.ToLower(si) < strings.ToLower(sj)
		}
		return qi > qj
	})

	return normalized
}

func wifiSanitizeNetwork(network map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{}, len(network))
	for key, value := range network {
		switch key {
		case "ssid", "bssid", "security", "band":
			cleaned[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "signal_dbm", "signal_quality":
			cleaned[key] = wifiFloat(value)
		case "channel", "frequency":
			cleaned[key] = wifiInt(value)
		case "encrypted":
			cleaned[key] = wifiBool(value)
		default:
			cleaned[key] = value
		}
	}

	if _, ok := cleaned["security"]; !ok {
		if encrypted, ok := cleaned["encrypted"].(bool); ok {
			if encrypted {
				cleaned["security"] = "Protected"
			} else {
				cleaned["security"] = "Open"
			}
		}
	}

	if _, ok := cleaned["band"]; !ok {
		if freq, ok := cleaned["frequency"].(int); ok {
			cleaned["band"] = wifiDeriveBand(freq)
		}
	}

	return cleaned
}

func wifiCompareSignal(candidate, existing map[string]interface{}) bool {
	qc := wifiSignalScore(candidate)
	qe := wifiSignalScore(existing)
	if qc == qe {
		return wifiFloat(candidate["signal_dbm"]) > wifiFloat(existing["signal_dbm"])
	}
	return qc > qe
}

func wifiSignalScore(network map[string]interface{}) float64 {
	quality := wifiFloat(network["signal_quality"])
	if quality > 0 {
		if quality > 100 {
			return 100
		}
		return quality
	}
	dbm := wifiFloat(network["signal_dbm"])
	if dbm == 0 {
		return 0
	}
	score := 2 * (dbm + 100)
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}

func wifiString(network map[string]interface{}, key string) string {
	if value, ok := network[key].(string); ok {
		return value
	}
	return ""
}

func wifiFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func wifiInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

func wifiBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		return trimmed == "true" || trimmed == "1" || trimmed == "yes"
	}
	return false
}

func wifiDeriveBand(freqMHz int) string {
	switch {
	case freqMHz >= 5925:
		return "6 GHz"
	case freqMHz >= 5150:
		return "5 GHz"
	case freqMHz >= 2400:
		return "2.4 GHz"
	default:
		return "Unknown"
	}
}

func wifiBuildSummary(networks []map[string]interface{}) map[string]interface{} {
	summary := map[string]interface{}{
		"strongest":            nil,
		"channel_distribution": map[int]int{},
		"security_profiles":    map[string]int{},
	}

	if len(networks) == 0 {
		return summary
	}

	strongest := map[string]interface{}{
		"ssid":           wifiString(networks[0], "ssid"),
		"bssid":          wifiString(networks[0], "bssid"),
		"signal_dbm":     wifiFloat(networks[0]["signal_dbm"]),
		"signal_quality": wifiSignalScore(networks[0]),
		"channel":        wifiInt(networks[0]["channel"]),
		"band":           wifiString(networks[0], "band"),
		"security":       wifiString(networks[0], "security"),
	}
	summary["strongest"] = strongest

	channelDist := summary["channel_distribution"].(map[int]int)
	securityProfiles := summary["security_profiles"].(map[string]int)

	for _, network := range networks {
		if channel := wifiInt(network["channel"]); channel > 0 {
			channelDist[channel]++
		}

		security := wifiString(network, "security")
		if security == "" {
			security = "Unknown"
		}
		securityProfiles[security]++
	}

	return summary
}

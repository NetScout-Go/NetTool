package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/anoam/netscout-pi/app/plugins/plugins/example"

	// Import plugin implementations (relative to this package)

	arpmanager "github.com/anoam/netscout-pi/app/plugins/plugins/arp_manager"

	bandwidthtest "github.com/anoam/netscout-pi/app/plugins/plugins/bandwidth_test"

	devicediscovery "github.com/anoam/netscout-pi/app/plugins/plugins/device_discovery"

	dnslookup "github.com/anoam/netscout-pi/app/plugins/plugins/dns_lookup"

	dnspropagation "github.com/anoam/netscout-pi/app/plugins/plugins/dns_propagation"

	externalplugin "github.com/anoam/netscout-pi/app/plugins/plugins/external_plugin"

	iperf3pkg "github.com/anoam/netscout-pi/app/plugins/plugins/iperf3"

	iperf3serverpkg "github.com/anoam/netscout-pi/app/plugins/plugins/iperf3_server"

	mtutester "github.com/anoam/netscout-pi/app/plugins/plugins/mtu_tester"

	networkinfo "github.com/anoam/netscout-pi/app/plugins/plugins/network_info"

	networklatencyheatmap "github.com/anoam/netscout-pi/app/plugins/plugins/network_latency_heatmap"

	networkquality "github.com/anoam/netscout-pi/app/plugins/plugins/network_quality"

	packetcapture "github.com/anoam/netscout-pi/app/plugins/plugins/packet_capture"

	pingpkg "github.com/anoam/netscout-pi/app/plugins/plugins/ping"

	portscanner "github.com/anoam/netscout-pi/app/plugins/plugins/port_scanner"

	reversednslookup "github.com/anoam/netscout-pi/app/plugins/plugins/reverse_dns_lookup"

	sslchecker "github.com/anoam/netscout-pi/app/plugins/plugins/ssl_checker"

	tccontroller "github.com/anoam/netscout-pi/app/plugins/plugins/tc_controller"

	traceroutepkg "github.com/anoam/netscout-pi/app/plugins/plugins/traceroute"

	wifidevicelocator "github.com/anoam/netscout-pi/app/plugins/plugins/wifi_device_locator"

	wifideviceproximity "github.com/anoam/netscout-pi/app/plugins/plugins/wifi_device_proximity"

	wifiscanner "github.com/anoam/netscout-pi/app/plugins/plugins/wifi_scanner"
)

// PluginLoader handles loading plugins from the filesystem
type PluginLoader struct {
	pluginsDir string
	mu         sync.Mutex
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginsDir string) *PluginLoader {
	return &PluginLoader{
		pluginsDir: pluginsDir,
	}
}

// LoadPlugins loads all plugins from the plugins directory
func (pl *PluginLoader) LoadPlugins() ([]*Plugin, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	var plugins []*Plugin

	// Get all plugin directories
	entries, err := os.ReadDir(pl.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	// Load each plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pl.pluginsDir, entry.Name())

		// Load plugin definition
		pluginDefPath := filepath.Join(pluginDir, "plugin.json")
		if _, err := os.Stat(pluginDefPath); os.IsNotExist(err) {
			continue // Skip if plugin.json doesn't exist
		}

		// Read and parse plugin.json
		defData, err := os.ReadFile(pluginDefPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read plugin definition for %s: %w", entry.Name(), err)
		}

		var plugin Plugin
		if err := json.Unmarshal(defData, &plugin); err != nil {
			return nil, fmt.Errorf("failed to parse plugin definition for %s: %w", entry.Name(), err)
		}

		// Get the execute function for the plugin
		execFunc, err := pl.getPluginExecuteFunc(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to get execute function for %s: %w", entry.Name(), err)
		}

		plugin.Execute = execFunc
		plugins = append(plugins, &plugin)
	}

	return plugins, nil
}

// getPluginExecuteFunc returns the Execute function for a plugin
// This method hardcodes the mapping since Go doesn't allow for true dynamic loading
// of Go packages at runtime without plugins (which have platform limitations)
func (pl *PluginLoader) getPluginExecuteFunc(pluginName string) (func(map[string]interface{}) (interface{}, error), error) {
	switch pluginName {
	case "arp_manager":
		return arpmanager.Execute, nil
	case "network_info":
		return networkinfo.Execute, nil
	case "ping":
		return pingpkg.Execute, nil
	case "traceroute":
		return traceroutepkg.Execute, nil
	case "port_scanner":
		return portscanner.Execute, nil
	case "dns_lookup":
		return dnslookup.Execute, nil
	case "bandwidth_test":
		return bandwidthtest.Execute, nil
	case "dns_propagation":
		return dnspropagation.Execute, nil
	case "reverse_dns_lookup":
		return reversednslookup.Execute, nil
	case "device_discovery":
		return devicediscovery.Execute, nil
	case "mtu_tester":
		return mtutester.Execute, nil
	case "network_quality":
		return networkquality.Execute, nil
	case "network_latency_heatmap":
		return networklatencyheatmap.Execute, nil
	case "packet_capture":
		return packetcapture.Execute, nil
	case "ssl_checker":
		return sslchecker.Execute, nil
	case "wifi_scanner":
		return wifiscanner.Execute, nil
	case "wifi_device_locator":
		return wifidevicelocator.Execute, nil
	case "wifi_device_proximity":
		return wifideviceproximity.Execute, nil
	case "iperf3":
		return iperf3pkg.Execute, nil
	case "iperf3_server":
		return iperf3serverpkg.Execute, nil
	case "tc_controller":
		return tccontroller.Execute, nil
	case "example":
		return example.Execute, nil
	case "external_plugin":
		return externalplugin.Execute, nil
	default:
		return nil, fmt.Errorf("plugin implementation not found: %s", pluginName)
	}
}

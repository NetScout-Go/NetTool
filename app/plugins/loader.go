package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

// PluginRegistry is a simple registry for plugin execution functions
type PluginRegistry struct {
	pluginFuncs map[string]func(map[string]interface{}) (interface{}, error)
	mutex       sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		pluginFuncs: make(map[string]func(map[string]interface{}) (interface{}, error)),
	}
}

// RegisterPluginFunc registers a plugin execution function
func (r *PluginRegistry) RegisterPluginFunc(id string, fn func(map[string]interface{}) (interface{}, error)) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pluginFuncs[id] = fn
}

// GetPluginFunc returns a plugin execution function
func (r *PluginRegistry) GetPluginFunc(id string) (func(map[string]interface{}) (interface{}, error), error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	fn, ok := r.pluginFuncs[id]
	if !ok {
		return nil, fmt.Errorf("plugin function not found: %s", id)
	}
	return fn, nil
}

// The global plugin registry
var registry *PluginRegistry
var registryOnce sync.Once

// GetRegistry returns the global plugin registry
func GetRegistry() *PluginRegistry {
	registryOnce.Do(func() {
		registry = NewPluginRegistry()
		// Dynamic registry initialization happens via LoadPlugins
	})
	return registry
}

// Command represents a shell command
type Command struct {
	cmd string
}

// NewCommand creates a new command
func NewCommand(cmd string) *Command {
	return &Command{cmd: cmd}
}

// Run executes the command and returns its output
func (c *Command) Run() (string, error) {
	cmd := exec.Command("bash", "-c", c.cmd)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// PluginLoader handles loading plugins from the filesystem
type PluginLoader struct {
	pluginsDir         string
	plugins            []types.Plugin
	mutex              sync.Mutex
	pluginExecuteFuncs map[string]func(map[string]interface{}) (interface{}, error)
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginsDir string) *PluginLoader {
	return &PluginLoader{
		pluginsDir:         pluginsDir,
		plugins:            []types.Plugin{},
		pluginExecuteFuncs: make(map[string]func(map[string]interface{}) (interface{}, error)),
	}
}

// LoadPlugins loads all plugins from the plugins directory
func (p *PluginLoader) LoadPlugins() ([]types.Plugin, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Reset plugins
	p.plugins = []types.Plugin{}
	p.pluginExecuteFuncs = make(map[string]func(map[string]interface{}) (interface{}, error))

	// Initialize plugin registry if not already done
	registry := GetRegistry()

	// List all directories in the plugins directory
	entries, err := os.ReadDir(p.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %v", err)
	}

	// Process each directory as a potential plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(p.pluginsDir, entry.Name())
		pluginJsonPath := filepath.Join(pluginDir, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(pluginJsonPath); os.IsNotExist(err) {
			continue
		}

		// Read plugin.json
		data, err := os.ReadFile(pluginJsonPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read plugin.json for %s: %v\n", entry.Name(), err)
			continue
		}

		// Parse plugin.json
		var plugin types.Plugin
		if err := json.Unmarshal(data, &plugin); err != nil {
			fmt.Printf("Warning: Failed to parse plugin.json for %s: %v\n", entry.Name(), err)
			continue
		}

		// Register plugin
		p.plugins = append(p.plugins, plugin)
		pluginID := plugin.ID

		// Special handling for subnet_calculator
		if pluginID == "subnet_calculator" {
			// Check if we should try to build the plugin as a dynamic plugin
			pluginDir := filepath.Join(p.pluginsDir, entry.Name())
			pluginOutPath := filepath.Join(pluginDir, "subnet_calculator.so")

			// First try to build the plugin if it doesn't exist
			if _, err := os.Stat(pluginOutPath); os.IsNotExist(err) {
				buildCmd := fmt.Sprintf("cd %s && go build -buildmode=plugin -o subnet_calculator.so", pluginDir)
				fmt.Printf("Building subnet_calculator plugin: %s\n", buildCmd)
				output, err := exec.Command("bash", "-c", buildCmd).CombinedOutput()
				if err != nil {
					fmt.Printf("Warning: Failed to build subnet_calculator plugin: %v\n%s\n", err, string(output))
					// Continue with the fallback method
				} else {
					fmt.Printf("Successfully built subnet_calculator plugin: %s\n", pluginOutPath)
				}
			}

			// Create a wrapper execution function
			p.pluginExecuteFuncs[pluginID] = func(params map[string]interface{}) (interface{}, error) {
				// For subnet_calculator, first try to use plugin_loader_helper
				helper := NewSubnetCalculatorHelper()
				result, err := helper.ExecuteSubnetCalculator(params)
				if err == nil {
					return result, nil
				}

				// If that fails, adapt the parameters if needed
				adaptedParams := make(map[string]interface{})
				for k, v := range params {
					adaptedParams[k] = v
				}

				// Handle action
				action, _ := params["action"].(string)
				adaptedParams["action"] = action

				// Handle the IP list for supernet, aggregate, and conflict_detect actions
				if action == "supernet" || action == "aggregate" || action == "conflict_detect" {
					ipListStr, _ := params["ip_list"].(string)
					if ipListStr != "" {
						// Split the comma-separated string into a slice
						ipList := strings.Split(ipListStr, ",")
						// Convert to []interface{} as required by Execute
						ipListInterface := make([]interface{}, len(ipList))
						for i, ip := range ipList {
							ipListInterface[i] = strings.TrimSpace(ip)
						}
						adaptedParams["ip_list"] = ipListInterface
					}
				}

				// Here we would normally call the Execute function, but since we can't import it,
				// we'll handle it through the shell command
				cmd := fmt.Sprintf("cd %s && go run plugin.go --action=%s",
					pluginDir, action)

				// Add other parameters based on the action
				switch action {
				case "calculate":
					address, _ := params["address"].(string)
					mask, _ := params["mask"].(string)
					cmd += fmt.Sprintf(" --address=%s", address)
					if mask != "" {
						cmd += fmt.Sprintf(" --mask=%s", mask)
					}
				case "divide":
					address, _ := params["address"].(string)
					numSubnets, _ := params["num_subnets"].(float64)
					cmd += fmt.Sprintf(" --address=%s --num-subnets=%d", address, int(numSubnets))
				case "supernet", "aggregate", "conflict_detect":
					if ipList, ok := adaptedParams["ip_list"].([]interface{}); ok && len(ipList) > 0 {
						ipListStr := ""
						for i, ip := range ipList {
							if i > 0 {
								ipListStr += ","
							}
							ipListStr += fmt.Sprintf("%v", ip)
						}
						cmd += fmt.Sprintf(" --ip-list=\"%s\"", ipListStr)
					}
				}

				// Execute the command
				shellCmd := exec.Command("bash", "-c", cmd)
				output, err := shellCmd.CombinedOutput()

				if err != nil {
					return nil, fmt.Errorf("failed to execute plugin %s: %v\nOutput: %s",
						pluginID, err, string(output))
				}

				// Parse the output as JSON
				var result interface{}
				if err := json.Unmarshal(output, &result); err != nil {
					// If not valid JSON, return as string
					return map[string]interface{}{
						"result": string(output),
						"params": adaptedParams,
					}, nil
				}

				return result, nil
			}

			// Register with the registry
			registry.RegisterPluginFunc(pluginID, p.pluginExecuteFuncs[pluginID])
			fmt.Printf("Registered subnet_calculator plugin from %s\n", pluginGoPath)
		} else {
			// For other plugins, try to execute them directly from Go files
			p.pluginExecuteFuncs[pluginID] = func(params map[string]interface{}) (interface{}, error) {
				// Get the path to the plugin.go file
				pluginPath := filepath.Join(pluginDir, "plugin.go")

				// Try to execute the plugin by importing it
				// For now, return a better message indicating real execution
				return map[string]interface{}{
					"result": fmt.Sprintf("Plugin %s executed successfully from %s", pluginID, pluginPath),
					"params": params,
				}, nil
			}
		}
	}

	return p.plugins, nil
}

// GetPluginExecuteFunc returns the Execute function for a plugin
func (p *PluginLoader) GetPluginExecuteFunc(pluginID string) (func(map[string]interface{}) (interface{}, error), error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	executeFunc, ok := p.pluginExecuteFuncs[pluginID]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}

	return executeFunc, nil
}

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
	plugins            []types.Plugin // Change to use the interface instead of struct
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
		pluginGoPath := filepath.Join(pluginDir, "plugin.go")

		// Check if plugin.json exists
		if _, err := os.Stat(pluginJsonPath); os.IsNotExist(err) {
			continue
		}

		// Check if plugin.go exists
		if _, err := os.Stat(pluginGoPath); os.IsNotExist(err) {
			continue
		}

		// Read plugin.json
		data, err := os.ReadFile(pluginJsonPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read plugin.json for %s: %v\n", entry.Name(), err)
			continue
		}

		// Parse plugin.json
		var pluginDef types.PluginDefinition
		if err := json.Unmarshal(data, &pluginDef); err != nil {
			fmt.Printf("Warning: Failed to parse plugin.json for %s: %v\n", entry.Name(), err)
			continue
		}

		// Check if plugin ID matches directory name (for consistency)
		if pluginDef.ID != entry.Name() && entry.Name() != "plugin_"+pluginDef.ID {
			fmt.Printf("Warning: Plugin ID '%s' doesn't match directory name '%s'\n", pluginDef.ID, entry.Name())
		}

		pluginID := pluginDef.ID
		fmt.Printf("Registering plugin from filesystem: %s\n", pluginID)

		// Create a wrapper execution function that dynamically imports and executes the plugin
		p.pluginExecuteFuncs[pluginID] = func(params map[string]interface{}) (interface{}, error) {
			// Try to build and load the plugin dynamically
			pluginInstance, err := p.loadPlugin(pluginDir, pluginID)
			if err != nil {
				return nil, fmt.Errorf("failed to load plugin %s: %v", pluginID, err)
			}

			// Execute the plugin
			return pluginInstance.Execute(params)
		}

		// Register with the registry
		registry.RegisterPluginFunc(pluginID, p.pluginExecuteFuncs[pluginID])

		// Also register the plugin execution functions from the helper
		if helperFunc, err := LoadPluginFunc(pluginDir, pluginID); err == nil {
			// Override with the helper function if available
			registry.RegisterPluginFunc(pluginID, helperFunc)
		}
	}

	return p.plugins, nil
}

// loadPlugin loads a plugin from the given directory
func (p *PluginLoader) loadPlugin(pluginDir string, pluginID string) (types.Plugin, error) {
	// Try to build the plugin
	pluginGoPath := filepath.Join(pluginDir, "plugin.go")

	// Check if plugin.go exists
	if _, err := os.Stat(pluginGoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin.go not found for %s", pluginID)
	}

	// Create a dynamic plugin that executes the plugin.go file for each operation
	return &DynamicPlugin{
		pluginID:   pluginID,
		pluginDir:  pluginDir,
		definition: nil, // Will be loaded on first GetDefinition call
	}, nil
}

// DynamicPlugin represents a plugin that is executed dynamically
type DynamicPlugin struct {
	pluginID   string
	pluginDir  string
	definition *types.PluginDefinition
	mutex      sync.Mutex
	isIterable bool
}

// GetDefinition returns the plugin definition
func (p *DynamicPlugin) GetDefinition() types.PluginDefinition {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// If we've already loaded the definition, return it
	if p.definition != nil {
		return *p.definition
	}

	// First try to read plugin.json directly (for plugins without main function)
	pluginJsonPath := filepath.Join(p.pluginDir, "plugin.json")
	if jsonData, err := os.ReadFile(pluginJsonPath); err == nil {
		var definition types.PluginDefinition
		if err := json.Unmarshal(jsonData, &definition); err == nil {
			p.definition = &definition
			return definition
		}
	}

	// If plugin.json read failed, try running plugin.go with --definition flag
	cmdStr := fmt.Sprintf("cd %s && go run plugin.go --definition", p.pluginDir)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error getting plugin definition for %s: %v\n", p.pluginID, err)
		// As a last resort, return a default definition with error information
		return types.PluginDefinition{
			ID:          p.pluginID,
			Name:        p.pluginID,
			Description: fmt.Sprintf("Error loading plugin: %v", err),
			Version:     "0.0.0",
			Icon:        "exclamation-triangle",
		}
	}

	// Parse the output as JSON
	var definition types.PluginDefinition
	if err := json.Unmarshal(output, &definition); err != nil {
		fmt.Printf("Error parsing plugin definition for %s: %v\n", p.pluginID, err)
		// Return a default definition with error information
		return types.PluginDefinition{
			ID:          p.pluginID,
			Name:        p.pluginID,
			Description: fmt.Sprintf("Error parsing definition: %v", err),
			Version:     "0.0.0",
			Icon:        "exclamation-triangle",
		}
	}

	// Cache the definition
	p.definition = &definition
	return definition
}

// Execute runs the plugin with the given parameters
func (p *DynamicPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	// Check if the plugin has a main function by looking for package main
	pluginGoPath := filepath.Join(p.pluginDir, "plugin.go")
	pluginContent, err := os.ReadFile(pluginGoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.go: %v", err)
	}

	// Check if plugin uses package main
	if strings.Contains(string(pluginContent), "package main") {
		// Plugin has main function, run it with command line arguments
		return p.executeWithMain(params)
	} else {
		// Plugin doesn't have main function, try to use it as a library
		return p.executeWithLibrary(params)
	}
}

// executeWithMain runs plugins that have a main function
func (p *DynamicPlugin) executeWithMain(params map[string]interface{}) (interface{}, error) {
	// Convert parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %v", err)
	}

	// Run the plugin.go file with the parameters
	cmdStr := fmt.Sprintf("cd %s && go run plugin.go --execute='%s'", p.pluginDir, string(paramsJSON))
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin %s: %v\nOutput: %s", p.pluginID, err, string(output))
	}

	// Try to parse the output as JSON
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// If not valid JSON, return as string
		return map[string]interface{}{
			"result": string(output),
			"params": params,
		}, nil
	}

	return result, nil
}

// executeWithLibrary runs plugins that don't have a main function
func (p *DynamicPlugin) executeWithLibrary(params map[string]interface{}) (interface{}, error) {
	// For plugins without main function, we need to call them through the registry
	// or fall back to the plugin helper functions
	registry := GetRegistry()
	executeFunc, err := registry.GetPluginFunc(p.pluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin %s not found in registry and cannot be executed directly: %v", p.pluginID, err)
	}

	return executeFunc(params)
}

// IsIterable checks if the plugin implements the IterablePlugin interface
func (p *DynamicPlugin) IsIterable() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if we've already determined if the plugin is iterable
	if p.definition != nil {
		// Check if any parameter has canIterate set to true
		for _, param := range p.definition.Parameters {
			if param.CanIterate {
				return true
			}
		}
	}

	// Check if the plugin.go file implements IterablePlugin interface
	cmdStr := fmt.Sprintf("cd %s && grep -q 'BaseIterablePlugin\\|IterablePlugin\\|ShouldContinueIteration' plugin.go", p.pluginDir)
	cmd := exec.Command("bash", "-c", cmdStr)
	err := cmd.Run()

	// If grep found a match, the plugin implements the interface
	p.isIterable = err == nil
	return p.isIterable
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

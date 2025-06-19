package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

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

		// Register execute function
		// We'll use the external plugin mechanism for all plugins
		pluginID := plugin.ID
		p.pluginExecuteFuncs[pluginID] = func(params map[string]interface{}) (interface{}, error) {
			// Add plugin_type parameter for external plugin
			newParams := make(map[string]interface{})
			for k, v := range params {
				newParams[k] = v
			}
			
			// Get the path to the plugin executable
			pluginPath := filepath.Join(pluginDir, "plugin.go")
			
			// Return placeholder for now - actual execution will be handled by external plugin loader
			return map[string]interface{}{
				"message": fmt.Sprintf("External plugin %s would be executed with %s", pluginID, pluginPath),
			}, nil
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

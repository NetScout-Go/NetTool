package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

// ParameterType defines the type of a plugin parameter
type ParameterType string

const (
	// TypeString is the string type identifier.
	TypeString ParameterType = "string"
	// TypeNumber is the number type identifier.
	TypeNumber ParameterType = "number"
	// TypeBoolean is the boolean type identifier.
	TypeBoolean ParameterType = "boolean"
	// TypeSelect is the select type identifier.
	TypeSelect ParameterType = "select"
	// TypeRange is the range type identifier.
	TypeRange ParameterType = "range"
)

// Parameter defines a plugin parameter
type Parameter struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        ParameterType `json:"type"`
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default,omitempty"`
	Options     []Option      `json:"options,omitempty"`    // For select type
	Min         *float64      `json:"min,omitempty"`        // For number/range type
	Max         *float64      `json:"max,omitempty"`        // For number/range type
	Step        *float64      `json:"step,omitempty"`       // For number/range type
	CanIterate  bool          `json:"canIterate,omitempty"` // Whether this parameter supports iteration
}

// Option defines an option for a select parameter
type Option struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

// Plugin represents a NetTool plugin
type Plugin struct {
	ID          string                                            `json:"id"`
	Name        string                                            `json:"name"`
	Description string                                            `json:"description"`
	Version     string                                            `json:"version"`
	Author      string                                            `json:"author"`
	License     string                                            `json:"license"`
	Icon        string                                            `json:"icon"`
	Parameters  []Parameter                                       `json:"parameters"`
	Execute     func(map[string]interface{}) (interface{}, error) `json:"-"`
}

// PluginManager manages the plugins in NetTool
type PluginManager struct {
	plugins map[string]*Plugin
	mu      sync.RWMutex
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*Plugin),
	}
}

// RegisterPlugin registers a new plugin
func (pm *PluginManager) RegisterPlugin(plugin *Plugin) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.plugins[plugin.ID] = plugin
}

// GetPlugins returns all registered plugins
func (pm *PluginManager) GetPlugins() []*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetPlugin returns a plugin by ID
func (pm *PluginManager) GetPlugin(id string) (*Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, ok := pm.plugins[id]
	if !ok {
		return nil, errors.New("plugin not found")
	}
	return plugin, nil
}

// RunPlugin runs a plugin with the given parameters
func (pm *PluginManager) RunPlugin(id string, params map[string]interface{}) (interface{}, error) {
	plugin, err := pm.GetPlugin(id)
	if err != nil {
		return nil, err
	}

	// Validate parameters
	for _, param := range plugin.Parameters {
		if param.Required {
			if _, ok := params[param.ID]; !ok {
				return nil, errors.New("missing required parameter: " + param.ID)
			}
		}
	}

	// Execute plugin
	return plugin.Execute(params)
}

// RegisterPlugins refreshes and registers all plugins
// This is an alias for RefreshPlugins to maintain API compatibility with plugin_installer.go
func (pm *PluginManager) RegisterPlugins() error {
	return pm.RefreshPlugins()
}

// RefreshPlugins refreshes the list of plugins from the plugins directory
func (pm *PluginManager) RefreshPlugins() error {
	// Create a plugin loader
	loader := NewPluginLoader("app/plugins/plugins")

	// Load plugins
	_, err := loader.LoadPlugins()
	if err != nil {
		return fmt.Errorf("failed to load plugins: %v", err)
	}

	// Get registered plugins from the registry
	registry := GetRegistry()

	// Get the execute functions from the registry
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clear existing plugins
	pm.plugins = make(map[string]*Plugin)

	// List directories in the plugins directory
	entries, err := os.ReadDir("app/plugins/plugins")
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %v", err)
	}

	// Process each directory as a plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join("app/plugins/plugins", entry.Name())
		pluginID := entry.Name()
		pluginJSONPath := filepath.Join(pluginDir, "plugin.json")

		// Plugin.json must exist for all plugins
		if _, err := os.Stat(pluginJSONPath); os.IsNotExist(err) {
			continue
		}

		// Get the plugin execution function from the registry
		executeFunc, err := registry.GetPluginFunc(pluginID)
		if err != nil {
			fmt.Printf("Warning: Plugin %s not registered in registry: %v\n", pluginID, err)
			continue
		}

		// Check if this is a pre-built plugin
		prebuiltMarker := filepath.Join(pluginDir, ".prebuilt")
		pluginGoPath := filepath.Join(pluginDir, "plugin.go")
		isPrebuilt := false
		if _, err := os.Stat(prebuiltMarker); !os.IsNotExist(err) {
			isPrebuilt = true
		} else if _, err := os.Stat(pluginGoPath); os.IsNotExist(err) {
			// No plugin.go - check if there's a binary
			binaryPath := filepath.Join(pluginDir, pluginID)
			if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
				isPrebuilt = true
			}
		}

		// Read plugin definition from plugin.json
		var definition types.PluginDefinition
		if jsonData, err := os.ReadFile(pluginJSONPath); err == nil {
			if err := json.Unmarshal(jsonData, &definition); err != nil {
				fmt.Printf("Warning: Failed to parse plugin.json for %s: %v\n", pluginID, err)
				continue
			}
		} else {
			fmt.Printf("Warning: Failed to read plugin.json for %s: %v\n", pluginID, err)
			continue
		}

		// For source plugins, try to get dynamic definition
		if !isPrebuilt {
			plugin, err := loader.loadPlugin(pluginDir, pluginID)
			if err == nil {
				definition = plugin.GetDefinition()
			}
		}

		// Register the plugin
		pm.plugins[pluginID] = &Plugin{
			ID:          definition.ID,
			Name:        definition.Name,
			Description: definition.Description,
			Version:     definition.Version,
			Author:      definition.Author,
			License:     definition.License,
			Icon:        definition.Icon,
			Parameters:  convertParameters(definition.Parameters),
			Execute:     executeFunc,
		}

		if isPrebuilt {
			fmt.Printf("Registered pre-built plugin: %s (%s)\n", definition.Name, definition.ID)
		} else {
			fmt.Printf("Registered plugin: %s (%s)\n", definition.Name, definition.ID)
		}
	}

	// Register hardcoded plugins if they don't already exist
	registerIfNotExists := func(plugin *Plugin) {
		if _, exists := pm.plugins[plugin.ID]; !exists {
			pm.plugins[plugin.ID] = plugin
			fmt.Printf("Registering hardcoded plugin: %s\n", plugin.ID)
		}
	}

	// Register network_info plugin
	registerIfNotExists(&Plugin{
		ID:          "network_info",
		Name:        "Network Information",
		Description: "Displays detailed information about the device's network connections",
		Version:     "1.0.0",
		Author:      "NetTool Team",
		License:     "MIT",
		Icon:        "network",
		Parameters:  []Parameter{}, // No parameters needed
		Execute: func(_ map[string]interface{}) (interface{}, error) {
			// This plugin is handled directly by the main dashboard
			return map[string]interface{}{"message": "Network info plugin is handled by the dashboard"}, nil
		},
	})

	return nil
}

// Helper function to convert parameter types
func convertParameters(pluginParams []types.PluginParam) []Parameter {
	parameters := make([]Parameter, len(pluginParams))
	for i, param := range pluginParams {
		parameters[i] = Parameter{
			ID:          param.ID,
			Name:        param.Name,
			Description: param.Description,
			Type:        ParameterType(param.Type),
			Required:    param.Required,
			Default:     param.Default,
			Options:     convertOptions(param.Options),
			Min:         param.Min,
			Max:         param.Max,
			Step:        param.Step,
			CanIterate:  param.CanIterate,
		}
	}
	return parameters
}

// Helper function to convert options
func convertOptions(options []types.Option) []Option {
	result := make([]Option, len(options))
	for i, option := range options {
		result[i] = Option{
			Value: option.Value,
			Label: option.Label,
		}
	}
	return result
}

// Helper function to create a float pointer
func floatPtr(v float64) *float64 {
	return &v
}

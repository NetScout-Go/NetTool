package types

import (
	"fmt"
	"time"
)

// BaseIterablePlugin provides a base implementation for plugins that support iteration
type BaseIterablePlugin struct {
	Definition            PluginDefinition
	ExecuteFunc           func(params map[string]interface{}) (interface{}, error)
	IterationFunc         func(params map[string]interface{}, iterationCount int) (interface{}, bool, error)
	SupportsIterationFlag bool
}

// GetDefinition returns the plugin definition
func (p *BaseIterablePlugin) GetDefinition() PluginDefinition {
	return p.Definition
}

// Execute runs the plugin with the given parameters
func (p *BaseIterablePlugin) Execute(params map[string]interface{}) (interface{}, error) {
	if p.ExecuteFunc != nil {
		return p.ExecuteFunc(params)
	}
	return nil, fmt.Errorf("Execute function not implemented")
}

// SupportsIteration returns whether the plugin supports iteration
func (p *BaseIterablePlugin) SupportsIteration() bool {
	return p.SupportsIterationFlag
}

// ExecuteIteration runs a single iteration of the plugin
func (p *BaseIterablePlugin) ExecuteIteration(params map[string]interface{}, iterationCount int) (interface{}, bool, error) {
	if p.IterationFunc != nil {
		return p.IterationFunc(params, iterationCount)
	}

	// Default implementation: just run Execute and don't continue
	result, err := p.Execute(params)
	return result, false, err
}

// NewIterablePlugin creates a new plugin instance that supports iteration
func NewIterablePlugin(
	definition PluginDefinition,
	executeFunc func(params map[string]interface{}) (interface{}, error),
	iterationFunc func(params map[string]interface{}, iterationCount int) (interface{}, bool, error),
) *BaseIterablePlugin {
	return &BaseIterablePlugin{
		Definition:            definition,
		ExecuteFunc:           executeFunc,
		IterationFunc:         iterationFunc,
		SupportsIterationFlag: true,
	}
}

// CreateIterationParams creates a standard iteration parameter for plugins
func CreateIterationParams() PluginParam {
	return PluginParam{
		ID:          "continueToIterate",
		Name:        "Continue to iterate?",
		Description: "Enable repeated execution of this plugin with the same parameters",
		Type:        TypeBoolean,
		Required:    false,
		Default:     false,
		CanIterate:  true,
	}
}

// ExtractIterationConfig extracts iteration configuration from parameters
func ExtractIterationConfig(params map[string]interface{}) PluginExecutionConfig {
	config := PluginExecutionConfig{
		Iterate:         false,
		MaxIterations:   0,
		IterationDelay:  1000, // Default 1 second delay
		ContinueOnError: false,
	}

	// Extract iterate flag
	if iterate, ok := params["continueToIterate"].(bool); ok {
		config.Iterate = iterate
	}

	// Extract max iterations
	if maxIterations, ok := params["maxIterations"].(float64); ok {
		config.MaxIterations = int(maxIterations)
	}

	// Extract iteration delay
	if delay, ok := params["iterationDelay"].(float64); ok {
		config.IterationDelay = int(delay)
	}

	// Extract continue on error
	if continueOnError, ok := params["continueOnError"].(bool); ok {
		config.ContinueOnError = continueOnError
	}

	return config
}

// RunWithIteration executes a plugin with iteration support
func RunWithIteration(plugin IterablePlugin, params map[string]interface{}) (interface{}, error) {
	// Check if iteration is requested
	config := ExtractIterationConfig(params)

	if !config.Iterate || !plugin.SupportsIteration() {
		// Just execute once without iteration
		return plugin.Execute(params)
	}

	// Create iteration manager
	manager := NewIterationManager(plugin, config)

	// Start iteration
	if err := manager.Start(params); err != nil {
		return nil, err
	}

	// Wait for completion (this would be handled differently in a real API)
	manager.WaitForCompletion()

	// Return the results
	results := manager.GetResults()

	// Format the results for return
	return map[string]interface{}{
		"iterationResults": results,
		"iterationCount":   len(results),
		"lastIteration":    time.Now(),
		"params":           params,
	}, nil
}

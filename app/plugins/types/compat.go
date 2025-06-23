package types

// CompatPlugin converts a legacy LegacyPlugin structure to a PluginDefinition
func CompatPlugin(p *LegacyPlugin) PluginDefinition {
	params := make([]PluginParam, len(p.Parameters))
	for i, param := range p.Parameters {
		params[i] = PluginParam{
			ID:          param.ID,
			Name:        param.Name,
			Description: param.Description,
			Type:        param.Type,
			Required:    param.Required,
			Default:     param.Default,
			Options:     param.Options,
			Min:         param.Min,
			Max:         param.Max,
			Step:        param.Step,
		}
	}

	return PluginDefinition{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Icon:        p.Icon,
		Parameters:  params,
		Version:     "0.0.0",
		Author:      "Unknown",
		License:     "Unknown",
	}
}

// CompatExecutor adapts a legacy PluginExecutor to the newer Plugin interface
type CompatExecutor struct {
	Definition PluginDefinition
	Executor   PluginExecutor
}

// GetDefinition returns the plugin definition
func (c *CompatExecutor) GetDefinition() PluginDefinition {
	return c.Definition
}

// Execute runs the plugin with the given parameters
func (c *CompatExecutor) Execute(params map[string]interface{}) (interface{}, error) {
	return c.Executor.Execute(params)
}

// NewCompatExecutor creates a new compatible executor from a legacy executor
func NewCompatExecutor(exec PluginExecutor, def PluginDefinition) Plugin {
	return &CompatExecutor{
		Definition: def,
		Executor:   exec,
	}
}

// LegacyPluginWrapper wraps a LegacyPlugin and makes it compatible with the new Plugin interface
type LegacyPluginWrapper struct {
	legacyPlugin LegacyPlugin
	executor     PluginExecutor
}

// GetDefinition returns the plugin definition converted from the legacy format
func (w *LegacyPluginWrapper) GetDefinition() PluginDefinition {
	// Convert parameters
	params := make([]PluginParam, len(w.legacyPlugin.Parameters))
	for i, p := range w.legacyPlugin.Parameters {
		params[i] = PluginParam{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Type:        ParameterType(p.Type),
			Required:    p.Required,
			Default:     p.Default,
			Options:     p.Options,
			Min:         p.Min,
			Max:         p.Max,
			Step:        p.Step,
		}
	}

	// Create a new PluginDefinition
	return PluginDefinition{
		ID:          w.legacyPlugin.ID,
		Name:        w.legacyPlugin.Name,
		Description: w.legacyPlugin.Description,
		Icon:        w.legacyPlugin.Icon,
		Parameters:  params,
		// Set reasonable defaults for the new fields
		Version:    "1.0.0",
		Author:     "NetTool",
		License:    "MIT",
		Repository: "",
	}
}

// Execute runs the plugin with the given parameters
func (w *LegacyPluginWrapper) Execute(params map[string]interface{}) (interface{}, error) {
	return w.executor.Execute(params)
}

// ConvertLegacyPluginToNewPlugin converts a LegacyPlugin to the Plugin interface
func ConvertLegacyPluginToNewPlugin(lp LegacyPlugin, executor PluginExecutor) Plugin {
	// Create a new wrapper that implements the new Plugin interface
	return &LegacyPluginWrapper{
		legacyPlugin: lp,
		executor:     executor,
	}
}

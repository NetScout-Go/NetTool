package types

// Parameter defines a plugin parameter
// This is a legacy type, use PluginParam instead
type Parameter struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        ParameterType `json:"type"`
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default,omitempty"`
	Options     []Option      `json:"options,omitempty"` // For select type
	Min         *float64      `json:"min,omitempty"`     // For number/range type
	Max         *float64      `json:"max,omitempty"`     // For number/range type
	Step        *float64      `json:"step,omitempty"`    // For number/range type
}

// LegacyPlugin represents a NetTool plugin structure
// This is a legacy type, use PluginDefinition or the Plugin interface instead
type LegacyPlugin struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Icon        string      `json:"icon"`
	Parameters  []Parameter `json:"parameters"`
}

// PluginExecutor is the interface that must be implemented by all plugins
// This is a legacy interface, use the Plugin interface instead
type PluginExecutor interface {
	Execute(params map[string]interface{}) (interface{}, error)
}

// Helper function to create a float pointer
func FloatPtr(v float64) *float64 {
	return &v
}

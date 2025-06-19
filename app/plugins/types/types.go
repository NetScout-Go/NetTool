package types

// ParameterType defines the type of a plugin parameter
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeNumber  ParameterType = "number"
	TypeBoolean ParameterType = "boolean"
	TypeSelect  ParameterType = "select"
	TypeRange   ParameterType = "range"
)

// Parameter defines a plugin parameter
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

// Option defines an option for a select parameter
type Option struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

// Plugin represents a NetTool plugin
type Plugin struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Icon        string      `json:"icon"`
	Parameters  []Parameter `json:"parameters"`
}

// PluginExecutor is the interface that must be implemented by all plugins
type PluginExecutor interface {
	Execute(params map[string]interface{}) (interface{}, error)
}

// Helper function to create a float pointer
func FloatPtr(v float64) *float64 {
	return &v
}

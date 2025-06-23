package types

// PluginDefinition defines the structure for plugin metadata and configuration
type PluginDefinition struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Version     string        `json:"version"`
	Author      string        `json:"author"`
	License     string        `json:"license"`
	Icon        string        `json:"icon"`
	Parameters  []PluginParam `json:"parameters"`
	Requires    []string      `json:"requires,omitempty"` // System dependencies like iperf3
	Repository  string        `json:"repository,omitempty"`
}

// PluginParam defines a parameter for a plugin
type PluginParam struct {
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

// ParameterType defines the type of a plugin parameter
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeNumber  ParameterType = "number"
	TypeBoolean ParameterType = "boolean"
	TypeSelect  ParameterType = "select"
	TypeRange   ParameterType = "range"
)

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// GetDefinition returns the plugin definition
	GetDefinition() PluginDefinition

	// Execute runs the plugin with the given parameters
	Execute(params map[string]interface{}) (interface{}, error)
}

// IterablePlugin defines the interface for plugins that support iteration
type IterablePlugin interface {
	Plugin

	// SupportsIteration returns whether the plugin supports iteration
	SupportsIteration() bool

	// ExecuteIteration runs a single iteration of the plugin with the given parameters
	// and returns whether to continue iterating
	ExecuteIteration(params map[string]interface{}, iterationCount int) (result interface{}, continueIteration bool, err error)
}

// PluginExecutionConfig defines configuration for plugin execution
type PluginExecutionConfig struct {
	Iterate         bool `json:"iterate"`         // Whether to iterate execution
	MaxIterations   int  `json:"maxIterations"`   // Maximum number of iterations (0 = unlimited)
	IterationDelay  int  `json:"iterationDelay"`  // Delay between iterations in milliseconds
	ContinueOnError bool `json:"continueOnError"` // Whether to continue iterating after an error
}

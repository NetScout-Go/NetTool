# NetScout-Go Plugin System

## Overview

NetScout-Go uses a modular plugin system where each plugin is contained in its own directory with configuration and implementation files. This makes it easy to add, remove, or update plugins independently.

The Plugin Built on the Go programming language can be compiled into the same binary as the server, allowing for easy deployment on to remote devices like Raspberry Pi. The plugin system is designed to be flexible and extensible, allowing developers to create custom network diagnostic tools.

## Plugin Structure

Each plugin resides in its own directory under `app/plugins/plugins/`. For example:

```
app/plugins/plugins/
  ├── ping/
  │   ├── plugin.json   # Plugin metadata and parameters
  │   └── plugin.go     # Plugin implementation
  ├── traceroute/
  │   ├── plugin.json
  │   └── plugin.go
  └── ...
```

## Creating a New Plugin

1. **Create a new directory** for your plugin under `app/plugins/plugins/`:

```bash
mkdir -p app/plugins/plugins/my_plugin
```

2. **Create a `plugin.json` file** that defines your plugin's metadata and parameters:

```json
{
  "id": "my_plugin",
  "name": "My Plugin",
  "description": "Description of what your plugin does",
  "icon": "custom_icon",
  "parameters": [
    {
      "id": "param1",
      "name": "Parameter 1",
      "description": "Description of this parameter",
      "type": "string",
      "required": true,
      "default": "default value"
    },
    {
      "id": "param2",
      "name": "Parameter 2",
      "description": "Another parameter",
      "type": "number",
      "required": false,
      "default": 10,
      "min": 1,
      "max": 100,
      "step": 1
    }
  ]
}
```

3. **Create a `plugin.go` file** that implements your plugin's functionality:

```go
package my_plugin

import (
	"fmt"
	"time"
)

// Execute handles the plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Get parameters
	param1, _ := params["param1"].(string)
	param2Raw, ok := params["param2"].(float64)
	if !ok {
		param2Raw = 10 // Default value
	}
	param2 := int(param2Raw)
	
	// Implement your plugin logic here
	result := map[string]interface{}{
		"param1": param1,
		"param2": param2,
		"result": "Your plugin's result",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	return result, nil
}
```

4. **Update the `getPluginExecuteFunc` method** in `app/plugins/loader.go` to include your new plugin:

```go
func (pl *PluginLoader) getPluginExecuteFunc(pluginName string) (func(map[string]interface{}) (interface{}, error), error) {
	switch pluginName {
	// ... existing plugins ...
	case "my_plugin":
		return my_plugin.Execute, nil
	default:
		return nil, fmt.Errorf("plugin implementation not found: %s", pluginName)
	}
}
```

5. **Import your plugin** in `app/plugins/loader.go`:

```go
import (
	// ... existing imports ...
	"github.com/anoam/netscout-pi/app/plugins/plugins/my_plugin"
)
```

## Parameter Types

The plugin system supports the following parameter types:

- `string`: Text input
- `number`: Numeric input with optional min/max/step
- `boolean`: True/false checkbox
- `select`: Dropdown selection with options
- `range`: Range slider with min/max/step

## Display Format

The frontend automatically formats the results based on the plugin ID. If you need a custom display format, you'll need to update:

1. The JavaScript frontend in `app/static/js/plugin-manager.js`
2. The plugin page template in `app/templates/plugin_page.html`

## Testing Your Plugin

1. Build and run the application
2. Navigate to your plugin page at `/plugin/my_plugin`
3. Configure the parameters and run the plugin
4. Check the results and API response at `/api/plugins/my_plugin/run`

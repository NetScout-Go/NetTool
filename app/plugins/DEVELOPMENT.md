# NetScout-Pi Plugin Development Guide

This guide provides detailed instructions for creating and integrating new plugins into NetScout-Pi.

## Plugin System Overview

NetScout-Pi uses a modular plugin system where each plugin is contained in its own directory with configuration and implementation files. This architecture makes it easy to add, remove, or update plugins independently without affecting the core application.

Each plugin consists of:

1. A **plugin.json** file defining metadata and parameters
2. A **plugin.go** file implementing the plugin's functionality

## Plugin Directory Structure

Plugins reside under the `app/plugins/plugins/` directory:

```text
app/plugins/plugins/
  ├── ping/
  │   ├── plugin.json   # Plugin metadata and parameters
  │   └── plugin.go     # Plugin implementation
  ├── traceroute/
  │   ├── plugin.json
  │   └── plugin.go
  └── ...
```

## Step-by-Step Plugin Creation Guide

### 1. Create the Plugin Directory

Create a new directory for your plugin under `app/plugins/plugins/`:

```bash
mkdir -p app/plugins/plugins/my_plugin
```

### 2. Create the Plugin Metadata File

Create a `plugin.json` file that defines your plugin's metadata and parameters:

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

#### Available Parameter Types

The plugin system supports the following parameter types:

- `string`: Text input field
- `number`: Numeric input with optional min/max/step
- `boolean`: True/false checkbox
- `select`: Dropdown selection with options
- `range`: Range slider with min/max/step

#### Parameter Properties

Each parameter can have the following properties:

| Property | Description | Required | Applies To |
|----------|-------------|----------|------------|
| id | Unique identifier for the parameter | Yes | All |
| name | Display name for the parameter | Yes | All |
| description | Help text for the parameter | Yes | All |
| type | Parameter type (string, number, boolean, select, range) | Yes | All |
| required | Whether the parameter is required | Yes | All |
| default | Default value for the parameter | No | All |
| min | Minimum allowed value | No | number, range |
| max | Maximum allowed value | No | number, range |
| step | Increment step | No | number, range |
| options | Array of options for select type | Yes | select |

For `select` parameters, the `options` property is an array of objects with `value` and `label` properties:

```json
"options": [
  {"value": "option1", "label": "Option 1"},
  {"value": "option2", "label": "Option 2"}
]
```

### 3. Implement the Plugin Logic

Create a `plugin.go` file that implements your plugin's functionality:

```go
package my_plugin

import (
    "fmt"
    "time"
)

// Execute handles the plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
    // Get parameters from the params map
    param1, _ := params["param1"].(string)
    param2Raw, ok := params["param2"].(float64)
    if !ok {
        param2Raw = 10 // Default value
    }
    param2 := int(param2Raw)
    
    // Implement your plugin logic here
    // This is just an example - replace with your actual implementation
    result := map[string]interface{}{
        "param1": param1,
        "param2": param2,
        "result": "Your plugin's result",
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    return result, nil
}
```

Important points for plugin implementation:

1. Your plugin package name should match the directory name
2. The `Execute` function is the entry point for your plugin
3. Parameters are passed as a map[string]interface{} and need to be type-asserted
4. Always provide fallback values for optional parameters
5. Return results as a map[string]interface{} for JSON serialization
6. Include error handling for potential failures
7. Include a timestamp in your results

### 4. Update the Plugin Loader

Open `app/plugins/loader.go` and update the `getPluginExecuteFunc` method to include your new plugin:

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

### 5. Import Your Plugin

In the same `app/plugins/loader.go` file, add an import for your plugin package:

```go
import (
    // ... existing imports ...
    "github.com/anoam/netscout-pi/app/plugins/plugins/my_plugin"
)
```

## Custom Result Formatting

By default, the frontend displays plugin results in a generic JSON format. If you want a custom display format for your plugin, you'll need to:

### 1. Update the JavaScript Frontend

Modify `app/static/js/plugin-manager.js` to add a custom display function for your plugin:

```javascript
// Display plugin results
displayResults: function(data, element) {
    // ... existing code ...
    
    switch (this.activePluginId) {
        // ... existing plugins ...
        case 'my_plugin':
            this.displayMyPluginResults(data, element);
            break;
        // ... existing default case ...
    }
},

// Custom display function for your plugin
displayMyPluginResults: function(data, element) {
    element.innerHTML = `
        <div class="result-card">
            <div class="result-header">My Plugin Results</div>
            <div class="result-body">
                <div class="result-row">
                    <div class="result-label">Parameter 1</div>
                    <div class="result-value">${data.param1}</div>
                </div>
                <div class="result-row">
                    <div class="result-label">Parameter 2</div>
                    <div class="result-value">${data.param2}</div>
                </div>
                <div class="result-row">
                    <div class="result-label">Result</div>
                    <div class="result-value">${data.result}</div>
                </div>
            </div>
        </div>
    `;
}
```

### 2. Update the Plugin Page Template (Optional)

If your plugin requires special handling in the template, modify `app/templates/plugin_page.html`.

## Adding Custom Styles

You can add custom styles for your plugin by:

1. Creating a new CSS file in `app/static/css/` directory
2. Referencing it in your custom display function
3. Or updating the existing `style.css` file with plugin-specific styles

## Testing Your Plugin

1. Build and run the application:

   ```bash
   go build
   ./netscout-pi
   ```

2. Navigate to your plugin page at `/plugin/my_plugin`

3. Configure the parameters and run the plugin

4. Check the results and API response at `/api/plugins/my_plugin/run`

## Debugging Tips

1. **Check the logs**: Look for any error messages in the server logs

2. **Test the API directly**: Use curl or Postman to call your plugin's API endpoint:

   ```bash
   curl -X POST http://localhost:8080/api/plugins/my_plugin/run \
     -H "Content-Type: application/json" \
     -d '{"param1": "value1", "param2": 42}'
   ```

3. **Validate plugin.json**: Ensure your plugin.json file is valid JSON

4. **Check for import errors**: Make sure your plugin is properly imported in loader.go

## Advanced Plugin Development

### Running External Commands

To run external commands in your plugin:

```go
cmd := exec.Command("command", "arg1", "arg2")
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

err := cmd.Run()
if err != nil {
    return nil, fmt.Errorf("command failed: %v: %s", err, stderr.String())
}

output := stdout.String()
// Process output...
```

### Asynchronous Operations

For long-running operations, consider implementing timeout handling:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "command", "args")
// ...
```

### Error Handling Best Practices

Always provide meaningful error messages and proper error handling:

```go
if err != nil {
    return nil, fmt.Errorf("failed to perform operation: %w", err)
}
```

### Dependency Management

If your plugin requires external dependencies:

1. Add them to the project's `go.mod` file
2. Import them in your plugin.go file
3. Document the dependencies in your plugin documentation

### Unit Testing

Create unit tests for your plugin in a `plugin_test.go` file:

```go
package my_plugin

import (
    "testing"
    "reflect"
)

func TestExecute(t *testing.T) {
    // Define test cases
    testCases := []struct {
        name          string
        params        map[string]interface{}
        expectedError bool
    }{
        {
            name: "Basic test",
            params: map[string]interface{}{
                "param1": "test",
                "param2": float64(10),
            },
            expectedError: false,
        },
        // Add more test cases...
    }

    // Run test cases
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := Execute(tc.params)
            
            // Check error
            if tc.expectedError && err == nil {
                t.Errorf("Expected error but got none")
            }
            if !tc.expectedError && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            
            // Validate result
            if err == nil {
                // Add assertions for expected results
            }
        })
    }
}
```

## Example Plugins

For reference, explore these existing plugins:

- **Ping**: Simple network connectivity test
- **Traceroute**: Network path tracing
- **Port Scanner**: Network port scanning
- **DNS Lookup**: Domain name resolution

## Icon Reference

When choosing an icon for your plugin, use one of the following values:

- `network`: Network/connectivity icon
- `ping`: Ping/echo icon
- `dns`: DNS/domain icon
- `port`: Port/service icon
- `route`: Route/path icon
- `speed`: Speed/performance icon
- `info`: Information icon
- `scan`: Scanning icon
- `custom_icon`: Your custom icon (requires additional frontend changes)

## Contribution Guidelines

When submitting a new plugin, ensure:

1. Your plugin follows the structure outlined in this guide
2. Include comprehensive documentation in your code
3. Add appropriate error handling
4. Create unit tests for your plugin
5. Update the plugin loader to include your plugin
6. Add any custom frontend elements needed for display
7. Document any dependencies or special requirements

## Plugin Security Considerations

When developing plugins that run system commands:

1. Validate and sanitize all user inputs to prevent command injection
2. Use specific command paths instead of relying on PATH environment
3. Limit permissions to only what is necessary
4. Avoid running commands as root/sudo unless absolutely necessary
5. Implement timeouts for all operations
6. Sanitize and validate command output before returning to the client

## Common Pitfalls and Solutions

1. **Type Assertion Errors**: Always check the type assertion with the "comma ok" idiom
2. **Missing Parameters**: Provide fallback values for all parameters
3. **Race Conditions**: Use proper synchronization for concurrent operations
4. **Resource Leaks**: Close all resources (files, connections) with defer statements
5. **Large Response Data**: Consider pagination or limiting data size for large results

## Troubleshooting Common Issues

### Plugin Not Loading

If your plugin doesn't appear in the dashboard:

1. Check that the plugin directory and files are named correctly
2. Verify that plugin.json is valid
3. Ensure the plugin is properly imported in loader.go
4. Check the server logs for any error messages

### Plugin Execution Errors

If your plugin fails to execute:

1. Test the plugin via the API directly
2. Check parameter handling in your Execute function
3. Verify that any external commands or dependencies are available
4. Check for permission issues if running system commands

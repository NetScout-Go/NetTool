# "Continue to Iterate?" Feature Documentation

This document outlines how to use the "Continue to iterate?" feature in NetTool, which allows running operations in an iterative mode across the UI, Go plugins, and CLI.

## Overview

The iteration feature allows users to:

- Run a plugin or operation multiple times
- View history of previous iterations
- Control when to stop the iterations
- Configure iteration settings

## Using Iteration in the UI

### Dashboard

The dashboard now includes an auto-refresh toggle and iteration controls:

1. Enable the auto-refresh toggle to automatically refresh the dashboard
2. Use the refresh interval dropdown to set how frequently the dashboard refreshes
3. View iteration status and history in the iteration panel
4. Respond to "Continue to iterate?" prompts in the modal dialog

### Plugin Page

When using an iterable plugin:

1. The plugin page will show an "Enable Iteration" checkbox
2. Configure iteration settings (interval, max iterations)
3. Click "Run" to start the iterative execution
4. Respond to "Continue to iterate?" prompts
5. View iteration history in the results section
6. Use the "Stop Iteration" button to manually stop

## Implementing an Iterable Plugin

To make a plugin support iteration:

1. Define the plugin parameters in `plugin.json`:

   ```json
   {
     "name": "Your Plugin",
     "parameters": [
       {
         "name": "param1",
         "type": "string",
         "required": true,
         "description": "Parameter 1",
         "canIterate": true
       }
     ]
   }
   ```

2. Implement the `IterablePlugin` interface in your Go plugin:

   ```go
   type YourPlugin struct {
     types.BaseIterablePlugin // Embed the base implementation
     // Your plugin fields
   }
   
   // Implement required methods
   func (p *YourPlugin) ShouldContinueIteration(config *types.PluginExecutionConfig) (bool, error) {
     // Your logic to determine if iteration should continue
     return true, nil
   }
   
   func (p *YourPlugin) Run(params map[string]interface{}) (interface{}, error) {
     // Your plugin logic
     return result, nil
   }
   ```

3. Register your plugin as iterable in the plugin loader

## Using Iteration from CLI

### Using iterate_plugin.sh

The `iterate_plugin.sh` script allows running any iterable plugin with iteration support:

```bash
./iterate_plugin.sh plugin_name param1=value1 param2=value2
```

Options:

- `--max-iterations=N`: Set maximum number of iterations
- `--interval=N`: Set interval between iterations (seconds)
- `--auto-continue=true|false`: Automatically continue without prompting
- `--save-results=true|false`: Save results to files

### Using continue_to_iterate.sh

For custom scripts, you can use the `continue_to_iterate.sh` utility:

```bash
source continue_to_iterate.sh

# Run a command with iteration
run_with_iteration "your_command" max_iterations delay_seconds

# Or use the prompt directly
while true; do
  # Your code here
  
  # Prompt to continue
  if ! continue_to_iterate $count "Continue running?"; then
    break
  fi
  
  count=$((count + 1))
done
```

### Example: Running Iterative Ping

```bash
./run_iterative_ping.sh 8.8.8.8 --count=4 --max-iterations=10 --interval=5
```

## Integration with Other Tools

You can integrate the iteration feature with other scripts and tools:

```bash
# Example: Run a network test every 5 minutes and save results
./iterate_plugin.sh network_info --interval=300 --save-results=true
```

## Troubleshooting

- If iteration doesn't start, check if the plugin supports iteration
- For CLI issues, ensure scripts have executable permissions
- Check logs for any errors in the iteration process

## Future Enhancements

- Scheduled iterations (run at specific times)
- Conditional iterations (continue based on result values)
- Remote iteration control via API

## How Iteration Works in NetTool

The iteration feature works at multiple levels:

1. **Plugin JSON Definition**: Plugins define which parameters can be iterated with the `canIterate` flag
2. **Plugin Implementation**: Plugins implement the `IterablePlugin` interface to handle iteration logic
3. **Plugin Loader**: The loader detects iterable plugins and manages their execution
4. **UI Integration**: The web interface provides controls for managing iterations
5. **CLI Tools**: Command-line tools support running iterative plugins

### Implementation Layers

#### 1. Plugin JSON Definition

All plugins must specify which parameters can be iterated over:

```json
{
  "parameters": [
    {
      "id": "host",
      "name": "Host",
      "type": "string",
      "required": true,
      "canIterate": true
    }
  ]
}
```

#### 2. Go Plugin Implementation

Plugins must implement the `IterablePlugin` interface:

```go
// Required interface
type IterablePlugin interface {
  Plugin
  SupportsIteration() bool
  ExecuteIteration(params map[string]interface{}, iterationCount int) (result interface{}, continueIteration bool, err error)
}
```

A complete iterable plugin implementation looks like:

```go
type YourPlugin struct {
  types.BaseIterablePlugin
  // Your plugin state fields
  Results        []Result
  IterationCount int
}

// Initialize a new plugin instance
func NewPlugin() *YourPlugin {
  p := &YourPlugin{}
  
  // Create the base iterable plugin
  p.BaseIterablePlugin = types.NewIterablePlugin(
    definition,
    p.execute,
    p.executeIteration,
  )
  
  return p
}

// Main execution function (implements Plugin interface)
func (p *YourPlugin) execute(params map[string]interface{}) (interface{}, error) {
  // Extract parameters
  param1, ok := params["param1"].(string)
  if !ok {
    return nil, fmt.Errorf("param1 is required")
  }
  
  // Check if we should use iteration
  config := types.ExtractIterationConfig(params)
  if config.Iterate {
    return types.RunWithIteration(p, params)
  }
  
  // Run a single execution
  return p.runOperation(param1)
}

// Implements IterablePlugin.ExecuteIteration
func (p *YourPlugin) executeIteration(params map[string]interface{}, iterationCount int) (interface{}, bool, error) {
  // Extract parameters
  param1, ok := params["param1"].(string)
  if !ok {
    return nil, false, fmt.Errorf("param1 is required")
  }
  
  // Run the operation
  result, err := p.runOperation(param1)
  if err != nil {
    return nil, false, err
  }
  
  // Add iteration metadata to the result
  resultMap, ok := result.(map[string]interface{})
  if ok {
    resultMap["iterationCount"] = iterationCount
    resultMap["timestamp"] = time.Now().Format(time.RFC3339)
    resultMap["iteration_data"] = map[string]interface{}{
      "can_iterate":        true,
      "supports_iteration": true,
      "iteration_summary":  fmt.Sprintf("Iteration %d completed", iterationCount),
    }
  }
  
  // Let the system handle prompting for continuation
  return result, true, nil
}

// Main function for CLI usage
func main() {
  // Create plugin instance
  plugin := NewPlugin()
  
  // Check command line arguments
  if len(os.Args) < 2 {
    fmt.Println("Usage: plugin.go --definition|--execute='{\"params\":...}'")
    os.Exit(1)
  }
  
  // Handle --definition argument
  if os.Args[1] == "--definition" {
    definition, err := json.Marshal(plugin.GetDefinition())
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    fmt.Println(string(definition))
    return
  }
  
  // Handle --execute argument
  if strings.HasPrefix(os.Args[1], "--execute=") {
    // Extract parameters JSON
    paramsJSON := strings.TrimPrefix(os.Args[1], "--execute=")
    
    // Parse parameters
    var params map[string]interface{}
    if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    
    // Execute plugin
    result, err := types.RunWithIteration(plugin, params)
    if err != nil {
      fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
      os.Exit(1)
    }
    
    // Output result as JSON
    resultJSON, err := json.Marshal(result)
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    fmt.Println(string(resultJSON))
    return
  }
  
  fmt.Println("Unknown command")
  os.Exit(1)
}
```

## Advanced Iteration Features

### Conditional Iteration

You can implement conditional iteration based on results:

```go
func (p *YourPlugin) executeIteration(params map[string]interface{}, iterationCount int) (interface{}, bool, error) {
  // ... perform operations ...
  
  // Only continue if we meet a condition
  shouldContinue := result.SomeValue > threshold
  
  return result, shouldContinue, nil
}
```

### Storing Iteration History

To maintain history across iterations:

```go
type YourPlugin struct {
  types.BaseIterablePlugin
  History []Result // Store history of all iterations
}

func (p *YourPlugin) executeIteration(params map[string]interface{}, iterationCount int) (interface{}, bool, error) {
  // ... perform operations ...
  
  // Add to history
  p.History = append(p.History, result)
  
  // Include history in result
  resultMap["history"] = p.History
  
  return resultMap, true, nil
}
```

### Time-Based Iteration

For time-based iteration:

```go
func (p *YourPlugin) executeIteration(params map[string]interface{}, iterationCount int) (interface{}, bool, error) {
  // ... perform operations ...
  
  // Check if we've been running too long
  if time.Since(p.StartTime) > maxDuration {
    return result, false, nil
  }
  
  return result, true, nil
}
```

## Troubleshooting Iteration

### Common Issues and Solutions

1. **Plugin JSON Parse Errors**

   Error: `Warning: Failed to parse plugin.json for [plugin_name]`
   
   Solution:
   - Use `jq` to validate and fix the JSON: `jq . plugin.json > fixed.json && mv fixed.json plugin.json`
   - Check for duplicate keys (especially `version`)
   - Ensure parameter objects have all required fields

2. **Iteration Not Working**

   Error: Plugin executes but doesn't prompt for iteration
   
   Check:
   - `plugin.json` has `canIterate: true` for relevant parameters
   - Plugin implements `IterablePlugin` interface
   - `ExecuteIteration` method returns `true` for the continue flag
   - UI has iteration controls enabled
   
3. **Plugin Registration Issues**

   Error: `Plugin not registered in registry`
   
   Solution:
   - Check that plugin.go has correct `init()` function
   - Ensure plugin struct embeds `BaseIterablePlugin`
   - Check import paths for types package
   
4. **CLI Iteration Problems**

   Error: CLI runs once but doesn't iterate
   
   Check:
   - Command includes `--iterate` or sets `continueToIterate` parameter
   - Plugin parameters match expected format
   - Plugin's go file returns correct JSON output format

### Debugging Iteration

For advanced debugging:

```bash
# Run with verbose output
DEBUG=1 ./iterate_plugin.sh your_plugin param1=value1

# Trace plugin execution
GO_DEBUG=1 ./run_iterative_ping.sh 8.8.8.8
```

## Performance Considerations

- Set reasonable `iterationDelay` values (1-5 seconds minimum)
- Use `maxIterations` to prevent infinite loops
- For resource-intensive operations, consider:
  - Limiting concurrent iterations
  - Implementing adaptive delays
  - Storing minimal history data

## Best Practices

1. Always define clear iteration parameters in plugin.json
2. Include iteration metadata in results
3. Handle errors gracefully during iterations
4. Provide clear user feedback on iteration progress
5. Allow users to cancel iterations
6. Store iteration history efficiently
7. Test iteration in both UI and CLI contexts
8. Add iteration support to all long-running operations

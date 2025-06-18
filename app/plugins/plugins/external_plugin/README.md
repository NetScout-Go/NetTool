# External Plugin Interface

This plugin provides an interface for running external plugins written in any programming language. It allows you to extend NetScout-Pi with plugins written in Python, Bash, Node.js, or any other language that can:

1. Read JSON from stdin
2. Process the parameters
3. Output JSON to stdout

## How It Works

The external plugin interface acts as a bridge between NetScout-Pi and external scripts or programs:

1. It converts the parameters from the dashboard to JSON
2. Executes the external program and passes the parameters via stdin
3. Reads the output from the external program
4. Parses the JSON output and returns it to the dashboard

## Parameters

- **Plugin Type**: Select from built-in Python, Bash, or custom plugin
- **Custom Plugin Path**: Absolute path to the custom plugin executable (only needed for custom plugins)
- **Timeout**: Maximum execution time in seconds (default: 30)
- **Custom Parameters**: Additional parameters in JSON format to pass to the external plugin

## Creating an External Plugin

External plugins can be written in any language as long as they:

1. Accept JSON input from stdin
2. Output valid JSON to stdout
3. Have executable permissions

### Example Python Plugin

```python
#!/usr/bin/env python3
import json
import sys
import time

def main():
    # Read parameters from stdin
    params_json = sys.stdin.read()
    params = json.loads(params_json)
    
    # Process parameters
    custom_params = json.loads(params.get("custom_params", "{}"))
    
    # Do your work here
    
    # Create result
    result = {
        "plugin_name": "My Python Plugin",
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "message": "Hello from external plugin!",
        "data": {
            # Your plugin's result data here
        }
    }
    
    # Output result as JSON to stdout
    print(json.dumps(result))

if __name__ == "__main__":
    main()
```

### Example Bash Plugin

```bash
#!/usr/bin/env bash

# Read JSON from stdin
input=$(cat)

# Extract parameters using jq (must be installed)
plugin_type=$(echo $input | jq -r '.plugin_type')
custom_params=$(echo $input | jq -r '.custom_params')

# Do your work here
timestamp=$(date +"%Y-%m-%d %H:%M:%S")

# Output JSON result
echo "{\"plugin_name\": \"Bash Plugin\", \"timestamp\": \"$timestamp\", \"message\": \"Hello from Bash plugin!\"}"
```

## Using External Plugins

1. In the NetScout-Pi dashboard, select the "External Plugin Interface"
2. Choose a plugin type:
   - **Python Plugin**: Uses the built-in Python example plugin
   - **Bash Plugin**: Uses the built-in Bash example plugin
   - **Custom Plugin**: Allows you to specify your own plugin path
3. For custom plugins, enter the absolute path to your plugin executable (and ensure it has executable permissions with `chmod +x your_plugin.py`)
4. Set the timeout and any custom parameters in JSON format
5. Run the plugin

## Error Handling

If the external plugin:

- Times out: An error will be returned with the timeout message
- Exits with a non-zero status: The error message will be returned
- Outputs non-JSON: The raw output will be returned as a string in the "raw_output" field
# Expected Output Format

External plugins should output a JSON object with the following structure:

```json
{
  "plugin_name": "Your Plugin Name",
  "timestamp": "2025-06-13 12:34:56",
  "message": "A human-readable message describing the result",
  "data": {
    // Your plugin's specific data structure here
    // This can be any valid JSON object
  }
}
```

The `data` field should contain the specific output from your plugin's functionality. This structure will be seamlessly integrated into the NetScout-Pi dashboard.

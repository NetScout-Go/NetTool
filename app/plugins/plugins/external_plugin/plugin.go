package external_plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Execute handles the execution of external plugins
func Execute(params map[string]interface{}) (interface{}, error) {
	// Small delay to avoid UI race conditions where a fast response might not register
	time.Sleep(500 * time.Millisecond)

	// Extract parameters
	pluginType, ok := params["plugin_type"].(string)
	if !ok || pluginType == "" {
		return nil, fmt.Errorf("plugin_type is required and must be a string")
	}

	// Determine the plugin path based on type
	var pluginPath string
	if pluginType == "custom" {
		// For custom plugins, use the provided path
		customPath, ok := params["plugin_path"].(string)
		if !ok || customPath == "" {
			return nil, fmt.Errorf("plugin_path is required for custom plugins")
		}
		pluginPath = customPath
	} else {
		// For built-in plugins, use the examples from the examples directory
		baseDir, err := filepath.Abs(filepath.Dir(""))
		if err != nil {
			return nil, fmt.Errorf("failed to determine base directory: %w", err)
		}

		examplesDir := filepath.Join(baseDir, "app/plugins/plugins/external_plugin/examples")

		switch pluginType {
		case "python":
			pluginPath = filepath.Join(examplesDir, "python_plugin.py")
		case "bash":
			pluginPath = filepath.Join(examplesDir, "bash_plugin.sh")
		default:
			return nil, fmt.Errorf("unknown plugin type: %s", pluginType)
		}
	}

	// Get timeout with default value of 30 seconds
	timeoutRaw, ok := params["timeout"].(float64)
	if !ok {
		timeoutRaw = 30
	}
	timeout := time.Duration(timeoutRaw) * time.Second

	// Parse custom parameters
	customParamsStr, ok := params["custom_params"].(string)
	if !ok || customParamsStr == "" {
		customParamsStr = "{}"
	}

	// Convert all parameters to JSON to pass to the external plugin
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Create a command with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute the external plugin with parameters as JSON input
	cmd := exec.CommandContext(ctx, pluginPath)
	cmd.Stdin = strings.NewReader(string(paramsJSON))

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("external plugin execution timed out after %v seconds", timeoutRaw)
		}
		return nil, fmt.Errorf("external plugin execution failed: %w", err)
	}

	// Parse the output as JSON
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// If the output isn't valid JSON, return it as a string
		return map[string]interface{}{
			"raw_output": string(output),
		}, nil
	}

	return result, nil
}

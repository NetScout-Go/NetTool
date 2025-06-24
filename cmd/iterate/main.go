package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NetScout-Go/NetTool/app/plugins"
	"github.com/NetScout-Go/NetTool/app/plugins/cli"
	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

// SimplePluginWrapper is a simple wrapper for plugin execution
type SimplePluginWrapper struct {
	id          string
	definition  types.PluginDefinition
	executeFunc func(map[string]interface{}) (interface{}, error)
}

// GetDefinition returns the plugin definition
func (s *SimplePluginWrapper) GetDefinition() types.PluginDefinition {
	return s.definition
}

// Execute runs the plugin with the given parameters
func (s *SimplePluginWrapper) Execute(params map[string]interface{}) (interface{}, error) {
	return s.executeFunc(params)
}

func main() {
	// Define command-line flags
	pluginID := flag.String("plugin", "", "ID of the plugin to run")
	paramsFile := flag.String("params", "", "Path to JSON file containing parameters")
	paramsJSON := flag.String("paramsJson", "{}", "JSON string of parameters")
	maxIterations := flag.Int("max", 0, "Maximum number of iterations (0 = unlimited)")
	delay := flag.Int("delay", 5, "Delay between iterations in seconds")
	outputFile := flag.String("output", "", "Path to save results (optional)")
	continueToIterate := flag.Bool("iterate", false, "Whether to run with iteration")
	flag.Parse()

	// Check if a plugin ID was provided
	if *pluginID == "" {
		fmt.Println("Error: Plugin ID is required")
		printUsage()
		os.Exit(1)
	}

	// Load the plugin
	pluginInstance, err := loadPlugin(*pluginID)
	if err != nil {
		fmt.Printf("Error loading plugin '%s': %v\n", *pluginID, err)
		os.Exit(1)
	}

	// Parse parameters
	params, err := parseParams(*paramsFile, *paramsJSON)
	if err != nil {
		fmt.Printf("Error parsing parameters: %v\n", err)
		os.Exit(1)
	}

	// Add iteration parameter if requested
	if *continueToIterate {
		params["continueToIterate"] = true
	}

	// Check if the plugin supports iteration
	var iterablePlugin types.IterablePlugin
	var isIterable bool

	iterablePlugin, isIterable = pluginInstance.(types.IterablePlugin)
	if *continueToIterate && !isIterable {
		fmt.Printf("Warning: Plugin '%s' does not support iteration. Running once.\n", *pluginID)
	}

	// Run the plugin
	if *continueToIterate && isIterable {
		// Create CLI for iterable plugin
		iterableCLI := cli.NewIterableCLI(iterablePlugin)
		iterableCLI.SetParams(params)
		iterableCLI.SetMaxIterations(*maxIterations)
		iterableCLI.SetIterationDelay(time.Duration(*delay) * time.Second)

		// Run with iteration
		if err := iterableCLI.Run(); err != nil {
			fmt.Printf("Error running plugin: %v\n", err)
			os.Exit(1)
		}

		// Save results if output file is specified
		if *outputFile != "" {
			if err := iterableCLI.SaveResultsToFile(*outputFile); err != nil {
				fmt.Printf("Error saving results: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Results saved to %s\n", *outputFile)
		}
	} else {
		// Run once without iteration
		result, err := pluginInstance.Execute(params)
		if err != nil {
			fmt.Printf("Error running plugin: %v\n", err)
			os.Exit(1)
		}

		// Print the result
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(resultJSON))

		// Save result if output file is specified
		if *outputFile != "" {
			if err := os.WriteFile(*outputFile, resultJSON, 0644); err != nil {
				fmt.Printf("Error saving result: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Result saved to %s\n", *outputFile)
		}
	}
}

// loadPlugin loads a plugin by ID
func loadPlugin(pluginID string) (types.Plugin, error) {
	// Find plugin directory - need to adjust path since we're in cmd/iterate
	pluginDir := filepath.Join("..", "..", "app", "plugins", "plugins", pluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}

	// Load the plugin definition
	definitionFile := filepath.Join(pluginDir, "plugin.json")
	data, err := os.ReadFile(definitionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin definition: %v", err)
	}

	// Parse plugin definition
	var definition types.PluginDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("failed to parse plugin definition: %v", err)
	}

	// We need to use the plugin loader to create the plugin properly
	// since DynamicPlugin fields are unexported
	loader := plugins.NewPluginLoader(filepath.Join("..", "..", "app", "plugins", "plugins"))

	// Load all plugins first to register them
	_, err = loader.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %v", err)
	}

	// Get the plugin execution function and wrap it in a simple plugin implementation
	executeFunc, err := loader.GetPluginExecuteFunc(pluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin execution function not found: %v", err)
	}

	// Create a simple plugin wrapper
	return &SimplePluginWrapper{
		id:          pluginID,
		definition:  definition,
		executeFunc: executeFunc,
	}, nil
}

// parseParams parses parameters from a file or JSON string
func parseParams(paramsFile, paramsJSON string) (map[string]interface{}, error) {
	// Initialize params
	params := make(map[string]interface{})

	// Parse from file if provided
	if paramsFile != "" {
		data, err := os.ReadFile(paramsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read parameters file: %v", err)
		}

		if err := json.Unmarshal(data, &params); err != nil {
			return nil, fmt.Errorf("failed to parse parameters JSON: %v", err)
		}
	} else if paramsJSON != "" {
		// Parse from JSON string
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return nil, fmt.Errorf("failed to parse parameters JSON: %v", err)
		}
	}

	return params, nil
}

// printUsage prints usage information
func printUsage() {
	fmt.Println("Usage: iterate [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -plugin string     ID of the plugin to run")
	fmt.Println("  -params string     Path to JSON file containing parameters")
	fmt.Println("  -paramsJson string JSON string of parameters")
	fmt.Println("  -iterate           Run with iteration support")
	fmt.Println("  -max int           Maximum number of iterations (0 = unlimited)")
	fmt.Println("  -delay int         Delay between iterations in seconds")
	fmt.Println("  -output string     Path to save results (optional)")
	fmt.Println("")
	fmt.Println("Example:")
	fmt.Println("  iterate -plugin iterative_ping -paramsJson '{\"host\":\"8.8.8.8\",\"count\":3}' -iterate -max 10 -delay 2")
}

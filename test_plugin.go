package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/NetScout-Go/NetTool/app/plugins"
)

func main() {
	// Create plugin loader
	loader := plugins.NewPluginLoader("app/plugins/plugins")

	// Load plugins
	_, err := loader.LoadPlugins()
	if err != nil {
		log.Fatalf("Failed to load plugins: %v", err)
	}

	// Get the registry
	registry := plugins.GetRegistry()

	// Test the arp_manager plugin
	fmt.Println("Testing arp_manager plugin...")
	execFunc, err := registry.GetPluginFunc("arp_manager")
	if err != nil {
		log.Fatalf("Failed to get arp_manager plugin: %v", err)
	}

	// Execute the plugin with test parameters
	params := map[string]interface{}{
		"action": "show",
	}

	result, err := execFunc(params)
	if err != nil {
		log.Fatalf("Failed to execute arp_manager plugin: %v", err)
	}

	// Print the result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Result: %s\n", string(resultJSON))
}

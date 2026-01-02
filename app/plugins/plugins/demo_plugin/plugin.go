// Demo Plugin - Showcases NetTool plugin features
// This plugin demonstrates all available parameter types, display formats,
// error handling, and dependency management for plugin developers.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ========================================
// Display Format Types
// ========================================

// DisplaySection defines a formatted output section
type DisplaySection struct {
	Type    string                 `json:"type"`              // "metrics", "table", "list"
	Title   string                 `json:"title"`             // Section title
	Icon    string                 `json:"icon,omitempty"`    // Lucide icon name
	Color   string                 `json:"color,omitempty"`   // cyan, green, orange, purple, red, primary
	Columns []Column               `json:"columns,omitempty"` // For table type
	Data    interface{}            `json:"data,omitempty"`    // Table rows or list items
	Metrics []Metric               `json:"metrics,omitempty"` // For metrics type
	Extra   map[string]interface{} `json:"extra,omitempty"`   // Additional hints (e.g., progress bar)
}

// Column defines a table column
type Column struct {
	Key   string `json:"key"`            // Data key
	Label string `json:"label"`          // Display label
	Type  string `json:"type,omitempty"` // text, number, status, bytes, progress
}

// Metric defines a single metric item
type Metric struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
	Icon  string      `json:"icon,omitempty"`
	Color string      `json:"color,omitempty"` // green, orange, red, cyan, purple
	Unit  string      `json:"unit,omitempty"`
}

// ========================================
// Result Types
// ========================================

// PluginResult is the standard result format
type PluginResult struct {
	Success          bool             `json:"success"`
	Timestamp        string           `json:"timestamp"`
	ExecutionTime    string           `json:"executionTime"`
	Error            string           `json:"error,omitempty"`
	ErrorCode        string           `json:"errorCode,omitempty"`
	ErrorDetails     string           `json:"errorDetails,omitempty"`
	Warnings         []string         `json:"warnings,omitempty"`
	DependencyStatus []DependencyInfo `json:"dependencyStatus,omitempty"`
	Data             interface{}      `json:"data,omitempty"`
	Display          []DisplaySection `json:"_display"` // Formatted display sections
}

// DependencyInfo shows dependency installation status
type DependencyInfo struct {
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	InstallCmd string `json:"installCmd,omitempty"`
}

// ========================================
// Main Entry Point
// ========================================

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--definition" {
			printDefinition()
			return
		}
		if strings.HasPrefix(arg, "--execute=") {
			paramsJSON := strings.TrimPrefix(arg, "--execute=")
			execute(paramsJSON)
			return
		}
	}
	fmt.Println("Usage: plugin.go --definition | --execute=<params_json>")
}

func printDefinition() {
	data, _ := os.ReadFile("plugin.json")
	fmt.Print(string(data))
}

func execute(paramsJSON string) {
	start := time.Now()

	// Parse parameters
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		outputError("PARSE_ERROR", "Failed to parse parameters", err.Error())
		return
	}

	// Get parameters with defaults
	demoMode := getString(params, "demo_mode", "metrics")
	textInput := getString(params, "text_input", "Hello NetTool!")
	numberInput := getFloat(params, "number_input", 42)
	rangeInput := getFloat(params, "range_input", 50)
	boolInput := getBool(params, "boolean_input", true)
	networkInterface := getString(params, "network_interface", "auto")

	// Check dependencies
	deps := checkDependencies()

	// Build result based on demo mode
	result := &PluginResult{
		Success:          true,
		Timestamp:        time.Now().Format(time.RFC3339),
		DependencyStatus: deps,
	}

	switch demoMode {
	case "metrics":
		result.Display = buildMetricsDemo(textInput, numberInput, rangeInput, boolInput, networkInterface)
	case "table":
		result.Display = buildTableDemo()
	case "all":
		result.Display = buildAllDemo(textInput, numberInput, rangeInput, boolInput)
	case "error":
		outputError("DEMO_ERROR", "This is a demonstration error", "This shows how errors are displayed to users with error codes and details.")
		return
	case "warning":
		result.Warnings = []string{
			"This is a warning message",
			"Warnings are non-critical issues",
			"They appear in a yellow banner",
		}
		result.Display = buildWarningDemo()
	}

	// Store raw parameter data
	result.Data = map[string]interface{}{
		"receivedParams": map[string]interface{}{
			"demo_mode":         demoMode,
			"text_input":        textInput,
			"number_input":      numberInput,
			"range_input":       rangeInput,
			"boolean_input":     boolInput,
			"network_interface": networkInterface,
		},
		"message": "This is the raw data. Click 'Show Raw' to see the full JSON response.",
	}

	result.ExecutionTime = fmt.Sprintf("%.2fs", time.Since(start).Seconds())
	outputJSON(result)
}

// ========================================
// Demo Display Builders
// ========================================

func buildMetricsDemo(text string, number, rangeVal float64, boolVal bool, iface string) []DisplaySection {
	return []DisplaySection{
		{
			Type:  "metrics",
			Title: "Metrics Display Demo",
			Icon:  "bar-chart-3",
			Color: "cyan",
			Metrics: []Metric{
				{Label: "Text Parameter", Value: text, Icon: "type"},
				{Label: "Number Parameter", Value: number, Icon: "hash"},
				{Label: "Range Parameter", Value: fmt.Sprintf("%.0f%%", rangeVal), Icon: "sliders"},
				{Label: "Boolean Parameter", Value: boolVal, Icon: "toggle-right"},
				{Label: "Interface", Value: iface, Icon: "network"},
			},
		},
		{
			Type:  "metrics",
			Title: "Status Colors Demo",
			Icon:  "palette",
			Color: "purple",
			Metrics: []Metric{
				{Label: "Success Status", Value: "OK", Icon: "check-circle", Color: "green"},
				{Label: "Warning Status", Value: "Caution", Icon: "alert-triangle", Color: "orange"},
				{Label: "Error Status", Value: "Failed", Icon: "x-circle", Color: "red"},
				{Label: "Info Status", Value: "Info", Icon: "info", Color: "cyan"},
			},
		},
		{
			Type:  "metrics",
			Title: "Progress Bar Demo",
			Icon:  "loader",
			Color: "green",
			Metrics: []Metric{
				{Label: "Progress Value", Value: fmt.Sprintf("%.0f%%", rangeVal), Icon: "pie-chart"},
				{Label: "Description", Value: "Use 'extra.progress' for progress bars", Icon: "info"},
			},
			Extra: map[string]interface{}{
				"progress": rangeVal, // This shows a progress bar!
			},
		},
	}
}

func buildTableDemo() []DisplaySection {
	return []DisplaySection{
		{
			Type:  "table",
			Title: "Table Display Demo",
			Icon:  "table",
			Color: "orange",
			Columns: []Column{
				{Key: "name", Label: "Name", Type: "text"},
				{Key: "status", Label: "Status", Type: "status"},
				{Key: "value", Label: "Value", Type: "number"},
				{Key: "size", Label: "Size", Type: "bytes"},
			},
			Data: []map[string]interface{}{
				{"name": "Item One", "status": "up", "value": 100, "size": 1048576},
				{"name": "Item Two", "status": "down", "value": 50, "size": 2097152},
				{"name": "Item Three", "status": "up", "value": 75, "size": 536870912},
			},
		},
		{
			Type:  "table",
			Title: "Progress Column Demo",
			Icon:  "activity",
			Color: "cyan",
			Columns: []Column{
				{Key: "name", Label: "Resource", Type: "text"},
				{Key: "usage", Label: "Usage", Type: "progress"},
			},
			Data: []map[string]interface{}{
				{"name": "CPU", "usage": 45.5},
				{"name": "Memory", "usage": 72.3},
				{"name": "Disk", "usage": 91.8},
			},
		},
	}
}

func buildAllDemo(text string, number, rangeVal float64, boolVal bool) []DisplaySection {
	sections := []DisplaySection{
		{
			Type:  "metrics",
			Title: "📊 Your Input Parameters",
			Icon:  "settings",
			Color: "primary",
			Metrics: []Metric{
				{Label: "Text", Value: text, Icon: "type"},
				{Label: "Number", Value: number, Icon: "hash"},
				{Label: "Range", Value: fmt.Sprintf("%.0f%%", rangeVal), Icon: "sliders"},
				{Label: "Boolean", Value: boolVal, Icon: "toggle-right"},
			},
			Extra: map[string]interface{}{"progress": rangeVal},
		},
		{
			Type:  "table",
			Title: "📋 Column Types Reference",
			Icon:  "book-open",
			Color: "cyan",
			Columns: []Column{
				{Key: "type", Label: "Column Type", Type: "text"},
				{Key: "description", Label: "Description", Type: "text"},
				{Key: "example", Label: "Example", Type: "text"},
			},
			Data: []map[string]interface{}{
				{"type": "text", "description": "Plain text display", "example": "Hello World"},
				{"type": "number", "description": "Numeric values", "example": "42"},
				{"type": "status", "description": "up/down indicator with color", "example": "up / down"},
				{"type": "bytes", "description": "Auto-format to KB/MB/GB", "example": "1.5 GB"},
				{"type": "progress", "description": "Progress bar with percentage", "example": "75%"},
			},
		},
		{
			Type:  "metrics",
			Title: "🎨 Available Colors",
			Icon:  "palette",
			Color: "purple",
			Metrics: []Metric{
				{Label: "primary", Value: "Orange theme color", Color: "primary"},
				{Label: "cyan", Value: "Cyan/Teal", Color: "cyan"},
				{Label: "green", Value: "Success/OK", Color: "green"},
				{Label: "orange", Value: "Warning", Color: "orange"},
				{Label: "red", Value: "Error/Danger", Color: "red"},
				{Label: "purple", Value: "Purple accent", Color: "purple"},
			},
		},
	}

	return sections
}

func buildWarningDemo() []DisplaySection {
	return []DisplaySection{
		{
			Type:  "metrics",
			Title: "Warnings Demo",
			Icon:  "alert-triangle",
			Color: "orange",
			Metrics: []Metric{
				{Label: "Status", Value: "Plugin completed with warnings", Icon: "alert-circle", Color: "orange"},
				{Label: "Note", Value: "Warnings appear in a yellow banner above", Icon: "info"},
			},
		},
	}
}

// ========================================
// Helper Functions
// ========================================

func checkDependencies() []DependencyInfo {
	deps := []DependencyInfo{
		{Name: "curl", InstallCmd: "apt install curl"},
	}

	for i := range deps {
		if path, err := exec.LookPath(deps[i].Name); err == nil {
			deps[i].Installed = true
			// Try to get version
			out, _ := exec.Command(deps[i].Name, "--version").Output()
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				deps[i].Version = strings.TrimSpace(lines[0])
			}
			_ = path
		}
	}

	return deps
}

func getString(params map[string]interface{}, key, defaultVal string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getFloat(params map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := params[key].(float64); ok {
		return v
	}
	return defaultVal
}

func getBool(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return defaultVal
}

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Print(string(data))
}

func outputError(code, message, details string) {
	result := PluginResult{
		Success:      false,
		Error:        message,
		ErrorCode:    code,
		ErrorDetails: details,
		Timestamp:    time.Now().Format(time.RFC3339),
	}
	outputJSON(result)
}

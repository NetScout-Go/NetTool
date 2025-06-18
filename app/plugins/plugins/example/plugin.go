package example

import (
	"strings"
	"time"
)

// Execute handles the example plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Get parameters
	message, _ := params["message"].(string)
	repeatParam, ok := params["repeat"].(float64)
	if !ok {
		repeatParam = 1 // Default value
	}
	repeat := int(repeatParam)

	// Create repeated message
	var repeated []string
	for i := 0; i < repeat; i++ {
		repeated = append(repeated, message)
	}

	// Create result
	result := map[string]interface{}{
		"message":   message,
		"repeat":    repeat,
		"result":    strings.Join(repeated, "\n"),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	return result, nil
}

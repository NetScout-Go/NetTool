package subnet_calculator

import (
	"strings"
)

// ExecuteAdapter adapts the input from the dashboard to the format expected by the plugin
func ExecuteAdapter(params map[string]interface{}) (interface{}, error) {
	// Copy the parameters to avoid modifying the original
	adaptedParams := make(map[string]interface{})
	for k, v := range params {
		adaptedParams[k] = v
	}

	// Handle action
	action, _ := params["action"].(string)
	adaptedParams["action"] = action

	// Handle the IP list for supernet calculation
	if action == "supernet" {
		ipListStr, _ := params["ip_list"].(string)
		if ipListStr != "" {
			// Split the comma-separated string into a slice
			ipList := strings.Split(ipListStr, ",")
			// Convert to []interface{} as required by Execute
			ipListInterface := make([]interface{}, len(ipList))
			for i, ip := range ipList {
				ipListInterface[i] = strings.TrimSpace(ip)
			}
			adaptedParams["ip_list"] = ipListInterface
		}
	}

	// Call the original Execute function
	return Execute(adaptedParams)
}

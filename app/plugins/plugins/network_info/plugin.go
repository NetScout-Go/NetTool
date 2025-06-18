package network_info

// Execute handles the network_info plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// This plugin is handled directly by the main dashboard
	return map[string]string{"status": "This plugin provides data for the main dashboard"}, nil
}

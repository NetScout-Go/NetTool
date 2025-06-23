package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// DnsLookupPlugin is the main plugin struct
type DnsLookupPlugin struct {
	Results        []interface{}
	StartTime      time.Time
	IterationCount int
}

// NewPlugin creates a new plugin instance
func NewPlugin() *DnsLookupPlugin {
	return &DnsLookupPlugin{
		StartTime: time.Now(),
		Results:   []interface{}{},
	}
}

// Execute handles the DNS lookup plugin execution
func (p *DnsLookupPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	// Check if we should use iteration
	continueToIterate, _ := params["continueToIterate"].(bool)
	if continueToIterate {
		return p.executeWithIteration(params)
	}

	// Run a single execution
	return p.performDnsLookup(params)
}

// executeWithIteration handles running the plugin in iteration mode
func (p *DnsLookupPlugin) executeWithIteration(params map[string]interface{}) (interface{}, error) {
	// Run the DNS lookup operation
	result, err := p.performDnsLookup(params)
	if err != nil {
		return nil, err
	}

	// Update state
	p.IterationCount++
	if resultMap, ok := result.(map[string]interface{}); ok {
		// Create a copy of the result for history to avoid reference issues
		historyCopy := make(map[string]interface{})
		for k, v := range resultMap {
			historyCopy[k] = v
		}
		p.Results = append(p.Results, historyCopy)

		// Add iteration metadata to the result
		resultMap["iterationCount"] = p.IterationCount
		resultMap["elapsedTime"] = time.Since(p.StartTime).String()

		// Create a summary for the UI
		domain := resultMap["domain"].(string)
		recordType := resultMap["recordType"].(string)
		results := resultMap["results"].(map[string]interface{})

		var recordCount int
		for _, records := range results {
			if recordsArr, ok := records.([]string); ok {
				recordCount += len(recordsArr)
			}
		}

		// Add iteration_data for UI display
		resultMap["iteration_data"] = map[string]interface{}{
			"can_iterate":        true,
			"supports_iteration": true,
			"iteration_summary": fmt.Sprintf(
				"Iteration %d: %s lookup for %s - %d records found",
				p.IterationCount,
				recordType,
				domain,
				recordCount,
			),
		}

		// Add history summary
		if len(p.Results) > 1 {
			history := make([]map[string]interface{}, 0)
			for i, res := range p.Results {
				if resMap, ok := res.(map[string]interface{}); ok {
					// Create a simplified history entry
					domain := resMap["domain"].(string)
					recordType := resMap["recordType"].(string)
					timestamp := resMap["timestamp"].(string)

					var recordCount int
					if results, ok := resMap["results"].(map[string]interface{}); ok {
						for _, records := range results {
							if recordsArr, ok := records.([]string); ok {
								recordCount += len(recordsArr)
							}
						}
					}

					historyEntry := map[string]interface{}{
						"iteration":   i + 1,
						"timestamp":   timestamp,
						"domain":      domain,
						"recordType":  recordType,
						"recordCount": recordCount,
					}
					history = append(history, historyEntry)
				}
			}
			resultMap["history"] = history
		}
	}

	return result, nil
}

// performDnsLookup handles the actual DNS lookup logic
func (p *DnsLookupPlugin) performDnsLookup(params map[string]interface{}) (interface{}, error) {
	domain, _ := params["domain"].(string)
	recordType, _ := params["recordType"].(string)

	if domain == "" {
		return nil, fmt.Errorf("domain parameter is required")
	}

	if recordType == "" {
		recordType = "A"
	}

	results := make(map[string]interface{})

	// Determine which record types to look up
	var recordTypes []string
	if recordType == "ALL" {
		recordTypes = []string{"A", "AAAA", "MX", "NS", "TXT", "CNAME"}
	} else {
		recordTypes = []string{recordType}
	}

	// Fallback to Go's built-in resolver
	for _, rt := range recordTypes {
		switch rt {
		case "A":
			ips, err := net.LookupIP(domain)
			if err == nil {
				var ipv4s []string
				for _, ip := range ips {
					if ipv4 := ip.To4(); ipv4 != nil {
						ipv4s = append(ipv4s, ipv4.String())
					}
				}
				if len(ipv4s) > 0 {
					results["A"] = ipv4s
				}
			}
		case "AAAA":
			ips, err := net.LookupIP(domain)
			if err == nil {
				var ipv6s []string
				for _, ip := range ips {
					if ipv4 := ip.To4(); ipv4 == nil {
						ipv6s = append(ipv6s, ip.String())
					}
				}
				if len(ipv6s) > 0 {
					results["AAAA"] = ipv6s
				}
			}
		case "MX":
			mxs, err := net.LookupMX(domain)
			if err == nil {
				var mxRecords []string
				for _, mx := range mxs {
					mxRecords = append(mxRecords, fmt.Sprintf("%d %s", mx.Pref, mx.Host))
				}
				results["MX"] = mxRecords
			}
		case "NS":
			nss, err := net.LookupNS(domain)
			if err == nil {
				var nsRecords []string
				for _, ns := range nss {
					nsRecords = append(nsRecords, ns.Host)
				}
				results["NS"] = nsRecords
			}
		case "TXT":
			txts, err := net.LookupTXT(domain)
			if err == nil {
				results["TXT"] = txts
			}
		case "CNAME":
			cname, err := net.LookupCNAME(domain)
			if err == nil {
				results["CNAME"] = []string{strings.TrimSuffix(cname, ".")}
			}
		}
	}

	return map[string]interface{}{
		"domain":     domain,
		"recordType": recordType,
		"results":    results,
		"timestamp":  time.Now().Format(time.RFC3339),
	}, nil
}

// Main function
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
		// Read plugin.json for definition
		definitionBytes, err := os.ReadFile("plugin.json")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(definitionBytes))
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
		result, err := plugin.Execute(params)
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

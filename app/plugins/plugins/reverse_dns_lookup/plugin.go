package reverse_dns_lookup

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// LookupResult represents the result of a single reverse DNS lookup
type LookupResult struct {
	IPAddress    string        `json:"ip_address"`
	Hostnames    []string      `json:"hostnames"`
	Status       string        `json:"status"` // "success", "not_found", "error"
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
}

// Execute handles the reverse DNS lookup plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	ipAddressesStr, ok := params["ip_addresses"].(string)
	if !ok || ipAddressesStr == "" {
		return nil, fmt.Errorf("IP addresses are required")
	}

	timeoutFloat, ok := params["timeout"].(float64)
	if !ok {
		timeoutFloat = 5 // Default timeout in seconds
	}
	timeout := time.Duration(timeoutFloat) * time.Second

	dnsServer, _ := params["dns_server"].(string)

	concurrentLookupsFloat, ok := params["concurrent_lookups"].(float64)
	if !ok {
		concurrentLookupsFloat = 5 // Default concurrent lookups
	}
	concurrentLookups := int(concurrentLookupsFloat)

	// Parse IP addresses
	var ipAddresses []string
	// First split by line breaks
	lines := strings.Split(ipAddressesStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if the line contains comma-separated values
		if strings.Contains(line, ",") {
			for _, ip := range strings.Split(line, ",") {
				ip = strings.TrimSpace(ip)
				if ip != "" {
					ipAddresses = append(ipAddresses, ip)
				}
			}
		} else {
			ipAddresses = append(ipAddresses, line)
		}
	}

	// Initialize result structure
	result := map[string]interface{}{
		"ip_count":   len(ipAddresses),
		"timeout":    timeoutFloat,
		"dns_server": dnsServer,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Process IP addresses in batches
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]LookupResult, 0, len(ipAddresses))

	// Create a semaphore to limit concurrent lookups
	semaphore := make(chan struct{}, concurrentLookups)

	for _, ip := range ipAddresses {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		go func(ip string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			// Perform lookup
			lookupResult := performReverseDNSLookup(ip, dnsServer, timeout)

			// Add result
			mu.Lock()
			results = append(results, lookupResult)
			mu.Unlock()
		}(ip)
	}

	wg.Wait()

	// Count statistics
	success := 0
	notFound := 0
	errors := 0
	var totalTime time.Duration

	for _, res := range results {
		switch res.Status {
		case "success":
			success++
		case "not_found":
			notFound++
		case "error":
			errors++
		}
		totalTime += res.ResponseTime
	}

	// Add statistics to result
	result["results"] = results
	result["success_count"] = success
	result["not_found_count"] = notFound
	result["error_count"] = errors

	if len(results) > 0 {
		result["average_response_time"] = totalTime.Seconds() / float64(len(results))
	}

	return result, nil
}

// performReverseDNSLookup performs a reverse DNS lookup for a single IP address
func performReverseDNSLookup(ipAddress, dnsServer string, timeout time.Duration) LookupResult {
	result := LookupResult{
		IPAddress: ipAddress,
		Status:    "error",
		Hostnames: []string{},
	}

	// Validate IP address
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		result.Error = "Invalid IP address format"
		return result
	}

	// Use system resolver if no DNS server specified
	if dnsServer == "" {
		startTime := time.Now()
		hostnames, err := net.LookupAddr(ipAddress)
		responseTime := time.Since(startTime)
		result.ResponseTime = responseTime

		if err != nil {
			if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "not found") {
				result.Status = "not_found"
			} else {
				result.Error = err.Error()
			}
			return result
		}

		// Trim trailing dots from hostnames
		for i, hostname := range hostnames {
			hostnames[i] = strings.TrimSuffix(hostname, ".")
		}

		result.Hostnames = hostnames
		result.Status = "success"
		return result
	}

	// Use specified DNS server with miekg/dns
	// Create reverse lookup query
	reverseIP, err := dns.ReverseAddr(ipAddress)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create reverse lookup address: %v", err)
		return result
	}

	m := new(dns.Msg)
	m.SetQuestion(reverseIP, dns.TypePTR)
	m.RecursionDesired = true

	// Create a DNS client
	c := new(dns.Client)
	c.Timeout = timeout

	// Ensure DNS server has port
	if !strings.Contains(dnsServer, ":") {
		dnsServer = dnsServer + ":53"
	}

	// Send the query
	r, rtt, err := c.Exchange(m, dnsServer)
	result.ResponseTime = rtt

	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Process response
	if r == nil || r.Rcode != dns.RcodeSuccess {
		if r != nil {
			if r.Rcode == dns.RcodeNameError {
				result.Status = "not_found"
				return result
			}
			result.Error = fmt.Sprintf("DNS query failed with code: %d", r.Rcode)
		} else {
			result.Error = "DNS query returned nil response"
		}
		return result
	}

	// No answer
	if len(r.Answer) == 0 {
		result.Status = "not_found"
		return result
	}

	// Process answers
	for _, answer := range r.Answer {
		if ptr, ok := answer.(*dns.PTR); ok {
			// Trim trailing dots from hostnames
			hostname := strings.TrimSuffix(ptr.Ptr, ".")
			result.Hostnames = append(result.Hostnames, hostname)
		}
	}

	if len(result.Hostnames) > 0 {
		result.Status = "success"
	} else {
		result.Status = "not_found"
	}

	return result
}

package dns_propagation

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// List of public DNS servers to check against
var defaultDNSServers = []struct {
	Name     string
	IP       string
	Country  string
	Provider string
}{
	{"Google DNS", "8.8.8.8", "USA", "Google"},
	{"Google DNS 2", "8.8.4.4", "USA", "Google"},
	{"Cloudflare", "1.1.1.1", "USA", "Cloudflare"},
	{"Cloudflare 2", "1.0.0.1", "USA", "Cloudflare"},
	{"Quad9", "9.9.9.9", "USA", "Quad9"},
	{"OpenDNS", "208.67.222.222", "USA", "OpenDNS"},
	{"OpenDNS 2", "208.67.220.220", "USA", "OpenDNS"},
	{"Level3", "4.2.2.1", "USA", "Level3"},
	{"Comodo Secure DNS", "8.26.56.26", "USA", "Comodo"},
	{"AdGuard DNS", "94.140.14.14", "Cyprus", "AdGuard"},
	{"CleanBrowsing", "185.228.168.168", "Multiple", "CleanBrowsing"},
	{"Yandex DNS", "77.88.8.8", "Russia", "Yandex"},
	{"Verisign", "64.6.64.6", "USA", "Verisign"},
	{"IBM Quad9", "9.9.9.10", "USA", "IBM"},
	{"Hurricane Electric", "74.82.42.42", "USA", "Hurricane Electric"},
}

// DNSResult represents the result of a DNS propagation check for a single server
type DNSResult struct {
	Server   string        `json:"server"`
	Name     string        `json:"name"`
	Country  string        `json:"country"`
	Provider string        `json:"provider"`
	Status   string        `json:"status"` // "propagated", "not_propagated", "error"
	Records  []string      `json:"records"`
	TTL      uint32        `json:"ttl"`
	Time     time.Duration `json:"time"`
	Error    string        `json:"error,omitempty"`
}

// Execute handles the DNS propagation checker plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	domain, ok := params["domain"].(string)
	if !ok || domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	recordType, ok := params["record_type"].(string)
	if !ok || recordType == "" {
		recordType = "A" // Default to A record
	}

	timeoutFloat, ok := params["timeout"].(float64)
	if !ok {
		timeoutFloat = 5 // Default timeout in seconds
	}
	timeout := time.Duration(timeoutFloat) * time.Second

	customServers := ""
	if customServersParam, ok := params["custom_dns_servers"].(string); ok {
		customServers = customServersParam
	}

	// Add custom DNS servers to the list
	servers := make([]struct {
		Name     string
		IP       string
		Country  string
		Provider string
	}, len(defaultDNSServers))
	copy(servers, defaultDNSServers)

	if customServers != "" {
		customList := strings.Split(customServers, ",")
		for _, server := range customList {
			server = strings.TrimSpace(server)
			if server != "" {
				servers = append(servers, struct {
					Name     string
					IP       string
					Country  string
					Provider string
				}{
					Name:     "Custom: " + server,
					IP:       server,
					Country:  "Unknown",
					Provider: "Custom",
				})
			}
		}
	}

	// Initialize result structure
	result := map[string]interface{}{
		"domain":       domain,
		"record_type":  recordType,
		"timestamp":    time.Now().Format(time.RFC3339),
		"servers":      len(servers),
		"timeout":      timeoutFloat,
		"server_count": len(servers),
	}

	// Check DNS propagation across servers
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]DNSResult, 0, len(servers))

	for _, server := range servers {
		wg.Add(1)
		go func(server struct {
			Name     string
			IP       string
			Country  string
			Provider string
		}) {
			defer wg.Done()

			dnsResult := checkDNSServer(domain, recordType, server.IP, server.Name, server.Country, server.Provider, timeout)

			mu.Lock()
			results = append(results, dnsResult)
			mu.Unlock()
		}(server)
	}

	wg.Wait()

	// Count statistics
	propagated := 0
	notPropagated := 0
	errors := 0
	var recordValues = make(map[string]int)

	for _, res := range results {
		switch res.Status {
		case "propagated":
			propagated++
			for _, record := range res.Records {
				recordValues[record]++
			}
		case "not_propagated":
			notPropagated++
		case "error":
			errors++
		}
	}

	// Find the most common record value
	var mostCommonRecord string
	var maxCount int
	for record, count := range recordValues {
		if count > maxCount {
			maxCount = count
			mostCommonRecord = record
		}
	}

	// Add statistics to result
	result["results"] = results
	result["propagated_count"] = propagated
	result["not_propagated_count"] = notPropagated
	result["error_count"] = errors
	result["propagation_percentage"] = float64(propagated) / float64(len(servers)) * 100
	result["most_common_record"] = mostCommonRecord
	result["most_common_record_count"] = maxCount

	// Determine overall status
	if propagated == len(servers) {
		result["status"] = "fully_propagated"
	} else if propagated > 0 {
		result["status"] = "partially_propagated"
	} else {
		result["status"] = "not_propagated"
	}

	return result, nil
}

// checkDNSServer checks DNS propagation on a single server
func checkDNSServer(domain, recordType, serverIP, serverName, country, provider string, timeout time.Duration) DNSResult {
	result := DNSResult{
		Server:   serverIP,
		Name:     serverName,
		Country:  country,
		Provider: provider,
		Status:   "error",
		Records:  []string{},
	}

	// Create a new DNS message
	m := new(dns.Msg)

	// Handle FQDN format
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}

	// Set query parameters
	var dnsType uint16
	switch recordType {
	case "A":
		dnsType = dns.TypeA
	case "AAAA":
		dnsType = dns.TypeAAAA
	case "CNAME":
		dnsType = dns.TypeCNAME
	case "MX":
		dnsType = dns.TypeMX
	case "TXT":
		dnsType = dns.TypeTXT
	case "NS":
		dnsType = dns.TypeNS
	default:
		dnsType = dns.TypeA
	}

	m.SetQuestion(domain, dnsType)
	m.RecursionDesired = true

	// Create a DNS client
	c := new(dns.Client)
	c.Timeout = timeout

	// Send the query
	r, rtt, err := c.Exchange(m, serverIP+":53")
	result.Time = rtt

	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Process response
	if r == nil || r.Rcode != dns.RcodeSuccess {
		if r != nil {
			result.Error = fmt.Sprintf("DNS query failed with code: %d", r.Rcode)
		} else {
			result.Error = "DNS query returned nil response"
		}
		return result
	}

	// No answer
	if len(r.Answer) == 0 {
		result.Status = "not_propagated"
		return result
	}

	// Process answers
	for _, answer := range r.Answer {
		var record string
		var ttl uint32

		switch rr := answer.(type) {
		case *dns.A:
			record = rr.A.String()
			ttl = rr.Hdr.Ttl
		case *dns.AAAA:
			record = rr.AAAA.String()
			ttl = rr.Hdr.Ttl
		case *dns.CNAME:
			record = rr.Target
			ttl = rr.Hdr.Ttl
		case *dns.MX:
			record = fmt.Sprintf("%d %s", rr.Preference, rr.Mx)
			ttl = rr.Hdr.Ttl
		case *dns.TXT:
			record = strings.Join(rr.Txt, " ")
			ttl = rr.Hdr.Ttl
		case *dns.NS:
			record = rr.Ns
			ttl = rr.Hdr.Ttl
		default:
			record = answer.String()
			ttl = answer.Header().Ttl
		}

		if record != "" {
			result.Records = append(result.Records, record)
			result.TTL = ttl
		}
	}

	if len(result.Records) > 0 {
		result.Status = "propagated"
	} else {
		result.Status = "not_propagated"
	}

	return result
}

package dns_lookup

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Execute handles the DNS lookup plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	domain, _ := params["domain"].(string)
	recordType, _ := params["recordType"].(string)

	results := make(map[string][]string)

	// Determine which record types to look up
	var recordTypes []string
	if recordType == "ALL" {
		recordTypes = []string{"A", "AAAA", "MX", "NS", "TXT", "CNAME"}
	} else {
		recordTypes = []string{recordType}
	}

	// Set up DNS config
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	c := new(dns.Client)

	// Function to lookup DNS records
	lookup := func(qtype uint16, typeName string) ([]string, error) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), qtype)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], "53"))
		if err != nil {
			return nil, err
		}

		if r.Rcode != dns.RcodeSuccess {
			return nil, fmt.Errorf("DNS lookup failed with code %d", r.Rcode)
		}

		var records []string
		for _, ans := range r.Answer {
			switch record := ans.(type) {
			case *dns.A:
				records = append(records, record.A.String())
			case *dns.AAAA:
				records = append(records, record.AAAA.String())
			case *dns.MX:
				records = append(records, fmt.Sprintf("%d %s", record.Preference, record.Mx))
			case *dns.NS:
				records = append(records, record.Ns)
			case *dns.TXT:
				records = append(records, record.Txt...)
			case *dns.CNAME:
				records = append(records, record.Target)
			}
		}

		return records, nil
	}

	// Perform lookups
	for _, rt := range recordTypes {
		var qtype uint16
		switch rt {
		case "A":
			qtype = dns.TypeA
		case "AAAA":
			qtype = dns.TypeAAAA
		case "MX":
			qtype = dns.TypeMX
		case "NS":
			qtype = dns.TypeNS
		case "TXT":
			qtype = dns.TypeTXT
		case "CNAME":
			qtype = dns.TypeCNAME
		default:
			continue
		}

		records, err := lookup(qtype, rt)
		if err == nil && len(records) > 0 {
			results[rt] = records
		}
	}

	// Fallback to Go's built-in resolver if dns package fails
	if len(results) == 0 {
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
	}

	return map[string]interface{}{
		"domain":     domain,
		"recordType": recordType,
		"results":    results,
		"timestamp":  time.Now().Format(time.RFC3339),
	}, nil
}

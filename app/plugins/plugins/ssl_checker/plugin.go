package ssl_checker

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Execute handles the SSL/TLS certificate checker plugin execution
func Execute(params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	hostname, ok := params["hostname"].(string)
	if !ok || hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}

	portFloat, ok := params["port"].(float64)
	if !ok {
		portFloat = 443 // Default port
	}
	port := int(portFloat)

	checkChain, ok := params["check_chain"].(bool)
	if !ok {
		checkChain = true // Default to check the chain
	}

	showDetails, ok := params["show_details"].(bool)
	if !ok {
		showDetails = true // Default to show details
	}

	// Initialize result structure
	result := map[string]interface{}{
		"hostname":     hostname,
		"port":         port,
		"check_chain":  checkChain,
		"show_details": showDetails,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	// First try using openssl for more detailed output
	opensslResult, opensslErr := checkWithOpenSSL(hostname, port)
	if opensslErr == nil {
		result["openssl_output"] = opensslResult
	}

	// Always perform our own Go-based check
	certs, issues, err := checkCertificate(hostname, port, checkChain)
	if err != nil {
		result["error"] = err.Error()
		return result, nil
	}

	// Process certificate information
	var certDetails []map[string]interface{}
	for i, cert := range certs {
		certInfo := map[string]interface{}{
			"subject":        cert.Subject.String(),
			"issuer":         cert.Issuer.String(),
			"valid_from":     cert.NotBefore.Format(time.RFC3339),
			"valid_until":    cert.NotAfter.Format(time.RFC3339),
			"days_remaining": math.Floor(time.Until(cert.NotAfter).Hours() / 24),
			"serial_number":  cert.SerialNumber.String(),
		}

		if showDetails {
			// Add more detailed information
			certInfo["version"] = cert.Version
			certInfo["signature_algorithm"] = cert.SignatureAlgorithm.String()

			// Add SANs (Subject Alternative Names)
			var sans []string
			for _, name := range cert.DNSNames {
				sans = append(sans, name)
			}
			for _, ip := range cert.IPAddresses {
				sans = append(sans, ip.String())
			}
			certInfo["subject_alternative_names"] = sans

			// Add key usage
			if cert.KeyUsage != 0 {
				var usages []string
				if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
					usages = append(usages, "Digital Signature")
				}
				if cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
					usages = append(usages, "Content Commitment")
				}
				if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
					usages = append(usages, "Key Encipherment")
				}
				if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
					usages = append(usages, "Data Encipherment")
				}
				if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
					usages = append(usages, "Key Agreement")
				}
				if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
					usages = append(usages, "Certificate Signing")
				}
				if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
					usages = append(usages, "CRL Signing")
				}
				if cert.KeyUsage&x509.KeyUsageEncipherOnly != 0 {
					usages = append(usages, "Encipher Only")
				}
				if cert.KeyUsage&x509.KeyUsageDecipherOnly != 0 {
					usages = append(usages, "Decipher Only")
				}
				certInfo["key_usage"] = usages
			}

			// Add extended key usage
			if len(cert.ExtKeyUsage) > 0 {
				var extUsages []string
				for _, usage := range cert.ExtKeyUsage {
					switch usage {
					case x509.ExtKeyUsageAny:
						extUsages = append(extUsages, "Any")
					case x509.ExtKeyUsageServerAuth:
						extUsages = append(extUsages, "Server Authentication")
					case x509.ExtKeyUsageClientAuth:
						extUsages = append(extUsages, "Client Authentication")
					case x509.ExtKeyUsageCodeSigning:
						extUsages = append(extUsages, "Code Signing")
					case x509.ExtKeyUsageEmailProtection:
						extUsages = append(extUsages, "Email Protection")
					case x509.ExtKeyUsageIPSECEndSystem:
						extUsages = append(extUsages, "IPSEC End System")
					case x509.ExtKeyUsageIPSECTunnel:
						extUsages = append(extUsages, "IPSEC Tunnel")
					case x509.ExtKeyUsageIPSECUser:
						extUsages = append(extUsages, "IPSEC User")
					case x509.ExtKeyUsageTimeStamping:
						extUsages = append(extUsages, "Time Stamping")
					case x509.ExtKeyUsageOCSPSigning:
						extUsages = append(extUsages, "OCSP Signing")
					}
				}
				certInfo["extended_key_usage"] = extUsages
			}
		}

		// Determine if this certificate is for the requested hostname
		if i == 0 {
			certInfo["is_leaf"] = true
			result["expires_soon"] = time.Until(cert.NotAfter) < 30*24*time.Hour
		} else {
			certInfo["is_leaf"] = false
		}

		certDetails = append(certDetails, certInfo)
	}

	result["certificates"] = certDetails
	result["issues"] = issues

	// Determine overall status
	if len(issues) == 0 {
		result["status"] = "valid"
	} else {
		result["status"] = "issues_found"
	}

	return result, nil
}

// checkCertificate connects to the server and retrieves certificate information
func checkCertificate(hostname string, port int, checkChain bool) ([]*x509.Certificate, []string, error) {
	// Connect to the server
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:%d", hostname, port), &tls.Config{
		InsecureSkipVerify: true, // We'll verify manually
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Get the certificate chain
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificates returned")
	}

	// Check for issues
	var issues []string

	// Verify certificate is valid for the hostname
	if err := certs[0].VerifyHostname(hostname); err != nil {
		issues = append(issues, fmt.Sprintf("Certificate not valid for hostname: %s", err.Error()))
	}

	// Check expiration
	now := time.Now()
	if now.Before(certs[0].NotBefore) {
		issues = append(issues, "Certificate is not yet valid")
	}
	if now.After(certs[0].NotAfter) {
		issues = append(issues, "Certificate has expired")
	}
	if time.Until(certs[0].NotAfter) < 30*24*time.Hour {
		issues = append(issues, fmt.Sprintf("Certificate will expire soon (%.0f days remaining)",
			math.Floor(time.Until(certs[0].NotAfter).Hours()/24)))
	}

	// Verify chain if requested
	if checkChain {
		opts := x509.VerifyOptions{
			DNSName: hostname,
			// Use system cert pool for verification
			Intermediates: x509.NewCertPool(),
		}

		// Add intermediates to the pool
		for i := 1; i < len(certs); i++ {
			opts.Intermediates.AddCert(certs[i])
		}

		_, err := certs[0].Verify(opts)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Certificate chain validation failed: %s", err.Error()))
		}
	}

	return certs, issues, nil
}

// checkWithOpenSSL runs openssl to get additional certificate information
func checkWithOpenSSL(hostname string, port int) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Check if openssl is available
	checkCmd := exec.CommandContext(ctx, "which", "openssl")
	if err := checkCmd.Run(); err != nil {
		return nil, fmt.Errorf("openssl not found")
	}

	// Run openssl command to get certificate info
	cmd := exec.CommandContext(ctx, "openssl", "s_client", "-connect", fmt.Sprintf("%s:%d", hostname, port), "-servername", hostname, "-showcerts")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBuffer([]byte("Q\n")) // Send Q to quit after getting the certificate

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("openssl command failed: %w", err)
	}

	// Parse openssl output
	output := stdout.String()

	// Extract the certificate chain
	var certs []string
	var currentCert strings.Builder
	inCert := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "-----BEGIN CERTIFICATE-----") {
			inCert = true
			currentCert.Reset()
			currentCert.WriteString(line)
			currentCert.WriteString("\n")
		} else if inCert {
			currentCert.WriteString(line)
			currentCert.WriteString("\n")

			if strings.Contains(line, "-----END CERTIFICATE-----") {
				inCert = false
				certs = append(certs, currentCert.String())
			}
		}
	}

	// Extract TLS protocol version and cipher info
	var protocol, cipher string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Protocol  :") {
			protocol = strings.TrimSpace(strings.TrimPrefix(line, "Protocol  :"))
		}
		if strings.Contains(line, "Cipher    :") {
			cipher = strings.TrimSpace(strings.TrimPrefix(line, "Cipher    :"))
		}
	}

	// Build result
	result := map[string]interface{}{
		"certificates": certs,
		"protocol":     protocol,
		"cipher":       cipher,
	}

	return result, nil
}

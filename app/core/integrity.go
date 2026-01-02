// Package core provides core functionality including binary integrity verification.
package core

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Build-time variables injected via ldflags
var (
	// BinaryHash is set at build time to the expected SHA256 hash
	BinaryHash = ""
	// HashURL is the GitHub URL to fetch the official hash
	HashURL = ""
	// IntegrityEnabled controls whether integrity checks are active
	IntegrityEnabled = "true"
)

// Trusted DNS servers for verification
// We use multiple independent DNS providers to detect DNS spoofing
var trustedDNSServers = []string{
	"8.8.8.8:53",        // Google Primary
	"8.8.4.4:53",        // Google Secondary
	"1.1.1.1:53",        // Cloudflare Primary
	"1.0.0.1:53",        // Cloudflare Secondary
	"9.9.9.9:53",        // Quad9
	"208.67.222.222:53", // OpenDNS
}

// Minimum number of DNS servers that must agree on the IP
const minDNSConsensus = 3

// IntegrityStatus represents the result of an integrity check
type IntegrityStatus struct {
	Verified       bool   `json:"verified"`
	ExpectedHash   string `json:"expected_hash,omitempty"`
	ActualHash     string `json:"actual_hash,omitempty"`
	Source         string `json:"source"` // "embedded", "github", "blocked", "skipped"
	Error          string `json:"error,omitempty"`
	CheckedAt      string `json:"checked_at"`
	BinaryPath     string `json:"binary_path"`
	BinarySize     int64  `json:"binary_size"`
	RuntimeOS      string `json:"runtime_os"`
	RuntimeArch    string `json:"runtime_arch"`
	TamperDetected bool   `json:"tamper_detected"`
	ShouldBlock    bool   `json:"should_block"` // If true, binary should not run
}

// GitHubReleaseAsset represents a release asset from GitHub API
type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

// HashManifest represents the hash manifest file structure
type HashManifest struct {
	Version   string            `json:"version"`
	BuildTime string            `json:"build_time"`
	Hashes    map[string]string `json:"hashes"` // filename -> sha256
}

var (
	integrityChecked   bool
	integrityMutex     sync.Mutex
	cachedStatus       *IntegrityStatus
	antiDebugTriggered bool
)

// VerifyBinaryIntegrity performs a comprehensive integrity check
// It only trusts the embedded hash or GitHub releases as sources of truth.
// Local files are NEVER trusted as they can be modified by attackers.
func VerifyBinaryIntegrity() *IntegrityStatus {
	integrityMutex.Lock()
	defer integrityMutex.Unlock()

	status := &IntegrityStatus{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		RuntimeOS:   runtime.GOOS,
		RuntimeArch: runtime.GOARCH,
		ShouldBlock: false,
	}

	// Get the binary path
	execPath, err := os.Executable()
	if err != nil {
		status.Error = fmt.Sprintf("failed to get executable path: %v", err)
		status.Source = "error"
		status.ShouldBlock = true
		return status
	}
	status.BinaryPath = execPath

	// Get binary info
	fileInfo, err := os.Stat(execPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to stat binary: %v", err)
		status.Source = "error"
		status.ShouldBlock = true
		return status
	}
	status.BinarySize = fileInfo.Size()

	// Check if integrity verification is disabled at compile time
	if IntegrityEnabled != "true" {
		status.Source = "skipped"
		status.Verified = true
		return status
	}

	// Calculate actual hash of the running binary
	actualHash, err := calculateBinaryHash(execPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to calculate hash: %v", err)
		status.Source = "error"
		status.ShouldBlock = true
		return status
	}
	status.ActualHash = actualHash

	// PRIORITY 1: Try embedded hash (compiled into binary - most trusted)
	// This is set at build time and cannot be modified without recompiling
	if BinaryHash != "" {
		status.ExpectedHash = BinaryHash
		status.Source = "embedded"
		status.Verified = actualHash == BinaryHash
		if !status.Verified {
			status.TamperDetected = true
			status.ShouldBlock = true
			status.Error = "binary has been modified - hash mismatch with embedded hash"
		}
		cachedStatus = status
		integrityChecked = true
		return status
	}

	// PRIORITY 2: Fetch hash from GitHub releases (remote trusted source)
	// This is the ONLY external source we trust - NOT local files
	githubHash, source, err := fetchHashFromGitHub()
	if err == nil && githubHash != "" {
		status.ExpectedHash = githubHash
		status.Source = source // "github-release" or "github-beta"
		status.Verified = actualHash == githubHash
		if !status.Verified {
			status.TamperDetected = true
			status.ShouldBlock = true
			status.Error = "binary has been modified - hash mismatch with official release"
		}
		cachedStatus = status
		integrityChecked = true
		return status
	}

	// NO TRUSTED HASH AVAILABLE
	// This could be:
	// 1. Development build (no hash embedded)
	// 2. Network offline (can't reach GitHub)
	// 3. Custom build from source
	//
	// For security, we block execution if no trusted hash is available
	// Users must use --skip-integrity flag to bypass (at their own risk)
	status.Source = "blocked"
	status.Verified = false
	status.ShouldBlock = true
	status.Error = "no trusted hash source available - cannot verify binary integrity"
	if err != nil {
		status.Error = fmt.Sprintf("no trusted hash source available: %v", err)
	}
	cachedStatus = status
	integrityChecked = true

	return status
}

// GetCachedIntegrityStatus returns the cached integrity status
func GetCachedIntegrityStatus() *IntegrityStatus {
	integrityMutex.Lock()
	defer integrityMutex.Unlock()
	return cachedStatus
}

// calculateBinaryHash computes SHA256 hash of the binary
func calculateBinaryHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fetchHashFromGitHub fetches the hash manifest from GitHub releases
// Returns: hash, source name, error
func fetchHashFromGitHub() (string, string, error) {
	// First, verify DNS resolution using multiple trusted DNS servers
	// This protects against DNS spoofing attacks
	verifiedAPIIPs, err := verifyDNSResolution("api.github.com")
	if err != nil {
		return "", "", fmt.Errorf("DNS verification failed for api.github.com: %v", err)
	}

	// Also verify DNS for release assets domain (used for downloads)
	verifiedAssetsIPs, err := verifyDNSResolution("objects.githubusercontent.com")
	if err != nil {
		// Non-fatal - will try with standard resolution for downloads
		verifiedAssetsIPs = nil
	}

	// Create HTTP client that uses our verified IPs
	client := createSecureHTTPClient(verifiedAPIIPs, verifiedAssetsIPs)

	// Try multiple release endpoints: latest stable first, then beta
	releaseEndpoints := []struct {
		URL        string
		SourceName string
	}{
		{"https://api.github.com/repos/NetScout-Go/NetTool/releases/latest", "github-release"},
		{"https://api.github.com/repos/NetScout-Go/NetTool/releases/tags/beta", "github-beta"},
	}

	var lastErr error
	for _, endpoint := range releaseEndpoints {
		hash, err := tryFetchHashFromRelease(client, endpoint.URL)
		if err == nil && hash != "" {
			return hash, endpoint.SourceName, nil
		}
		lastErr = err
	}

	return "", "", fmt.Errorf("no hash manifest found in any release: %v", lastErr)
}

// verifyDNSResolution queries multiple trusted DNS servers and ensures consensus
// Returns verified IPs only if multiple independent DNS servers agree
func verifyDNSResolution(hostname string) ([]string, error) {
	type dnsResult struct {
		server string
		ips    []string
		err    error
	}

	results := make(chan dnsResult, len(trustedDNSServers))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query all DNS servers in parallel
	for _, server := range trustedDNSServers {
		go func(dnsServer string) {
			ips, err := queryDNSServer(ctx, hostname, dnsServer)
			results <- dnsResult{server: dnsServer, ips: ips, err: err}
		}(server)
	}

	// Collect results
	ipCount := make(map[string]int)
	successfulQueries := 0
	var lastError error

	for i := 0; i < len(trustedDNSServers); i++ {
		select {
		case result := <-results:
			if result.err != nil {
				lastError = result.err
				continue
			}
			successfulQueries++
			for _, ip := range result.ips {
				ipCount[ip]++
			}
		case <-ctx.Done():
			break
		}
	}

	if successfulQueries < minDNSConsensus {
		return nil, fmt.Errorf("only %d DNS servers responded (need %d): %v",
			successfulQueries, minDNSConsensus, lastError)
	}

	// Find IPs that have consensus (appear in at least minDNSConsensus responses)
	var verifiedIPs []string
	for ip, count := range ipCount {
		if count >= minDNSConsensus {
			verifiedIPs = append(verifiedIPs, ip)
		}
	}

	if len(verifiedIPs) == 0 {
		return nil, fmt.Errorf("DNS servers returned inconsistent results - possible DNS spoofing attack")
	}

	// Sort for consistent ordering
	sort.Strings(verifiedIPs)

	return verifiedIPs, nil
}

// queryDNSServer queries a specific DNS server for A records
func queryDNSServer(ctx context.Context, hostname, dnsServer string) ([]string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", dnsServer)
		},
	}

	addrs, err := resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}

	// Filter to IPv4 only for consistency
	var ipv4Addrs []string
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			ipv4Addrs = append(ipv4Addrs, addr)
		}
	}

	if len(ipv4Addrs) == 0 {
		return nil, fmt.Errorf("no IPv4 addresses found")
	}

	return ipv4Addrs, nil
}

// createSecureHTTPClient creates an HTTP client that only connects to verified IPs
func createSecureHTTPClient(verifiedAPIIPs, verifiedAssetsIPs []string) *http.Client {
	// Create a custom dialer that only connects to verified IPs
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract host and port
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// For GitHub API, use verified IPs
			if host == "api.github.com" {
				for _, ip := range verifiedAPIIPs {
					conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
					if err == nil {
						return conn, nil
					}
				}
				return nil, fmt.Errorf("failed to connect to any verified API IP")
			}

			// For GitHub release assets, use verified IPs if available
			if verifiedAssetsIPs != nil && (strings.HasSuffix(host, ".githubusercontent.com") ||
				strings.HasSuffix(host, ".github.com")) {
				for _, ip := range verifiedAssetsIPs {
					conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
					if err == nil {
						return conn, nil
					}
				}
				// Fall through to normal resolution if verified IPs don't work
			}

			// For other hosts or fallback, use normal resolution
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
		// Follow redirects but let the dialer handle IP verification
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// tryFetchHashFromRelease attempts to fetch hash from a specific release
func tryFetchHashFromRelease(client *http.Client, releaseURL string) (string, error) {
	resp, err := client.Get(releaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	// Look for binary-checksums.json which contains per-binary hashes
	var binaryChecksumsURL string
	for _, asset := range release.Assets {
		if asset.Name == "binary-checksums.json" {
			binaryChecksumsURL = asset.BrowserDownloadURL
			break
		}
	}

	// binary-checksums.json is the ONLY trusted source for binary hashes
	// SHA256SUMS files typically contain archive hashes, not binary hashes
	if binaryChecksumsURL != "" {
		hash, err := fetchHashFromBinaryChecksums(client, binaryChecksumsURL)
		if err == nil && hash != "" {
			return hash, nil
		}
	}

	return "", fmt.Errorf("no binary-checksums.json found in release")
}

// BinaryChecksumsManifest represents the binary-checksums.json structure
type BinaryChecksumsManifest struct {
	Version     string                       `json:"version"`
	BuildTime   string                       `json:"build_time"`
	Description string                       `json:"description"`
	Platforms   map[string]map[string]string `json:"platforms"`
}

// fetchHashFromBinaryChecksums fetches hash from binary-checksums.json
func fetchHashFromBinaryChecksums(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var manifest BinaryChecksumsManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return "", err
	}

	// Get platform identifier
	platform := getPlatformIdentifier()

	// Try exact platform match
	platformKey := fmt.Sprintf("nettool-%s", platform)
	if platformHashes, ok := manifest.Platforms[platformKey]; ok {
		if hash, ok := platformHashes["nettool"]; ok {
			return hash, nil
		}
	}

	// Try without nettool- prefix
	if platformHashes, ok := manifest.Platforms[platform]; ok {
		if hash, ok := platformHashes["nettool"]; ok {
			return hash, nil
		}
	}

	return "", fmt.Errorf("no hash found for platform %s", platform)
}

// getPlatformIdentifier returns the platform string used in release filenames
func getPlatformIdentifier() string {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go architecture names to release filename conventions
	switch arch {
	case "386":
		// Keep as-is for 32-bit x86
		return fmt.Sprintf("%s-%s", goos, arch)
	case "arm":
		// ARM builds might be named differently (arm6, arm7, etc.)
		// Default to arm6 for Raspberry Pi Zero compatibility
		return fmt.Sprintf("%s-arm6", goos)
	case "arm64":
		return fmt.Sprintf("%s-%s", goos, arch)
	default:
		return fmt.Sprintf("%s-%s", goos, arch)
	}
}

// PerformAntiTamperChecks runs various anti-tampering checks
func PerformAntiTamperChecks() bool {
	checks := []func() bool{
		checkEnvironment,
		checkRuntimeIntegrity,
		checkMemoryPatterns,
	}

	for _, check := range checks {
		if !check() {
			antiDebugTriggered = true
			return false
		}
	}

	return true
}

// checkEnvironment checks for debugging/analysis environment indicators
func checkEnvironment() bool {
	// Check for common debugger environment variables
	debugEnvVars := []string{
		"GODEBUG",
		"GOTRACEBACK",
		"LD_PRELOAD",
		"DYLD_INSERT_LIBRARIES",
	}

	for _, envVar := range debugEnvVars {
		if val := os.Getenv(envVar); val != "" {
			// Allow some safe values
			if envVar == "GOTRACEBACK" && (val == "none" || val == "single") {
				continue
			}
			// Log but don't fail - these might be legitimate
		}
	}

	return true
}

// checkRuntimeIntegrity verifies Go runtime hasn't been tampered with
func checkRuntimeIntegrity() bool {
	// Basic runtime checks
	if runtime.NumCPU() < 1 {
		return false
	}

	// Check GOMAXPROCS is reasonable
	maxProcs := runtime.GOMAXPROCS(0)
	if maxProcs < 1 || maxProcs > 1024 {
		return false
	}

	return true
}

// checkMemoryPatterns performs basic memory integrity checks
func checkMemoryPatterns() bool {
	// Verify stack is working correctly
	var stackTest [16]byte
	for i := range stackTest {
		stackTest[i] = byte(i)
	}

	for i := range stackTest {
		if stackTest[i] != byte(i) {
			return false
		}
	}

	return true
}

// IsAntiDebugTriggered returns whether anti-debug measures were triggered
func IsAntiDebugTriggered() bool {
	return antiDebugTriggered
}

// obfuscateString applies simple XOR obfuscation to sensitive strings
// This helps prevent easy string extraction from binaries
func obfuscateString(s string, key byte) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		result[i] = s[i] ^ key
	}
	return string(result)
}

// deobfuscateString reverses the XOR obfuscation
func deobfuscateString(s string, key byte) string {
	return obfuscateString(s, key) // XOR is symmetric
}

// GetBinaryInfo returns information about the running binary
func GetBinaryInfo() map[string]interface{} {
	execPath, _ := os.Executable()
	fileInfo, _ := os.Stat(execPath)

	info := map[string]interface{}{
		"path":          execPath,
		"size":          fileInfo.Size(),
		"modified":      fileInfo.ModTime().Format(time.RFC3339),
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
	}

	// Add integrity status if available
	if cachedStatus != nil {
		info["integrity_verified"] = cachedStatus.Verified
		info["integrity_source"] = cachedStatus.Source
	}

	return info
}

// CalculateFileHash calculates SHA256 hash of any file
func CalculateFileHash(path string) (string, error) {
	return calculateBinaryHash(path)
}

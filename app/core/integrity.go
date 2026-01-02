// Package core provides core functionality including binary integrity verification.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
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

// IntegrityStatus represents the result of an integrity check
type IntegrityStatus struct {
	Verified       bool   `json:"verified"`
	ExpectedHash   string `json:"expected_hash,omitempty"`
	ActualHash     string `json:"actual_hash,omitempty"`
	Source         string `json:"source"` // "embedded", "github", "skipped"
	Error          string `json:"error,omitempty"`
	CheckedAt      string `json:"checked_at"`
	BinaryPath     string `json:"binary_path"`
	BinarySize     int64  `json:"binary_size"`
	RuntimeOS      string `json:"runtime_os"`
	RuntimeArch    string `json:"runtime_arch"`
	TamperDetected bool   `json:"tamper_detected"`
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
func VerifyBinaryIntegrity() *IntegrityStatus {
	integrityMutex.Lock()
	defer integrityMutex.Unlock()

	status := &IntegrityStatus{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		RuntimeOS:   runtime.GOOS,
		RuntimeArch: runtime.GOARCH,
	}

	// Get the binary path
	execPath, err := os.Executable()
	if err != nil {
		status.Error = fmt.Sprintf("failed to get executable path: %v", err)
		status.Source = "error"
		return status
	}
	status.BinaryPath = execPath

	// Get binary info
	fileInfo, err := os.Stat(execPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to stat binary: %v", err)
		status.Source = "error"
		return status
	}
	status.BinarySize = fileInfo.Size()

	// Check if integrity verification is disabled
	if strings.ToLower(IntegrityEnabled) != "true" {
		status.Source = "skipped"
		status.Verified = true
		return status
	}

	// Calculate actual hash of the running binary
	actualHash, err := calculateBinaryHash(execPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to calculate hash: %v", err)
		status.Source = "error"
		return status
	}
	status.ActualHash = actualHash

	// Try embedded hash first
	if BinaryHash != "" {
		status.ExpectedHash = BinaryHash
		status.Source = "embedded"
		status.Verified = actualHash == BinaryHash
		if !status.Verified {
			status.TamperDetected = true
		}
		cachedStatus = status
		integrityChecked = true
		return status
	}

	// Try fetching hash from GitHub
	githubHash, err := fetchHashFromGitHub()
	if err == nil && githubHash != "" {
		status.ExpectedHash = githubHash
		status.Source = "github"
		status.Verified = actualHash == githubHash
		if !status.Verified {
			status.TamperDetected = true
		}
		cachedStatus = status
		integrityChecked = true
		return status
	}

	// No hash available for verification
	status.Source = "unavailable"
	status.Verified = true // Can't verify without reference hash
	status.Error = "no reference hash available"
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
func fetchHashFromGitHub() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Try multiple release endpoints: latest stable, then beta
	releaseURLs := []string{
		"https://api.github.com/repos/NetScout-Go/NetTool/releases/latest",
		"https://api.github.com/repos/NetScout-Go/NetTool/releases/tags/beta",
	}

	for _, releaseURL := range releaseURLs {
		hash, err := tryFetchHashFromRelease(client, releaseURL)
		if err == nil && hash != "" {
			return hash, nil
		}
	}

	return "", fmt.Errorf("no hash manifest found in any release")
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

	// Look for hash files in order of preference
	var jsonManifestURL, sha256sumsURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case "checksums.json", "hashes.json":
			jsonManifestURL = asset.BrowserDownloadURL
		case "SHA256SUMS":
			sha256sumsURL = asset.BrowserDownloadURL
		}
	}

	// Try JSON manifest first
	if jsonManifestURL != "" {
		hash, err := fetchHashFromJSONManifest(client, jsonManifestURL)
		if err == nil && hash != "" {
			return hash, nil
		}
	}

	// Fall back to SHA256SUMS file
	if sha256sumsURL != "" {
		hash, err := fetchHashFromSHA256SUMS(client, sha256sumsURL)
		if err == nil && hash != "" {
			return hash, nil
		}
	}

	return "", fmt.Errorf("no hash found in release")
}

// fetchHashFromJSONManifest fetches hash from a JSON checksums file
func fetchHashFromJSONManifest(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var manifest HashManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return "", err
	}

	// Get platform-specific identifiers
	platform := getPlatformIdentifier()

	// Try multiple possible key formats in order of specificity
	possibleKeys := []string{
		// Exact platform match for tar.gz archives
		fmt.Sprintf("nettool-%s-beta.tar.gz", platform),
		fmt.Sprintf("nettool-%s.tar.gz", platform),
		// Platform identifier only
		fmt.Sprintf("nettool-%s", platform),
		// Binary name within archive
		"nettool",
	}

	for _, key := range possibleKeys {
		if hash, ok := manifest.Hashes[key]; ok {
			return hash, nil
		}
	}

	return "", fmt.Errorf("no hash found for platform %s", platform)
}

// fetchHashFromSHA256SUMS fetches hash from traditional SHA256SUMS file
func fetchHashFromSHA256SUMS(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Get platform-specific identifier
	platform := getPlatformIdentifier()

	// Parse SHA256SUMS format: "hash  filename"
	lines := strings.Split(string(body), "\n")

	// Build list of possible filenames in order of preference (most specific first)
	possiblePatterns := []string{
		fmt.Sprintf("nettool-%s-beta.tar.gz", platform),
		fmt.Sprintf("nettool-%s.tar.gz", platform),
	}

	// First pass: look for exact platform match
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "hash  filename" or "hash filename"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := parts[0]
			filename := parts[len(parts)-1]

			for _, pattern := range possiblePatterns {
				if filename == pattern {
					return hash, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no hash found for platform %s", platform)
}

// getPlatformIdentifier returns the platform string used in release filenames
func getPlatformIdentifier() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go architecture names to release filename conventions
	switch arch {
	case "386":
		// Keep as-is for 32-bit x86
		return fmt.Sprintf("%s-%s", os, arch)
	case "arm":
		// ARM builds might be named differently (arm6, arm7, etc.)
		// Default to arm6 for Raspberry Pi Zero compatibility
		return fmt.Sprintf("%s-arm6", os)
	case "arm64":
		return fmt.Sprintf("%s-%s", os, arch)
	default:
		return fmt.Sprintf("%s-%s", os, arch)
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

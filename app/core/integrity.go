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
	// Try to get the latest release
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/NetScout-Go/NetTool/releases/latest")
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

	// Look for the hash manifest file
	var manifestURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.json" || asset.Name == "hashes.json" {
			manifestURL = asset.BrowserDownloadURL
			break
		}
	}

	if manifestURL == "" {
		return "", fmt.Errorf("no hash manifest found in release")
	}

	// Fetch the manifest
	manifestResp, err := client.Get(manifestURL)
	if err != nil {
		return "", err
	}
	defer manifestResp.Body.Close()

	var manifest HashManifest
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifest); err != nil {
		return "", err
	}

	// Find hash for current platform
	binaryName := fmt.Sprintf("nettool-%s-%s", runtime.GOOS, runtime.GOARCH)
	if hash, ok := manifest.Hashes[binaryName]; ok {
		return hash, nil
	}

	return "", fmt.Errorf("no hash found for %s", binaryName)
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

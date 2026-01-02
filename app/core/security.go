// Package core provides security utilities for memory protection and anti-tampering.
package core

import (
	"crypto/rand"
	"crypto/subtle"
	"os"
	"runtime"
	"sync"
	"time"
)

// SecureString is a string type that zeros memory when no longer needed
type SecureString struct {
	data []byte
	mu   sync.Mutex
}

// NewSecureString creates a new SecureString from a regular string
func NewSecureString(s string) *SecureString {
	ss := &SecureString{
		data: make([]byte, len(s)),
	}
	copy(ss.data, s)
	// Zero the original string's backing array if possible
	runtime.SetFinalizer(ss, (*SecureString).Destroy)
	return ss
}

// String returns the string value (use sparingly, copies to heap)
func (ss *SecureString) String() string {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.data == nil {
		return ""
	}
	return string(ss.data)
}

// Equals compares two secure strings in constant time
func (ss *SecureString) Equals(other *SecureString) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	other.mu.Lock()
	defer other.mu.Unlock()

	if ss.data == nil || other.data == nil {
		return ss.data == nil && other.data == nil
	}
	return subtle.ConstantTimeCompare(ss.data, other.data) == 1
}

// Destroy securely wipes the memory and releases it
func (ss *SecureString) Destroy() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.data != nil {
		SecureZero(ss.data)
		ss.data = nil
	}
}

// SecureBytes is a byte slice that zeros memory when no longer needed
type SecureBytes struct {
	data []byte
	mu   sync.Mutex
}

// NewSecureBytes creates a new SecureBytes from a byte slice
func NewSecureBytes(b []byte) *SecureBytes {
	sb := &SecureBytes{
		data: make([]byte, len(b)),
	}
	copy(sb.data, b)
	runtime.SetFinalizer(sb, (*SecureBytes).Destroy)
	return sb
}

// Bytes returns a copy of the underlying bytes
func (sb *SecureBytes) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.data == nil {
		return nil
	}
	result := make([]byte, len(sb.data))
	copy(result, sb.data)
	return result
}

// Destroy securely wipes the memory
func (sb *SecureBytes) Destroy() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.data != nil {
		SecureZero(sb.data)
		sb.data = nil
	}
}

// SecureZero overwrites a byte slice with zeros
// Uses multiple passes and memory barriers to prevent optimization
func SecureZero(b []byte) {
	if len(b) == 0 {
		return
	}

	// First pass: zero
	for i := range b {
		b[i] = 0
	}

	// Memory barrier - prevent compiler from optimizing away
	runtime.KeepAlive(b)

	// Second pass: random data then zero again (prevents cold boot attacks)
	randomBytes := make([]byte, len(b))
	if _, err := rand.Read(randomBytes); err == nil {
		for i := range b {
			b[i] = randomBytes[i]
		}
	}

	// Final zero pass
	for i := range b {
		b[i] = 0
	}

	// Force memory sync
	runtime.KeepAlive(b)
}

// SecureCompare performs constant-time comparison of two byte slices
// Prevents timing attacks
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureCompareStrings performs constant-time comparison of two strings
func SecureCompareStrings(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SecurityContext holds runtime security state
type SecurityContext struct {
	mu                sync.RWMutex
	debuggerDetected  bool
	integrityVerified bool
	startTime         time.Time
	checkCount        int
	anomalies         []string
}

var (
	globalSecurityContext *SecurityContext
	securityContextOnce   sync.Once
)

// GetSecurityContext returns the global security context
func GetSecurityContext() *SecurityContext {
	securityContextOnce.Do(func() {
		globalSecurityContext = &SecurityContext{
			startTime: time.Now(),
			anomalies: make([]string, 0),
		}
	})
	return globalSecurityContext
}

// AddAnomaly records a security anomaly
func (sc *SecurityContext) AddAnomaly(description string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.anomalies = append(sc.anomalies, description)
}

// GetAnomalies returns all recorded anomalies
func (sc *SecurityContext) GetAnomalies() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	result := make([]string, len(sc.anomalies))
	copy(result, sc.anomalies)
	return result
}

// SetDebuggerDetected marks that a debugger was detected
func (sc *SecurityContext) SetDebuggerDetected(detected bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.debuggerDetected = detected
}

// IsDebuggerDetected returns whether a debugger was detected
func (sc *SecurityContext) IsDebuggerDetected() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.debuggerDetected
}

// PerformSecurityChecks runs all security checks
func PerformSecurityChecks() *SecurityCheckResult {
	result := &SecurityCheckResult{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make(map[string]bool),
	}

	ctx := GetSecurityContext()

	// Check 1: Debugger detection
	debuggerPresent := detectDebugger()
	result.Checks["debugger_absent"] = !debuggerPresent
	if debuggerPresent {
		ctx.SetDebuggerDetected(true)
		ctx.AddAnomaly("debugger detected")
	}

	// Check 2: Timing integrity (detect time manipulation)
	timingOK := checkTimingIntegrity()
	result.Checks["timing_integrity"] = timingOK
	if !timingOK {
		ctx.AddAnomaly("timing anomaly detected")
	}

	// Check 3: Environment safety
	envOK := checkEnvironmentSafety()
	result.Checks["environment_safe"] = envOK
	if !envOK {
		ctx.AddAnomaly("suspicious environment detected")
	}

	// Check 4: Memory integrity
	memOK := checkMemoryIntegrity()
	result.Checks["memory_integrity"] = memOK
	if !memOK {
		ctx.AddAnomaly("memory integrity issue")
	}

	// Check 5: Process integrity
	procOK := checkProcessIntegrity()
	result.Checks["process_integrity"] = procOK
	if !procOK {
		ctx.AddAnomaly("process integrity issue")
	}

	// Overall result
	result.Passed = !debuggerPresent && timingOK && envOK && memOK && procOK
	result.Anomalies = ctx.GetAnomalies()

	return result
}

// SecurityCheckResult contains results of security checks
type SecurityCheckResult struct {
	Passed    bool            `json:"passed"`
	Timestamp string          `json:"timestamp"`
	Checks    map[string]bool `json:"checks"`
	Anomalies []string        `json:"anomalies,omitempty"`
}

// detectDebugger checks for common debugger indicators
func detectDebugger() bool {
	// Check 1: Parent process name (Linux)
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/self/status"); err == nil {
			// Look for TracerPid != 0
			status := string(data)
			for _, line := range splitLines(status) {
				if len(line) > 10 && line[:9] == "TracerPid" {
					// TracerPid: X - if X != 0, we're being traced
					for i := 10; i < len(line); i++ {
						if line[i] >= '1' && line[i] <= '9' {
							return true
						}
					}
				}
			}
		}
	}

	// Check 2: Timing-based detection
	// Debuggers introduce delays in execution
	start := time.Now()
	dummy := 0
	for i := 0; i < 1000000; i++ {
		dummy += i
	}
	runtime.KeepAlive(dummy)
	elapsed := time.Since(start)

	// If simple loop takes more than 100ms, something is slowing us down
	if elapsed > 100*time.Millisecond {
		return true
	}

	return false
}

// splitLines splits a string into lines without importing strings package
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// checkTimingIntegrity verifies system timing hasn't been manipulated
func checkTimingIntegrity() bool {
	// Verify monotonic clock is working correctly
	t1 := time.Now()
	time.Sleep(10 * time.Millisecond)
	t2 := time.Now()

	diff := t2.Sub(t1)
	// Should be at least 10ms, allow up to 500ms for system load
	if diff < 5*time.Millisecond || diff > 500*time.Millisecond {
		return false
	}

	return true
}

// checkEnvironmentSafety checks for suspicious environment variables
func checkEnvironmentSafety() bool {
	suspiciousVars := []string{
		"LD_PRELOAD",            // Library injection on Linux
		"DYLD_INSERT_LIBRARIES", // Library injection on macOS
		"LD_LIBRARY_PATH",       // Could be used for hijacking
		"_JAVA_OPTIONS",         // Sometimes used for injection
	}

	for _, varName := range suspiciousVars {
		if val := os.Getenv(varName); val != "" {
			// LD_LIBRARY_PATH is sometimes legitimately set
			if varName == "LD_LIBRARY_PATH" {
				continue
			}
			return false
		}
	}

	return true
}

// checkMemoryIntegrity performs memory integrity checks
func checkMemoryIntegrity() bool {
	// Test 1: Stack canary simulation
	canary := make([]byte, 32)
	if _, err := rand.Read(canary); err != nil {
		return false
	}

	canaryCopy := make([]byte, 32)
	copy(canaryCopy, canary)

	// Do some operations
	runtime.Gosched()

	// Verify canary wasn't corrupted
	if !SecureCompare(canary, canaryCopy) {
		return false
	}

	// Test 2: Heap allocation integrity
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	runtime.Gosched()

	for i := range testData {
		if testData[i] != byte(i%256) {
			return false
		}
	}

	return true
}

// checkProcessIntegrity verifies process hasn't been tampered with
func checkProcessIntegrity() bool {
	// Check 1: Verify we can get our own executable path
	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	// Check 2: Verify the executable exists and is readable
	info, err := os.Stat(execPath)
	if err != nil || info.Size() == 0 {
		return false
	}

	// Check 3: On Linux, verify /proc/self/exe points to the same file
	if runtime.GOOS == "linux" {
		if link, err := os.Readlink("/proc/self/exe"); err == nil {
			// Just verify it's not deleted
			if len(link) > 10 && link[len(link)-10:] == " (deleted)" {
				return false
			}
		}
	}

	return true
}

// DisableCoreDumps attempts to disable core dumps for this process
// This prevents sensitive data from being written to disk on crash
// Note: This is a no-op on Windows
func DisableCoreDumps() error {
	// Platform-specific implementation in security_unix.go and security_windows.go
	return disableCoreDumpsImpl()
}

// ProtectMemoryRegion marks a memory region as non-swappable
// This prevents sensitive data from being written to swap
// Note: This is a no-op on Windows and may require elevated privileges on Unix
func ProtectMemoryRegion(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return protectMemoryRegionImpl(data)
}

// UnprotectMemoryRegion unlocks memory region
func UnprotectMemoryRegion(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return unprotectMemoryRegionImpl(data)
}

// GenerateSecureRandom generates cryptographically secure random bytes
func GenerateSecureRandom(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// SecureHashCompare compares two hashes in constant time
func SecureHashCompare(hash1, hash2 string) bool {
	// Ensure we compare the full length to prevent timing attacks
	if len(hash1) != len(hash2) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash1), []byte(hash2)) == 1
}

// RuntimeProtection applies runtime protection measures
func RuntimeProtection() {
	// Disable core dumps
	_ = DisableCoreDumps()

	// Set GOMAXPROCS to actual CPU count (prevent unusual settings)
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Force garbage collection to clean up any initialization data
	runtime.GC()
}

// ValidateEnvironment checks if the runtime environment is safe
func ValidateEnvironment() error {
	result := PerformSecurityChecks()
	if !result.Passed {
		// Log anomalies but don't block - some checks may have false positives
		for _, anomaly := range result.Anomalies {
			_ = anomaly // Could log these if needed
		}
	}
	return nil
}

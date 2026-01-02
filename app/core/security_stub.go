//go:build windows
// +build windows

package core

// Stub implementations for Windows (development only)
// Real security features only work on Unix

func disableCoreDumpsImpl() error              { return nil }
func protectMemoryRegionImpl(_ []byte) error   { return nil }
func unprotectMemoryRegionImpl(_ []byte) error { return nil }

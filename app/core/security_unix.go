//go:build !windows
// +build !windows

package core

import (
	"syscall"
	"unsafe"
)

// disableCoreDumpsImpl disables core dumps on Unix systems
func disableCoreDumpsImpl() error {
	var rLimit syscall.Rlimit
	rLimit.Cur = 0
	rLimit.Max = 0

	// Try to set core dump limit to 0
	// This may fail without root privileges, but that's okay
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit)
	return nil
}

// protectMemoryRegionImpl locks memory on Unix systems to prevent swapping
func protectMemoryRegionImpl(data []byte) error {
	// mlock the memory to prevent swapping
	// This requires CAP_IPC_LOCK or running as root
	addr := unsafe.Pointer(&data[0])
	_, _, errno := syscall.Syscall(
		syscall.SYS_MLOCK,
		uintptr(addr),
		uintptr(len(data)),
		0,
	)
	if errno != 0 {
		// Silently fail - may not have permission
		return nil
	}
	return nil
}

// unprotectMemoryRegionImpl unlocks memory region on Unix systems
func unprotectMemoryRegionImpl(data []byte) error {
	addr := unsafe.Pointer(&data[0])
	_, _, _ = syscall.Syscall(
		syscall.SYS_MUNLOCK,
		uintptr(addr),
		uintptr(len(data)),
		0,
	)
	return nil
}

//go:build !linux && !darwin

package main

import "os"

// peakRSSBytes has no portable implementation outside Linux/Darwin via the
// standard library's syscall.Rusage. Callers must fall back to in-process
// runtime.MemStats sampling on these platforms; this returns 0 to signal
// "unavailable" rather than a misleading value.
func peakRSSBytes(ps *os.ProcessState) int64 {
	return 0
}

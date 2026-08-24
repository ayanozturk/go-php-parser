//go:build linux || darwin

package main

import (
	"os"
	"runtime"
	"syscall"
)

// peakRSSBytes extracts the child process's peak resident set size from the
// OS-reported rusage after Wait(). Linux reports ru_maxrss in kilobytes;
// Darwin reports it in bytes. Both are normalized to bytes here so report
// consumers never need to know which platform produced the number.
func peakRSSBytes(ps *os.ProcessState) int64 {
	if ps == nil {
		return 0
	}
	usage, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil {
		return 0
	}
	if runtime.GOOS == "linux" {
		return int64(usage.Maxrss) * 1024
	}
	return int64(usage.Maxrss)
}

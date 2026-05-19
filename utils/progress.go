package utils

import (
	"fmt"
	"os"
	"time"
)

// ProgressBar provides a simple progress bar for terminal output.
type ProgressBar struct {
	total     int
	label     string
	isTTY     bool
	startTime time.Time
}

// NewProgressBar creates a new ProgressBar instance.
func NewProgressBar(total int, label string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		label:     label,
		isTTY:     isatty(os.Stdout.Fd()),
		startTime: time.Now(),
	}
}

// Print prints the progress bar with elapsed time, files/sec, and ETA if output is a terminal.
func (pb *ProgressBar) Print(current int) {
	if !pb.isTTY || pb.total <= 0 {
		return
	}
	if current < 0 {
		current = 0
	}
	if current > pb.total {
		current = pb.total
	}
	percent := float64(current) / float64(pb.total) * 100
	elapsed := time.Since(pb.startTime).Seconds()

	var etaStr string
	var rateStr string
	if elapsed > 0 && current > 0 {
		rate := float64(current) / elapsed
		rateStr = fmt.Sprintf(" %.0f files/s", rate)
		if current < pb.total {
			remaining := float64(pb.total-current) / rate
			etaStr = fmt.Sprintf(" ETA %ds", int(remaining))
		}
	}

	fmt.Printf("\r%s: %3.0f%% [%d/%d] %.1fs%s%s   ",
		pb.label, percent, current, pb.total, elapsed, rateStr, etaStr)
	if current == pb.total {
		fmt.Println()
	}
}

// isatty returns true if the given file descriptor is a terminal.
func isatty(fd uintptr) bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

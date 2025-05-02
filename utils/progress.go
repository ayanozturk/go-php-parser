package utils

import (
	"fmt"
	"os"
)

// ProgressBar provides a simple progress bar for terminal output.
type ProgressBar struct {
	total int
	label string
	isTTY bool
}

// NewProgressBar creates a new ProgressBar instance.
func NewProgressBar(total int, label string) *ProgressBar {
	return &ProgressBar{
		total: total,
		label: label,
		isTTY: isatty(os.Stdout.Fd()),
	}
}

// Print prints the progress bar if output is a terminal.
func (pb *ProgressBar) Print(current int) {
	if !pb.isTTY || pb.total == 0 {
		return
	}
	percent := float64(current) / float64(pb.total) * 100
	fmt.Printf("\r%s: %3.0f%% [%d/%d]", pb.label, percent, current, pb.total)
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

package utils

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProgressBar_Print_TTY(t *testing.T) {
	pb := &ProgressBar{total: 10, label: "Test", isTTY: true, startTime: time.Now()}

	output := captureStdout(t, func() {
		for i := 1; i <= 10; i++ {
			pb.Print(i)
		}
	})

	if len(output) == 0 {
		t.Errorf("Expected output for TTY, got none")
	}
}

func TestProgressBar_Print_NonTTY(t *testing.T) {
	pb := &ProgressBar{total: 10, label: "Test", isTTY: false, startTime: time.Now()}

	output := captureStdout(t, func() {
		for i := 1; i <= 10; i++ {
			pb.Print(i)
		}
	})

	if output != "" {
		t.Errorf("Expected no output for non-TTY, got: %q", output)
	}
}

func TestProgressBar_Print_ClampsCurrent(t *testing.T) {
	pb := &ProgressBar{total: 10, label: "Test", isTTY: true, startTime: time.Now()}

	output := captureStdout(t, func() {
		pb.Print(-3)
		pb.Print(15)
	})

	if !strings.Contains(output, "Test:   0% [0/10]") || !strings.Contains(output, "Test: 100% [10/10]") {
		t.Fatalf("expected clamped progress output to contain 0%% and 100%% states, got %q", output)
	}
}

func TestProgressBar_Print_NonPositiveTotal(t *testing.T) {
	for _, total := range []int{0, -5} {
		pb := &ProgressBar{total: total, label: "Test", isTTY: true}

		output := captureStdout(t, func() {
			pb.Print(1)
		})

		if output != "" {
			t.Fatalf("expected no output for total %d, got %q", total, output)
		}
	}
}

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(5, "Load")
	if pb.total != 5 || pb.label != "Load" {
		t.Errorf("ProgressBar not initialized correctly")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = writePipe
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("failed to close stdout pipe: %v", err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("failed to read stdout pipe: %v", err)
	}
	if err := readPipe.Close(); err != nil {
		t.Fatalf("failed to close stdout read pipe: %v", err)
	}

	return string(output)
}

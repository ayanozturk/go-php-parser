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

func TestProgressBar_Print_RateAndETA(t *testing.T) {
	pb := &ProgressBar{
		total:     100,
		label:     "Scan",
		isTTY:     true,
		startTime: time.Now().Add(-2 * time.Second),
	}

	output := captureStdout(t, func() {
		pb.Print(0)  // no rate/ETA while current == 0
		pb.Print(50) // mid-run: rate + ETA
	})

	if !strings.Contains(output, "Scan:   0% [0/100]") {
		t.Fatalf("expected zero-progress line, got %q", output)
	}
	if !strings.Contains(output, "files/s") {
		t.Fatalf("expected files/s rate for mid progress, got %q", output)
	}
	if !strings.Contains(output, "ETA") {
		t.Fatalf("expected ETA for incomplete progress, got %q", output)
	}
	if strings.Contains(output, "100%") {
		t.Fatalf("did not expect completion line, got %q", output)
	}
	if strings.HasSuffix(output, "\n") {
		t.Fatalf("mid-progress line must not end with newline, got %q", output)
	}
}

func TestProgressBar_Print_CompleteOmitsETA(t *testing.T) {
	pb := &ProgressBar{
		total:     4,
		label:     "Done",
		isTTY:     true,
		startTime: time.Now().Add(-time.Second),
	}
	output := captureStdout(t, func() {
		pb.Print(4)
	})
	if !strings.Contains(output, "files/s") {
		t.Fatalf("expected rate on completion, got %q", output)
	}
	if strings.Contains(output, "ETA") {
		t.Fatalf("completion line must not include ETA, got %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("expected trailing newline on completion, got %q", output)
	}
}

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(5, "Load")
	if pb.total != 5 || pb.label != "Load" {
		t.Errorf("ProgressBar not initialized correctly")
	}
	if pb.startTime.IsZero() {
		t.Fatal("expected startTime to be set")
	}
	if pb.isTTY != isatty(os.Stdout.Fd()) {
		t.Fatalf("isTTY=%v, want isatty(stdout)=%v", pb.isTTY, isatty(os.Stdout.Fd()))
	}
}

func TestIsattyHonorsFD(t *testing.T) {
	// Unknown descriptor must not fall back to blindly stating stdout.
	if isatty(^uintptr(0)) {
		t.Fatal("unknown fd must not report as a TTY")
	}
	if fileForFD(^uintptr(0)) != nil {
		t.Fatal("unknown fd must map to nil file")
	}
	if fileForFD(os.Stdout.Fd()) != os.Stdout {
		t.Fatal("stdout fd should map to os.Stdout")
	}
	if fileForFD(os.Stderr.Fd()) != os.Stderr {
		t.Fatal("stderr fd should map to os.Stderr")
	}
	if fileForFD(os.Stdin.Fd()) != os.Stdin {
		t.Fatal("stdin fd should map to os.Stdin")
	}

	// Pipe fds are not char devices.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	if isCharDeviceFile(r) || isCharDeviceFile(w) {
		t.Fatal("pipe ends must not be char devices")
	}
}

func TestIsCharDeviceFile(t *testing.T) {
	if isCharDeviceFile(nil) {
		t.Fatal("nil file must not be a char device")
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if isCharDeviceFile(r) {
		t.Fatal("pipe reader must not be a char device")
	}
	_ = w.Close()
	_ = r.Close()
	// Stat on a closed file fails → false (covers the error branch).
	if isCharDeviceFile(r) {
		t.Fatal("closed file must not report as char device")
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

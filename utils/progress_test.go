package utils

import (
	"bytes"
	"os"
	"testing"
)

func TestProgressBar_Print_TTY(t *testing.T) {
	pb := &ProgressBar{total: 10, label: "Test", isTTY: true}
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	for i := 1; i <= 10; i++ {
		pb.Print(i)
	}

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if len(output) == 0 {
		t.Errorf("Expected output for TTY, got none")
	}
}

func TestProgressBar_Print_NonTTY(t *testing.T) {
	pb := &ProgressBar{total: 10, label: "Test", isTTY: false}
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	for i := 1; i <= 10; i++ {
		pb.Print(i)
	}

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("Expected no output for non-TTY, got: %q", output)
	}
}

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(5, "Load")
	if pb.total != 5 || pb.label != "Load" {
		t.Errorf("ProgressBar not initialized correctly")
	}
}

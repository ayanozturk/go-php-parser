package helper

import (
	"flag"
	"os"
	"testing"
)

func TestParseCLIArgsDefaults(t *testing.T) {
	// Save and restore original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	args := ParseCLIArgs(nil)
	if args.Profile {
		t.Errorf("Expected Profile to be false by default")
	}
	if args.CommandName != "style" {
		t.Errorf("Expected CommandName to be 'style', got %s", args.CommandName)
	}
	if args.parallelism != 2 {
		t.Errorf("Expected parallelism to be 2 by default")
	}
	if args.Fix {
		t.Errorf("Expected Fix to be false by default")
	}
}

func TestParseCLIArgsWithFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	// Flags must come before positional arguments for Go's flag package
	os.Args = []string{"cmd", "-profile", "-output", "out.log", "-o", "short.log", "-debug", "-p", "4", "-fix", "lint", "file.php"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	args := ParseCLIArgs(nil)
	if !args.Profile {
		t.Errorf("Expected Profile to be true")
	}
	if args.CommandName != "lint" {
		t.Errorf("Expected CommandName to be 'lint', got %s", args.CommandName)
	}
	if args.filePath != "file.php" {
		t.Errorf("Expected filePath to be 'file.php', got %s", args.filePath)
	}
	if args.outputFile != "out.log" {
		t.Errorf("Expected outputFile to be 'out.log', got %s", args.outputFile)
	}
	if args.outputFileShort != "short.log" {
		t.Errorf("Expected outputFileShort to be 'short.log', got %s", args.outputFileShort)
	}
	if !args.debug {
		t.Errorf("Expected debug to be true")
	}
	if args.parallelism != 4 {
		t.Errorf("Expected parallelism to be 4, got %d", args.parallelism)
	}
	if !args.Fix {
		t.Errorf("Expected Fix to be true")
	}
}

func TestSetupOutputFileStdout(t *testing.T) {
	args := CliArgs{}
	w := SetupOutputFile(args)
	if w != os.Stdout {
		t.Errorf("Expected os.Stdout when no output file is set")
	}
}

func TestSetupOutputFileOutputFile(t *testing.T) {
	f, err := os.CreateTemp("", "testout*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	args := CliArgs{outputFile: f.Name()}
	w := SetupOutputFile(args)
	if w == os.Stdout {
		t.Errorf("Expected file writer, got os.Stdout")
	}
	if w == nil {
		t.Errorf("Expected file writer, got nil")
	}
}

func TestSetupOutputFileOutputFileShort(t *testing.T) {
	f, err := os.CreateTemp("", "testout*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	args := CliArgs{outputFileShort: f.Name()}
	w := SetupOutputFile(args)
	if w == os.Stdout {
		t.Errorf("Expected file writer, got os.Stdout")
	}
	if w == nil {
		t.Errorf("Expected file writer, got nil")
	}
}

func removeProfileFiles() {
	_ = os.Remove("cpu.prof")
	_ = os.Remove("mem.prof")
}

func TestSetupProfilingDisabled(t *testing.T) {
	cleanup := SetupProfiling(false)
	if cleanup == nil {
		t.Fatal("Expected a non-nil cleanup function")
	}
	// Should be a no-op
	cleanup()
}

func TestSetupProfilingEnabled(t *testing.T) {
	removeProfileFiles()

	cleanup := SetupProfiling(true)
	if cleanup == nil {
		t.Fatal("Expected a non-nil cleanup function")
	}
	// Should create cpu.prof
	if _, err := os.Stat("cpu.prof"); os.IsNotExist(err) {
		t.Error("cpu.prof should exist after profiling started")
	}
	// Call cleanup, which should create mem.prof
	cleanup()
	if _, err := os.Stat("mem.prof"); os.IsNotExist(err) {
		t.Error("mem.prof should exist after profiling stopped")
	}
	removeProfileFiles()
}

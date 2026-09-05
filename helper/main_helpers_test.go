package helper

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/config"
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
	if args.parallelism != runtime.NumCPU() {
		t.Errorf("Expected parallelism to default to NumCPU (%d), got %d", runtime.NumCPU(), args.parallelism)
	}
	if args.Fix {
		t.Errorf("Expected Fix to be false by default")
	}
}

func TestParseCLIArgsWithFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	// Flags must come before positional arguments for Go's flag package
	os.Args = []string{"cmd", "-config", "custom.yaml", "-profile", "-output", "out.log", "-o", "short.log", "-debug", "-p", "4", "-fix", "lint", "file.php"}
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
	if args.ConfigPath != "custom.yaml" {
		t.Errorf("Expected ConfigPath to be 'custom.yaml', got %s", args.ConfigPath)
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

func TestPrintFileListSorted(t *testing.T) {
	var buf bytes.Buffer

	PrintFileList(&buf, []string{"src/Z.php", "src/A.php", "src/M.php"})

	got := strings.TrimSpace(buf.String())
	want := strings.Join([]string{"src/A.php", "src/M.php", "src/Z.php"}, "\n")
	if got != want {
		t.Fatalf("unexpected file list:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintFileListEmpty(t *testing.T) {
	var buf bytes.Buffer

	PrintFileList(&buf, nil)

	if got, want := strings.TrimSpace(buf.String()), "No files selected by config."; got != want {
		t.Fatalf("unexpected empty file list message: got %q, want %q", got, want)
	}
}

func TestRunAnalyzeExitCodes(t *testing.T) {
	dir := t.TempDir()
	level := 0
	cfg := &config.Config{AnalysisLevel: &level}

	tests := []struct {
		name     string
		path     string
		source   string
		wantExit int
	}{
		{name: "clean", path: filepath.Join(dir, "Clean.php"), source: "<?php\nclass Clean {}\n", wantExit: 0},
		{name: "diagnostic", path: filepath.Join(dir, "Diagnostic.php"), source: "<?php\nnew MissingType();\n", wantExit: 1},
		{name: "parser error", path: filepath.Join(dir, "Invalid.php"), source: "<?php\nfunction broken(\n", wantExit: 1},
		{name: "read error", path: filepath.Join(dir, "Missing.php"), wantExit: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.source != "" {
				if err := os.WriteFile(test.path, []byte(test.source), 0644); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			}
			var output bytes.Buffer
			outcome := RunScanOrCommand(CliArgs{CommandName: "analyze", filePath: test.path, parallelism: 2}, cfg, nil, &output, &MemStats{})
			if outcome.ExitCode != test.wantExit {
				t.Fatalf("unexpected exit code: got %d, want %d; output:\n%s", outcome.ExitCode, test.wantExit, output.String())
			}
		})
	}
}

func TestRunAnalyzeFolderScopesReport(t *testing.T) {
	dir := t.TempDir()
	insideDir := filepath.Join(dir, "inside")
	outsideDir := filepath.Join(dir, "outside")
	if err := os.MkdirAll(insideDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	insideFile := filepath.Join(insideDir, "Inside.php")
	outsideFile := filepath.Join(outsideDir, "Outside.php")
	ignoredFile := filepath.Join(insideDir, "Skip.php.txt")
	if err := os.WriteFile(insideFile, []byte("<?php\nnew MissingType();\n"), 0644); err != nil {
		t.Fatalf("write inside: %v", err)
	}
	if err := os.WriteFile(outsideFile, []byte("<?php\nnew MissingType();\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.WriteFile(ignoredFile, []byte("ignored"), 0644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	level := 0
	cfg := &config.Config{
		Path:          dir,
		Extensions:    []string{"php"},
		Ignore:        []string{},
		AnalysisLevel: &level,
	}

	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", filePath: insideDir, parallelism: 1},
		cfg,
		[]string{insideFile, outsideFile},
		&output,
		&MemStats{},
	)

	if outcome.ExitCode != 1 {
		t.Fatalf("expected exit 1 (diagnostic), got %d; output:\n%s", outcome.ExitCode, output.String())
	}
	if !strings.Contains(output.String(), insideFile) {
		t.Fatalf("expected diagnostic for inside file %q; output:\n%s", insideFile, output.String())
	}
	if strings.Contains(output.String(), outsideFile) {
		t.Fatalf("outside file %q should not appear in report when targeting %q; output:\n%s", outsideFile, insideDir, output.String())
	}
}

func TestRunAnalyzeFolderRespectsIgnore(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "scoped")
	ignoredDir := filepath.Join(target, "vendor")
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	keepFile := filepath.Join(target, "Keep.php")
	skipFile := filepath.Join(ignoredDir, "Skip.php")
	if err := os.WriteFile(keepFile, []byte("<?php\nclass Keep {}\n"), 0644); err != nil {
		t.Fatalf("write keep: %v", err)
	}
	if err := os.WriteFile(skipFile, []byte("<?php\nclass Skip {}\n"), 0644); err != nil {
		t.Fatalf("write skip: %v", err)
	}

	level := 0
	cfg := &config.Config{
		Path:          dir,
		Extensions:    []string{"php"},
		Ignore:        []string{"vendor"},
		AnalysisLevel: &level,
	}

	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", filePath: target, parallelism: 1},
		cfg,
		[]string{keepFile, skipFile},
		&output,
		&MemStats{},
	)

	if outcome.ExitCode != 0 {
		t.Fatalf("expected exit 0 (clean), got %d; output:\n%s", outcome.ExitCode, output.String())
	}
	if strings.Contains(output.String(), skipFile) {
		t.Fatalf("ignored file %q should not appear in report; output:\n%s", skipFile, output.String())
	}
}

func TestRunAnalyzeDoesNotTypeCheckVendorOnScanPath(t *testing.T) {
	dir := t.TempDir()
	hostFile := filepath.Join(dir, "App.php")
	vendorFile := filepath.Join(dir, "vendor", "pkg", "Lib.php")
	if err := os.MkdirAll(filepath.Dir(vendorFile), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(hostFile, []byte("<?php\nfunction use_lib(): VendorLib { return new VendorLib(); }\n"), 0644); err != nil {
		t.Fatalf("write host: %v", err)
	}
	if err := os.WriteFile(vendorFile, []byte("<?php\nclass VendorLib { public function broken(): int { return \"nope\"; } }\n"), 0644); err != nil {
		t.Fatalf("write vendor: %v", err)
	}

	level := 10
	cfg := &config.Config{
		Path:          dir,
		Includes:      []string{filepath.Join(dir, "vendor")},
		Extensions:    []string{"php"},
		Ignore:        []string{"vendor"},
		AnalysisLevel: &level,
	}

	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", parallelism: 1},
		cfg,
		[]string{hostFile, vendorFile},
		&output,
		&MemStats{},
	)
	if strings.Contains(output.String(), vendorFile) {
		t.Fatalf("vendor file should not be type-checked; output:\n%s", output.String())
	}
	if strings.Contains(output.String(), "VendorLib not found") {
		t.Fatalf("host file should resolve vendored symbols; output:\n%s", output.String())
	}
	if outcome.ExitCode != 0 && strings.Contains(output.String(), vendorFile) {
		t.Fatalf("unexpected vendor diagnostic exit %d; output:\n%s", outcome.ExitCode, output.String())
	}
}

func TestRunAnalyzeFolderMissingPathErrors(t *testing.T) {
	level := 0
	cfg := &config.Config{
		Path:          t.TempDir(),
		Extensions:    []string{"php"},
		AnalysisLevel: &level,
	}
	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", filePath: filepath.Join(t.TempDir(), "does-not-exist"), parallelism: 1},
		cfg,
		nil,
		&output,
		&MemStats{},
	)
	if outcome.ExitCode != 2 {
		t.Fatalf("expected exit 2 for missing path, got %d; output:\n%s", outcome.ExitCode, output.String())
	}
	if !strings.Contains(output.String(), "could not stat") {
		t.Fatalf("expected stat error in output, got:\n%s", output.String())
	}
}

func TestRunAnalyzeFolderNoMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "empty")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "readme.md"), []byte("# nothing"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	level := 0
	cfg := &config.Config{
		Path:          dir,
		Extensions:    []string{"php"},
		AnalysisLevel: &level,
	}
	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", filePath: target, parallelism: 1},
		cfg,
		nil,
		&output,
		&MemStats{},
	)
	if outcome.ExitCode != 0 {
		t.Fatalf("expected exit 0 for empty folder, got %d; output:\n%s", outcome.ExitCode, output.String())
	}
	if !strings.Contains(output.String(), "No analyzable files") {
		t.Fatalf("expected helpful message, got:\n%s", output.String())
	}
}

func TestRunAnalyzeExplicitEmptyFolderDoesNotFallBackToConfiguredProject(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "empty")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	outsideFile := filepath.Join(dir, "Outside.php")
	if err := os.WriteFile(outsideFile, []byte("<?php\nnew MissingType();\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	level := 0
	cfg := &config.Config{Path: dir, Extensions: []string{"php"}, AnalysisLevel: &level}
	var output bytes.Buffer
	outcome := RunScanOrCommand(
		CliArgs{CommandName: "analyze", filePath: target, parallelism: 1},
		cfg,
		[]string{outsideFile},
		&output,
		&MemStats{},
	)
	if outcome.ExitCode != 0 {
		t.Fatalf("expected an explicit empty target to stay empty, got exit %d; output:\n%s", outcome.ExitCode, output.String())
	}
	if strings.Contains(output.String(), outsideFile) {
		t.Fatalf("configured project diagnostic leaked into explicit empty target; output:\n%s", output.String())
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

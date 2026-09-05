package command

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeFilesUsesOneProjectSnapshotDeterministically(t *testing.T) {
	dir := t.TempDir()
	declaration := filepath.Join(dir, "Service.php")
	consumer := filepath.Join(dir, "Consumer.php")
	writeAnalyzeFixture(t, declaration, `<?php
namespace Example;
class Service {}
`)
	writeAnalyzeFixture(t, consumer, `<?php
namespace Example;
function build(): void {
    new Service();
    new MissingService();
}
`)

	level := 0
	serial := AnalyzeFiles([]string{consumer, declaration, consumer}, &level, nil, 1)
	parallel := AnalyzeFiles([]string{declaration, consumer}, &level, nil, 4)
	if !reflect.DeepEqual(serial, parallel) {
		t.Fatalf("analysis changed with worker count:\nserial: %#v\nparallel: %#v", serial, parallel)
	}
	if serial.FilesDiscovered != 2 || serial.FilesAnalyzed != 2 {
		t.Fatalf("unexpected file accounting: %#v", serial)
	}
	if hasAnalysisMessage(serial, "Service is not found") {
		t.Fatalf("expected cross-file Service declaration to resolve, got %#v", serial.Issues)
	}
	if !hasAnalysisMessage(serial, "MissingService not found") {
		t.Fatalf("expected missing-class diagnostic, got %#v", serial.Issues)
	}

	var serialOutput, parallelOutput bytes.Buffer
	PrintAnalyzeResult(&serialOutput, serial)
	PrintAnalyzeResult(&parallelOutput, parallel)
	if serialOutput.String() != parallelOutput.String() {
		t.Fatalf("analysis output changed with worker count:\n%s\n%s", serialOutput.String(), parallelOutput.String())
	}
}

func TestAnalyzeFilesAccountsForParseAndReadFailures(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "Valid.php")
	invalid := filepath.Join(dir, "Invalid.php")
	missing := filepath.Join(dir, "Missing.php")
	writeAnalyzeFixture(t, valid, "<?php\nclass Valid {}\n")
	writeAnalyzeFixture(t, invalid, "<?php\nfunction broken(\n")

	level := 0
	result := AnalyzeFiles([]string{missing, invalid, valid}, &level, nil, 2)
	if result.FilesDiscovered != 3 || result.FilesAnalyzed != 1 {
		t.Fatalf("unexpected file accounting: %#v", result)
	}
	if len(result.ParseErrors) != 1 || result.ParseErrors[0].File != invalid || len(result.ParseErrors[0].Errors) == 0 {
		t.Fatalf("unexpected parser failure accounting: %#v", result.ParseErrors)
	}
	if len(result.ReadErrors) != 1 || result.ReadErrors[0].File != missing {
		t.Fatalf("unexpected read failure accounting: %#v", result.ReadErrors)
	}

	var output bytes.Buffer
	PrintAnalyzeResult(&output, result)
	for _, fragment := range []string{"ANALYSIS RESULTS", "parser errors", "read error", "files=3 analyzed=1"} {
		if !strings.Contains(output.String(), fragment) {
			t.Fatalf("analysis output missing %q:\n%s", fragment, output.String())
		}
	}
}

func TestAnalyzeFilesUsesSemanticFacts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ReturnType.php")
	writeAnalyzeFixture(t, path, `<?php
function identifier(): string {
    $value = 42;
    return $value;
}
`)
	level := 10
	result := AnalyzeFiles([]string{path}, &level, nil, 1)
	if !hasAnalysisCode(result, "A.RETURN.TYPE") {
		t.Fatalf("expected snapshot-backed return-type diagnostic, got %#v", result.Issues)
	}
}

func TestAnalyzeFilesDoesNotTypeCheckVendor(t *testing.T) {
	dir := t.TempDir()
	host := filepath.Join(dir, "App.php")
	vendored := filepath.Join(dir, "vendor", "pkg", "Lib.php")
	if err := os.MkdirAll(filepath.Dir(vendored), 0755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	writeAnalyzeFixture(t, host, `<?php
function use_lib(): VendorLib {
    return new VendorLib();
}
`)
	writeAnalyzeFixture(t, vendored, `<?php
class VendorLib {
    public function broken(): int {
        return "nope";
    }
}
function missing_vendor_call() {
    unknown_vendor_fn();
}
`)

	level := 10
	result := AnalyzeFiles([]string{host, vendored}, &level, nil, 2)
	if result.FilesDiscovered != 2 || result.FilesAnalyzed != 1 {
		t.Fatalf("unexpected file accounting: discovered=%d analyzed=%d", result.FilesDiscovered, result.FilesAnalyzed)
	}
	for _, issue := range result.Issues {
		if issue.Filename == vendored {
			t.Fatalf("vendored file was type-checked: %#v", issue)
		}
	}
	if hasAnalysisMessage(result, "VendorLib not found") {
		t.Fatalf("host file should resolve the vendored class, got %#v", result.Issues)
	}
}

func writeAnalyzeFixture(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func hasAnalysisMessage(result AnalyzeResult, message string) bool {
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, message) {
			return true
		}
	}
	return false
}

func hasAnalysisCode(result AnalyzeResult, code string) bool {
	for _, issue := range result.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

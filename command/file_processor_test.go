package command

import (
	"os"
	"testing"
)

func TestProcessStyleFilesParallel(t *testing.T) {
	dir := "../debug"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read testdata dir: %v", err)
	}
	var paths []string
	for _, f := range files {
		if !f.IsDir() {
			paths = append(paths, dir+"/"+f.Name())
		}
	}
	if len(paths) == 0 {
		t.Fatal("no test files found")
	}

	// Run with 1 worker (serial)
	issuesSerial, errSerial, linesSerial := ProcessStyleFilesParallel(paths, []string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"}, 1)
	// Run with 4 workers (parallel)
	issuesParallel, errParallel, linesParallel := ProcessStyleFilesParallel(paths, []string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"}, 4)

	if len(issuesSerial) != len(issuesParallel) {
		t.Errorf("issue count mismatch: serial %d, parallel %d", len(issuesSerial), len(issuesParallel))
	}
	if errSerial != errParallel {
		t.Errorf("error count mismatch: serial %d, parallel %d", errSerial, errParallel)
	}
	if linesSerial != linesParallel {
		t.Errorf("line count mismatch: serial %d, parallel %d", linesSerial, linesParallel)
	}

	// Spot check: ensure at least one issue is found
	if len(issuesSerial) == 0 {
		t.Error("expected at least one issue in test files")
	}
}

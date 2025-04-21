package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testconfig-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := []byte(`path: ./testdata
extensions:
  - php
  - inc
ignore:
  - vendor
  - testdata
`)
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expected := &Config{
		Path:       "./testdata",
		Extensions: []string{"php", "inc"},
		Ignore:     []string{"vendor", "testdata"},
	}
	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("unexpected config: got %+v, want %+v", cfg, expected)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempFile, err := os.CreateTemp("", "badconfig-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.Write([]byte("not: [valid: yaml")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()
	_, err = LoadConfig(tempFile.Name())
	if err == nil {
		t.Error("expected YAML error, got nil")
	}
}

func TestGetFilesToScan(t *testing.T) {
	dir, err := os.MkdirTemp("", "scanroot-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create files and dirs
	os.Mkdir(filepath.Join(dir, "vendor"), 0755)
	os.Mkdir(filepath.Join(dir, "skipme"), 0755)
	file1 := filepath.Join(dir, "a.php")
	file2 := filepath.Join(dir, "b.inc")
	file3 := filepath.Join(dir, "c.txt")
	file4 := filepath.Join(dir, "vendor", "d.php")
	file5 := filepath.Join(dir, "skipme", "e.inc")
	files := []string{file1, file2, file3, file4, file5}
	for _, f := range files {
		os.WriteFile(f, []byte("test"), 0644)
	}

	cfg := &Config{
		Path:       dir,
		Extensions: []string{"php", "inc"},
		Ignore:     []string{"vendor", "skipme"},
	}

	scanned, err := GetFilesToScan(cfg)
	if err != nil {
		t.Fatalf("GetFilesToScan failed: %v", err)
	}

	expected := []string{file1, file2}
	if !reflect.DeepEqual(sorted(scanned), sorted(expected)) {
		t.Errorf("unexpected files to scan: got %v, want %v", scanned, expected)
	}
}

func sorted(s []string) []string {
	copyS := append([]string{}, s...)
	if len(copyS) > 1 {
		for i := 0; i < len(copyS)-1; i++ {
			for j := i + 1; j < len(copyS); j++ {
				if copyS[i] > copyS[j] {
					copyS[i], copyS[j] = copyS[j], copyS[i]
				}
			}
		}
	}
	return copyS
}

func TestGetFilesToScan_Error(t *testing.T) {
	cfg := &Config{
		Path:       "/nonexistent/path/for/coverage",
		Extensions: []string{"php"},
		Ignore:     []string{},
	}
	files, err := GetFilesToScan(cfg)
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
	if files != nil && len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
}

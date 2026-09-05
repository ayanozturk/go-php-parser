package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/overrides"
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
overrides:
  PSR1.Classes.ClassDeclaration.PascalCase:
    classes:
      - "/Legacy_.*/"
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
		Overrides: map[string]overrides.RuleOverride{
			"PSR1.Classes.ClassDeclaration.PascalCase": {
				Classes: []string{"/Legacy_.*/"},
			},
		},
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

func TestLoadConfigRejectsNegativeAnalysisLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go-phpcs.yaml")
	if err := os.WriteFile(path, []byte("path: .\nanalysis_level: -1\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "analysis_level") {
		t.Fatalf("expected analysis-level validation error, got %v", err)
	}
}

func TestDiscoverConfigPrefersGoPHPCSYAML(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"config.yaml", "go-phpcs.yml", "go-phpcs.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("path: .\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	got, err := DiscoverConfig(dir)
	if err != nil {
		t.Fatalf("DiscoverConfig failed: %v", err)
	}

	want := filepath.Join(dir, "go-phpcs.yaml")
	if got != want {
		t.Fatalf("unexpected discovered config: got %q, want %q", got, want)
	}
}

func TestDiscoverConfigFallsBackToLegacyConfigYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("path: .\n"), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	got, err := DiscoverConfig(dir)
	if err != nil {
		t.Fatalf("DiscoverConfig failed: %v", err)
	}

	want := filepath.Join(dir, "config.yaml")
	if got != want {
		t.Fatalf("unexpected discovered config: got %q, want %q", got, want)
	}
}

func TestDiscoverConfigNoConfig(t *testing.T) {
	_, err := DiscoverConfig(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestPrintEffectiveConfig(t *testing.T) {
	level := 0
	cfg := &Config{
		Path:          "./src",
		Extensions:    []string{"php", "inc"},
		Ignore:        []string{"vendor"},
		Rules:         []string{"PSR12.Files.EndFileNewline"},
		AnalysisLevel: &level,
		Overrides: overrides.RuleOverrides{
			"Z.Rule": {Classes: []string{"LegacyZ"}},
			"A.Rule": {Classes: []string{"/LegacyA.*/"}},
		},
	}

	var buf bytes.Buffer
	PrintEffectiveConfig(&buf, cfg, "go-phpcs.yaml")

	want := `config_file: "go-phpcs.yaml"
path: "./src"
includes: []
extensions:
  - "php"
  - "inc"
ignore:
  - "vendor"
rules:
  - "PSR12.Files.EndFileNewline"
analysis_level: 0
overrides:
  "A.Rule":
    classes:
      - "/LegacyA.*/"
  "Z.Rule":
    classes:
      - "LegacyZ"
`
	if got := buf.String(); got != want {
		t.Fatalf("unexpected effective config:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintEffectiveConfigEmptyValues(t *testing.T) {
	cfg := &Config{}

	var buf bytes.Buffer
	PrintEffectiveConfig(&buf, cfg, "")

	want := `config_file: ""
path: ""
includes: []
extensions: []
ignore: []
rules: []
analysis_level: null
overrides: {}
`
	if got := buf.String(); got != want {
		t.Fatalf("unexpected effective config:\ngot:\n%s\nwant:\n%s", got, want)
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

func TestGetIncludeFilesIndexesIgnoredVendorRoot(t *testing.T) {
	root := t.TempDir()
	vendorFile := filepath.Join(root, "vendor", "pkg", "Lib.php")
	if err := os.MkdirAll(filepath.Dir(vendorFile), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(vendorFile, []byte("<?php\nclass Lib {}\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	nestedIgnored := filepath.Join(root, "vendor", "node_modules", "skip.php")
	if err := os.MkdirAll(filepath.Dir(nestedIgnored), 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(nestedIgnored, []byte("<?php\n"), 0644); err != nil {
		t.Fatalf("write nested: %v", err)
	}

	cfg := &Config{
		Path:       root,
		Includes:   []string{filepath.Join(root, "vendor")},
		Extensions: []string{"php"},
		Ignore:     []string{"vendor", "node_modules"},
	}
	host, err := GetFilesToScan(cfg)
	if err != nil {
		t.Fatalf("host scan: %v", err)
	}
	if len(host) != 0 {
		t.Fatalf("host scan should ignore vendor, got %#v", host)
	}
	includes, err := GetIncludeFiles(cfg)
	if err != nil {
		t.Fatalf("include scan: %v", err)
	}
	if len(includes) != 1 || includes[0] != vendorFile {
		t.Fatalf("expected vendor include %q, got %#v", vendorFile, includes)
	}
}

func TestGetIncludeFilesSkipsMissingDirectory(t *testing.T) {
	cfg := &Config{
		Includes:   []string{filepath.Join(t.TempDir(), "vendor")},
		Extensions: []string{"php"},
		Ignore:     []string{"vendor"},
	}
	files, err := GetIncludeFiles(cfg)
	if err != nil {
		t.Fatalf("missing include dir should be skipped, got %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no include files, got %#v", files)
	}
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
	if len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
}

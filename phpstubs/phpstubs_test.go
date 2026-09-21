package phpstubs

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNormalizePHPVersionFallsBackToDefault(t *testing.T) {
	if NormalizePHPVersion("") != DefaultPHPVersion {
		t.Fatalf("expected default %s", DefaultPHPVersion)
	}
	if NormalizePHPVersion("   ") != DefaultPHPVersion {
		t.Fatalf("whitespace-only should default to %s", DefaultPHPVersion)
	}
	if NormalizePHPVersion("8.4.12") != "8.4" {
		t.Fatalf("expected 8.4, got %s", NormalizePHPVersion("8.4.12"))
	}
	if NormalizePHPVersion(" 8.2.0 ") != "8.2" {
		t.Fatalf("expected trimmed 8.2, got %s", NormalizePHPVersion(" 8.2.0 "))
	}
	if NormalizePHPVersion("8.5") != "8.5" {
		t.Fatalf("expected 8.5, got %s", NormalizePHPVersion("8.5"))
	}
	if NormalizePHPVersion("7.4") != DefaultPHPVersion {
		t.Fatalf("unsupported versions should fall back to %s", DefaultPHPVersion)
	}
	if NormalizePHPVersion("8") != DefaultPHPVersion {
		t.Fatalf("major-only versions should fall back to %s", DefaultPHPVersion)
	}
}

func TestSharedStubsAreBundled(t *testing.T) {
	names := SharedNames()
	if len(names) == 0 {
		t.Fatal("expected shared stubs")
	}
	want := []string{"Standard", "Reflection", "Json", "SimpleXML", "Mbstring", "Pdo", "Ddtrace"}
	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}
	for _, name := range want {
		if !have[name] {
			t.Fatalf("missing shared stub %s in %v", name, names)
		}
		data, err := ReadShared(name)
		if err != nil {
			t.Fatalf("read shared %s: %v", name, err)
		}
		if len(data) == 0 || !strings.Contains(string(data), "<?php") {
			t.Fatalf("shared %s: unexpected contents", name)
		}
		if got := SharedFileName(name); got != "phpstub:shared/"+name+".php" {
			t.Fatalf("SharedFileName(%s)=%q", name, got)
		}
		// .php suffix and surrounding space should normalize identically.
		if got := SharedFileName(name + ".php"); got != SharedFileName(" "+name+" ") {
			t.Fatalf("SharedFileName suffix/space mismatch for %s", name)
		}
	}
}

func TestBundledStubsExistPerSupportedVersion(t *testing.T) {
	for _, version := range supportedPHPVersions {
		names := Names(version)
		if len(names) == 0 {
			t.Fatalf("expected bundled stubs for PHP %s", version)
		}
		for _, name := range []string{"Core", "SPL"} {
			data, err := Read(version, name)
			if err != nil {
				t.Fatalf("read %s %s: %v", version, name, err)
			}
			if len(data) == 0 {
				t.Fatalf("empty stub %s/%s", version, name)
			}
			wantFile := "phpstub:" + version + "/" + name + ".php"
			if got := FileName(version, name); got != wantFile {
				t.Fatalf("FileName(%s,%s)=%q want %q", version, name, got, wantFile)
			}
			if got := FileName(version, name+".php"); got != wantFile {
				t.Fatalf("FileName with .php suffix: %q", got)
			}
		}
	}
}

func TestReadRejectsPathEscape(t *testing.T) {
	if _, err := Read("8.3", "../shared/Standard"); err == nil {
		t.Fatal("Read should reject ../ path escape")
	}
	if _, err := ReadShared("../8.3/Core"); err == nil {
		t.Fatal("ReadShared should reject ../ path escape")
	}
	if _, err := Read("8.3", "foo/bar"); err == nil {
		t.Fatal("Read should reject nested names")
	}
	if _, err := Read("8.3", ""); err == nil {
		t.Fatal("Read should reject empty name")
	}
	if _, err := Read("8.3", ".."); err == nil {
		t.Fatal("Read should reject ..")
	}
	if _, err := Read("8.3", "MissingExtension"); err == nil {
		t.Fatal("Read should error for missing stub")
	}
}

func TestFileNameSanitizesInvalidNames(t *testing.T) {
	if got := FileName("8.3", "../shared/Standard"); got != "phpstub:8.3/.php" {
		t.Fatalf("escaped FileName=%q", got)
	}
	if got := SharedFileName(""); got != "phpstub:shared/.php" {
		t.Fatalf("empty SharedFileName=%q", got)
	}
	if got := FileName("9.9", "Core"); !strings.HasPrefix(got, "phpstub:"+DefaultPHPVersion+"/") {
		t.Fatalf("unsupported version FileName should normalize, got %q", got)
	}
}

func TestStubBaseNamesSkipsDirsAndNonPHP(t *testing.T) {
	fsys := fstest.MapFS{
		"v/Core.php":     {Data: []byte("<?php")},
		"v/notes.txt":    {Data: []byte("nope")},
		"v/nested/x.php": {Data: []byte("<?php")}, // implies nested/ dir entry
		"missing-only.txt": {Data: []byte("x")},
	}
	got := stubBaseNames(fsys, "v")
	if len(got) != 1 || got[0] != "Core" {
		t.Fatalf("stubBaseNames=%v, want [Core]", got)
	}
	if got := stubBaseNames(fsys, "does-not-exist"); got != nil {
		t.Fatalf("missing dir should return nil, got %v", got)
	}
	// Ensure the skip branches are the reason nested/x.php is excluded:
	entries, err := fs.ReadDir(fsys, "v")
	if err != nil {
		t.Fatal(err)
	}
	var sawDir, sawNonPHP bool
	for _, e := range entries {
		if e.IsDir() {
			sawDir = true
		}
		if !e.IsDir() && !strings.HasSuffix(e.Name(), ".php") {
			sawNonPHP = true
		}
	}
	if !sawDir || !sawNonPHP {
		t.Fatalf("fixture missing dir/non-php entries: dir=%v nonphp=%v entries=%v", sawDir, sawNonPHP, entries)
	}
}

func TestReadSharedAcceptsPHPSuffix(t *testing.T) {
	a, err := ReadShared("Json.php")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ReadShared("Json")
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("ReadShared .php suffix should be insignificant")
	}
}

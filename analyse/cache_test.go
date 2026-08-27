package analyse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheStoreAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache_test_")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple project index
	idx := NewProjectIndex()
	idx.Classes["myclass"] = ResolvedClass{
		Name: "MyClass",
		Kind: "class",
	}

	// File checksums
	checksums := map[string]string{
		"file1.php": "abc123",
		"file2.php": "def456",
	}

	// Store
	cm := NewCacheManager(tmpDir)
	if err := cm.Store(idx, checksums); err != nil {
		t.Fatalf("store: %v", err)
	}

	// Verify cache file exists
	cachePath := filepath.Join(tmpDir, cacheFile)
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	// Load with matching checksums
	loadedIdx, valid := cm.Load(checksums)
	if !valid {
		t.Fatalf("cache should be valid with matching checksums")
	}
	if loadedIdx == nil {
		t.Fatalf("loaded index is nil")
	}
	if loadedIdx.Classes["myclass"].Name != "MyClass" {
		t.Fatalf("class not preserved in cache")
	}

	// Load with different checksums (cache miss)
	differentChecksums := map[string]string{
		"file1.php": "changed",
		"file2.php": "def456",
	}
	loadedIdx2, valid2 := cm.Load(differentChecksums)
	if valid2 {
		t.Fatalf("cache should be invalid with different checksums")
	}
	if loadedIdx2 != nil {
		t.Fatalf("should not load when checksums differ")
	}
}

func TestFileChecksums(t *testing.T) {
	files := map[string][]byte{
		"file1.php": []byte("<?php class A {}"),
		"file2.php": []byte("<?php class B {}"),
	}

	checksums := FileChecksums(files)

	if len(checksums) != 2 {
		t.Fatalf("expected 2 checksums, got %d", len(checksums))
	}

	// Same content should produce same checksum
	checksums2 := FileChecksums(map[string][]byte{
		"file1.php": []byte("<?php class A {}"),
	})
	if checksums["file1.php"] != checksums2["file1.php"] {
		t.Fatalf("checksum should be deterministic")
	}

	// Different content should produce different checksum
	checksums3 := FileChecksums(map[string][]byte{
		"file1.php": []byte("<?php class A { changed }"),
	})
	if checksums["file1.php"] == checksums3["file1.php"] {
		t.Fatalf("different content should produce different checksum")
	}
}

func TestProjectHash(t *testing.T) {
	checksums := map[string]string{
		"file1.php": "abc123",
		"file2.php": "def456",
	}

	hash1 := ProjectHash(checksums)
	hash2 := ProjectHash(checksums)

	if hash1 != hash2 {
		t.Fatalf("project hash should be deterministic")
	}

	// Different checksums should produce different hash
	checksums2 := map[string]string{
		"file1.php": "changed",
		"file2.php": "def456",
	}
	hash3 := ProjectHash(checksums2)
	if hash1 == hash3 {
		t.Fatalf("different files should produce different hash")
	}
}

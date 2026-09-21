package sharedcache

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetStoreDeleteCachedFileContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	want := []byte("<?php echo 1;\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	DeleteCachedFileContent(path)

	got, err := GetCachedFileContent(path)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("first read content = %q, want %q", got, want)
	}

	// Mutate on disk; cache must still serve the stored bytes.
	if err := os.WriteFile(path, []byte("<?php echo 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cached, err := GetCachedFileContent(path)
	if err != nil {
		t.Fatalf("cached read: %v", err)
	}
	if string(cached) != string(want) {
		t.Fatalf("cache miss after disk change: got %q want %q", cached, want)
	}

	DeleteCachedFileContent(path)
	afterDelete, err := GetCachedFileContent(path)
	if err != nil {
		t.Fatalf("read after delete: %v", err)
	}
	if string(afterDelete) != "<?php echo 2;\n" {
		t.Fatalf("after delete expected disk content, got %q", afterDelete)
	}

	StoreCachedFileContent(path, []byte("stored"))
	stored, err := GetCachedFileContent(path)
	if err != nil {
		t.Fatalf("store/get: %v", err)
	}
	if string(stored) != "stored" {
		t.Fatalf("StoreCachedFileContent not honored: %q", stored)
	}

	DeleteCachedFileContent(path)
}

func TestGetCachedFileContentMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.php")
	DeleteCachedFileContent(path)
	_, err := GetCachedFileContent(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSplitLinesCachedEmpty(t *testing.T) {
	ClearLinesCache()
	if got := SplitLinesCached(nil); got != nil {
		t.Fatalf("nil content: got %#v", got)
	}
	if got := SplitLinesCached([]byte{}); got != nil {
		t.Fatalf("empty content: got %#v", got)
	}
	DeleteCachedLines(nil)
	DeleteCachedLines([]byte{})
}

func TestSplitLinesCachedStaleFingerprint(t *testing.T) {
	ClearLinesCache()

	content := []byte("aa\nbb\n")
	first := SplitLinesCached(content)
	if len(first) != 3 || first[0] != "aa" || first[1] != "bb" {
		t.Fatalf("unexpected first split: %#v", first)
	}

	// Same backing pointer, shorter length → stale fingerprint, must recompute.
	short := content[:2] // "aa"
	second := SplitLinesCached(short)
	if len(second) != 1 || second[0] != "aa" {
		t.Fatalf("unexpected short split: %#v", second)
	}
	if &second[0] == &first[0] {
		t.Fatal("expected recomputed lines after length fingerprint change")
	}

	// Restore full view with a different first byte (same pointer, original length).
	content[0] = 'Z'
	third := SplitLinesCached(content)
	if third[0] != "Za" {
		t.Fatalf("expected first-byte fingerprint miss, got %#v", third)
	}
	if &third[0] == &second[0] {
		t.Fatal("expected recomputed lines after first-byte fingerprint change")
	}

	// Same pointer + length, different last byte → recompute.
	prevLast := &third[0]
	content[len(content)-1] = 'X'
	fourth := SplitLinesCached(content)
	if &fourth[0] == prevLast {
		t.Fatal("expected recomputed lines after last-byte fingerprint change")
	}

	DeleteCachedLines(content)
	ClearLinesCache()
}

func TestSplitLinesCachedEvictionRecount(t *testing.T) {
	ClearLinesCache()
	prev := linesCacheEvictionThreshold
	linesCacheEvictionThreshold = 2
	defer func() {
		linesCacheEvictionThreshold = prev
		ClearLinesCache()
	}()

	// Three distinct backing arrays → third insert trips eviction.
	a := []byte("one\n")
	b := []byte("two\n")
	c := []byte("three\n")

	SplitLinesCached(a)
	SplitLinesCached(b)
	if atomic.LoadInt64(&linesCacheCount) != 2 {
		t.Fatalf("after two inserts count=%d want 2", atomic.LoadInt64(&linesCacheCount))
	}

	SplitLinesCached(c)
	// Eviction cleared prior entries; the entry for c must be counted as 1.
	if got := atomic.LoadInt64(&linesCacheCount); got != 1 {
		t.Fatalf("after eviction+store count=%d want 1", got)
	}

	// Prior entries must be gone; re-split a should allocate fresh lines.
	againA := SplitLinesCached(a)
	if againA[0] != "one" {
		t.Fatalf("unexpected lines after eviction: %#v", againA)
	}
	if atomic.LoadInt64(&linesCacheCount) != 2 {
		t.Fatalf("after re-insert a count=%d want 2", atomic.LoadInt64(&linesCacheCount))
	}
}

func TestDeleteCachedLinesSaturatingDecrement(t *testing.T) {
	ClearLinesCache()
	content := []byte("x\ny\n")
	SplitLinesCached(content)
	if atomic.LoadInt64(&linesCacheCount) != 1 {
		t.Fatalf("count=%d want 1", atomic.LoadInt64(&linesCacheCount))
	}

	// Simulate ClearLinesCache racing ahead of an individual delete.
	atomic.StoreInt64(&linesCacheCount, 0)
	DeleteCachedLines(content)
	if got := atomic.LoadInt64(&linesCacheCount); got != 0 {
		t.Fatalf("saturating decrement failed: count=%d", got)
	}
	ClearLinesCache()
}

func TestClearLinesCacheEmptiesEntries(t *testing.T) {
	ClearLinesCache()
	content := []byte("line\n")
	lines := SplitLinesCached(content)
	ClearLinesCache()
	if atomic.LoadInt64(&linesCacheCount) != 0 {
		t.Fatalf("count after clear=%d", atomic.LoadInt64(&linesCacheCount))
	}
	again := SplitLinesCached(content)
	if &again[0] == &lines[0] {
		t.Fatal("expected fresh lines after ClearLinesCache")
	}
	ClearLinesCache()
}

func TestGetCachedTokensMiss(t *testing.T) {
	ClearTokenCache()
	if got := GetCachedTokens("never-tokenized.php"); got != nil {
		t.Fatalf("expected nil miss, got %#v", got)
	}
}

func TestBatchTokenizeFilesEmptyAndOverwrite(t *testing.T) {
	ClearTokenCache()
	BatchTokenizeFiles(nil)
	BatchTokenizeFiles(map[string][]byte{})

	files := map[string][]byte{"x.php": []byte("<?php echo 1;")}
	BatchTokenizeFiles(files)
	first := GetCachedTokens("x.php")
	if len(first) == 0 {
		t.Fatal("expected tokens")
	}

	BatchTokenizeFiles(map[string][]byte{"x.php": []byte("<?php echo 2;")})
	second := GetCachedTokens("x.php")
	if len(second) == 0 {
		t.Fatal("expected overwritten tokens")
	}
	ClearTokenCache()
}

func TestClearTokenCacheConcurrent(t *testing.T) {
	ClearTokenCache()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := filepath.Join("concurrent", string(rune('a'+i%26))+string(rune('0'+i/26))+".php")
			BatchTokenizeFiles(map[string][]byte{name: []byte("<?php;")})
			_ = GetCachedTokens(name)
			if i%4 == 0 {
				ClearTokenCache()
			}
		}(i)
	}
	wg.Wait()
	ClearTokenCache()
}

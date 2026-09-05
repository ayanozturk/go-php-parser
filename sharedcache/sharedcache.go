package sharedcache

import (
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ayanozturk/go-php-parser/token"
)

var fileContentCache sync.Map
var linesCache sync.Map
var linesCacheCount int64

// linesCacheEvictionThreshold bounds the split-lines cache's live entry
// count as a memory safety valve for very long-lived processes, not as a
// per-corpus budget: linesCacheCount now tracks the actual number of live
// entries (incremented on store, decremented on individual eviction), so a
// single full pass over a corpus at or above this size no longer triggers
// a full-cache clear mid-analysis. It previously counted cumulative
// stores-since-last-clear rather than live entries and was set to 10,000,
// which is smaller than several real-world corpora this project already
// benchmarks against (e.g. test_projects/drupal has 10,856 files),
// causing the cache to thrash: clear itself before a single warm-loop
// iteration even finished, forcing every remaining file to re-split.
const linesCacheEvictionThreshold = 200000

// GetCachedFileContent loads file content from cache or disk, and stores it in cache if not present.
func GetCachedFileContent(filename string) ([]byte, error) {
	if val, ok := fileContentCache.Load(filename); ok {
		return val.([]byte), nil
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	fileContentCache.Store(filename, content)
	return content, nil
}

// StoreCachedFileContent stores already-read file content for later reuse.
func StoreCachedFileContent(filename string, content []byte) {
	fileContentCache.Store(filename, content)
}

// DeleteCachedFileContent removes a file's content from the cache to free memory.
func DeleteCachedFileContent(filename string) {
	fileContentCache.Delete(filename)
}

type linesCacheEntry struct {
	length    int
	firstByte byte
	lastByte  byte
	lines     []string
}

// SplitLinesCached converts content into lines once per backing array.
func SplitLinesCached(content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	key := uintptr(unsafe.Pointer(&content[0]))
	first := content[0]
	last := content[len(content)-1]

	if v, ok := linesCache.Load(key); ok {
		entry := v.(linesCacheEntry)
		if entry.length == len(content) && entry.firstByte == first && entry.lastByte == last {
			return entry.lines
		}
		// Stale entry for a reused pointer (backing array freed and a new
		// allocation landed at the same address): it will be overwritten
		// below without double-counting linesCacheCount.
	} else if atomic.AddInt64(&linesCacheCount, 1) > linesCacheEvictionThreshold {
		ClearLinesCache()
	}

	table := token.NewLineTable(content)
	lines := make([]string, len(table))
	for i := range table {
		lines[i] = string(table.LineBytes(content, i+1))
	}
	linesCache.Store(key, linesCacheEntry{
		length:    len(content),
		firstByte: first,
		lastByte:  last,
		lines:     lines,
	})
	return lines
}

// DeleteCachedLines removes cached split lines for content.
func DeleteCachedLines(content []byte) {
	if len(content) == 0 {
		return
	}
	if _, ok := linesCache.LoadAndDelete(uintptr(unsafe.Pointer(&content[0]))); ok {
		atomic.AddInt64(&linesCacheCount, -1)
	}
}

// ClearLinesCache removes all cached split-line entries.
func ClearLinesCache() {
	atomic.StoreInt64(&linesCacheCount, 0)
	linesCache.Range(func(k, _ interface{}) bool {
		linesCache.Delete(k)
		return true
	})
}

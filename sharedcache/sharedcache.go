package sharedcache

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var fileContentCache sync.Map
var linesCache sync.Map
var linesCacheCount int64

const linesCacheEvictionThreshold = 10000

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
	}

	if atomic.AddInt64(&linesCacheCount, 1) > linesCacheEvictionThreshold {
		ClearLinesCache()
	}

	lines := strings.Split(string(content), "\n")
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
	linesCache.Delete(uintptr(unsafe.Pointer(&content[0])))
}

// ClearLinesCache removes all cached split-line entries.
func ClearLinesCache() {
	atomic.StoreInt64(&linesCacheCount, 0)
	linesCache.Range(func(k, _ interface{}) bool {
		linesCache.Delete(k)
		return true
	})
}

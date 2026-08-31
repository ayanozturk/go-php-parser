package analyse

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheManager handles disk-based caching of ProjectIndex.
// Avoids re-parsing and re-indexing unchanged files on warm runs.
type CacheManager struct {
	cacheDir string
	mu       sync.Mutex
}

// CacheEntry wraps ProjectIndex with metadata for validation.
type CacheEntry struct {
	Version       int                    `json:"version"`
	Timestamp     int64                  `json:"timestamp"`
	ProjectHash   string                 `json:"projectHash"`   // Hash of all source files
	FileChecksums map[string]string      `json:"fileChecksums"` // path -> content MD5
	OldChecksums  map[string]string      `json:"old_checksums"` // prev run checksums for diff
	Index         SerializedProjectIndex `json:"index"`
}

// SerializedProjectIndex is a serializable view of ProjectIndex.
// Omits caches/derivations; only keeps essential symbol tables.
type SerializedProjectIndex struct {
	Classes    map[string]ResolvedClass               `json:"classes"`
	Methods    map[string]map[string]ResolvedMethod   `json:"methods"`
	Properties map[string]map[string]ResolvedProperty `json:"properties"`
	Functions  map[string]ResolvedFunction            `json:"functions"`
}

const (
	cacheVersion = 1
	cacheFile    = "go-phpcs-index.json"
)

// NewCacheManager creates a cache manager. cacheDir should be a writable directory.
func NewCacheManager(cacheDir string) *CacheManager {
	return &CacheManager{cacheDir: cacheDir}
}

// Load attempts to load a cached ProjectIndex if it exists and is valid.
// Returns (index, valid, error). If not valid or missing, returns (nil, false, nil).
func (cm *CacheManager) Load(fileChecksums map[string]string) (*ProjectIndex, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cachePath := filepath.Join(cm.cacheDir, cacheFile)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false // Cache miss (expected on cold run)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false // Cache corrupted
	}

	// Validate version
	if entry.Version != cacheVersion {
		return nil, false
	}

	// Validate file checksums: if any file changed, cache is invalid
	for path, expectedHash := range fileChecksums {
		cachedHash, ok := entry.FileChecksums[path]
		if !ok || cachedHash != expectedHash {
			return nil, false
		}
	}

	// Deserialize
	idx := NewProjectIndex()
	idx.Classes = entry.Index.Classes
	idx.Methods = entry.Index.Methods
	idx.Properties = entry.Index.Properties
	idx.Functions = entry.Index.Functions

	return idx, true
}

// GetChangedFiles returns list of files that changed since last cache.
// Entry should be loaded from cache. Returns empty if all files unchanged.
func (cm *CacheManager) GetChangedFiles(entry *CacheEntry, newChecksums map[string]string) []string {
	changed := make([]string, 0)

	// Check for modified or added files
	for path, newHash := range newChecksums {
		oldHash, exists := entry.OldChecksums[path]
		if !exists || oldHash != newHash {
			changed = append(changed, path)
		}
	}

	// Check for deleted files
	for path := range entry.OldChecksums {
		if _, exists := newChecksums[path]; !exists {
			changed = append(changed, path)
		}
	}

	return changed
}

// Store persists the ProjectIndex to disk for future warm runs.
func (cm *CacheManager) Store(index *ProjectIndex, fileChecksums map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry := CacheEntry{
		Version:       cacheVersion,
		Timestamp:     time.Now().Unix(),
		FileChecksums: fileChecksums,
		OldChecksums:  fileChecksums, // Store for next diff
		Index: SerializedProjectIndex{
			Classes:    index.Classes,
			Methods:    index.Methods,
			Properties: index.Properties,
			Functions:  index.Functions,
		},
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cm.cacheDir, cacheFile)
	if err := os.MkdirAll(cm.cacheDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// FileChecksums computes MD5 hash for each file.
func FileChecksums(files map[string][]byte) map[string]string {
	checksums := make(map[string]string, len(files))
	for path, content := range files {
		h := md5.New()
		h.Write(content)
		checksums[path] = fmt.Sprintf("%x", h.Sum(nil))
	}
	return checksums
}

// ProjectHash computes a combined hash of all file checksums for quick full validation.
func ProjectHash(checksums map[string]string) string {
	h := md5.New()
	// Iterate in sorted order for determinism
	for _, path := range sortedMapKeys(checksums) {
		io.WriteString(h, path)
		io.WriteString(h, checksums[path])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple bubble sort; for production use sort.Strings
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

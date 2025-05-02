package sharedcache

import (
	"os"
	"sync"
)

var fileContentCache sync.Map

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

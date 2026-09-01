package analyse

import (
	"strings"
	"sync"
	"unsafe"
)

const (
	identLowerCacheMaxEntries  = 16384
	identLowerCacheMaxKeyBytes = 1 << 20
)

type boundedStringCache struct {
	values   sync.Map
	mu       sync.Mutex
	order    []string
	head     int
	keyBytes int
}

var identLowerCache = &boundedStringCache{order: make([]string, 0, identLowerCacheMaxEntries)}

func (c *boundedStringCache) load(key string) (string, bool) {
	value, ok := c.values.Load(key)
	if !ok {
		return "", false
	}
	return value.(string), true
}

func (c *boundedStringCache) store(key, value string) {
	if len(key) > parsedTypeCacheMaxKeySize {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, loaded := c.values.Load(key); loaded {
		return
	}
	for len(c.order)-c.head >= identLowerCacheMaxEntries || c.keyBytes+len(key) > identLowerCacheMaxKeyBytes {
		oldest := c.order[c.head]
		c.head++
		c.keyBytes -= len(oldest)
		c.values.Delete(oldest)
	}
	if c.head > 0 && c.head*2 >= len(c.order) {
		copy(c.order, c.order[c.head:])
		c.order = c.order[:len(c.order)-c.head]
		c.head = 0
	}
	c.values.Store(key, value)
	c.order = append(c.order, key)
	c.keyBytes += len(key)
}

// asciiLowerIdent lowercases PHP identifiers without the unicode.ToLower
// tables. Non-ASCII input falls back to strings.ToLower. Already-lowercase
// ASCII is returned unchanged and does not allocate. Mixed-case ASCII results
// are interned so repeated lookups reuse one lowered string.
func asciiLowerIdent(s string) string {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 {
			return strings.ToLower(s)
		}
		if c >= 'A' && c <= 'Z' {
			if cached, ok := identLowerCache.load(s); ok {
				return cached
			}
			b := make([]byte, len(s))
			copy(b, s)
			b[i] = c + ('a' - 'A')
			for j := i + 1; j < len(s); j++ {
				cj := s[j]
				if cj >= 0x80 {
					return strings.ToLower(s)
				}
				if cj >= 'A' && cj <= 'Z' {
					b[j] = cj + ('a' - 'A')
				}
			}
			lowered := bytesAsString(b)
			identLowerCache.store(s, lowered)
			return lowered
		}
	}
	return s
}

func bytesAsString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

package analyse

import (
	"strings"
	"sync"
	"unsafe"
)

const (
	identLowerCacheMaxEntries  = 8192
	identLowerCacheMaxKeyBytes = 1 << 19
)

type boundedStringCache struct {
	mu       sync.RWMutex
	values   map[string]string
	order    []string
	head     int
	keyBytes int
}

const identLowerCacheShardCount = 8

type shardedStringCache struct {
	shards [identLowerCacheShardCount]boundedStringCache
}

var identLowerCache = &shardedStringCache{}

func (c *boundedStringCache) load(key string) (string, bool) {
	c.mu.RLock()
	value, ok := c.values[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	return value, true
}

func (c *boundedStringCache) store(key, value string) {
	if len(key) > parsedTypeCacheMaxKeySize {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, loaded := c.values[key]; loaded {
		return
	}
	if len(key) > identLowerCacheMaxKeyBytes/identLowerCacheShardCount {
		return
	}
	if c.values == nil {
		// Grow maps only with observed keys; eagerly reserving every shard's
		// maximum made the bounded cache's cold RSS unnecessarily large.
		c.values = make(map[string]string)
		c.order = make([]string, 0, identLowerCacheMaxEntries/identLowerCacheShardCount)
	}
	maxEntries := identLowerCacheMaxEntries / identLowerCacheShardCount
	maxKeyBytes := identLowerCacheMaxKeyBytes / identLowerCacheShardCount
	for len(c.order)-c.head >= maxEntries || c.keyBytes+len(key) > maxKeyBytes {
		oldest := c.order[c.head]
		c.head++
		c.keyBytes -= len(oldest)
		delete(c.values, oldest)
	}
	if c.head > 0 && c.head*2 >= len(c.order) {
		copy(c.order, c.order[c.head:])
		c.order = c.order[:len(c.order)-c.head]
		c.head = 0
	}
	c.values[key] = value
	c.order = append(c.order, key)
	c.keyBytes += len(key)
}

func (c *shardedStringCache) shard(key string) *boundedStringCache {
	// Identifier keys are short. FNV-1a gives a stable, low-cost spread without
	// putting all writes behind sync.Map's global miss promotion lock.
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return &c.shards[h&(identLowerCacheShardCount-1)]
}

func (c *shardedStringCache) load(key string) (string, bool) {
	return c.shard(key).load(key)
}

func (c *shardedStringCache) store(key, value string) {
	c.shard(key).store(key, value)
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

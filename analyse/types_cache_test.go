package analyse

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var parseTypeBenchmarkRun atomic.Uint64

func resetParsedTypeCache(t testing.TB) {
	t.Helper()
	previous := parsedTypeCache
	parsedTypeCache = newBoundedTypeCache()
	t.Cleanup(func() {
		parsedTypeCache = previous
	})
}

func TestParseTypeCacheIsBounded(t *testing.T) {
	resetParsedTypeCache(t)

	const attemptedEntries = 5000
	for i := 0; i < attemptedEntries; i++ {
		ParseType(fmt.Sprintf("SecurityFuzzType%d", i))
	}

	entries, keyBytes := parsedTypeCache.size()
	if entries > parsedTypeCacheMaxEntries {
		t.Fatalf("type cache retained %d entries after %d unique workspace types; want at most %d", entries, attemptedEntries, parsedTypeCacheMaxEntries)
	}
	if keyBytes > parsedTypeCacheMaxKeyBytes {
		t.Fatalf("type cache retained %d key bytes; want at most %d", keyBytes, parsedTypeCacheMaxKeyBytes)
	}
}

func TestParseTypeCacheBoundsRetainedKeyBytes(t *testing.T) {
	resetParsedTypeCache(t)
	prefix := strings.Repeat("T", parsedTypeCacheMaxKeySize-16)
	for i := 0; i < 5000; i++ {
		ParseType(fmt.Sprintf("%s%015d", prefix, i))
	}

	_, keyBytes := parsedTypeCache.size()
	if keyBytes > parsedTypeCacheMaxKeyBytes {
		t.Fatalf("type cache retained %d key bytes; want at most %d", keyBytes, parsedTypeCacheMaxKeyBytes)
	}
}

func TestParseTypeCacheSkipsOversizedKeys(t *testing.T) {
	resetParsedTypeCache(t)
	raw := "WorkspaceType" + strings.Repeat("X", parsedTypeCacheMaxKeySize)
	want := ParseType(raw).String()

	if entries, _ := parsedTypeCache.size(); entries != 0 {
		t.Fatalf("oversized type was retained in cache; entries = %d", entries)
	}
	if got := ParseType(raw).String(); got != want {
		t.Fatalf("uncached parsing changed result: got %q, want %q", got, want)
	}
}

func TestParseTypeCacheConcurrentEvictionPreservesResults(t *testing.T) {
	resetParsedTypeCache(t)

	const workers = 16
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				raw := fmt.Sprintf("Worker%dType%d|null", worker, i)
				if got := ParseType(raw).String(); !strings.Contains(got, "Worker") || !strings.Contains(got, "null") {
					t.Errorf("ParseType(%q) returned %q", raw, got)
					return
				}
			}
		}(worker)
	}
	wg.Wait()

	entries, keyBytes := parsedTypeCache.size()
	if entries > parsedTypeCacheMaxEntries || keyBytes > parsedTypeCacheMaxKeyBytes {
		t.Fatalf("concurrent cache exceeded bounds: entries=%d keyBytes=%d", entries, keyBytes)
	}
}

func TestParseTypeCacheConcurrentDNFSerializationIsStable(t *testing.T) {
	resetParsedTypeCache(t)

	const raw = "(LeftContract&MarkerContract)|RightChoice|null"
	const want = "(LeftContract&MarkerContract)|RightChoice|null"
	ParseType(raw)

	const workers = 16
	const iterations = 1000
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if got := ParseType(raw).dnfString(); got != want {
					t.Errorf("ParseType(%q).dnfString() = %q, want %q", raw, got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkParseTypeCacheHit(b *testing.B) {
	resetParsedTypeCache(b)
	const raw = "array<string, list<WorkspaceEntity>>|null"
	ParseType(raw)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseType(raw)
	}
}

func BenchmarkParseTypeCacheMiss(b *testing.B) {
	resetParsedTypeCache(b)
	run := parseTypeBenchmarkRun.Add(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseType(fmt.Sprintf("WorkspaceEntity%d_%d|null", run, i))
	}
}

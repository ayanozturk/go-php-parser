package analyse

import "testing"

func TestSemanticFactStorePreservesKindsAndReconstructsFacts(t *testing.T) {
	t.Parallel()
	const (
		filename = "src/Service.php"
		start    = 24
		end      = 31
	)

	store := make(semanticFactStore)
	builtIn := SemanticFact{
		Key:     SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: FactKindInferredType},
		Subject: "method:service:load",
		Type:    "User",
		Value:   "non-null",
	}
	if !store.put(builtIn) {
		t.Fatal("expected first built-in fact to be stored")
	}
	if store.put(SemanticFact{
		Key:     builtIn.Key,
		Subject: "method:service:other",
		Type:    "Admin",
		Value:   "replaced",
	}) {
		t.Fatal("same file/span/kind should be rejected as a duplicate")
	}
	if got, ok := store.fact(builtIn.Key); !ok || got != builtIn {
		t.Fatalf("duplicate changed or failed to reconstruct built-in fact: %#v, %v", got, ok)
	}
	if !store.has(builtIn.Key) {
		t.Fatal("stored built-in fact was not found by has")
	}

	for _, fact := range []SemanticFact{
		{
			Key:     SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: FactKindReference},
			Subject: "class:user",
			Value:   "reference",
		},
		{
			Key:     SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: FactKindNarrowed},
			Subject: "class:user",
			Type:    "User",
			Value:   "instanceof",
		},
	} {
		if !store.put(fact) {
			t.Fatalf("different built-in kind at the same span was rejected: %#v", fact.Key)
		}
		if got, ok := store.fact(fact.Key); !ok || got != fact {
			t.Fatalf("failed to reconstruct fact for kind %q: %#v, %v", fact.Key.Kind, got, ok)
		}
	}

	customKind := FactKind("acme/name/confidence")
	custom := SemanticFact{
		Key:     SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: customKind},
		Subject: "symbol:user",
		Type:    "high",
		Value:   "external",
	}
	if !store.put(custom) {
		t.Fatal("custom namespaced kind at a shared span was rejected")
	}
	if !store.has(custom.Key) {
		t.Fatal("custom namespaced kind was not found by has")
	}
	if got, ok := store.fact(custom.Key); !ok || got != custom {
		t.Fatalf("custom kind lookup did not reconstruct the original fact: %#v, %v", got, ok)
	}
	if store.putParts(custom.Key, "symbol:other", "low", "replaced") {
		t.Fatal("same file/span/custom-kind should be rejected as a duplicate")
	}

	otherFile := custom
	otherFile.Key.File = "src/Other.php"
	if !store.put(otherFile) {
		t.Fatal("same span and custom kind in another file was rejected")
	}
	if got, ok := store.fact(otherFile.Key); !ok || got != otherFile {
		t.Fatalf("fact reconstruction crossed file partition: %#v, %v", got, ok)
	}
	missing := otherFile.Key
	missing.Kind = FactKind("acme/name/missing")
	if store.has(missing) {
		t.Fatal("lookup for an unknown custom kind unexpectedly succeeded")
	}

	generatedKey := SemanticFactKey{File: filename, StartOffset: 40, EndOffset: 45, Kind: FactKindInferredType}
	if !store.putGeneratedInferred(generatedKey, "method:service:generated", "Generated") {
		t.Fatal("expected generated inferred fact to be stored")
	}
	generated := SemanticFact{Key: generatedKey, Subject: "method:service:generated", Type: "Generated"}
	if got, ok := store.fact(generatedKey); !ok || got != generated {
		t.Fatalf("generated inferred fact was not reconstructed: %#v, %v", got, ok)
	}
	if store.put(SemanticFact{Key: generatedKey, Type: "Explicit", Value: "external"}) {
		t.Fatal("explicit inferred fact replaced an existing generated fact")
	}

	if ^uint(0)>>63 == 1 {
		largeOffset := maxCompactSourceOffset + 1
		large := int(largeOffset)
		largeKey := SemanticFactKey{File: filename, StartOffset: large, EndOffset: large + 1, Kind: FactKindInferredType}
		if !store.putGeneratedInferred(largeKey, "function:large", "Large") {
			t.Fatal("large-offset generated inferred fact was rejected")
		}
		if got, ok := store.fact(largeKey); !ok || got.Key != largeKey || got.Subject != "function:large" || got.Type != "Large" {
			t.Fatalf("large-offset fallback did not preserve the fact: %#v, %v", got, ok)
		}
	}
}

// Keep this key deliberately equivalent to the pre-partitioning key shape.
// It is local to the test so the production representation can change while
// this remains a stable allocation comparison.
type legacySemanticFactSpanKey struct {
	StartOffset int
	EndOffset   int
	Kind        FactKind
}

type legacySemanticFact struct {
	Subject SymbolID
	Type    string
	Value   string
}

type legacySemanticFactStore map[string]map[legacySemanticFactSpanKey]legacySemanticFact

var (
	semanticFactStoreBenchmarkSink       semanticFactStore
	legacySemanticFactStoreBenchmarkSink legacySemanticFactStore
)

func TestSemanticFactStoreBuiltInInsertionAllocationCeiling(t *testing.T) {
	keys := semanticFactStoreBenchmarkKeys(256)
	candidate := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			store := make(semanticFactStore)
			for _, key := range keys {
				if !store.putGeneratedInferred(key, "function:benchmark", "User") {
					b.Fatal("benchmark built-in insertion unexpectedly collided")
				}
			}
			semanticFactStoreBenchmarkSink = store
		}
	})
	legacy := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			store := make(legacySemanticFactStore)
			for _, key := range keys {
				fileFacts := store[key.File]
				if fileFacts == nil {
					fileFacts = make(map[legacySemanticFactSpanKey]legacySemanticFact)
					store[key.File] = fileFacts
				}
				fileFacts[legacySemanticFactSpanKey{
					StartOffset: key.StartOffset,
					EndOffset:   key.EndOffset,
					Kind:        key.Kind,
				}] = legacySemanticFact{Subject: "function:benchmark", Type: "User"}
			}
			legacySemanticFactStoreBenchmarkSink = store
		}
	})

	candidateBytes := candidate.AllocedBytesPerOp()
	legacyBytes := legacy.AllocedBytesPerOp()
	candidateAllocs := candidate.AllocsPerOp()
	legacyAllocs := legacy.AllocsPerOp()
	t.Logf("built-in insertion: %d bytes/op, %d allocs/op; legacy: %d bytes/op, %d allocs/op", candidateBytes, candidateAllocs, legacyBytes, legacyAllocs)

	if candidateBytes >= legacyBytes {
		t.Fatalf("built-in insertion allocation = %d bytes/op, want less than legacy %d bytes/op", candidateBytes, legacyBytes)
	}
	// Allocation count is a secondary guard because map bucket sizes can change
	// how many growth allocations the runtime performs. Keep it bounded while
	// using allocated bytes for the strict legacy comparison above.
	if candidateAllocs > 24 {
		t.Fatalf("built-in insertion allocations = %d/op, want <= 24", candidateAllocs)
	}
	// Keep the comparison from passing by moving work out of the measured
	// insertion path: each inserted fact should stay well below this ceiling.
	const maxBytesPerFact = 512
	if candidateBytes > int64(len(keys)*maxBytesPerFact) {
		t.Fatalf("built-in insertion allocation = %d bytes/op, want <= %d bytes/op", candidateBytes, len(keys)*maxBytesPerFact)
	}
}

func semanticFactStoreBenchmarkKeys(count int) []SemanticFactKey {
	keys := make([]SemanticFactKey, count)
	for i := range keys {
		start := i * 3
		keys[i] = SemanticFactKey{
			File:        "benchmark.php",
			StartOffset: start,
			EndOffset:   start + 2,
			Kind:        FactKindInferredType,
		}
	}
	return keys
}

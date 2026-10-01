package analyse

import (
	"strings"
	"testing"
)

func TestResolvePHPDocOffsetAccess(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
		ok   bool
	}{
		{name: "shape literal key", typ: `array{foo: int, bar: string}['foo']`, want: "int", ok: true},
		{name: "generic array value", typ: `array<int, string>[int]`, want: "string", ok: true},
		{name: "list value", typ: `list<bool>[int]`, want: "bool", ok: true},
		{name: "nested shape access", typ: `array{nested: array{active: bool}}['nested']['active']`, want: "bool", ok: true},
		{name: "unknown shape key", typ: `array{foo: int}['missing']`},
		{name: "unresolved template", typ: `T[K]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolvePHPDocOffsetAccess(tt.typ, FileTypeContext{})
			if ok != tt.ok || got != tt.want {
				t.Fatalf("resolvePHPDocOffsetAccess(%q) = (%q, %v), want (%q, %v)", tt.typ, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestResolvePHPDocKeyValueProjections(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
		ok   bool
	}{
		{name: "shape keys", typ: `key-of<array{foo: int, bar: string}>`, want: "'bar'|'foo'", ok: true},
		{name: "mixed literal keys", typ: `key-of<array{foo: int, 0: bool}>`, want: "'foo'|0", ok: true},
		{name: "escaped string key", typ: `key-of<array{'it\'s': int}>`, want: `'it\'s'`, ok: true},
		{name: "shape values", typ: `value-of<array{foo: int, bar: string}>`, want: "int|string", ok: true},
		{name: "union shape values", typ: `value-of<array{foo: int}|array{bar: string}>`, want: "int|string", ok: true},
		{name: "generic array keys", typ: `key-of<array<int, string>>`, want: "int", ok: true},
		{name: "generic array values", typ: `value-of<array<int, string>>`, want: "string", ok: true},
		{name: "unresolved template", typ: `value-of<T>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolvePHPDocIndexedTypes(tt.typ, FileTypeContext{})
			if ok != tt.ok || got != tt.want {
				t.Fatalf("resolvePHPDocIndexedTypes(%q) = (%q, %v), want (%q, %v)", tt.typ, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestArgumentTypesInferPHPDocOffsetAccessAndGenericArrayOffsets(t *testing.T) {
	const source = `<?php
/** @phpstan-type Summary array{count: int, label: string} */
class SummaryProvider {
    /** @return Summary['count'] */
    public function count(): int { return 1; }
}

/** @param array<int, string> $items */
function check(array $items, SummaryProvider $provider): void {
    acceptInt($provider->count());
    acceptString($items[0]);
    acceptString($provider->count());
    acceptInt($items[0]);
}

function acceptInt(int $value): void {}
function acceptString(string $value): void {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"offset.php": source}, 5)
	var mismatches []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == "A.ARG.TYPE" {
			mismatches = append(mismatches, issue)
		}
	}
	if len(mismatches) != 2 {
		t.Fatalf("expected the indexed shape type and generic value type to flag their reversed calls, got %#v", mismatches)
	}
}

func TestArgumentTypesCheckPHPDocCallableParametersAndReturnTypes(t *testing.T) {
	const source = `<?php
/** @param callable(int): string $callback */
function invokeCallable(callable $callback): void {
    acceptString($callback(1));
    acceptString($callback('wrong'));
}

/** @param Closure(int): string $callback */
function invokeClosure(Closure $callback): void {
    acceptString($callback(1));
}

function acceptString(string $value): void {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"callable.php": source}, 5)
	var mismatches []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == "A.ARG.TYPE" {
			mismatches = append(mismatches, issue)
		}
	}
	if len(mismatches) != 1 || !strings.Contains(mismatches[0].Message, "Callable") {
		t.Fatalf("expected only the invalid callable argument to be reported, got %#v", mismatches)
	}
}

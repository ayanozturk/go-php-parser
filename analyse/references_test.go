package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestIndexSyntaxFileForReferencesUsesBindSyntaxResult(t *testing.T) {
	srcA := []byte("<?php\nnamespace App;\nclass Foo {}\n")
	srcB := []byte("<?php\nnamespace App;\nfunction f(Foo $x) {}\n")
	resA := syntax.Parse(srcA)
	resB := syntax.Parse(srcB)

	g := NewProjectUsageGraph()
	IndexSyntaxFileForReferences(g, "file://a.php", resA)
	IndexSyntaxFileForReferences(g, "file://b.php", resB)

	// Cursor on Foo in B's parameter type.
	offset := strings.Index(string(srcB), "Foo")
	if offset < 0 {
		t.Fatal("Foo not found in srcB")
	}
	hits := ReferencesAt(g, "file://b.php", resB, offset)
	if len(hits) < 2 {
		t.Fatalf("want class decl + type use, got %d: %+v", len(hits), hits)
	}
	for _, h := range hits {
		if h.Kind != "class" && h.Kind != "type" {
			t.Fatalf("unexpected kind in Foo refs: %+v", h)
		}
		if h.Resolved != `App\Foo` {
			t.Fatalf("Resolved=%q want App\\Foo", h.Resolved)
		}
		if h.URI != "file://a.php" && h.URI != "file://b.php" {
			t.Fatalf("unexpected URI: %+v", h)
		}
	}
}

func TestReferencesAtSharesParseWithBindSyntaxResult(t *testing.T) {
	src := []byte(`<?php
namespace App;
class Foo {
    public function bar() {}
    public function m() { $this->bar(); }
}
function f(Foo $x) { return $x; }
`)
	res := syntax.Parse(src)
	g := NewProjectUsageGraph()
	IndexSyntaxFileForReferences(g, "file://one.php", res)

	// Parameter type Foo — class-like kindsCompatible joins decl + type use.
	offset := strings.Index(string(src), "function f(Foo")
	if offset < 0 {
		t.Fatal("function f(Foo not found")
	}
	offset = strings.Index(string(src)[offset:], "Foo") + offset
	hits := ReferencesAt(g, "file://one.php", res, offset)
	if len(hits) < 2 {
		t.Fatalf("want >=2 Foo refs, got %d: %+v", len(hits), hits)
	}

	// Re-query without re-parsing: same res proves BindSyntaxResult path.
	again := RenameTargetsAt(g, "file://one.php", res, offset)
	if len(again) != len(hits) {
		t.Fatalf("RenameTargetsAt=%d ReferencesAt=%d", len(again), len(hits))
	}

	// Method call $this->bar() inside Foo matches method decl (same Owner).
	barCall := strings.LastIndex(string(src), "bar")
	if barCall < 0 {
		t.Fatal("bar not found")
	}
	barHits := ReferencesAt(g, "file://one.php", res, barCall)
	methods := 0
	for _, h := range barHits {
		if h.Kind == "method" && h.Written == "bar" && h.Owner == `App\Foo` {
			methods++
		}
	}
	if methods < 2 {
		t.Fatalf("want method decl+call for bar, got %d hits=%+v", methods, barHits)
	}
}

func TestReferencesAtMissReturnsNil(t *testing.T) {
	src := []byte("<?php\nclass Foo {}\n")
	res := syntax.Parse(src)
	g := NewProjectUsageGraph()
	IndexSyntaxFileForReferences(g, "file://m.php", res)
	if hits := ReferencesAt(g, "file://m.php", res, 0); hits != nil {
		t.Fatalf("offset 0 should miss, got %+v", hits)
	}
	if hits := ReferencesAt(nil, "file://m.php", res, 10); hits != nil {
		t.Fatalf("nil graph should return nil, got %+v", hits)
	}
}

func TestReferencesAtRequiresFullParseNotIndex(t *testing.T) {
	// Body `new Foo()` inside a method (Owner=App\Foo) must still match the
	// top-level class decl (Owner="") via softened class-like Owner filter.
	src := []byte("<?php\nnamespace App;\nclass Foo {\n  public function m() { return new Foo(); }\n}\n")
	full := syntax.Parse(src)
	index := syntax.ParseForIndex(src)

	gFull := NewProjectUsageGraph()
	IndexSyntaxFileForReferences(gFull, "file://f.php", full)
	offset := strings.LastIndex(string(src), "Foo")
	fullHits := ReferencesAt(gFull, "file://f.php", full, offset)

	gIdx := NewProjectUsageGraph()
	IndexSyntaxFileForReferences(gIdx, "file://f.php", index)
	idxHits := ReferencesAt(gIdx, "file://f.php", index, offset)

	// Full parse sees the body `new Foo()` use; index-mode may not.
	if len(fullHits) < 2 {
		t.Fatalf("full parse should find decl+body use, got %d: %+v", len(fullHits), fullHits)
	}
	if len(idxHits) >= len(fullHits) {
		t.Fatalf("index-mode should not exceed full-parse body refs: full=%d idx=%d", len(fullHits), len(idxHits))
	}
}

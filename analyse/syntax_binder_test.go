package analyse

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestBinderTypeFromSyntax(t *testing.T) {
	p := syntax.NewParser([]byte(`?\App\Foo|string`))
	g := p.ParseType()
	f := &syntax.File{Source: []byte(`?\App\Foo|string`), Green: g}
	syntax.BindRed(f)
	b := NewBinder("App", nil)
	got := b.TypeFromSyntax(f.Root)
	if got.IsEmpty() {
		t.Fatal("expected non-empty type")
	}
}

func TestBinderTypeFromSyntaxCallable(t *testing.T) {
	p := syntax.NewParser([]byte(`callable(Foo): Bar`))
	g := p.ParseType()
	f := &syntax.File{Source: []byte(`callable(Foo): Bar`), Green: g}
	syntax.BindRed(f)
	b := NewBinder("App", nil)
	got := b.TypeFromSyntax(f.Root)
	if got.String() != "callable" {
		t.Fatalf("got type %q, want callable", got.String())
	}
	seen := map[string]bool{}
	for _, u := range b.Graph.Uses {
		if u.Kind == "type" {
			seen[u.Written] = true
		}
	}
	if !seen["Foo"] || !seen["Bar"] {
		t.Fatalf("expected Foo and Bar type uses, got %+v", b.Graph.Uses)
	}
}

func TestBinderResolvesImports(t *testing.T) {
	b := NewBinder("App", map[string]string{"foo": "Vendor\\Foo"})
	if got := b.resolve("Bar"); got != "App\\Bar" {
		t.Fatalf("got %q", got)
	}
	if got := b.resolve("foo"); got != "Vendor\\Foo" {
		t.Fatalf("alias got %q", got)
	}
	if got := b.resolve(`\Abs\X`); got != `Abs\X` {
		t.Fatalf("fqn got %q", got)
	}
}

func TestProjectUsageGraphCrossFile(t *testing.T) {
	g := NewProjectUsageGraph()
	a := BindSyntaxFile("file://a.php", []byte("<?php\nclass Foo {}\n"), "", nil)
	b := BindSyntaxFile("file://b.php", []byte("<?php\nfunction f(Foo $x) {}\n"), "", nil)
	g.PutFile("file://a.php", a.Uses)
	g.PutFile("file://b.php", b.Uses)
	hits := g.FindByResolved("Foo")
	if len(hits) < 2 {
		t.Fatalf("expected cross-file Foo uses, got %d: %+v", len(hits), hits)
	}
	uris := map[string]bool{}
	for _, h := range hits {
		uris[h.URI] = true
	}
	if !uris["file://a.php"] || !uris["file://b.php"] {
		t.Fatalf("expected both files in hits, got %v", uris)
	}
	g.RemoveFile("file://b.php")
	hits = g.FindByResolved("Foo")
	for _, h := range hits {
		if h.URI == "file://b.php" {
			t.Fatal("expected b.php uses removed")
		}
	}
}

func TestMultiNamespaceScopesDoNotShareAliases(t *testing.T) {
	src := `<?php
namespace A {
use X\Foo as Alias;
class C { function f(Alias $a) {} }
}
namespace B {
use Y\Foo as Alias;
class D { function g(Alias $a) {} }
}
`
	graph := BindFile("file://multi.php", []byte(src), BindModeReferences)
	var aAlias, bAlias []NameUse
	for _, u := range graph.Uses {
		if u.Written != "Alias" || u.Kind != "type" {
			continue
		}
		if strings.HasPrefix(u.Resolved, `X\`) {
			aAlias = append(aAlias, u)
		}
		if strings.HasPrefix(u.Resolved, `Y\`) {
			bAlias = append(bAlias, u)
		}
	}
	if len(aAlias) == 0 || len(bAlias) == 0 {
		t.Fatalf("expected Alias resolved per-namespace; uses=%+v", graph.Uses)
	}
	for _, u := range aAlias {
		if u.Resolved != `X\Foo` {
			t.Fatalf("namespace A Alias resolved to %q", u.Resolved)
		}
	}
	for _, u := range bAlias {
		if u.Resolved != `Y\Foo` {
			t.Fatalf("namespace B Alias resolved to %q", u.Resolved)
		}
	}
}

func TestProjectLookupDoesNotMixUnqualifiedSpellings(t *testing.T) {
	g := NewProjectUsageGraph()
	a := BindFile("file://a.php", []byte("<?php\nnamespace A;\nclass Foo {}\n"), BindModeReferences)
	b := BindFile("file://b.php", []byte("<?php\nnamespace B;\nclass Foo {}\n"), BindModeReferences)
	g.PutFile("file://a.php", a.Uses)
	g.PutFile("file://b.php", b.Uses)

	if hits := g.FindByName("Foo"); len(hits) != 0 {
		t.Fatalf("unqualified FindByName must not mix namespace Foo spellings, got %+v", hits)
	}
	aHits := g.FindByResolved(`A\Foo`)
	bHits := g.FindByResolved(`B\Foo`)
	if len(aHits) == 0 || len(bHits) == 0 {
		t.Fatalf("expected resolved hits; a=%d b=%d", len(aHits), len(bHits))
	}
	for _, h := range aHits {
		if strings.EqualFold(h.Resolved, `B\Foo`) {
			t.Fatal("A\\Foo lookup mixed B\\Foo")
		}
	}
	for _, h := range bHits {
		if strings.EqualFold(h.Resolved, `A\Foo`) {
			t.Fatal("B\\Foo lookup mixed A\\Foo")
		}
	}
}

func TestFindMatchingRespectsOwnerAndKind(t *testing.T) {
	src := `<?php
namespace App;
class Foo {
    public function bar() {}
}
class Other {
    public function bar() {}
}
`
	graph := BindFile("file://m.php", []byte(src), BindModeReferences)
	g := NewProjectUsageGraph()
	g.PutFile("file://m.php", graph.Uses)

	var fooBar NameUse
	found := false
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "bar" && u.Owner == `App\Foo` {
			fooBar = u
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected App\\Foo::bar use, got %+v", graph.Uses)
	}
	hits := g.FindMatching(fooBar)
	if len(hits) == 0 {
		t.Fatal("expected matching hits for Foo::bar")
	}
	for _, h := range hits {
		if h.Owner != `App\Foo` {
			t.Fatalf("cross-type false hit: %+v", h)
		}
		if h.Kind != "method" {
			t.Fatalf("kind mismatch: %+v", h)
		}
	}
}

func TestUseAtOffsetAndExactLeafSpan(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {}\n"
	graph := BindFile("file://c.php", []byte(src), BindModeReferences)
	var classUse NameUse
	found := false
	for _, u := range graph.Uses {
		if u.Kind == "class" && u.Written == "Foo" {
			classUse = u
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected class Foo use, got %+v", graph.Uses)
	}
	leaf := string([]byte(src)[classUse.Span.Start:classUse.Span.End])
	if leaf != "Foo" {
		t.Fatalf("exact edit span text %q, want Foo (span=%v)", leaf, classUse.Span)
	}
	mid := classUse.Span.Start + 1
	got, ok := UseAtOffset(graph.Uses, mid)
	if !ok || got.Resolved != `App\Foo` {
		t.Fatalf("UseAtOffset failed: ok=%v got=%+v", ok, got)
	}
}

func TestBindModeDeclarationVsReferences(t *testing.T) {
	src := `<?php
namespace App;
function outer(Foo $a) {
    bar(new Baz());
}
`
	decl := BindFile("file://d.php", []byte(src), BindModeDeclarations)
	refs := BindFile("file://d.php", []byte(src), BindModeReferences)
	declNames := map[string]bool{}
	for _, u := range decl.Uses {
		declNames[u.Written] = true
	}
	refNames := map[string]bool{}
	for _, u := range refs.Uses {
		refNames[u.Written] = true
	}
	if !declNames["Foo"] {
		t.Fatalf("declaration mode should bind signature type Foo; got %v", declNames)
	}
	if BindModeName(BindModeDeclarations) == BindModeName(BindModeReferences) {
		t.Fatal("BindMode names must distinguish declaration vs references")
	}
	_ = refNames
}

func TestUTF16PositionFromMultibyte(t *testing.T) {
	src := []byte("<?php\n//é\nclass Foo {}\n")
	graph := BindFile("file://u.php", src, BindModeReferences)
	var classUse NameUse
	for _, u := range graph.Uses {
		if u.Kind == "class" && u.Written == "Foo" {
			classUse = u
			break
		}
	}
	if classUse.Written == "" {
		t.Fatal("missing class use")
	}
	if classUse.StartLine != 2 {
		t.Fatalf("StartLine=%d want 2", classUse.StartLine)
	}
	line := strings.Split(string(src), "\n")[classUse.StartLine]
	prefix := line[:strings.Index(line, "Foo")]
	wantCol := 0
	for _, r := range prefix {
		wantCol += utf16.RuneLen(r)
	}
	if classUse.StartChar != wantCol {
		t.Fatalf("StartChar=%d want UTF-16 %d (prefix %q)", classUse.StartChar, wantCol, prefix)
	}
}

func TestImportAliasAsBinding(t *testing.T) {
	src := "<?php\nnamespace App;\nuse Vendor\\Thing as Alias;\nfunction f(Alias $x) {}\n"
	graph := BindFile("file://alias.php", []byte(src), BindModeReferences)
	found := false
	for _, u := range graph.Uses {
		if u.Written == "Alias" && u.Kind == "type" {
			if u.Resolved != `Vendor\Thing` {
				t.Fatalf("Alias resolved to %q", u.Resolved)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Alias type use, got %+v", graph.Uses)
	}
}

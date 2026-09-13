package analyse

import (
	"testing"

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
	hits := g.FindByName("Foo")
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
	hits = g.FindByName("Foo")
	for _, h := range hits {
		if h.URI == "file://b.php" {
			t.Fatal("expected b.php uses removed")
		}
	}
}


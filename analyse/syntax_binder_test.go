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
	aUses := g.UsesForURI("file://a.php")
	if len(aUses) == 0 {
		t.Fatal("UsesForURI(a) empty")
	}
	for _, u := range aUses {
		if u.URI != "file://a.php" {
			t.Fatalf("UsesForURI leaked other URI: %+v", u)
		}
	}
	if g.UsesForURI("file://missing.php") != nil {
		t.Fatal("expected nil for missing uri")
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
	assertFindMatchingScoped(t, `<?php
namespace App;
class Foo {
    public function bar() {}
}
class Other {
    public function bar() {}
}
`, "file://m.php", "method", "bar", `App\Foo`)
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

func TestBindSyntaxResultSharesParse(t *testing.T) {
	src := []byte("<?php\nclass Foo {}\nfunction f(Foo $x) {}\n")
	res := syntax.ParseForIndex(src)
	a := BindSyntaxResult("file://once.php", res)
	b := BindSyntaxFileForIndex("file://once.php", src, "", nil)
	if len(a.Uses) == 0 || len(a.Uses) != len(b.Uses) {
		t.Fatalf("BindSyntaxResult uses=%d BindSyntaxFileForIndex uses=%d", len(a.Uses), len(b.Uses))
	}
}

func findUse(uses []NameUse, kind, written, owner string) (NameUse, bool) {
	for _, u := range uses {
		if u.Kind == kind && u.Written == written && (owner == "" || u.Owner == owner) {
			return u, true
		}
	}
	return NameUse{}, false
}

func assertFindMatchingScoped(t *testing.T, src, uri, kind, written, owner string) {
	t.Helper()
	graph := BindFile(uri, []byte(src), BindModeReferences)
	g := NewProjectUsageGraph()
	g.PutFile(uri, graph.Uses)
	needle, ok := findUse(graph.Uses, kind, written, owner)
	if !ok {
		t.Fatalf("expected %s %q on %s, got %+v", kind, written, owner, graph.Uses)
	}
	hits := g.FindMatching(needle)
	if len(hits) == 0 {
		t.Fatalf("expected matching hits for %s::%s", owner, written)
	}
	for _, h := range hits {
		if h.Owner != owner {
			t.Fatalf("cross-type false hit: %+v", h)
		}
		if h.Kind != kind {
			t.Fatalf("kind mismatch: %+v", h)
		}
	}
}

func TestPropertyDeclBinding(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public string $prop;\n}\n"
	graph := BindFile("file://prop.php", []byte(src), BindModeReferences)
	u, ok := findUse(graph.Uses, "property", "$prop", `App\Foo`)
	if !ok {
		t.Fatalf("expected property $prop on App\\Foo, got %+v", graph.Uses)
	}
	if u.Resolved != `App\prop` {
		t.Fatalf("Resolved=%q want App\\prop", u.Resolved)
	}
	leaf := string([]byte(src)[u.Span.Start:u.Span.End])
	if leaf != "$prop" {
		t.Fatalf("exact leaf %q want $prop", leaf)
	}
}

func TestPropertyDeclGrouped(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public $a, $b;\n}\n"
	graph := BindFile("file://group.php", []byte(src), BindModeReferences)
	if _, ok := findUse(graph.Uses, "property", "$a", `App\Foo`); !ok {
		t.Fatalf("missing $a: %+v", graph.Uses)
	}
	if _, ok := findUse(graph.Uses, "property", "$b", `App\Foo`); !ok {
		t.Fatalf("missing $b: %+v", graph.Uses)
	}
}

func TestPropertyHooksDeclBindsNameOnly(t *testing.T) {
	src := "<?php\nnamespace App;\nclass C {\n    public string $x { get => $this->x; set => $this->x = $value; }\n}\n"
	graph := BindFile("file://hooks.php", []byte(src), BindModeReferences)
	declCount := 0
	useCount := 0
	for _, u := range graph.Uses {
		if u.Kind != "property" || u.Owner != `App\C` {
			continue
		}
		switch u.Written {
		case "$x":
			declCount++
		case "x":
			useCount++
		}
	}
	if declCount != 1 {
		t.Fatalf("want 1 property decl $x, got %d uses=%+v", declCount, graph.Uses)
	}
	if useCount < 1 {
		t.Fatalf("want at least one property use x from hooks, got %d", useCount)
	}
	for _, u := range graph.Uses {
		if u.Kind == "property" && u.Written == "$this" {
			t.Fatalf("must not bind $this as property decl: %+v", u)
		}
		if u.Kind == "property" && u.Written == "$value" {
			t.Fatalf("must not bind hook param $value as property decl: %+v", u)
		}
	}
}

func TestMemberAccessInstanceProperty(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public string $p;\n    public function m() { return $this->p; }\n}\n"
	graph := BindFile("file://mem.php", []byte(src), BindModeReferences)
	decl, ok := findUse(graph.Uses, "property", "$p", `App\Foo`)
	if !ok {
		t.Fatalf("missing decl: %+v", graph.Uses)
	}
	use, ok := findUse(graph.Uses, "property", "p", `App\Foo`)
	if !ok {
		t.Fatalf("missing use: %+v", graph.Uses)
	}
	if decl.Resolved != use.Resolved {
		t.Fatalf("decl Resolved %q != use Resolved %q", decl.Resolved, use.Resolved)
	}
}

func TestNullsafeMemberAccessProperty(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public string $p;\n    public function m(?Foo $o) { return $o?->p; }\n}\n"
	graph := BindFile("file://nullsafe.php", []byte(src), BindModeReferences)
	if _, ok := findUse(graph.Uses, "property", "p", `App\Foo`); !ok {
		t.Fatalf("missing nullsafe property use: %+v", graph.Uses)
	}
}

func TestStaticMemberAccessPropertyAndConst(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public static $stat;\n    public const K = 1;\n    public function m() { $a = self::$stat; $b = self::K; }\n}\n"
	graph := BindFile("file://static.php", []byte(src), BindModeReferences)
	if _, ok := findUse(graph.Uses, "property", "$stat", `App\Foo`); !ok {
		t.Fatalf("missing static property decl/use $stat: %+v", graph.Uses)
	}
	if _, ok := findUse(graph.Uses, "const", "K", `App\Foo`); !ok {
		t.Fatalf("missing const use K: %+v", graph.Uses)
	}
}

func TestMemberAccessMethodCall(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public function bar() {}\n    public function m() { $this->bar(); }\n}\n"
	graph := BindFile("file://call.php", []byte(src), BindModeReferences)
	bars := 0
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "bar" && u.Owner == `App\Foo` {
			bars++
		}
		if u.Kind == "property" && u.Written == "bar" {
			t.Fatalf("method call must not bind as property: %+v", u)
		}
		if u.Kind == "function" && u.Written == "bar" && u.Owner == `App\Foo` {
			t.Fatalf("class method must not double-bind as function: %+v", u)
		}
	}
	if bars < 2 {
		t.Fatalf("want decl+call for bar(), got %d: %+v", bars, graph.Uses)
	}
}

// TestReservedWordMethodDeclAndCallBind is a regression test for a real bug:
// PHP allows most reserved words as method names (e.g. `function declare()`),
// and the parser correctly wraps such names in UnqualifiedName - but
// NameText's token-type allowlist didn't recognize the keyword's own token
// kind (T_DECLARE, not T_STRING), so the method decl silently bound with an
// empty name. The call site (`$obj->declare()`) had the same bug one level
// down, in the binder's memberNameToken. Together these caused a real false
// "Call to an undefined method" diagnostic on ordinary code.
func TestReservedWordMethodDeclAndCallBind(t *testing.T) {
	src := "<?php\nnamespace App;\nclass Foo {\n    public function declare(string $n): string { return $n; }\n}\nclass Bar {\n    public function m(Foo $f) { $f->declare('x'); }\n}\n"
	graph := BindFile("file://reserved.php", []byte(src), BindModeReferences)
	decl, ok := findUse(graph.Uses, "method", "declare", `App\Foo`)
	if !ok || !decl.Declaration {
		t.Fatalf("missing method decl for declare(): %+v", graph.Uses)
	}
	calls := 0
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "declare" && !u.Declaration {
			calls++
			if u.Owner != `App\Foo` {
				t.Fatalf("call to declare() has wrong owner %q, want App\\Foo: %+v", u.Owner, u)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("want exactly 1 call use for declare(), got %d: %+v", calls, graph.Uses)
	}
}

func TestDynamicBracedStaticMemberWalksNestedPropertyRef(t *testing.T) {
	// Foo::{$m->p} must bind nested property p (same as a bare $m->p).
	// $m is typed Foo so the property use's owner resolves to App\Foo,
	// matching what a bare (typed-receiver) $m->p would resolve to; an
	// untyped $m has no owner to resolve regardless of the outer expression.
	src := `<?php
namespace App;
class Foo {
    public string $p;
    public function m(Foo $m) { return Foo::{$m->p}; }
}
`
	graph := BindFile("file://dyn-static.php", []byte(src), BindModeReferences)
	if _, ok := findUse(graph.Uses, "property", "p", `App\Foo`); !ok {
		t.Fatalf("expected nested property use p inside Foo::{$m->p}: %+v", graph.Uses)
	}
	// Dynamic slot itself must not bind as const/method/property named "{" or "$m".
	for _, u := range graph.Uses {
		if u.Written == "{" || u.Written == "$" {
			t.Fatalf("must not bind brace/dynamic slot as member: %+v", u)
		}
	}
}

func TestDynamicBracedInstanceMemberNoSpuriousPropertyUse(t *testing.T) {
	// $this->{$m} has no static member name — do not invent a property use on $p.
	src := `<?php
namespace App;
class Foo {
    public string $p;
    public function m($m) { return $this->{$m}; }
}
`
	graph := BindFile("file://dyn-inst.php", []byte(src), BindModeReferences)
	for _, u := range graph.Uses {
		if u.Kind == "property" && u.Written == "p" && u.Owner == `App\Foo` {
			// Decl is $p; a use would be written "p". Decl written is "$p".
			if u.Written == "p" {
				t.Fatalf("dynamic $this->{$m} must not bind property use p: %+v", u)
			}
		}
	}
	decl, ok := findUse(graph.Uses, "property", "$p", `App\Foo`)
	if !ok {
		t.Fatalf("missing property decl $p: %+v", graph.Uses)
	}
	_ = decl
}

func TestDynamicStaticCallBracedNoMemberBind(t *testing.T) {
	src := `<?php
namespace App;
class Foo {
    public function bar() {}
    public function m($m) { Foo::{$m}(); }
}
`
	graph := BindFile("file://dyn-call.php", []byte(src), BindModeReferences)
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "$m" {
			t.Fatalf("must not bind $m as method: %+v", u)
		}
		if u.Kind == "method" && u.Written == "m" && u.Resolved != `App\Foo::m` {
			// only the enclosing method decl/name is expected among "m"
		}
		if (u.Kind == "method" || u.Kind == "const" || u.Kind == "property") &&
			(u.Written == "{" || u.Written == "$" || u.Written == "$m") {
			t.Fatalf("dynamic Foo::{$m}() must not bind the braced slot: %+v", u)
		}
	}
	// Nested empty: still bind the declared bar method once (decl only).
	bars := 0
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "bar" && u.Owner == `App\Foo` {
			bars++
		}
	}
	if bars != 1 {
		t.Fatalf("want only bar() decl (no dynamic call bind), got %d: %+v", bars, graph.Uses)
	}
}

func TestFindMatchingRespectsOwnerAndKindProperty(t *testing.T) {
	assertFindMatchingScoped(t, `<?php
namespace App;
class Foo {
    public string $p;
    public function m() { return $this->p; }
}
class Other {
    public string $p;
}
`, "file://pmatch.php", "property", "p", `App\Foo`)
}

func TestFindMatchingClassLikeAllowsEmptyOwner(t *testing.T) {
	// Class decl is bound with Owner=""; uses inside methods set Owner to the
	// enclosing class. FindMatching must still join them on Resolved+Kind.
	src := `<?php
namespace App;
class Foo {}
class Bar {
    public function m() { return new Foo(); }
}
`
	graph := BindFile("file://owner.php", []byte(src), BindModeReferences)
	g := NewProjectUsageGraph()
	g.PutFile("file://owner.php", graph.Uses)

	var bodyUse NameUse
	found := false
	for _, u := range graph.Uses {
		if u.Written == "Foo" && u.Owner == `App\Bar` && isClassLikeBindKind(u.Kind) {
			bodyUse = u
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Foo use inside Bar method, got %+v", graph.Uses)
	}
	hits := g.FindMatching(bodyUse)
	sawDecl, sawUse := false, false
	for _, h := range hits {
		if h.Resolved != `App\Foo` {
			t.Fatalf("unexpected Resolved %+v", h)
		}
		if h.Owner == "" && (h.Kind == "class" || h.Kind == "name") {
			sawDecl = true
		}
		if h.Owner == `App\Bar` {
			sawUse = true
		}
	}
	if !sawDecl || !sawUse {
		t.Fatalf("want decl (Owner=\"\") + body use (Owner=App\\Bar); hits=%+v", hits)
	}

	// Members still require Owner: Bar::m must not match a same-named method on Foo.
	var barMethod NameUse
	for _, u := range graph.Uses {
		if u.Kind == "method" && u.Written == "m" && u.Owner == `App\Bar` {
			barMethod = u
			break
		}
	}
	if barMethod.Written == "" {
		t.Fatal("missing Bar::m")
	}
	for _, h := range g.FindMatching(barMethod) {
		if h.Owner != "" && h.Owner != `App\Bar` {
			t.Fatalf("method FindMatching leaked other owner: %+v", h)
		}
	}
}

func TestBindModeDeclarationsBindsPropertyNotBodyRefs(t *testing.T) {
	src := `<?php
namespace App;
class Foo {
    public string $p;
    public function m() { return $this->p; }
}
`
	decl := BindFile("file://mode.php", []byte(src), BindModeDeclarations)
	refs := BindFile("file://mode.php", []byte(src), BindModeReferences)
	if _, ok := findUse(decl.Uses, "property", "$p", `App\Foo`); !ok {
		t.Fatalf("declaration mode should bind property $p: %+v", decl.Uses)
	}
	if _, ok := findUse(decl.Uses, "property", "p", `App\Foo`); ok {
		t.Fatalf("declaration mode must not bind body ->p uses: %+v", decl.Uses)
	}
	if _, ok := findUse(refs.Uses, "property", "p", `App\Foo`); !ok {
		t.Fatalf("references mode should bind ->p: %+v", refs.Uses)
	}
}

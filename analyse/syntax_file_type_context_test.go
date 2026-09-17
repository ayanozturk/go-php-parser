package analyse

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// TestCollectFileTypeContextFromSyntaxMatchesASTPath is the Phase 2 parity
// guard: for every fixture, CollectFileTypeContextFromSyntax must agree with
// the existing ast.Node-based CollectFileTypeContext on Namespace, Aliases,
// FunctionAliases, ConstAliases, and Classes (name/extends/implements).
// ClassNodes and Constants are known, documented gaps (see
// syntax_file_type_context.go's doc comment) and are intentionally excluded
// from this comparison.
func TestCollectFileTypeContextFromSyntaxMatchesASTPath(t *testing.T) {
	fixtures := []string{
		// no namespace, single class
		`<?php
class Foo {}
`,
		// namespace + use aliases (class, function, const) + extends/implements
		`<?php
namespace App;

use App\Contracts\Countable as Cnt;
use function App\Helpers\format_name;
use const App\Config\MAX_SIZE;

interface Cnt2 {}

class Bar extends \App\Base implements Cnt2, \Traversable {
    public function m() {}
}
`,
		// braced namespace form
		`<?php
namespace App\Braced {
    use App\Other\Thing;

    class Baz extends Thing {}
}
`,
		// unbraced namespace form with multiple classes/interfaces after it
		`<?php
namespace App\Unbraced;

interface Shape {}
interface Named {}

class Circle implements Shape, Named {}

class Square extends Circle {}
`,
		// no namespace, interface extending multiple parents
		`<?php
interface A {}
interface B {}
interface C extends A, B {}
`,
		// relative/self-referential extends using a plain unqualified name
		`<?php
namespace App\Deep\Nested;

class Base {}
class Derived extends Base {}
`,
		// fully-qualified extends/implements (leading backslash)
		`<?php
namespace App;

class Widget extends \App\Base implements \App\Contracts\Renderable {}
`,
	}

	for i, src := range fixtures {
		src := src
		t.Run(t.Name()+"/"+string(rune('A'+i)), func(t *testing.T) {
			nodes, diags := syntax.ParseAST([]byte(src))
			if len(diags) != 0 {
				t.Fatalf("unexpected parse diagnostics: %v", diags)
			}
			res := syntax.Parse([]byte(src))

			astCtx := CollectFileTypeContext(nodes)
			cstCtx := CollectFileTypeContextFromSyntax(res.File.Root)

			if astCtx.Namespace != cstCtx.Namespace {
				t.Errorf("Namespace mismatch: ast=%q cst=%q", astCtx.Namespace, cstCtx.Namespace)
			}
			assertStringMapsEqual(t, "Aliases", astCtx.Aliases, cstCtx.Aliases)
			assertStringMapsEqual(t, "FunctionAliases", astCtx.FunctionAliases, cstCtx.FunctionAliases)
			assertStringMapsEqual(t, "ConstAliases", astCtx.ConstAliases, cstCtx.ConstAliases)
			assertResolvedClassMapsEqual(t, astCtx.Classes, cstCtx.Classes)
		})
	}
}

func assertStringMapsEqual(t *testing.T, label string, ast, cst map[string]string) {
	t.Helper()
	if len(ast) != len(cst) {
		t.Errorf("%s size mismatch: ast=%v cst=%v", label, ast, cst)
		return
	}
	for k, v := range ast {
		if cst[k] != v {
			t.Errorf("%s[%q] mismatch: ast=%q cst=%q", label, k, v, cst[k])
		}
	}
}

func assertResolvedClassMapsEqual(t *testing.T, ast, cst map[string]ResolvedClass) {
	t.Helper()
	if len(ast) != len(cst) {
		t.Errorf("Classes size mismatch: ast keys=%v cst keys=%v", sortedKeys(ast), sortedKeys(cst))
		return
	}
	for k, v := range ast {
		other, ok := cst[k]
		if !ok {
			t.Errorf("Classes[%q] missing from CST-direct result", k)
			continue
		}
		if v.Name != other.Name {
			t.Errorf("Classes[%q].Name mismatch: ast=%q cst=%q", k, v.Name, other.Name)
		}
		if !reflect.DeepEqual(v.Extends, other.Extends) {
			t.Errorf("Classes[%q].Extends mismatch: ast=%v cst=%v", k, v.Extends, other.Extends)
		}
		if !reflect.DeepEqual(v.Implements, other.Implements) {
			t.Errorf("Classes[%q].Implements mismatch: ast=%v cst=%v", k, v.Implements, other.Implements)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestCollectFileTypeContextFromSyntaxNilRoot ensures the nil-root guard
// returns an empty-but-initialized context rather than panicking.
func TestCollectFileTypeContextFromSyntaxNilRoot(t *testing.T) {
	ctx := CollectFileTypeContextFromSyntax(nil)
	if ctx.Namespace != "" {
		t.Fatalf("expected empty namespace for nil root, got %q", ctx.Namespace)
	}
	if len(ctx.Classes) != 0 {
		t.Fatalf("expected no classes for nil root, got %v", ctx.Classes)
	}
}

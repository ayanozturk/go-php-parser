package lower_test

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
	"github.com/ayanozturk/go-php-parser/syntax/lower"
)

const fixtureUser = `<?php
namespace App;
use Vendor\Base;
class User extends Base {
    public string $name;
    public function id(): int {}
    private function set_name(string $n): void {}
}
`

func TestLowerFileFixture(t *testing.T) {
	res := syntax.ParseForIndex([]byte(fixtureUser))
	nodes := lower.File(res.File.Root, res.File)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d %#v", len(nodes), nodes)
	}
	ns, ok := nodes[0].(*ast.NamespaceNode)
	if !ok {
		t.Fatalf("expected NamespaceNode, got %T", nodes[0])
	}
	if ns.Name != "App" {
		t.Fatalf("namespace name: got %q", ns.Name)
	}
	if len(ns.Body) < 2 {
		t.Fatalf("namespace body too short: %d", len(ns.Body))
	}
	use, ok := ns.Body[0].(*ast.UseNode)
	if !ok {
		t.Fatalf("expected UseNode, got %T", ns.Body[0])
	}
	if use.Path != `Vendor\Base` {
		t.Fatalf("use path: got %q", use.Path)
	}
	cls, ok := ns.Body[1].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", ns.Body[1])
	}
	if cls.Name != "User" || cls.Extends != "Base" {
		t.Fatalf("class: name=%q extends=%q", cls.Name, cls.Extends)
	}
	if len(cls.Properties) != 1 {
		t.Fatalf("properties: %d", len(cls.Properties))
	}
	prop := cls.Properties[0].(*ast.PropertyNode)
	if prop.Name != "name" || ast.TypeText(prop.TypeHint) != "string" {
		t.Fatalf("property: name=%q type=%q", prop.Name, ast.TypeText(prop.TypeHint))
	}
	if len(cls.Methods) != 2 {
		t.Fatalf("methods: %d", len(cls.Methods))
	}
	id := cls.Methods[0].(*ast.FunctionNode)
	if id.Name != "id" || ast.TypeText(id.ReturnType) != "int" {
		t.Fatalf("id: name=%q ret=%q", id.Name, ast.TypeText(id.ReturnType))
	}
	set := cls.Methods[1].(*ast.FunctionNode)
	if set.Name != "set_name" || ast.TypeText(set.ReturnType) != "void" {
		t.Fatalf("set_name: name=%q ret=%q", set.Name, ast.TypeText(set.ReturnType))
	}
	if len(set.Params) != 1 {
		t.Fatalf("set_name params: %d", len(set.Params))
	}
	param := set.Params[0].(*ast.ParamNode)
	if param.Name != "n" || ast.TypeText(param.TypeHint) != "string" {
		t.Fatalf("param: name=%q type=%q", param.Name, ast.TypeText(param.TypeHint))
	}
}

func TestLowerDoesNotPanicOnEnumTrait(t *testing.T) {
	src := []byte(`<?php
enum Suit { case Hearts; }
trait T { public function x(): void {} }
class C {}
`)
	res := syntax.ParseForIndex(src)
	nodes := lower.File(res.File.Root, res.File)
	foundClass := false
	foundEnum := false
	foundTrait := false
	for _, n := range nodes {
		switch x := n.(type) {
		case *ast.ClassNode:
			if x.Name == "C" {
				foundClass = true
			}
		case *ast.EnumNode:
			if x.Name == "Suit" && len(x.Cases) == 1 && x.Cases[0].Name == "Hearts" {
				foundEnum = true
			}
		case *ast.TraitNode:
			if x.Name != nil && x.Name.Name == "T" {
				foundTrait = true
			}
		}
	}
	if !foundClass || !foundEnum || !foundTrait {
		t.Fatalf("expected class/enum/trait among %v (class=%v enum=%v trait=%v)", nodes, foundClass, foundEnum, foundTrait)
	}
}

func TestLowerPHPDocOnMethod(t *testing.T) {
	src := []byte(`<?php
class C {
    /**
     * @return int
     */
    public function m() {}
}
`)
	res := syntax.ParseForIndex(src)
	nodes := lower.File(res.File.Root, res.File)
	cls := nodes[0].(*ast.ClassNode)
	fn := cls.Methods[0].(*ast.FunctionNode)
	if fn.PHPDoc == nil || fn.PHPDoc.ReturnType != "int" {
		t.Fatalf("expected @return int PHPDoc, got %+v", fn.PHPDoc)
	}
}

func TestLowerPHPDocOnAttributedMethod(t *testing.T) {
	cases := []struct {
		name string
		src  string
		get  func(ast.Node) (*ast.PHPDocNode, []ast.Node)
	}{
		{
			name: "class",
			src:  "class C { /** @return int */ #[Example] public function m() {} }",
			get: func(node ast.Node) (*ast.PHPDocNode, []ast.Node) {
				fn := node.(*ast.ClassNode).Methods[0].(*ast.FunctionNode)
				return fn.PHPDoc, fn.Attributes
			},
		},
		{
			name: "trait",
			src:  "trait T { /** @return int */ #[Example] public function m() {} }",
			get: func(node ast.Node) (*ast.PHPDocNode, []ast.Node) {
				fn := node.(*ast.TraitNode).Body[0].(*ast.FunctionNode)
				return fn.PHPDoc, fn.Attributes
			},
		},
		{
			name: "enum",
			src:  "enum E { /** @return int */ #[Example] public function m() {} }",
			get: func(node ast.Node) (*ast.PHPDocNode, []ast.Node) {
				fn := node.(*ast.EnumNode).Methods[0].(*ast.FunctionNode)
				return fn.PHPDoc, fn.Attributes
			},
		},
		{
			name: "interface",
			src:  "interface I { /** @return int */ #[Example] public function m(); }",
			get: func(node ast.Node) (*ast.PHPDocNode, []ast.Node) {
				fn := node.(*ast.InterfaceNode).Members[0].(*ast.InterfaceMethodNode)
				return fn.PHPDoc, fn.Attributes
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := syntax.ParseForIndex([]byte("<?php " + tc.src))
			nodes := lower.File(res.File.Root, res.File)
			doc, attrs := tc.get(nodes[0])
			if doc == nil || doc.ReturnType != "int" {
				t.Fatalf("expected @return int PHPDoc through attribute, got %+v", doc)
			}
			if len(attrs) != 1 {
				t.Fatalf("expected one method attribute, got %d", len(attrs))
			}
		})
	}
}

func TestLowerPropertyHooks(t *testing.T) {
	src := []byte(`<?php
class H {
    public string $name {
        get => $this->name;
        set(string $v) { $this->name = $v; }
    }
}
`)
	// Hook CST always carries full expr/stmt trees (SkipFunctionBodies does not
	// blob hooks), so ParseForIndex still lowers Expr/Body when present.
	res := syntax.ParseForIndex(src)
	nodes := lower.File(res.File.Root, res.File)
	cls := nodes[0].(*ast.ClassNode)
	prop := cls.Properties[0].(*ast.PropertyNode)
	if len(prop.Hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(prop.Hooks))
	}
	if prop.Hooks[0].Name != "get" || prop.Hooks[1].Name != "set" {
		t.Fatalf("hook names: %+v", prop.Hooks)
	}
	get, set := prop.Hooks[0], prop.Hooks[1]
	if get.Expr == nil || get.Body != nil {
		t.Fatalf("get hook: want Expr set and Body nil, got Expr=%T Body=%d", get.Expr, len(get.Body))
	}
	if _, ok := get.Expr.(*ast.PropertyFetchNode); !ok {
		t.Fatalf("get Expr type: %T want PropertyFetchNode", get.Expr)
	}
	if set.Parameter == "" || !strings.Contains(set.Parameter, "$v") {
		t.Fatalf("set Parameter: %q", set.Parameter)
	}
	if set.Expr != nil || len(set.Body) == 0 {
		t.Fatalf("set hook: want Body stmts and Expr nil, got Expr=%T Body=%d", set.Expr, len(set.Body))
	}
}

func TestLowerTraitUseInClass(t *testing.T) {
	src := []byte(`<?php
class C {
    use T1, T2;
    public int $x;
}
`)
	res := syntax.ParseForIndex(src)
	nodes := lower.File(res.File.Root, res.File)
	cls := nodes[0].(*ast.ClassNode)
	if len(cls.Properties) < 2 {
		t.Fatalf("expected trait use + property, got %d", len(cls.Properties))
	}
	tu, ok := cls.Properties[0].(*ast.TraitUseNode)
	if !ok {
		t.Fatalf("expected TraitUseNode first, got %T", cls.Properties[0])
	}
	if len(tu.Traits) != 2 {
		t.Fatalf("traits: %v", tu.Traits)
	}
}

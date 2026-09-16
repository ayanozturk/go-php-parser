package lower_test

import (
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
	for _, n := range nodes {
		if c, ok := n.(*ast.ClassNode); ok && c.Name == "C" {
			foundClass = true
		}
	}
	if !foundClass {
		t.Fatalf("expected class C among %v", nodes)
	}
}

package syntax_test

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"github.com/ayanozturk/go-php-parser/syntax"
)

const indexFixture = `<?php
namespace App;
use Vendor\Base;
class User extends Base {
    public string $name;
    public function id(): int {}
    private function set_name(string $n): void {}
}
`

func TestParseASTForIndexFixture(t *testing.T) {
	nodes, diags := syntax.ParseASTForIndex([]byte(indexFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	ns := nodes[0].(*ast.NamespaceNode)
	if ns.Name != "App" {
		t.Fatalf("namespace %q", ns.Name)
	}
	cls := ns.Body[1].(*ast.ClassNode)
	if cls.Name != "User" || cls.Extends != "Base" {
		t.Fatalf("class %+v", cls)
	}
}

func TestParseASTForIndexProjectIndexParity(t *testing.T) {
	src := indexFixture
	classic := parser.New(lexer.New(src), false)
	classic.SkipFunctionBodies = true
	nodesC := classic.Parse()
	nodesS, _ := syntax.ParseASTForIndex([]byte(src))

	idxC := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesC})
	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	classC, okC := idxC.ResolveClass(`App\User`)
	classS, okS := idxS.ResolveClass(`App\User`)
	if !okC || !okS {
		t.Fatalf("ResolveClass: classic=%v syntax=%v", okC, okS)
	}
	if classC.Name != classS.Name {
		t.Fatalf("class name classic=%q syntax=%q", classC.Name, classS.Name)
	}
	if len(classC.Extends) != len(classS.Extends) || (len(classS.Extends) > 0 && classC.Extends[0] != classS.Extends[0]) {
		t.Fatalf("extends classic=%v syntax=%v", classC.Extends, classS.Extends)
	}

	methodC, okC := idxC.ResolveMethod(`App\User`, "id")
	methodS, okS := idxS.ResolveMethod(`App\User`, "id")
	if !okC || !okS {
		t.Fatalf("ResolveMethod id: classic=%v syntax=%v", okC, okS)
	}
	if methodC.ReturnType != methodS.ReturnType {
		t.Fatalf("id return classic=%q syntax=%q", methodC.ReturnType, methodS.ReturnType)
	}

	setC, okC := idxC.ResolveMethod(`App\User`, "set_name")
	setS, okS := idxS.ResolveMethod(`App\User`, "set_name")
	if !okC || !okS {
		t.Fatalf("ResolveMethod set_name: classic=%v syntax=%v", okC, okS)
	}
	if setC.ReturnType != setS.ReturnType {
		t.Fatalf("set_name return classic=%q syntax=%q", setC.ReturnType, setS.ReturnType)
	}
	if len(setC.Params) != 1 || len(setS.Params) != 1 {
		t.Fatalf("set_name params classic=%d syntax=%d", len(setC.Params), len(setS.Params))
	}
	if setC.Params[0].Name != setS.Params[0].Name || setC.Params[0].Type != setS.Params[0].Type {
		t.Fatalf("param classic=%+v syntax=%+v", setC.Params[0], setS.Params[0])
	}

	propC, okC := idxC.ResolveProperty(`App\User`, "name")
	propS, okS := idxS.ResolveProperty(`App\User`, "name")
	if !okC || !okS {
		t.Fatalf("ResolveProperty: classic=%v syntax=%v", okC, okS)
	}
	if propC.Type != propS.Type {
		t.Fatalf("property type classic=%q syntax=%q", propC.Type, propS.Type)
	}
}

func TestParseASTDoesNotPanicOnEnumTrait(t *testing.T) {
	src := []byte("<?php\nenum E { case A; }\ntrait T {}\ninterface I { public function f(): void; }\n")
	nodes, _ := syntax.ParseASTForIndex(src)
	foundIface := false
	for _, n := range nodes {
		if _, ok := n.(*ast.InterfaceNode); ok {
			foundIface = true
		}
	}
	if !foundIface {
		t.Fatalf("expected InterfaceNode in %v", nodes)
	}
}

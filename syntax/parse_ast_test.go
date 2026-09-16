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
	foundEnum := false
	foundTrait := false
	for _, n := range nodes {
		switch n.(type) {
		case *ast.InterfaceNode:
			foundIface = true
		case *ast.EnumNode:
			foundEnum = true
		case *ast.TraitNode:
			foundTrait = true
		}
	}
	if !foundIface || !foundEnum || !foundTrait {
		t.Fatalf("expected Interface/Enum/Trait in %v (i=%v e=%v t=%v)", nodes, foundIface, foundEnum, foundTrait)
	}
}

func TestParseASTForIndexPHPDocReturnParity(t *testing.T) {
	src := `<?php
class C {
    /**
     * @return int
     */
    public function m() {}
}
`
	classic := parser.New(lexer.New(src), false)
	classic.SkipFunctionBodies = true
	nodesC := classic.Parse()
	nodesS, _ := syntax.ParseASTForIndex([]byte(src))

	idxC := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesC})
	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	methodC, okC := idxC.ResolveMethod(`C`, "m")
	methodS, okS := idxS.ResolveMethod(`C`, "m")
	if !okC || !okS {
		t.Fatalf("ResolveMethod: classic=%v syntax=%v", okC, okS)
	}
	if methodC.ReturnType != methodS.ReturnType {
		t.Fatalf("return classic=%q syntax=%q", methodC.ReturnType, methodS.ReturnType)
	}
	if methodS.ReturnType != "int" {
		t.Fatalf("expected PHPDoc return int, got %q", methodS.ReturnType)
	}
}

func TestParseASTForIndexEnumTraitKindParity(t *testing.T) {
	src := `<?php
enum Suit { case Hearts; }
trait Logger { public function log(): void {} }
`
	classic := parser.New(lexer.New(src), false)
	classic.SkipFunctionBodies = true
	nodesC := classic.Parse()
	nodesS, _ := syntax.ParseASTForIndex([]byte(src))

	idxC := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesC})
	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	enumC, okC := idxC.ResolveClass("Suit")
	enumS, okS := idxS.ResolveClass("Suit")
	if !okC || !okS {
		t.Fatalf("ResolveClass Suit: classic=%v syntax=%v", okC, okS)
	}
	if enumC.Kind != enumS.Kind || enumS.Kind != "enum" {
		t.Fatalf("Suit kind classic=%q syntax=%q", enumC.Kind, enumS.Kind)
	}

	traitC, okC := idxC.ResolveClass("Logger")
	traitS, okS := idxS.ResolveClass("Logger")
	if !okC || !okS {
		t.Fatalf("ResolveClass Logger: classic=%v syntax=%v", okC, okS)
	}
	if traitC.Kind != traitS.Kind || traitS.Kind != "trait" {
		t.Fatalf("Logger kind classic=%q syntax=%q", traitC.Kind, traitS.Kind)
	}
}

func TestParseASTForIndexPropertyHooks(t *testing.T) {
	src := []byte(`<?php
class H {
    public string $name {
        get => $this->name;
        set(string $v) { $this->name = $v; }
    }
}
`)
	nodes, diags := syntax.ParseASTForIndex(src)
	if len(diags) != 0 {
		t.Fatalf("diags: %v", diags)
	}
	cls := nodes[0].(*ast.ClassNode)
	prop := cls.Properties[0].(*ast.PropertyNode)
	if len(prop.Hooks) != 2 {
		t.Fatalf("hooks=%d", len(prop.Hooks))
	}
}

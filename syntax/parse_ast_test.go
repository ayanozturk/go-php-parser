package syntax_test

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
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
	nodesS, diags := syntax.ParseASTForIndex([]byte(src))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	classS, okS := idxS.ResolveClass(`App\User`)
	if !okS {
		t.Fatalf("ResolveClass App\\User: ok=%v", okS)
	}
	if classS.Name != `App\User` {
		t.Fatalf("class name=%q want App\\User", classS.Name)
	}
	if len(classS.Extends) != 1 || classS.Extends[0] != `Vendor\Base` {
		t.Fatalf("extends=%v want [Vendor\\Base]", classS.Extends)
	}

	methodS, okS := idxS.ResolveMethod(`App\User`, "id")
	if !okS {
		t.Fatalf("ResolveMethod id: ok=%v", okS)
	}
	if methodS.ReturnType != "int" {
		t.Fatalf("id return=%q want int", methodS.ReturnType)
	}

	setS, okS := idxS.ResolveMethod(`App\User`, "set_name")
	if !okS {
		t.Fatalf("ResolveMethod set_name: ok=%v", okS)
	}
	if setS.ReturnType != "void" {
		t.Fatalf("set_name return=%q want void", setS.ReturnType)
	}
	if len(setS.Params) != 1 {
		t.Fatalf("set_name params=%d want 1", len(setS.Params))
	}
	if setS.Params[0].Name != "n" || setS.Params[0].Type != "string" {
		t.Fatalf("param=%+v want {Name:n Type:string}", setS.Params[0])
	}

	propS, okS := idxS.ResolveProperty(`App\User`, "name")
	if !okS {
		t.Fatalf("ResolveProperty name: ok=%v", okS)
	}
	if propS.Type != "string" {
		t.Fatalf("property type=%q want string", propS.Type)
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
	nodesS, diags := syntax.ParseASTForIndex([]byte(src))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	methodS, okS := idxS.ResolveMethod(`C`, "m")
	if !okS {
		t.Fatalf("ResolveMethod: ok=%v", okS)
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
	nodesS, diags := syntax.ParseASTForIndex([]byte(src))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	enumS, okS := idxS.ResolveClass("Suit")
	if !okS {
		t.Fatalf("ResolveClass Suit: ok=%v", okS)
	}
	if enumS.Kind != "enum" {
		t.Fatalf("Suit kind=%q want enum", enumS.Kind)
	}

	traitS, okS := idxS.ResolveClass("Logger")
	if !okS {
		t.Fatalf("ResolveClass Logger: ok=%v", okS)
	}
	if traitS.Kind != "trait" {
		t.Fatalf("Logger kind=%q want trait", traitS.Kind)
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

func TestParseASTForIndexKeepsMethodsAfterEncapsedCurly(t *testing.T) {
	// SkipFunctionBodies must not treat "{$var}" / ${var} / heredoc {$var}
	// closers as method-body ends, or later methods escape as top-level functions.
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "double-quoted curly",
			src: `<?php
namespace App\Tests;
final class EmployeeCalendarDataTest {
    public function testSomething(): void {
        $label = 'day';
        $msg = "label {$label}";
        $this->createDayAvailabilityData();
    }
    private function createDayAvailabilityData(): void {}
}
`,
		},
		{
			name: "dollar-curly",
			src: `<?php
namespace App\Tests;
final class EmployeeCalendarDataTest {
    public function testSomething(): void {
        $label = 'day';
        $msg = "label ${label}";
        $this->createDayAvailabilityData();
    }
    private function createDayAvailabilityData(): void {}
}
`,
		},
		{
			name: "heredoc curly",
			src: "<?php\nnamespace App\\Tests;\nfinal class EmployeeCalendarDataTest {\n" +
				"    public function testSomething(): void {\n" +
				"        $label = 'day';\n" +
				"        $msg = <<<TXT\nlabel {$label}\nTXT;\n" +
				"        $this->createDayAvailabilityData();\n" +
				"    }\n" +
				"    private function createDayAvailabilityData(): void {}\n" +
				"}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, diags := syntax.ParseASTForIndex([]byte(tc.src))
			if len(diags) != 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			idx := analyse.BuildProjectIndex(map[string][]ast.Node{"t.php": nodes})
			if _, ok := idx.ResolveMethod(`App\Tests\EmployeeCalendarDataTest`, "createDayAvailabilityData"); !ok {
				t.Fatalf("helper must remain a class method under ParseASTForIndex")
			}
			if _, ok := idx.ResolveFunction(`App\Tests\createDayAvailabilityData`); ok {
				t.Fatalf("helper must not be indexed as a namespaced function")
			}
			// Identity still holds for the skip-bodies green tree.
			tree := syntax.ParseForIndex([]byte(tc.src))
			if got := syntax.Print(tree.File.Root); got != tc.src {
				t.Fatalf("identity mismatch\ngot:\n%s\nwant:\n%s", got, tc.src)
			}
		})
	}
}


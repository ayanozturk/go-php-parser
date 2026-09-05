package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func TestInferHoverTargetAtPositionIgnoresNewKeyword(t *testing.T) {
	php := `<?php
class Builder {}
class BuilderTest {
    public function testArguments(): void
    {
        $configuration = (new Builder)->fromParameters(['command', 'argument']);
    }
}`
	l := lexer.New(php)
	p := parser.New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if target, ok := InferHoverTargetAtPosition(nodes, 5, 27, "new", nil); ok {
		t.Fatalf("expected no hover target for new keyword, got %#v", target)
	}
}

func TestInferHoverTypeBindsGenericAssignmentAndNarrowsAfterTerminatingNullGuard(t *testing.T) {
	php := `<?php
namespace App;

class Record {}

/** @template T */
abstract class GenericStore {
    /** @return T|null */
    public function lookup(string $id): ?object {}
}
/** @extends GenericStore<Record> */
class RecordStore extends GenericStore {}
class RecordProcessor {
    public function process(Record $record): void {}
}
class Controller {
    private RecordStore $store;
    private RecordProcessor $processor;
    public function run(string $id): void {
        $record = $this->store->lookup($id);
        if (!$record) {
            throw new \RuntimeException();
        }
        $this->processor->process($record);
    }
}`
	nodes := parseHoverFixture(t, php)
	project := BuildProjectIndex(map[string][]ast.Node{"test.php": nodes})
	ctx := &AnalysisContext{Resolver: project}

	assigned, ok := InferHoverTargetAtPosition(nodes, 20, 10, "record", ctx)
	if !ok || assigned.Type != `App\Record|null` {
		t.Fatalf("expected generic assignment hover type App\\Record|null, got %#v, %t", assigned, ok)
	}
	afterGuard, ok := InferHoverTargetAtPosition(nodes, 24, 31, "record", ctx)
	if !ok || afterGuard.Type != `App\Record` {
		t.Fatalf("expected non-null hover type after terminating guard, got %#v, %t", afterGuard, ok)
	}
}

func TestInferHoverTypeResolvesInheritedMethodOnCurrentObject(t *testing.T) {
	basePHP := `<?php
namespace Web;
use Domain\Member;
abstract class BaseEndpoint {
    protected function currentMember(): Member {}
}`
	childPHP := `<?php
namespace Web;
class TeamEndpoint extends BaseEndpoint {
    public function run(): void {
        $member = $this->currentMember();
    }
}`
	baseNodes := parseHoverFixture(t, basePHP)
	childNodes := parseHoverFixture(t, childPHP)
	project := BuildProjectIndex(map[string][]ast.Node{
		"base.php":  baseNodes,
		"child.php": childNodes,
	})
	ctx := &AnalysisContext{Resolver: project}

	target, ok := InferHoverTargetAtPosition(childNodes, 5, 10, "member", ctx)
	if !ok || target.Type != `Domain\Member` {
		t.Fatalf("expected inherited current-object method return type, got %#v, %t", target, ok)
	}
}

func TestInferHoverTargetAtPositionWithFilenameUsesSemanticFact(t *testing.T) {
	const filename = "src/Example.php"
	nodes := parseHoverFixture(t, `<?php
function run(): void {
    $value = "raw";
}
`)
	function := nodes[0].(*ast.FunctionNode)
	assignment := function.Body[0].(*ast.ExpressionStmt).Expr.(*ast.AssignmentNode)
	variable := assignment.Left.(*ast.VariableNode)
	key := inferredTypeFactKey(filename, variable)
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{key: {Key: key, Type: "int"}}}
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx := &AnalysisContext{Resolver: project, Facts: reader}
	pos := variable.GetPos()

	target, ok := InferHoverTargetAtPositionWithFilename(nodes, filename, pos.Line, pos.Column, variable.Name, ctx)
	if !ok || target.Type != "int" || target.Kind != HoverTargetVariable {
		t.Fatalf("expected filename-aware hover to use int fact, got %#v, %v", target, ok)
	}
	if typ, ok := InferTypeAtPositionWithFilename(nodes, filename, pos.Line, pos.Column, variable.Name, ctx); !ok || typ != "int" {
		t.Fatalf("expected filename-aware type API to use int fact, got %q, %v", typ, ok)
	}
	legacy, ok := InferHoverTargetAtPosition(nodes, pos.Line, pos.Column, variable.Name, ctx)
	if !ok || legacy.Type != "string" {
		t.Fatalf("expected legacy hover API to retain string fallback inference, got %#v, %v", legacy, ok)
	}
}

func TestInferHoverTypeResolvesBackedEnumNativeProperties(t *testing.T) {
	php := `<?php
namespace App\Module\Subscription\Enum;

enum InvoiceStatus: int {
    case PENDING = 1;
    case PAID = 2;

    public function isPending(): bool {
        return $this->value === self::PENDING->value;
    }
}`
	nodes := parseHoverFixture(t, php)
	project := BuildProjectIndex(map[string][]ast.Node{"invoice.php": nodes})
	ctx := &AnalysisContext{Resolver: project}

	lines := strings.Split(php, "\n")
	line := 0
	for i, text := range lines {
		if strings.Contains(text, "$this->value") {
			line = i + 1
			break
		}
	}
	if line == 0 {
		t.Fatal("expected $this->value in fixture")
	}
	thisValueCol := strings.Index(lines[line-1], "value") + 1
	caseValueCol := strings.LastIndex(lines[line-1], "value") + 1

	thisValue, ok := InferHoverTargetAtPosition(nodes, line, thisValueCol, "value", ctx)
	if !ok || thisValue.Kind != HoverTargetProperty || thisValue.Type != "int" || thisValue.ReceiverClass != `App\Module\Subscription\Enum\InvoiceStatus` {
		t.Fatalf("expected $this->value hover as int on InvoiceStatus, got %#v, %v", thisValue, ok)
	}
	caseValue, ok := InferHoverTargetAtPosition(nodes, line, caseValueCol, "value", ctx)
	if !ok || caseValue.Kind != HoverTargetProperty || caseValue.Type != "int" || caseValue.ReceiverClass != `App\Module\Subscription\Enum\InvoiceStatus` {
		t.Fatalf("expected enum-case ->value hover as int on InvoiceStatus, got %#v, %v", caseValue, ok)
	}
}

func parseHoverFixture(t *testing.T, php string) []ast.Node {
	t.Helper()
	l := lexer.New(php)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return nodes
}

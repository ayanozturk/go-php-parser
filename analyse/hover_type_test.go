package analyse

import (
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
	ctx := &AnalysisContext{Resolver: project, Project: project}

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
	ctx := &AnalysisContext{Resolver: project, Project: project}

	target, ok := InferHoverTargetAtPosition(childNodes, 5, 10, "member", ctx)
	if !ok || target.Type != `Domain\Member` {
		t.Fatalf("expected inherited current-object method return type, got %#v, %t", target, ok)
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

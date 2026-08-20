package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"testing"
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
	l := lexer.New(php)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
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

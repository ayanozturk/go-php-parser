package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

type returnCompletenessMissingFlow struct{}

func (returnCompletenessMissingFlow) StatementReachable(FlowStatementKey) (bool, bool) {
	return false, false
}

func (returnCompletenessMissingFlow) ScopeMayFallThrough(FlowScopeKey) (bool, bool) {
	return false, false
}

func (returnCompletenessMissingFlow) ControlFlowGraph(FlowScopeKey) (ControlFlowGraph, bool) {
	return ControlFlowGraph{}, false
}

type returnCompletenessKnownFlow struct {
	mayFallThrough bool
}

func (returnCompletenessKnownFlow) StatementReachable(FlowStatementKey) (bool, bool) {
	return false, false
}

func (f returnCompletenessKnownFlow) ScopeMayFallThrough(FlowScopeKey) (bool, bool) {
	return f.mayFallThrough, true
}

func (returnCompletenessKnownFlow) ControlFlowGraph(FlowScopeKey) (ControlFlowGraph, bool) {
	return ControlFlowGraph{}, false
}

func parseReturnCompletenessPHP(t *testing.T, code string) []ast.Node {
	t.Helper()
	p := parser.New(lexer.New(code), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse PHP: %v", errs)
	}
	return nodes
}

func returnCompletenessIssues(t *testing.T, code string, flow FlowGraphReader) []AnalysisIssue {
	t.Helper()
	const filename = "return-completeness.php"
	nodes := parseReturnCompletenessPHP(t, code)
	ctx := &AnalysisContext{Flow: flow}
	if flow == nil {
		snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
		if err != nil {
			t.Fatalf("build semantic snapshot: %v", err)
		}
		ctx = snapshot.NewAnalysisContext()
	}

	issues := (&ReturnTypeRule{}).CheckIssues(nodes, filename, ctx)
	completeness := make([]AnalysisIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "A.RETURN.TYPE" && strings.Contains(issue.Message, "not all paths return a value") {
			completeness = append(completeness, issue)
		}
	}
	return completeness
}

func TestReturnCompletenessReportsEveryDeclaredNonVoidFallthrough(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function integerResult(): int {}
function nullableResult(): ?int {}
function mixedResult(): mixed {}
function neverResult(): never {}
`, nil)

	if len(issues) != 4 {
		t.Fatalf("completeness issues = %#v; want one for each non-void declaration", issues)
	}
	for _, want := range []string{
		"Function integerResult: declared return type int but not all paths return a value",
		"Function nullableResult: declared return type int|null but not all paths return a value",
		"Function mixedResult: declared return type mixed but not all paths return a value",
		"Function neverResult: declared return type never but not all paths return a value",
	} {
		found := false
		for _, issue := range issues {
			if issue.Message == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing issue %q in %#v", want, issues)
		}
	}
}

func TestReturnCompletenessDistinguishesExhaustiveAndPartialBranches(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function exhaustive(bool $first, bool $second): int {
    if ($first) {
        return 1;
    } elseif ($second) {
        return 2;
    } else {
        return 3;
    }
}

function partial(bool $condition): int {
    if ($condition) {
        return 1;
    }
}
`, nil)

	if len(issues) != 1 || !strings.Contains(issues[0].Message, "Function partial:") {
		t.Fatalf("completeness issues = %#v; want only partial()", issues)
	}
}

func TestReturnCompletenessSkipsVoidAbstractInterfaceAndGeneratorFunctions(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function sideEffect(): void {}

abstract class AbstractRepository {
    abstract public function load(): int;
}

interface Repository {
    public function find(): int;
}

function values(): iterable {
    yield 1;
}
`, nil)

	if len(issues) != 0 {
		t.Fatalf("unexpected completeness issues for excluded declarations: %#v", issues)
	}
}

func TestReturnCompletenessTreatsThrowAndExitAsTermination(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function failure(): int {
    throw new RuntimeException();
}

function terminate(): string {
    exit(1);
}

function terminateBare(): string {
    die;
}
`, nil)

	if len(issues) != 0 {
		t.Fatalf("unexpected completeness issues for terminating functions: %#v", issues)
	}
}

func TestReturnCompletenessChecksMethodsAndClosures(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
class Service {
    public function load(): int {}
}

$callback = function (): string {};
`, nil)

	if len(issues) != 2 {
		t.Fatalf("completeness issues = %#v; want method and closure issues", issues)
	}
	wants := []string{"Function load:", "Function closure:"}
	for _, want := range wants {
		found := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %q issue in %#v", want, issues)
		}
	}
}

func TestReturnCompletenessFallsBackWhenGraphScopeIsMissing(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function exhaustive(bool $condition): int {
    if ($condition) {
        return 1;
    } else {
        throw new RuntimeException();
    }
}

function partial(bool $condition): int {
    if ($condition) {
        return 1;
    }
}
`, returnCompletenessMissingFlow{})

	if len(issues) != 1 || !strings.Contains(issues[0].Message, "Function partial:") {
		t.Fatalf("fallback completeness issues = %#v; want only partial()", issues)
	}
}

func TestReturnCompletenessUsesKnownGraphFallthrough(t *testing.T) {
	const function = `<?php function result(): int {}`
	if issues := returnCompletenessIssues(t, function, returnCompletenessKnownFlow{mayFallThrough: false}); len(issues) != 0 {
		t.Fatalf("known non-fallthrough graph produced issues: %#v", issues)
	}
	if issues := returnCompletenessIssues(t, function, returnCompletenessKnownFlow{mayFallThrough: true}); len(issues) != 1 {
		t.Fatalf("known fallthrough graph produced issues = %#v; want one", issues)
	}
}

func TestNestedGeneratorDoesNotExemptEnclosingFunction(t *testing.T) {
	issues := returnCompletenessIssues(t, `<?php
function outer(): int {
    $generator = function (): iterable {
        yield 1;
    };
}
`, nil)

	if len(issues) != 1 || !strings.Contains(issues[0].Message, "Function outer:") {
		t.Fatalf("completeness issues = %#v; want only outer()", issues)
	}
}

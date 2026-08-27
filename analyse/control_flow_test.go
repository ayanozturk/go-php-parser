package analyse

import (
	"reflect"
	"sync"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func parseControlFlowPHP(t *testing.T, source string) []ast.Node {
	t.Helper()
	p := parser.New(lexer.New(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse control-flow fixture: %v", errs)
	}
	return nodes
}

func controlFlowUnreachableIssues(t *testing.T, source string) []AnalysisIssue {
	t.Helper()
	const filename = "control-flow.php"
	nodes := parseControlFlowPHP(t, source)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	return unreachableIssues(RunAnalysisRulesWithContext(filename, nodes, snapshot.NewAnalysisContext()))
}

func unreachableIssues(issues []AnalysisIssue) []AnalysisIssue {
	filtered := make([]AnalysisIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "Generic.CodeAnalysis.UnreachableCode" {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func TestControlFlowExhaustiveIfElseIfElseTerminates(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function choose($first, $second): void {
    if ($first) {
        return;
    } elseif ($second) {
        throw new RuntimeException();
    } else {
        exit();
    }

    $unreachable = true;
}`)

	if len(issues) != 1 || issues[0].Line != 11 {
		t.Fatalf("expected only line 11 to be unreachable, got %#v", issues)
	}
}

func TestControlFlowIfElseIfWithoutElseMayFallThrough(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function choose($first, $second): void {
    if ($first) {
        return;
    } elseif ($second) {
        throw new RuntimeException();
    }

    $reachable = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected the missing else path to fall through, got %#v", issues)
	}
}

func TestControlFlowNestedExhaustiveBranchMarksFollowingStatementsUnreachable(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function nested($outer, $inner): void {
    if ($outer) {
        if ($inner) {
            return;
        } else {
            throw new RuntimeException();
        }

        $unreachableInOuterBranch = true;
    } else {
        exit();
    }

    $unreachableAfterOuterBranch = true;
}`)

	wantLines := []int{10, 15}
	gotLines := make([]int, 0, len(issues))
	for _, issue := range issues {
		gotLines = append(gotLines, issue.Line)
	}
	if !reflect.DeepEqual(gotLines, wantLines) {
		t.Fatalf("unexpected unreachable lines: got %v, want %v (%#v)", gotLines, wantLines, issues)
	}
}

func TestControlFlowOuterLoopsRemainConservative(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function loops($items): void {
    while (true) {
        return;
        $unreachableInWhile = true;
    }
    $reachableAfterWhile = true;

    foreach ($items as $item) {
        throw new RuntimeException();
        $unreachableInForeach = true;
    }
    $reachableAfterForeach = true;
}`)

	wantLines := []int{5, 11}
	gotLines := make([]int, 0, len(issues))
	for _, issue := range issues {
		gotLines = append(gotLines, issue.Line)
	}
	if !reflect.DeepEqual(gotLines, wantLines) {
		t.Fatalf("unexpected unreachable lines: got %v, want %v (%#v)", gotLines, wantLines, issues)
	}
}

func TestControlFlowForCreatesChildScopeAndTracksBodyReachability(t *testing.T) {
	const filename = "for-control-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function repeat(): void {
    for ($index = 0; $index < 2; $index++) {
        return;
        $never = 1;
    }
    $after = 2;
}`)
	function := controlFlowFunction(t, nodes, "repeat")
	loop, ok := function.Body[0].(*ast.ForNode)
	if !ok {
		t.Fatalf("expected ForNode in function body, got %T", function.Body[0])
	}
	if len(loop.Body) != 2 {
		t.Fatalf("expected two statements in for body, got %d", len(loop.Body))
	}

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	childScope, ok := FlowScopeKeyForNode(filename, "for", loop, loop.Body)
	if !ok {
		t.Fatal("expected an exact child scope key for the for loop")
	}
	graph, ok := snapshot.ControlFlowGraph(childScope)
	if !ok {
		t.Fatalf("expected control-flow graph for for child scope: %#v", childScope)
	}
	if graph.Scope() != childScope {
		t.Fatalf("graph scope = %#v, want %#v", graph.Scope(), childScope)
	}
	if got, found := snapshot.StatementReachable(flowStatementKey(filename, loop.Body[1])); !found || got {
		t.Fatalf("statement after return in for body reachability = %v, %v; want false, true", got, found)
	}
	if got, found := snapshot.StatementReachable(flowStatementKey(filename, function.Body[1])); !found || !got {
		t.Fatalf("statement after conservative for loop reachability = %v, %v; want true, true", got, found)
	}
}

func TestControlFlowSnapshotReadsAreConcurrentAndDeterministic(t *testing.T) {
	const filename = "concurrent-control-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function choose($value): void {
    if ($value) {
        return;
    } else {
        throw new RuntimeException();
    }
    $unreachable = true;
}`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	want := unreachableIssues(RunAnalysisRulesWithContext(filename, nodes, snapshot.NewAnalysisContext()))
	if len(want) != 1 {
		t.Fatalf("expected one baseline unreachable issue, got %#v", want)
	}

	const readers = 32
	var wg sync.WaitGroup
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := unreachableIssues(RunAnalysisRulesWithContext(filename, nodes, snapshot.NewAnalysisContext()))
			if !reflect.DeepEqual(got, want) {
				t.Errorf("concurrent snapshot result differs: got %#v, want %#v", got, want)
			}
		}()
	}
	wg.Wait()
}

func TestControlFlowUnreachableDiagnosticsMatchLegacyFallback(t *testing.T) {
	fixtures := []string{
		`<?php function sample($value): void { return; $a = 1; $b = 2; }`,
		`<?php function sample($value): void { if ($value) { return; } $a = 1; }`,
		`<?php function sample($value): void { if ($value) { return; } else { throw new RuntimeException(); } $a = 1; }`,
		`<?php function sample($items): void { foreach ($items as $item) { exit(); $a = 1; } $b = 2; }`,
	}
	const filename = "parity.php"
	rule := &UnreachableCodeRule{}
	for _, fixture := range fixtures {
		nodes := parseControlFlowPHP(t, fixture)
		snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
		if err != nil {
			t.Fatalf("build semantic snapshot: %v", err)
		}
		legacy := rule.CheckIssues(nodes, filename)
		fromGraph := rule.CheckIssuesWithContext(nodes, filename, snapshot.NewAnalysisContext())
		if !reflect.DeepEqual(fromGraph, legacy) {
			t.Fatalf("graph-backed diagnostics changed legacy behavior:\nlegacy: %#v\ngraph: %#v", legacy, fromGraph)
		}
	}
}

func TestControlFlowGraphReturnsDefensiveBlocksWithDeterministicIDs(t *testing.T) {
	const filename = "graph.php"
	nodes := parseControlFlowPHP(t, `<?php
function graph($value): void {
    $first = $value;
    return;
    $last = true;
}`)
	function := controlFlowFunction(t, nodes, "graph")
	scope := flowScopeKey(filename, "function", function, function.Body)

	firstSnapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build first semantic snapshot: %v", err)
	}
	secondSnapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build second semantic snapshot: %v", err)
	}
	first, ok := firstSnapshot.ControlFlowGraph(scope)
	if !ok {
		t.Fatalf("expected graph for scope %#v", scope)
	}
	second, ok := secondSnapshot.ControlFlowGraph(scope)
	if !ok {
		t.Fatalf("expected graph from repeated snapshot for scope %#v", scope)
	}

	wantIDs := []FlowNodeID{0, 1, 2, 3, 4}
	blocks := first.Blocks()
	gotIDs := make([]FlowNodeID, 0, len(blocks))
	for _, block := range blocks {
		gotIDs = append(gotIDs, block.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected deterministic block IDs: got %v, want %v", gotIDs, wantIDs)
	}
	if !reflect.DeepEqual(blocks, second.Blocks()) {
		t.Fatalf("repeated snapshot changed graph:\nfirst %#v\nsecond %#v", blocks, second.Blocks())
	}

	blocks[0].ID = 99
	blocks[0].Successors[0] = 99
	blocksAgain := first.Blocks()
	if blocksAgain[0].ID != 0 || !reflect.DeepEqual(blocksAgain[0].Successors, []FlowNodeID{1}) {
		t.Fatalf("graph leaked mutation through Blocks: %#v", blocksAgain[0])
	}
	fromSnapshot, ok := firstSnapshot.ControlFlowGraph(scope)
	if !ok || !reflect.DeepEqual(fromSnapshot.Blocks(), blocksAgain) {
		t.Fatalf("snapshot graph changed after caller mutation: %#v, %v", fromSnapshot.Blocks(), ok)
	}
}

func TestControlFlowScopeFallthroughAndStatementReachability(t *testing.T) {
	const filename = "fallthrough.php"
	nodes := parseControlFlowPHP(t, `<?php
function exhaustive($value): void {
    if ($value) {
        return;
    } else {
        throw new RuntimeException();
    }
    $unreachable = true;
}

function partial($value): void {
    if ($value) {
        return;
    }
    $reachable = true;
}`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	exhaustive := controlFlowFunction(t, nodes, "exhaustive")
	partial := controlFlowFunction(t, nodes, "partial")

	if got, ok := snapshot.ScopeMayFallThrough(flowScopeKey(filename, "function", exhaustive, exhaustive.Body)); !ok || got {
		t.Fatalf("exhaustive function fallthrough = %v, %v; want false, true", got, ok)
	}
	if got, ok := snapshot.ScopeMayFallThrough(flowScopeKey(filename, "function", partial, partial.Body)); !ok || !got {
		t.Fatalf("partial function fallthrough = %v, %v; want true, true", got, ok)
	}
	if got, ok := snapshot.StatementReachable(flowStatementKey(filename, exhaustive.Body[len(exhaustive.Body)-1])); !ok || got {
		t.Fatalf("statement after exhaustive branch reachability = %v, %v; want false, true", got, ok)
	}
	if got, ok := snapshot.StatementReachable(flowStatementKey(filename, partial.Body[len(partial.Body)-1])); !ok || !got {
		t.Fatalf("statement after partial branch reachability = %v, %v; want true, true", got, ok)
	}
}

func TestControlFlowIndexesStatementsInsideElseScopeWithoutNativeEndSpan(t *testing.T) {
	const filename = "else-scope.php"
	nodes := parseControlFlowPHP(t, `<?php
function choose($value): void {
    if ($value) {
        $reachable = true;
    } else {
        return;
        $unreachable = true;
    }
}`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	function := controlFlowFunction(t, nodes, "choose")
	branch, ok := function.Body[0].(*ast.IfNode)
	if !ok || branch.Else == nil || len(branch.Else.Body) != 2 {
		t.Fatalf("unexpected if/else AST: %#v", function.Body)
	}
	key, ok := FlowStatementKeyForNode(filename, branch.Else.Body[1])
	if !ok {
		t.Fatal("expected an exact key for the unreachable else statement")
	}
	if reachable, found := snapshot.StatementReachable(key); !found || reachable {
		t.Fatalf("else statement reachability = %v, %v; want false, true", reachable, found)
	}
	scope, ok := FlowScopeKeyForNode(filename, "else", branch.Else, branch.Else.Body)
	if !ok {
		t.Fatal("expected derived scope key for else node without a native end span")
	}
	if fallsThrough, found := snapshot.ScopeMayFallThrough(scope); !found || fallsThrough {
		t.Fatalf("else scope fallthrough = %v, %v; want false, true", fallsThrough, found)
	}
}

func TestControlFlowAmbiguousAndZeroSpansAreConservative(t *testing.T) {
	const filename = "ambiguous.php"
	nodes := parseControlFlowPHP(t, `<?php
function duplicate(): void {
    return;
}`)
	function := controlFlowFunction(t, nodes, "duplicate")
	duplicateTopLevel := []ast.Node{function, function}
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: duplicateTopLevel}, nil)
	if err != nil {
		t.Fatalf("build ambiguous semantic snapshot: %v", err)
	}
	if got, ok := snapshot.StatementReachable(flowStatementKey(filename, function)); ok {
		t.Fatalf("ambiguous statement reachability = %v, %v; want unknown", got, ok)
	}

	zeroSpan := &ast.ExpressionStmt{}
	zeroSnapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: {zeroSpan}}, nil)
	if err != nil {
		t.Fatalf("build zero-span semantic snapshot: %v", err)
	}
	if got, ok := zeroSnapshot.StatementReachable(FlowStatementKey{File: filename}); ok {
		t.Fatalf("zero-span statement reachability = %v, %v; want unknown", got, ok)
	}
	if _, ok := zeroSnapshot.ControlFlowGraph(FlowScopeKey{}); ok {
		t.Fatal("zero scope key unexpectedly resolved a graph")
	}
	if got, ok := zeroSnapshot.ScopeMayFallThrough(FlowScopeKey{}); ok {
		t.Fatalf("zero scope fallthrough = %v, %v; want unknown", got, ok)
	}
}

func controlFlowFunction(t *testing.T, nodes []ast.Node, name string) *ast.FunctionNode {
	t.Helper()
	for _, node := range nodes {
		if function, ok := node.(*ast.FunctionNode); ok && function.Name == name {
			return function
		}
	}
	t.Fatalf("function %q not found", name)
	return nil
}

func TestControlFlowSwitchExhaustiveDefaultTerminates(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function choice($value): void {
    switch ($value) {
    case 1:
        return;
    case 2:
        throw new RuntimeException();
    default:
        exit();
    }

    $unreachable = true;
}`)

	if len(issues) != 1 || issues[0].Line != 12 {
		t.Fatalf("expected only line 12 to be unreachable, got %#v", issues)
	}
}

func TestControlFlowSwitchWithoutDefaultMayFallThrough(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function choice($value): void {
    switch ($value) {
    case 1:
        return;
    case 2:
        throw new RuntimeException();
    }

    $reachable = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected missing default to allow fallthrough, got %#v", issues)
	}
}

func TestControlFlowSwitchFallthroughBetweenCases(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function choice($value): void {
    switch ($value) {
    case 1:
    case 2:
        return;
    default:
        exit();
    }

    $unreachable = true;
}`)

	if len(issues) != 1 || issues[0].Line != 11 {
		t.Fatalf("expected only line 11 to be unreachable, got %#v", issues)
	}
}

func TestControlFlowTryThrowsWithFinallyMayFallThrough(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function mayThrow(): void {
    try {
        throw new RuntimeException();
    } finally {
        $log = 1;
    }

    $reachable = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected try with finally (not terminating) to allow fallthrough, got %#v", issues)
	}
}

func TestControlFlowTryExhaustiveCatchesTerminate(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function tryAndCatch(): void {
    try {
        throw new RuntimeException();
    } catch (RuntimeException $e) {
        return;
    } catch (LogicException $e) {
        exit();
    }

    $unreachable = true;
}`)

	if len(issues) != 1 || issues[0].Line != 11 {
		t.Fatalf("expected only line 11 to be unreachable, got %#v", issues)
	}
}

func TestControlFlowTryWithSingleCatchMayFallThrough(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function tryCatchFallthrough(): void {
    try {
        doSomething();
    } catch (RuntimeException $e) {
        return;
    }

    $reachable = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected try/catch to allow fallthrough when try may not throw, got %#v", issues)
	}
}

func TestControlFlowFinallyAlwaysExecutesAndTerminates(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function tryFinallyTerminate(): void {
    try {
        doSomething();
    } finally {
        exit();
    }

    $unreachable = true;
}`)

	if len(issues) != 1 || issues[0].Line != 9 {
		t.Fatalf("expected only line 9 to be unreachable (finally terminated), got %#v", issues)
	}
}

func TestControlFlowFinallyDoesNotTerminateAllowsFallthrough(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function tryFinallyFallthrough(): void {
    try {
        throw new RuntimeException();
    } catch (RuntimeException $e) {
        return;
    } finally {
        $log = 1;
    }

    $reachable = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected finally without termination to allow fallthrough, got %#v", issues)
	}
}

func TestControlFlowSwitchInsideLoopTerminatesOneCase(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function switchInLoop(): void {
    while ($condition) {
        switch ($value) {
        case 1:
            return;
        case 2:
            return;
        default:
            return;
        }

        $unreachableAfterSwitch = true;
    }

    $reachableAfterLoop = true;
}`)

	wantUnreachable := 13
	if len(issues) != 1 || issues[0].Line != wantUnreachable {
		t.Fatalf("expected only line %d unreachable (all cases return), got %#v", wantUnreachable, issues)
	}
}

func TestControlFlowTryInsideLoopWithBreakInCatch(t *testing.T) {
	issues := controlFlowUnreachableIssues(t, `<?php
function tryInLoop(): void {
    foreach ($items as $item) {
        try {
            doSomething($item);
        } catch (SkipException $e) {
            continue;
        }

        $reachableAfterTry = true;
    }

    $reachableAfterForeach = true;
}`)

	if len(issues) != 0 {
		t.Fatalf("expected all code reachable (catch handles exception with continue), got %#v", issues)
	}
}

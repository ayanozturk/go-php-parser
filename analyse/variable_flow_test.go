package analyse

import (
	"reflect"
	"sync"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestVariableFlowJoinsConditionalBranches(t *testing.T) {
	const filename = "branch-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function choose(bool $condition): void {
    if ($condition) {
        $both = 1;
        $thenOnly = 1;
    } else {
        $both = 2;
        $elseOnly = 2;
    }
    echo $both;
    echo $thenOnly;
    echo $elseOnly;
    echo $never;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	assertVariableReadState(t, snapshot, filename, "both", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "thenOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "elseOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "never", VariableUndefined)
}

func TestVariableFlowPreservesConditionAssignmentsAndExhaustiveDefinitions(t *testing.T) {
	const filename = "condition-assignment.php"
	nodes := parseControlFlowPHP(t, `<?php
function choose(bool $condition): void {
    if ($assigned = $condition) {
        $fromEveryPath = 1;
    } else {
        $fromEveryPath = 2;
    }
    echo $assigned;
    echo $fromEveryPath;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "assigned", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "fromEveryPath", VariableDefinitelyDefined)
}

func TestVariableFlowConvergesLoopsAndDistinguishesGuaranteedDoBody(t *testing.T) {
	const filename = "loop-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function repeat(bool $condition, array $items): void {
    while ($condition) {
        echo $carried;
        $carried = 1;
        $whileOnly = 1;
    }
    echo $carried;
    echo $whileOnly;

    for ($index = 0; $index < 2; $index++) {
        $forOnly = $index;
    }
    echo $index;
    echo $forOnly;

    foreach ($items as $item) {
        $foreachOnly = $item;
    }
    echo $item;
    echo $foreachOnly;

    do {
        $doValue = 1;
    } while ($condition);
    echo $doValue;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	assertVariableReadState(t, snapshot, filename, "carried", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "whileOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "index", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "forOnly", VariablePossiblyDefined)
	assertVariableReadStates(t, snapshot, filename, "item", []VariableDefinedness{VariableDefinitelyDefined, VariablePossiblyDefined})
	assertVariableReadState(t, snapshot, filename, "foreachOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "doValue", VariableDefinitelyDefined)
}

func TestVariableFlowUsesConditionAssignmentsAndRejectsFalseLoopExits(t *testing.T) {
	const filename = "loop-condition-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function conditions(): void {
    while ($assigned = nextValue()) {}
    echo $assigned;

    while (false) { echo $impossible; }

    while (true) {}
    echo $afterInfinite;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "assigned", VariableDefinitelyDefined)
	for _, read := range snapshot.VariableReadsForFile(filename) {
		if read.Key.Name == "impossible" || read.Key.Name == "afterInfinite" {
			t.Fatalf("impossible loop path gained a variable-flow fact: %#v", read)
		}
	}
}

func TestVariableFlowJoinsTryCatchFinallyAndSwitch(t *testing.T) {
	const filename = "try-switch-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function inspect(bool $condition, int $kind): void {
    try {
        if ($condition) {
            $tryOnly = 1;
        }
    } catch (RuntimeException $error) {
        $catchOnly = $error;
    } finally {
        $finalValue = 1;
    }
    echo $tryOnly;
    echo $catchOnly;
    echo $finalValue;

    switch ($kind) {
        case 1:
            $everyCase = 1;
            break;
        default:
            $everyCase = 2;
            break;
    }
    echo $everyCase;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	assertVariableReadState(t, snapshot, filename, "tryOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "catchOnly", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "finalValue", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "everyCase", VariableDefinitelyDefined)
}

func TestVariableFlowFinallyReadsTerminatingPathsWithoutRevivingThem(t *testing.T) {
	const filename = "terminating-finally.php"
	nodes := parseControlFlowPHP(t, `<?php
function finish(bool $condition): void {
    try {
        if ($condition) { $possible = 1; }
        return;
    } finally {
        echo $possible;
        $finalOnly = 1;
    }
    echo $finalOnly;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "possible", VariablePossiblyDefined)
	for _, read := range snapshot.VariableReadsForFile(filename) {
		if read.Key.Name == "finalOnly" {
			t.Fatalf("unreachable statement after terminating try gained a variable-flow fact: %#v", read)
		}
	}
}

func TestVariableFlowJoinsConditionalExpressionSideEffects(t *testing.T) {
	const filename = "expression-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function expressions(bool $condition): void {
    $condition && ($short = 1);
    echo $short;

    $condition ? ($both = 1) : ($both = 2);
    echo $both;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "short", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "both", VariableDefinitelyDefined)
}

func TestVariableFlowHandlesGlobalAndStaticDeclarations(t *testing.T) {
	const filename = "declarations-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function declarations(): void {
    global $shared;
    static $counter = 0;
    echo $shared;
    echo $counter;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "shared", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "counter", VariableDefinitelyDefined)
}

func TestVariableFlowIsolatesClosureStateAndCapturesOuterReads(t *testing.T) {
	const filename = "closure-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function closures(): void {
    $outer = 1;
    $closure = function ($parameter) use ($outer) {
        echo $outer;
        echo $parameter;
        echo $missing;
        $inside = 1;
    };
    $arrow = fn ($value) => $outer + $value;
    echo $inside;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "outer", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "parameter", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "value", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "missing", VariableUndefined)
	assertVariableReadState(t, snapshot, filename, "inside", VariableUndefined)
}

func TestVariableFlowFactsAreDeterministicDefensiveAndConcurrent(t *testing.T) {
	const filename = "stable-variable-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function stable(bool $condition): void {
    if ($condition) { $value = 1; }
    echo $value;
}`)
	first := variableFlowSnapshot(t, filename, nodes)
	second := variableFlowSnapshot(t, filename, nodes)
	want := first.VariableReadsForFile(filename)
	if !reflect.DeepEqual(want, second.VariableReadsForFile(filename)) {
		t.Fatalf("repeated snapshots changed variable-flow facts:\nfirst %#v\nsecond %#v", want, second.VariableReadsForFile(filename))
	}

	mutated := first.VariableReadsForFile(filename)
	mutated[0].Key.Name = "changed"
	if reflect.DeepEqual(mutated, first.VariableReadsForFile(filename)) {
		t.Fatal("variable-flow reader leaked caller mutation")
	}

	const readers = 32
	var wait sync.WaitGroup
	for i := 0; i < readers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if got := first.VariableReadsForFile(filename); !reflect.DeepEqual(got, want) {
				t.Errorf("concurrent variable-flow read changed: %#v", got)
			}
		}()
	}
	wait.Wait()
}

func TestVariableFlowRuleStartsUndefinedDiagnosticsAtLevelOne(t *testing.T) {
	const filename = "variable-levels.php"
	nodes := parseControlFlowPHP(t, `<?php
function choose(bool $condition): void {
    if ($condition) { $possible = 1; }
    echo $possible;
    echo $missing;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	zeroContext := snapshot.NewAnalysisContext()
	zeroIssues := (&PHPStanLevel0Rule{}).CheckIssues(filename, nodes, zeroContext)
	if hasIssueContaining(zeroIssues, level1VariablesCode, "might not be defined") {
		t.Fatalf("level zero emitted undefined-variable diagnostics: %#v", zeroIssues)
	}

	oneContext := snapshot.NewAnalysisContext()
	oneIssues := checkUndefinedVariables(filename, nodes, oneContext)
	if !hasIssueContaining(oneIssues, level1VariablesCode, "Variable $possible might not be defined.") {
		t.Fatalf("level one did not emit possible-variable diagnostic: %#v", oneIssues)
	}
	if !hasIssueContaining(oneIssues, level1VariablesCode, "Variable $missing might not be defined.") {
		t.Fatalf("level one did not emit always-undefined diagnostic: %#v", oneIssues)
	}
}

func TestVariableFlowStateCloneSharesUntilFirstWrite(t *testing.T) {
	original := initialVariableFlowState()
	original.set("seed", VariableDefinitelyDefined)
	clone := cloneVariableFlowState(original)

	if reflect.ValueOf(original.values).Pointer() != reflect.ValueOf(clone.values).Pointer() {
		t.Fatal("read-only clone did not share its backing state")
	}
	clone.set("branch", VariableDefinitelyDefined)
	if reflect.ValueOf(original.values).Pointer() == reflect.ValueOf(clone.values).Pointer() {
		t.Fatal("first clone write did not detach its backing state")
	}
	if original.definedness("branch") != VariableUndefined {
		t.Fatal("clone write leaked into original state")
	}
	if clone.definedness("seed") != VariableDefinitelyDefined {
		t.Fatal("clone detach lost existing state")
	}
}

func TestVariableFlowStateKeepsPredefinedVariablesImplicit(t *testing.T) {
	state := initialVariableFlowState()
	if len(state.values) != 0 {
		t.Fatalf("initial state materialized predefined variables: %#v", state.values)
	}
	for _, name := range predefinedVariables {
		if state.definedness(name) != VariableDefinitelyDefined {
			t.Fatalf("predefined variable $%s is not definitely defined", name)
		}
	}
}

func TestVariableFlowCompactAndSuppressedReadsUseJoinedFacts(t *testing.T) {
	const filename = "compact-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function collect(bool $condition): array {
    if ($condition) { $possible = 1; }
    isset($suppressed);
    empty($alsoSuppressed);
    return compact('possible', 'missing');
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)
	assertVariableReadState(t, snapshot, filename, "possible", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "missing", VariableUndefined)
	for _, read := range snapshot.VariableReadsForFile(filename) {
		if read.Key.Name == "suppressed" || read.Key.Name == "alsoSuppressed" {
			t.Fatalf("isset/empty read should be suppressed, got %#v", read)
		}
	}
}

func variableFlowSnapshot(t *testing.T, filename string, nodes []ast.Node) *SemanticSnapshot {
	t.Helper()
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	return snapshot
}

func assertVariableReadState(t *testing.T, reader VariableFlowReader, filename, name string, want VariableDefinedness) {
	t.Helper()
	var matched []VariableReadFact
	for _, read := range reader.VariableReadsForFile(filename) {
		if read.Key.Name == name {
			matched = append(matched, read)
		}
	}
	if len(matched) == 0 {
		t.Fatalf("no variable-flow read for $%s in %#v", name, reader.VariableReadsForFile(filename))
	}
	for _, read := range matched {
		if read.State != want {
			t.Fatalf("$%s state = %v, want %v (all reads %#v)", name, read.State, want, matched)
		}
	}
}

func assertVariableReadStates(t *testing.T, reader VariableFlowReader, filename, name string, want []VariableDefinedness) {
	t.Helper()
	var got []VariableDefinedness
	for _, read := range reader.VariableReadsForFile(filename) {
		if read.Key.Name == name {
			got = append(got, read.State)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("$%s states = %v, want %v", name, got, want)
	}
}

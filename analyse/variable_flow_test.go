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

func TestVariableFlowModelsExplicitClosureCapturesAndReferenceWrites(t *testing.T) {
	const filename = "reference-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function fill(&$out): void { $out = 1; }
function inspect(): void {
    $closure = function () use ($missing): void { echo $missing; };
    $writer = function () use (&$captured): void { $captured = 1; };
    echo $captured;
    fill($output);
    echo $output;
    parse_str('key=value', $parsed);
    echo $parsed;
    sort($items);
    echo $items;
    $alias =& $source;
    echo $alias;
    echo $source;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	assertVariableReadStates(t, snapshot, filename, "missing", []VariableDefinedness{VariableUndefined, VariableDefinitelyDefined})
	for _, name := range []string{"captured", "output", "parsed", "alias"} {
		assertVariableReadState(t, snapshot, filename, name, VariableDefinitelyDefined)
	}
	assertVariableReadState(t, snapshot, filename, "source", VariableUndefined)
	assertVariableReadStates(t, snapshot, filename, "items", []VariableDefinedness{VariableUndefined, VariableDefinitelyDefined})

	levelOne := checkUndefinedVariables(filename, nodes, snapshot.NewAnalysisContext())
	if !hasIssueContaining(levelOne, level1VariablesCode, "Variable $missing might not be defined.") {
		t.Fatalf("by-value undefined capture was not reported: %#v", levelOne)
	}
	for _, name := range []string{"captured", "output", "parsed", "alias"} {
		if hasIssueContaining(levelOne, level1VariablesCode, "$"+name) {
			t.Fatalf("reference-defined $%s was reported: %#v", name, levelOne)
		}
	}
	if !hasIssueContaining(levelOne, level1VariablesCode, "Variable $source might not be defined.") {
		t.Fatalf("reference source direct read was not reported: %#v", levelOne)
	}
	if !hasIssueContaining(levelOne, level1VariablesCode, "Variable $items might not be defined.") {
		t.Fatalf("input-output reference did not preserve its input read: %#v", levelOne)
	}
}

func TestVariableFlowModelsResolvedMethodAndConstructorReferenceWrites(t *testing.T) {
	const filename = "method-reference-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
class ParentWriter {
    public static function fillParent(&$out): void { $out = 1; }
}
class Writer extends ParentWriter {
    public function __construct(&$out) { $out = 1; }
    public function fill(&$out): void { $out = 1; }
    public static function fillStatic(&$out): void { $out = 1; }
    public function run(object $dynamic): void {
        $this->fill($instance);
        echo $instance;
        self::fillStatic($selfStatic);
        echo $selfStatic;
        parent::fillParent($parentStatic);
        echo $parentStatic;
		new self($selfConstructed);
		echo $selfConstructed;
        $dynamic->fill($unresolved);
    }
}
Writer::fillStatic($static);
echo $static;
new Writer($constructed);
echo $constructed;
(new Writer($temporary))->fill($newReceiver);
echo $temporary;
echo $newReceiver;
`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	for _, name := range []string{"instance", "selfStatic", "parentStatic", "selfConstructed", "static", "constructed", "temporary", "newReceiver"} {
		assertVariableReadState(t, snapshot, filename, name, VariableDefinitelyDefined)
	}
	assertVariableReadState(t, snapshot, filename, "unresolved", VariableUndefined)

	levelOne := checkUndefinedVariables(filename, nodes, snapshot.NewAnalysisContext())
	for _, name := range []string{"instance", "selfStatic", "parentStatic", "selfConstructed", "static", "constructed", "temporary", "newReceiver"} {
		if hasIssueContaining(levelOne, level1VariablesCode, "$"+name) {
			t.Fatalf("resolved reference-defined $%s was reported: %#v", name, levelOne)
		}
	}
	if !hasIssueContaining(levelOne, level1VariablesCode, "Variable $unresolved might not be defined.") {
		t.Fatalf("dynamic method reference input was not reported: %#v", levelOne)
	}
}

func TestVariableFlowPropagatesMultiLevelLoopTransfersAcrossScopes(t *testing.T) {
	const filename = "multi-level-loop-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function nestedBreak(bool $condition): void {
    do {
        do {
            $beforeBreak = 1;
            if ($condition) { $conditionalBreak = 1; }
            break 2;
        } while (true);
        $afterInnerBreak = 1;
    } while (true);
    echo $beforeBreak;
    echo $conditionalBreak;
    echo $afterInnerBreak;
}
function nestedContinue(): void {
    do {
        do {
            $beforeContinue = 1;
            continue 2;
        } while (false);
        $afterInnerContinue = 1;
    } while (false);
    echo $beforeContinue;
    echo $afterInnerContinue;
}
function switchContinue(int $value): void {
    do {
        switch ($value) {
            default:
                continue 2;
        }
        $afterSwitch = 1;
    } while (false);
    echo $afterSwitch;
}
function breakThroughFinally(): void {
    do {
        do {
            try {
                break 2;
            } finally {
                $inFinally = 1;
            }
        } while (true);
        $afterFinally = 1;
    } while (true);
    echo $inFinally;
    echo $afterFinally;
}
`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	assertVariableReadState(t, snapshot, filename, "beforeBreak", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "conditionalBreak", VariablePossiblyDefined)
	assertVariableReadState(t, snapshot, filename, "afterInnerBreak", VariableUndefined)
	assertVariableReadState(t, snapshot, filename, "beforeContinue", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "afterInnerContinue", VariableUndefined)
	assertVariableReadState(t, snapshot, filename, "afterSwitch", VariableUndefined)
	assertVariableReadState(t, snapshot, filename, "inFinally", VariableDefinitelyDefined)
	assertVariableReadState(t, snapshot, filename, "afterFinally", VariableUndefined)
}

func TestVariableFlowModelsKnownDynamicReadsAndExtractEffects(t *testing.T) {
	const filename = "dynamic-variable-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function dynamicVariables(): void {
    $knownName = 'missingDynamicTarget';
    echo $$knownName;
    echo $$missingNameVariable;

    $writeName = 'dynamicWriteTarget';
    $$writeName = 1;
    echo $dynamicWriteTarget;
    ${'literalDynamicWrite'} = 1;
    echo $literalDynamicWrite;

    extract(['directExtracted' => 1]);
    echo $directExtracted;
    $assignedValues = ['assignedExtracted' => 1];
    extract($assignedValues);
    echo $assignedExtracted;

    extract(['prefixedOriginal' => 1], EXTR_PREFIX_ALL, 'prefix');
    echo $prefixedOriginal;
    echo $prefix_prefixedOriginal;
}
function unknownExtract(array $unknownValues): void {
    extract($unknownValues);
    echo $possibleUnknownExtract;
}
function conditionalExtract(bool $condition): void {
    if ($condition) {
        extract(['conditionalExtracted' => 1]);
    }
    echo $conditionalExtracted;
}
`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	for _, name := range []string{"missingDynamicTarget", "missingNameVariable", "dynamicWriteTarget", "literalDynamicWrite", "prefix_prefixedOriginal"} {
		assertVariableReadState(t, snapshot, filename, name, VariableUndefined)
	}
	for _, name := range []string{"directExtracted", "assignedExtracted", "prefixedOriginal"} {
		assertVariableReadState(t, snapshot, filename, name, VariableDefinitelyDefined)
	}
	for _, name := range []string{"possibleUnknownExtract", "conditionalExtracted"} {
		assertVariableReadState(t, snapshot, filename, name, VariablePossiblyDefined)
	}
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

func TestVariableFlowSnapshotLazilyMaterializesDefinitelyDefinedReads(t *testing.T) {
	const filename = "lazy-variable-flow.php"
	nodes := parseControlFlowPHP(t, `<?php
function inspect(): void {
    $defined = 1;
    echo $defined;
    echo $missing;
}`)
	snapshot := variableFlowSnapshot(t, filename, nodes)

	diagnosticReads := snapshot.variableReads[filename]
	if len(diagnosticReads) != 1 || diagnosticReads[0].name != "missing" {
		t.Fatalf("eager diagnostic reads = %#v, want only $missing", diagnosticReads)
	}
	lazy := snapshot.completeVariableReads[filename]
	if lazy == nil || lazy.reads != nil {
		t.Fatalf("complete reads were materialized eagerly: %#v", lazy)
	}

	complete := snapshot.VariableReadsForFile(filename)
	if len(complete) != 2 {
		t.Fatalf("complete reads = %#v, want defined and undefined reads", complete)
	}
	if lazy.reads == nil || lazy.nodes != nil {
		t.Fatal("lazy materialization did not cache facts and release AST roots")
	}

	var ranged []VariableReadFact
	snapshot.rangeVariableReadsForFile(filename, func(read VariableReadFact) {
		ranged = append(ranged, read)
	})
	if len(ranged) != 1 || ranged[0].Key.Name != "missing" {
		t.Fatalf("internal diagnostic range = %#v, want only $missing", ranged)
	}
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
	zeroIssues := (&Level0Rule{}).CheckIssues(filename, nodes, zeroContext)
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
	analyzer := &variableFlowAnalyzer{variableIDs: make(map[string]int)}
	original := initialVariableFlowState(analyzer)
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

func TestVariableFlowStateConstantMetadataDetachesAndJoins(t *testing.T) {
	analyzer := &variableFlowAnalyzer{variableIDs: make(map[string]int)}
	original := initialVariableFlowState(analyzer)
	original.set("name", VariableDefinitelyDefined)
	original.setKnownString("name", "target")
	original.set("values", VariableDefinitelyDefined)
	original.setKnownExtractShape("values", variableFlowExtractShape{keys: []string{"first"}, complete: true})

	clone := cloneVariableFlowState(original)
	clone.set("name", VariableDefinitelyDefined)
	clone.set("values", VariableDefinitelyDefined)
	if value, ok := original.knownString(&ast.VariableNode{Name: "name"}); !ok || value != "target" {
		t.Fatalf("clone write cleared original known string: %q, %v", value, ok)
	}
	valuesID := analyzer.variableIDs["values"]
	if shape, ok := original.knownExtractShapeValues()[valuesID]; !ok || !shape.complete || !reflect.DeepEqual(shape.keys, []string{"first"}) {
		t.Fatalf("clone write cleared original extract shape: %#v, %v", shape, ok)
	}
	if _, ok := clone.knownStringValues()[analyzer.variableIDs["name"]]; ok {
		t.Fatal("ordinary clone assignment retained stale known string")
	}
	if _, ok := clone.knownExtractShapeValues()[valuesID]; ok {
		t.Fatal("ordinary clone assignment retained stale extract shape")
	}

	matching := cloneVariableFlowState(original)
	joined := joinedVariableFlowState(original, matching)
	if value, ok := joined.knownString(&ast.VariableNode{Name: "name"}); !ok || value != "target" {
		t.Fatalf("matching join lost known string: %q, %v", value, ok)
	}
	joinedDifferent := joinedVariableFlowState(original, clone)
	if _, ok := joinedDifferent.knownString(&ast.VariableNode{Name: "name"}); ok {
		t.Fatal("divergent join retained a path-specific known string")
	}
}

func TestVariableFlowStateKeepsPredefinedVariablesImplicit(t *testing.T) {
	analyzer := &variableFlowAnalyzer{variableIDs: make(map[string]int)}
	state := initialVariableFlowState(analyzer)
	if len(state.values) != 0 {
		t.Fatalf("initial state materialized predefined variables: %#v", state.values)
	}
	for _, name := range predefinedVariables {
		if state.definedness(name) != VariableDefinitelyDefined {
			t.Fatalf("predefined variable $%s is not definitely defined", name)
		}
	}
}

func TestVariableFlowStateJoinsCompactVariableSlots(t *testing.T) {
	analyzer := &variableFlowAnalyzer{variableIDs: make(map[string]int)}
	entry := initialVariableFlowState(analyzer)
	left := cloneVariableFlowState(entry)
	right := cloneVariableFlowState(entry)
	left.set("branch", VariableDefinitelyDefined)

	joined := joinedVariableFlowState(left, right)
	if joined.definedness("branch") != VariablePossiblyDefined {
		t.Fatalf("joined branch state = %v, want possibly defined", joined.definedness("branch"))
	}
	if got := len(joined.values); got != 1 {
		t.Fatalf("joined slot count = %d, want 1", got)
	}
	right.set("branch", VariableDefinitelyDefined)
	if joinedVariableFlowState(left, right).definedness("branch") != VariableDefinitelyDefined {
		t.Fatal("exhaustive slot join did not preserve definite state")
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

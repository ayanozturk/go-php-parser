package analyse

import (
	"reflect"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

const flowStorageTestFile = "control-flow-storage.php"

var emptyFlowStorageProject = map[string][]ast.Node{}

func newFlowStorageSnapshot(t *testing.T) *SemanticSnapshot {
	t.Helper()
	snapshot, err := NewSemanticSnapshot(emptyFlowStorageProject, nil)
	if err != nil {
		t.Fatalf("build empty semantic snapshot: %v", err)
	}
	return snapshot
}

func flowStoragePosition(offset int) ast.Position {
	return ast.Position{Line: 1, Column: offset, Offset: offset}
}

func flowStorageStatement(start, end int) ast.Node {
	return &ast.ExpressionStmt{Pos: flowStoragePosition(start), EndPos: flowStoragePosition(end)}
}

func flowStorageReturn(start, end int) ast.Node {
	return &ast.ReturnNode{Pos: flowStoragePosition(start), EndPos: flowStoragePosition(end)}
}

func flowStorageOwner(start, end int) *ast.BlockNode {
	return &ast.BlockNode{Pos: flowStoragePosition(start), EndPos: flowStoragePosition(end)}
}

func flowStorageScope(owner *ast.BlockNode) FlowScopeKey {
	return FlowScopeKey{
		File:        flowStorageTestFile,
		StartOffset: owner.Pos.Offset,
		EndOffset:   owner.EndPos.Offset,
		Kind:        "block",
	}
}

func TestFlowStorageAddFlowScopeIndexesContracts(t *testing.T) {
	snapshot := newFlowStorageSnapshot(t)
	first := flowStorageStatement(10, 20)
	terminating := flowStorageReturn(21, 30)
	unreachable := flowStorageStatement(31, 40)
	owner := flowStorageOwner(0, 50)
	parent := FlowScopeKey{File: flowStorageTestFile, EndOffset: 1, Kind: "file"}
	scope := flowStorageScope(owner)

	snapshot.addFlowScope(flowStorageTestFile, "block", owner, []ast.Node{first, terminating, unreachable}, parent)

	if got := snapshot.resolveScopeAtLevel(scope, 1); got != parent {
		t.Fatalf("scope parent = %#v; want %#v", got, parent)
	}
	if got := snapshot.resolveScopeAtLevel(scope, 2); got.File != "" {
		t.Fatalf("scope unexpectedly resolved beyond its parent: %#v", got)
	}

	graph, ok := snapshot.ControlFlowGraph(scope)
	if !ok {
		t.Fatalf("expected graph for scope %#v", scope)
	}
	if graph.Scope() != scope {
		t.Fatalf("graph scope = %#v, want %#v", graph.Scope(), scope)
	}
	if graph.MayFallThrough() {
		t.Fatal("scope containing a reachable return should not fall through")
	}

	for _, want := range []struct {
		statement ast.Node
		reachable bool
	}{
		{first, true},
		{terminating, true},
		{unreachable, false},
	} {
		key, keyOK := FlowStatementKeyForNode(flowStorageTestFile, want.statement)
		if !keyOK {
			t.Fatalf("expected statement key for %#v", want.statement)
		}
		if got, found := snapshot.StatementReachable(key); !found || got != want.reachable {
			t.Fatalf("statement %v reachability = %v, %v; want %v, true", key, got, found, want.reachable)
		}
	}

	// A repeated scope key keeps the first graph and its indexed statements.
	replacement := flowStorageStatement(100, 110)
	snapshot.addFlowScope(flowStorageTestFile, "block", owner, []ast.Node{replacement}, parent)
	if blocks := graph.Blocks(); len(blocks) != 5 {
		t.Fatalf("duplicate scope replaced the original graph with %d blocks", len(blocks))
	}
	if key, keyOK := FlowStatementKeyForNode(flowStorageTestFile, replacement); !keyOK {
		t.Fatal("expected replacement statement key")
	} else if _, found := snapshot.StatementReachable(key); found {
		t.Fatal("duplicate scope indexed statements from the discarded graph")
	}

	if got, found := snapshot.StatementReachable(FlowStatementKey{StartOffset: 10, EndOffset: 20}); found || got {
		t.Fatalf("statement key without a file = %v, %v; want false, false", got, found)
	}
}

func TestFlowStorageControlFlowGraphBlocksAreDefensiveCopies(t *testing.T) {
	snapshot := newFlowStorageSnapshot(t)
	owner := flowStorageOwner(200, 260)
	statements := []ast.Node{
		flowStorageStatement(210, 220),
		flowStorageStatement(221, 230),
	}
	scope := flowStorageScope(owner)
	snapshot.addFlowScope(flowStorageTestFile, "block", owner, statements, FlowScopeKey{})

	graph, ok := snapshot.ControlFlowGraph(scope)
	if !ok {
		t.Fatalf("expected graph for scope %#v", scope)
	}
	before := graph.Blocks()
	if len(before) != 4 || !reflect.DeepEqual(before[0].Successors, []FlowNodeID{1}) {
		t.Fatalf("unexpected initial graph blocks: %#v", before)
	}

	mutated := graph.Blocks()
	mutated[0].ID = 99
	mutated[0].Successors[0] = 99
	mutated[1].Statement.File = "mutated.php"
	mutated[1].Successors[0] = 99

	after := graph.Blocks()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("Blocks leaked caller mutation:\nbefore %#v\nafter %#v", before, after)
	}
	fromSnapshot, ok := snapshot.ControlFlowGraph(scope)
	if !ok || !reflect.DeepEqual(fromSnapshot.Blocks(), before) {
		t.Fatalf("snapshot graph changed after caller mutation: %#v, %v", fromSnapshot.Blocks(), ok)
	}
}

func TestFlowStorageDuplicateStatementSpansBecomeUnknown(t *testing.T) {
	snapshot := newFlowStorageSnapshot(t)
	shared := flowStorageStatement(310, 320)
	firstOwner := flowStorageOwner(300, 330)
	secondOwner := flowStorageOwner(400, 440)
	firstScope := flowStorageScope(firstOwner)
	secondScope := flowStorageScope(secondOwner)

	snapshot.addFlowScope(flowStorageTestFile, "block", firstOwner, []ast.Node{shared}, FlowScopeKey{})
	snapshot.addFlowScope(flowStorageTestFile, "block", secondOwner, []ast.Node{
		flowStorageReturn(401, 410),
		shared,
	}, FlowScopeKey{})

	key, ok := FlowStatementKeyForNode(flowStorageTestFile, shared)
	if !ok {
		t.Fatal("expected shared statement key")
	}
	if got, found := snapshot.StatementReachable(key); found || got {
		t.Fatalf("ambiguous statement reachability = %v, %v; want false, false", got, found)
	}
	if got, found := snapshot.StatementReachable(key); found || got {
		t.Fatalf("ambiguous statement became known on repeated read = %v, %v", got, found)
	}

	if got, found := snapshot.ScopeMayFallThrough(firstScope); !found || !got {
		t.Fatalf("first scope fallthrough = %v, %v; want true, true", got, found)
	}
	if got, found := snapshot.ScopeMayFallThrough(secondScope); !found || got {
		t.Fatalf("second scope fallthrough = %v, %v; want false, true", got, found)
	}

	// The span is only ambiguous within one file; an equal span in another
	// file remains independently queryable.
	otherFile := "other-control-flow-storage.php"
	otherOwner := flowStorageOwner(300, 330)
	snapshot.addFlowScope(otherFile, "block", otherOwner, []ast.Node{shared}, FlowScopeKey{})
	otherKey, ok := FlowStatementKeyForNode(otherFile, shared)
	if !ok {
		t.Fatal("expected other-file statement key")
	}
	if got, found := snapshot.StatementReachable(otherKey); !found || !got {
		t.Fatalf("equal span in another file = %v, %v; want true, true", got, found)
	}
}

func TestFlowStorageFallbackPreservesCustomKindsAndLargeOffsets(t *testing.T) {
	snapshot := newFlowStorageSnapshot(t)
	largeOffset := maxCompactSourceOffset + 1
	if uint64(int(largeOffset)) != largeOffset {
		t.Skip("large source offsets require a 64-bit int")
	}
	start := int(largeOffset)
	owner := flowStorageOwner(start, start+100)
	statement := flowStorageStatement(start+10, start+20)
	parent := FlowScopeKey{File: "parent-" + flowStorageTestFile, StartOffset: start - 100, EndOffset: start - 1, Kind: "custom-parent"}
	scope := FlowScopeKey{File: flowStorageTestFile, StartOffset: start, EndOffset: start + 100, Kind: "custom-child"}

	snapshot.addFlowScope(flowStorageTestFile, scope.Kind, owner, []ast.Node{statement}, parent)
	graph, ok := snapshot.ControlFlowGraph(scope)
	if !ok || graph.Scope() != scope {
		t.Fatalf("large custom scope graph = %#v, %v; want %#v, true", graph.Scope(), ok, scope)
	}
	if got := snapshot.resolveScopeAtLevel(scope, 1); got != parent {
		t.Fatalf("large custom scope parent = %#v, want %#v", got, parent)
	}
	key, ok := FlowStatementKeyForNode(flowStorageTestFile, statement)
	if !ok {
		t.Fatal("expected a large-offset statement key")
	}
	if reachable, found := snapshot.StatementReachable(key); !found || !reachable {
		t.Fatalf("large-offset statement reachability = %v, %v; want true, true", reachable, found)
	}
}

type flowStorageFixture struct {
	owner      *ast.BlockNode
	statements []ast.Node
}

func makeFlowStorageFixtures(scopeCount, statementsPerScope int) []flowStorageFixture {
	fixtures := make([]flowStorageFixture, scopeCount)
	for i := range fixtures {
		base := 1000 + i*10000
		statements := make([]ast.Node, statementsPerScope)
		for j := range statements {
			start := base + 10 + j*10
			statements[j] = flowStorageStatement(start, start+5)
		}
		fixtures[i] = flowStorageFixture{
			owner:      flowStorageOwner(base, base+9000),
			statements: statements,
		}
	}
	return fixtures
}

func addFlowStorageFixtures(snapshot *SemanticSnapshot, fixtures []flowStorageFixture) {
	for _, fixture := range fixtures {
		snapshot.addFlowScope(flowStorageTestFile, "block", fixture.owner, fixture.statements, FlowScopeKey{})
	}
}

func measureFlowStorageAllocations(fixtures []flowStorageFixture) float64 {
	return testing.AllocsPerRun(20, func() {
		snapshot, err := NewSemanticSnapshot(emptyFlowStorageProject, nil)
		if err != nil {
			panic(err)
		}
		addFlowStorageFixtures(snapshot, fixtures)
	})
}

func TestFlowStorageManyLinearScopesHaveBoundedAllocations(t *testing.T) {
	small := makeFlowStorageFixtures(16, 1)
	large := makeFlowStorageFixtures(16, 64)
	baseline := testing.AllocsPerRun(20, func() {
		if _, err := NewSemanticSnapshot(emptyFlowStorageProject, nil); err != nil {
			panic(err)
		}
	})
	smallAllocs := measureFlowStorageAllocations(small) - baseline
	largeAllocs := measureFlowStorageAllocations(large) - baseline

	t.Logf("control-flow storage allocations: baseline=%.2f small=%.2f large=%.2f", baseline, smallAllocs, largeAllocs)
	if largeAllocs > 140 {
		t.Fatalf("many linear scopes/statements allocations = %.2f, want <= 140 beyond snapshot setup", largeAllocs)
	}
	if largeAllocs-smallAllocs > 24 {
		t.Fatalf("adding statements caused %.2f extra allocations beyond scope setup, want <= 24", largeAllocs-smallAllocs)
	}
}

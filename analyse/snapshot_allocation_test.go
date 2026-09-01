package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestSemanticFactStorePutPartsKeepsFirstDuplicate(t *testing.T) {
	t.Parallel()
	store := make(semanticFactStore)
	key := SemanticFactKey{File: "narrowing.php", StartOffset: 4, EndOffset: 8, Kind: FactKindNarrowed}
	if !store.putParts(key, "User", "User", "instanceof") {
		t.Fatal("expected first narrowing fact to be stored")
	}
	if store.putParts(key, "Admin", "Admin", "instanceof") {
		t.Fatal("duplicate span should keep the first fact")
	}
	if !store.has(key) {
		t.Fatal("expected stored fact to be visible through has")
	}
	fact, ok := store.fact(key)
	if !ok {
		t.Fatal("expected stored fact to be readable")
	}
	if fact.Type != "User" || fact.Subject != "User" || fact.Value != "instanceof" {
		t.Fatalf("duplicate insert mutated stored fact: %#v", fact)
	}
}

func TestLinearControlFlowGraphAvoidsPerStatementSuccessorSlices(t *testing.T) {
	nodes := parseControlFlowPHP(t, `<?php
function graph($value): void {
    $first = $value;
    $second = $first;
    $third = $second;
    return;
    $last = true;
}`)
	function := controlFlowFunction(t, nodes, "graph")
	scope := flowScopeKey("graph.php", "function", function, function.Body)

	graph := buildLinearControlFlowGraph(scope, function.Body)
	if !graph.blockReachable(0) || !graph.blockReachable(1) || !graph.blockReachable(2) || !graph.blockReachable(3) || !graph.blockReachable(4) {
		t.Fatalf("expected entry and reachable statements to be marked: %#v", graph.reachable)
	}
	if graph.blockReachable(5) || graph.MayFallThrough() {
		t.Fatal("expected the return to make the trailing statement and exit unreachable")
	}
	for _, block := range graph.blocks {
		if block.successors.n <= 2 && block.successors.more != nil {
			t.Fatalf("compact successors allocated an overflow slice for n=%d: %#v", block.successors.n, block.successors)
		}
	}

	blocks := graph.Blocks()
	if got, want := len(blocks), 7; got != want {
		t.Fatalf("block count = %d, want %d", got, want)
	}
	if !reflectDeepEqualFlowIDs(blocks[0].Successors, []FlowNodeID{1}) {
		t.Fatalf("entry successors = %v, want [1]", blocks[0].Successors)
	}
	blocks[0].Successors[0] = 99
	if graph.blocks[0].successors.a != 1 {
		t.Fatal("Blocks leaked successor mutation into stored graph")
	}

	allocs := testing.AllocsPerRun(50, func() {
		built := buildLinearControlFlowGraph(scope, function.Body)
		if !built.blockReachable(1) {
			t.Fatal("linear graph lost reachability")
		}
	})
	if allocs > 4 {
		t.Fatalf("linear control-flow construction allocations/run = %.2f, want <= 4", allocs)
	}
}

func TestLoopControlFlowGraphStoresReachabilityOnce(t *testing.T) {
	t.Parallel()
	nodes := parseControlFlowPHP(t, `<?php
function run($ready): void {
    while ($ready) {
        $seen = true;
        break;
    }
}`)
	function := controlFlowFunction(t, nodes, "run")
	loop, ok := function.Body[0].(*ast.WhileNode)
	if !ok {
		t.Fatalf("expected while node, got %T", function.Body[0])
	}
	scope := flowScopeKey("loop.php", "while", loop, loop.Body)
	graph := buildLoopControlFlowGraph(scope, loop, loop.Body)
	if len(graph.reachable) != len(graph.blocks) {
		t.Fatalf("reachable bitmap length = %d, want %d", len(graph.reachable), len(graph.blocks))
	}
	if !graph.blockReachable(0) || !graph.blockReachable(1) {
		t.Fatal("expected loop header and body to be reachable")
	}
}

func reflectDeepEqualFlowIDs(got, want []FlowNodeID) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestASCIILowerIdentReplacesHotUnicodeFolding(t *testing.T) {
	t.Parallel()
	if got, want := asciiLowerIdent("Get_Posts"), "get_posts"; got != want {
		t.Fatalf("asciiLowerIdent=%q, want %q", got, want)
	}
	name := "already_lower"
	if got := asciiLowerIdent(name); got != name {
		t.Fatalf("already-lowercase ident allocated or copied: %q vs %q", got, name)
	}
}

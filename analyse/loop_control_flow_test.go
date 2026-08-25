package analyse

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

type loopFlowFixture struct {
	filename   string
	owner      ast.Node
	statements []ast.Node
	graph      ControlFlowGraph
	snapshot   *SemanticSnapshot
}

func TestLoopControlFlowConditionalExitAndBackEdge(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		kind            string
		conditionBlock  FlowNodeID
		backEdgeTarget  FlowNodeID
		conditionTarget FlowNodeID
	}{
		{name: "for", source: `<?php function run($items): void { for ($index = 0; $index < 2; $index++) { $seen = $index; } }`, kind: "for", conditionBlock: 0, backEdgeTarget: 0, conditionTarget: 1},
		{name: "while", source: `<?php function run($ready): void { while ($ready) { $seen = true; } }`, kind: "while", conditionBlock: 0, backEdgeTarget: 0, conditionTarget: 1},
		{name: "foreach", source: `<?php function run($items): void { foreach ($items as $item) { $seen = $item; } }`, kind: "foreach", conditionBlock: 0, backEdgeTarget: 0, conditionTarget: 1},
		{name: "do", source: `<?php function run($ready): void { do { $seen = true; } while ($ready); }`, kind: "do", conditionBlock: 2, backEdgeTarget: 2, conditionTarget: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoopFlowFixture(t, test.name+".php", test.source, test.kind)
			blocks := fixture.graph.Blocks()
			exitID := blocks[len(blocks)-1].ID

			assertFlowSuccessor(t, blocks[test.conditionBlock], test.conditionTarget)
			assertFlowSuccessor(t, blocks[test.conditionBlock], exitID)
			bodyID := flowBlockForStatement(t, blocks, fixture.filename, fixture.statements[0]).ID
			assertFlowSuccessor(t, blocks[bodyID], test.backEdgeTarget)
			if !fixture.graph.MayFallThrough() {
				t.Fatal("conditional loop should retain a reachable exit")
			}
		})
	}
}

func TestLoopControlTransfersTargetLoopEdgesAndMakeFollowingStatementUnreachable(t *testing.T) {
	loopTemplates := []struct {
		name   string
		kind   string
		format string
	}{
		{name: "for", kind: "for", format: `<?php function run($items, $ready): void { for ($index = 0; $index < 2; $index++) { %s $never = true; } }`},
		{name: "while", kind: "while", format: `<?php function run($items, $ready): void { while ($ready) { %s $never = true; } }`},
		{name: "foreach", kind: "foreach", format: `<?php function run($items, $ready): void { foreach ($items as $item) { %s $never = true; } }`},
		{name: "do", kind: "do", format: `<?php function run($items, $ready): void { do { %s $never = true; } while ($ready); }`},
	}
	transfers := []struct {
		name      string
		statement string
		target    func([]FlowBlock) FlowNodeID
	}{
		{name: "break", statement: "break;", target: func(blocks []FlowBlock) FlowNodeID { return blocks[len(blocks)-1].ID }},
		{name: "continue", statement: "continue;", target: func(blocks []FlowBlock) FlowNodeID {
			if len(blocks) == 5 { // do: entry, two statements, condition, exit
				return blocks[len(blocks)-2].ID
			}
			return 0
		}},
	}

	for _, loop := range loopTemplates {
		for _, transfer := range transfers {
			t.Run(loop.name+"/"+transfer.name, func(t *testing.T) {
				fixture := newLoopFlowFixture(t, loop.name+"-"+transfer.name+".php", fmt.Sprintf(loop.format, transfer.statement), loop.kind)
				blocks := fixture.graph.Blocks()
				transferBlock := flowBlockForStatement(t, blocks, fixture.filename, fixture.statements[0])
				if got, want := transferBlock.Successors, []FlowNodeID{transfer.target(blocks)}; !reflect.DeepEqual(got, want) {
					t.Fatalf("transfer successors = %v, want %v", got, want)
				}
				if got, found := fixture.snapshot.StatementReachable(flowStatementKey(fixture.filename, fixture.statements[1])); !found || got {
					t.Fatalf("statement after %s reachability = %v, %v; want false, true", transfer.name, got, found)
				}
			})
		}
	}
}

func TestLoopControlNumericLevelOneUsesDirectTargets(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantExit  bool
	}{
		{name: "break", statement: "break 1;", wantExit: true},
		{name: "continue", statement: "continue 1;", wantExit: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoopFlowFixture(t, test.name+"-level.php", fmt.Sprintf(`<?php function run(): void { for (;;) { %s $never = true; } }`, test.statement), "for")
			blocks := fixture.graph.Blocks()
			transfer := flowBlockForStatement(t, blocks, fixture.filename, fixture.statements[0])
			wantTarget := FlowNodeID(0)
			if test.wantExit {
				wantTarget = blocks[len(blocks)-1].ID
			}
			if got := transfer.Successors; !reflect.DeepEqual(got, []FlowNodeID{wantTarget}) {
				t.Fatalf("numeric level-one %s successors = %v, want [%d]", test.name, got, wantTarget)
			}
		})
	}
}

func TestMultiLevelLoopControlDoesNotTargetInnerLoop(t *testing.T) {
	for _, statement := range []string{"break 2;", "continue 2;"} {
		t.Run(statement, func(t *testing.T) {
			fixture := newLoopFlowFixture(t, "multi-level.php", fmt.Sprintf(`<?php function run(): void { for (;;) { %s $never = true; } }`, statement), "for")
			blocks := fixture.graph.Blocks()
			transfer := flowBlockForStatement(t, blocks, fixture.filename, fixture.statements[0])
			if len(transfer.Successors) != 0 {
				t.Fatalf("multi-level transfer successors = %v; want no incorrect inner-loop target", transfer.Successors)
			}
		})
	}
}

func TestLoopControlExhaustiveIfTransfersDoNotReachFollowingStatement(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		source string
	}{
		{name: "for", kind: "for", source: `<?php function run($stop): void { for ($index = 0; $index < 2; $index++) { if ($stop) { break; } else { continue; } $never = true; } }`},
		{name: "while", kind: "while", source: `<?php function run($stop): void { while (true) { if ($stop) { break; } else { continue; } $never = true; } }`},
		{name: "foreach", kind: "foreach", source: `<?php function run($items, $stop): void { foreach ($items as $item) { if ($stop) { break; } else { continue; } $never = true; } }`},
		{name: "do", kind: "do", source: `<?php function run($stop): void { do { if ($stop) { break; } else { continue; } $never = true; } while (true); }`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoopFlowFixture(t, "nested-"+test.name+".php", test.source, test.kind)
			blocks := fixture.graph.Blocks()
			transfer := flowBlockForStatement(t, blocks, fixture.filename, fixture.statements[0])
			exitID := blocks[len(blocks)-1].ID
			continueID := FlowNodeID(0)
			if test.kind == "do" {
				continueID = blocks[len(blocks)-2].ID
			}
			if len(transfer.Successors) != 2 {
				t.Fatalf("exhaustive transfer successors = %v, want exactly continue and exit targets", transfer.Successors)
			}
			assertFlowSuccessor(t, transfer, continueID)
			assertFlowSuccessor(t, transfer, exitID)
			if got, found := fixture.snapshot.StatementReachable(flowStatementKey(fixture.filename, fixture.statements[1])); !found || got {
				t.Fatalf("statement after exhaustive transfer reachability = %v, %v; want false, true", got, found)
			}
		})
	}
}

func TestInfiniteForFallthroughRequiresReachableBreak(t *testing.T) {
	withoutBreak := newLoopFlowFixture(t, "infinite-continue.php", `<?php function run(): void { for (;;) { continue; } }`, "for")
	if withoutBreak.graph.MayFallThrough() {
		t.Fatal("for (;;) with no reachable break should not fall through")
	}

	withBreak := newLoopFlowFixture(t, "infinite-break.php", `<?php function run($stop): void { for (;;) { if ($stop) { break; } continue; } }`, "for")
	if !withBreak.graph.MayFallThrough() {
		t.Fatal("for (;;) with a reachable conditional break should fall through")
	}
}

func TestLoopControlFlowBlocksAreDefensiveAndDeterministic(t *testing.T) {
	const source = `<?php function run($ready): void { do { if ($ready) { continue; } break; } while ($ready); }`
	first := newLoopFlowFixture(t, "stable-loop.php", source, "do")
	second := newLoopFlowFixture(t, "stable-loop.php", source, "do")
	firstBlocks := first.graph.Blocks()
	if !reflect.DeepEqual(firstBlocks, second.graph.Blocks()) {
		t.Fatalf("repeated loop graph changed:\nfirst %#v\nsecond %#v", firstBlocks, second.graph.Blocks())
	}
	for id, block := range firstBlocks {
		if block.ID != FlowNodeID(id) {
			t.Fatalf("block %d ID = %d, want deterministic ID %d", id, block.ID, id)
		}
	}

	firstBlocks[0].ID = 99
	firstBlocks[0].Successors[0] = 99
	again := first.graph.Blocks()
	if again[0].ID != 0 || again[0].Successors[0] == 99 {
		t.Fatalf("loop graph leaked caller mutation: %#v", again[0])
	}
}

func newLoopFlowFixture(t *testing.T, filename, source, kind string) loopFlowFixture {
	t.Helper()
	nodes := parseControlFlowPHP(t, source)
	function := controlFlowFunction(t, nodes, "run")
	if len(function.Body) != 1 {
		t.Fatalf("function body count = %d, want one loop", len(function.Body))
	}
	owner := function.Body[0]
	var statements []ast.Node
	switch loop := owner.(type) {
	case *ast.ForNode:
		statements = loop.Body
	case *ast.WhileNode:
		statements = loop.Body
	case *ast.ForeachNode:
		statements = loop.Body
	case *ast.DoWhileNode:
		statements = loop.Body
	default:
		t.Fatalf("loop node = %T, want supported loop", owner)
	}
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	scope, ok := FlowScopeKeyForNode(filename, kind, owner, statements)
	if !ok {
		t.Fatalf("build %s loop scope key", kind)
	}
	graph, ok := snapshot.ControlFlowGraph(scope)
	if !ok {
		t.Fatalf("find %s loop graph for %#v", kind, scope)
	}
	return loopFlowFixture{filename: filename, owner: owner, statements: statements, graph: graph, snapshot: snapshot}
}

func flowBlockForStatement(t *testing.T, blocks []FlowBlock, filename string, statement ast.Node) FlowBlock {
	t.Helper()
	want := flowStatementKey(filename, statement)
	for _, block := range blocks {
		if block.Statement == want {
			return block
		}
	}
	t.Fatalf("no flow block for statement %#v", want)
	return FlowBlock{}
}

func assertFlowSuccessor(t *testing.T, block FlowBlock, want FlowNodeID) {
	t.Helper()
	for _, successor := range block.Successors {
		if successor == want {
			return
		}
	}
	t.Fatalf("block %d successors = %v, want edge to %d", block.ID, block.Successors, want)
}

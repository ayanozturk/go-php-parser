package analyse

import "github.com/ayanozturk/go-php-parser/ast"

// FlowNodeID is a stable node identifier within one control-flow graph.
type FlowNodeID uint32

// FlowScopeKey identifies one lexical statement scope by source span and kind.
// File scopes use Kind "file" and start at offset zero.
type FlowScopeKey struct {
	File        string
	StartOffset int
	EndOffset   int
	Kind        string
}

// FlowStatementKey identifies one statement by its exact source span.
type FlowStatementKey struct {
	File        string
	StartOffset int
	EndOffset   int
}

// FlowGraphReader is the read-only flow contract shared by analysis rules.
type FlowGraphReader interface {
	StatementReachable(key FlowStatementKey) (bool, bool)
	ScopeMayFallThrough(key FlowScopeKey) (bool, bool)
	ControlFlowGraph(key FlowScopeKey) (ControlFlowGraph, bool)
}

// FlowBlock is an immutable value returned to graph consumers. Entry and exit
// blocks have a zero Statement key.
type FlowBlock struct {
	ID         FlowNodeID
	Statement  FlowStatementKey
	Successors []FlowNodeID
}

// ControlFlowGraph is a compact graph for one lexical scope. Compound
// statements are atomic in their parent graph and have separate child graphs;
// loop child graphs also contain zero-statement header/condition/exit blocks.
// Blocks returns defensive copies.
type ControlFlowGraph struct {
	scope          FlowScopeKey
	blocks         []FlowBlock
	mayFallThrough bool
}

func (g ControlFlowGraph) Scope() FlowScopeKey { return g.scope }

func (g ControlFlowGraph) MayFallThrough() bool { return g.mayFallThrough }

func (g ControlFlowGraph) Blocks() []FlowBlock {
	blocks := make([]FlowBlock, len(g.blocks))
	for i, block := range g.blocks {
		blocks[i] = block
		blocks[i].Successors = append([]FlowNodeID(nil), block.Successors...)
	}
	return blocks
}

type flowOutcome uint8

const (
	flowNormal flowOutcome = 1 << iota
	flowTerminate
	flowBreak
	flowContinue
	flowEscape
)

func buildControlFlowGraph(scope FlowScopeKey, owner ast.Node, statements []ast.Node) ControlFlowGraph {
	if isLoopFlowScope(scope.Kind) {
		return buildLoopControlFlowGraph(scope, owner, statements)
	}
	return buildLinearControlFlowGraph(scope, statements)
}

func buildLinearControlFlowGraph(scope FlowScopeKey, statements []ast.Node) ControlFlowGraph {
	blocks := make([]FlowBlock, len(statements)+2)
	exitID := FlowNodeID(len(statements) + 1)
	blocks[0] = FlowBlock{ID: 0}
	if len(statements) == 0 {
		blocks[0].Successors = []FlowNodeID{exitID}
	}

	mayReachNext := true
	for i, statement := range statements {
		id := FlowNodeID(i + 1)
		blocks[i+1] = FlowBlock{ID: id, Statement: flowStatementKey(scope.File, statement)}
		if i == 0 {
			blocks[0].Successors = []FlowNodeID{id}
		}
		if !mayReachNext {
			continue
		}
		if statementFlowOutcomes(statement)&flowNormal == 0 {
			mayReachNext = false
			continue
		}
		next := exitID
		if i+1 < len(statements) {
			next = FlowNodeID(i + 2)
		}
		blocks[i+1].Successors = []FlowNodeID{next}
	}
	blocks[len(blocks)-1] = FlowBlock{ID: exitID}
	return ControlFlowGraph{scope: scope, blocks: blocks, mayFallThrough: mayReachNext}
}

func buildLoopControlFlowGraph(scope FlowScopeKey, owner ast.Node, statements []ast.Node) ControlFlowGraph {
	isDo := scope.Kind == "do"
	extraBlocks := 2
	if isDo {
		extraBlocks = 3
	}
	blocks := make([]FlowBlock, len(statements)+extraBlocks)
	conditionID := FlowNodeID(0)
	exitID := FlowNodeID(len(statements) + 1)
	if isDo {
		conditionID = FlowNodeID(len(statements) + 1)
		exitID++
	}

	firstBodyID := conditionID
	if len(statements) > 0 {
		firstBodyID = 1
	}
	mayIterate, mayExit := loopConditionPaths(owner)
	if isDo {
		blocks[0] = FlowBlock{ID: 0, Successors: []FlowNodeID{firstBodyID}}
		conditionSuccessors := make([]FlowNodeID, 0, 2)
		if mayIterate {
			conditionSuccessors = appendUniqueFlowNode(conditionSuccessors, firstBodyID)
		}
		if mayExit {
			conditionSuccessors = appendUniqueFlowNode(conditionSuccessors, exitID)
		}
		blocks[conditionID] = FlowBlock{ID: conditionID, Successors: conditionSuccessors}
	} else {
		headerSuccessors := make([]FlowNodeID, 0, 2)
		if mayIterate {
			headerSuccessors = appendUniqueFlowNode(headerSuccessors, firstBodyID)
		}
		if mayExit {
			headerSuccessors = appendUniqueFlowNode(headerSuccessors, exitID)
		}
		blocks[0] = FlowBlock{ID: 0, Successors: headerSuccessors}
	}

	continueTarget := FlowNodeID(0)
	if isDo {
		continueTarget = conditionID
	}
	for i, statement := range statements {
		id := FlowNodeID(i + 1)
		outcomes := statementFlowOutcomes(statement)
		successors := make([]FlowNodeID, 0, 3)
		if outcomes&flowNormal != 0 {
			next := continueTarget
			if i+1 < len(statements) {
				next = FlowNodeID(i + 2)
			}
			successors = appendUniqueFlowNode(successors, next)
		}
		if outcomes&flowContinue != 0 {
			successors = appendUniqueFlowNode(successors, continueTarget)
		}
		if outcomes&flowBreak != 0 {
			successors = appendUniqueFlowNode(successors, exitID)
		}
		blocks[i+1] = FlowBlock{ID: id, Statement: flowStatementKey(scope.File, statement), Successors: successors}
	}
	blocks[exitID] = FlowBlock{ID: exitID}
	return newControlFlowGraph(scope, blocks, exitID)
}

func newControlFlowGraph(scope FlowScopeKey, blocks []FlowBlock, exitID FlowNodeID) ControlFlowGraph {
	reachable := reachableFlowNodes(blocks)
	return ControlFlowGraph{scope: scope, blocks: blocks, mayFallThrough: reachable[exitID]}
}

func reachableFlowNodes(blocks []FlowBlock) map[FlowNodeID]bool {
	reachable := make(map[FlowNodeID]bool, len(blocks))
	if len(blocks) == 0 {
		return reachable
	}
	queue := []FlowNodeID{0}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if reachable[id] || int(id) >= len(blocks) {
			continue
		}
		reachable[id] = true
		queue = append(queue, blocks[id].Successors...)
	}
	return reachable
}

func appendUniqueFlowNode(nodes []FlowNodeID, node FlowNodeID) []FlowNodeID {
	for _, existing := range nodes {
		if existing == node {
			return nodes
		}
	}
	return append(nodes, node)
}

func isLoopFlowScope(kind string) bool {
	switch kind {
	case "for", "while", "foreach", "do":
		return true
	default:
		return false
	}
}

func loopConditionPaths(owner ast.Node) (mayIterate, mayExit bool) {
	var condition ast.Node
	switch loop := owner.(type) {
	case *ast.ForNode:
		if len(loop.Conditions) == 0 {
			return true, false
		}
		condition = loop.Conditions[len(loop.Conditions)-1]
	case *ast.WhileNode:
		condition = loop.Condition
	case *ast.DoWhileNode:
		condition = loop.Condition
	case *ast.ForeachNode:
		return true, true
	default:
		return true, true
	}
	switch literal := condition.(type) {
	case *ast.BooleanLiteral:
		return literal.Value, !literal.Value
	case *ast.BooleanNode:
		return literal.Value, !literal.Value
	}
	return true, true
}

func statementFlowOutcomes(node ast.Node) flowOutcome {
	switch n := node.(type) {
	case *ast.ReturnNode, *ast.ThrowNode:
		return flowTerminate
	case *ast.BreakNode:
		if loopControlLevelOne(n.Level) {
			return flowBreak
		}
		return flowEscape
	case *ast.ContinueNode:
		if loopControlLevelOne(n.Level) {
			return flowContinue
		}
		return flowEscape
	case *ast.ExpressionStmt:
		if call, ok := n.Expr.(*ast.FunctionCallNode); ok && isBuiltinTerminatorCall(call) {
			return flowTerminate
		}
		return flowNormal
	case *ast.IfNode:
		outcomes := statementsFlowOutcomes(n.Body)
		for _, elseif := range n.ElseIfs {
			outcomes |= statementsFlowOutcomes(elseif.Body)
		}
		if n.Else == nil {
			outcomes |= flowNormal
		} else {
			outcomes |= statementsFlowOutcomes(n.Else.Body)
		}
		return outcomes
	default:
		return flowNormal
	}
}

func statementsFlowOutcomes(statements []ast.Node) flowOutcome {
	outcomes := flowNormal
	for _, statement := range statements {
		next := outcomes &^ flowNormal
		if outcomes&flowNormal != 0 {
			next |= statementFlowOutcomes(statement)
		}
		outcomes = next
	}
	return outcomes
}

func loopControlLevelOne(level ast.Node) bool {
	if level == nil {
		return true
	}
	switch literal := level.(type) {
	case *ast.IntegerLiteral:
		return literal.Value == 1
	case *ast.IntegerNode:
		return literal.Value == 1
	default:
		return false
	}
}

// FlowStatementKeyForNode returns an exact statement key when the node has a
// usable source span.
func FlowStatementKeyForNode(filename string, node ast.Node) (FlowStatementKey, bool) {
	key := flowStatementKey(filename, node)
	return key, key.File != ""
}

func flowStatementKey(filename string, node ast.Node) FlowStatementKey {
	if node == nil {
		return FlowStatementKey{}
	}
	start, end := node.GetPos(), node.GetEndPos()
	if filename == "" || start.Offset < 0 || end.Offset <= start.Offset {
		return FlowStatementKey{}
	}
	return FlowStatementKey{File: filename, StartOffset: start.Offset, EndOffset: end.Offset}
}

// FlowScopeKeyForNode returns the lexical-scope key used by a snapshot. The
// final statement end is used when a parser helper node lacks its own end span.
func FlowScopeKeyForNode(filename, kind string, owner ast.Node, statements []ast.Node) (FlowScopeKey, bool) {
	key := flowScopeKey(filename, kind, owner, statements)
	return key, key.File != ""
}

func flowScopeKey(filename, kind string, owner ast.Node, statements []ast.Node) FlowScopeKey {
	if filename == "" || kind == "" {
		return FlowScopeKey{}
	}
	if kind == "file" {
		end := 0
		for _, statement := range statements {
			if statement != nil && statement.GetEndPos().Offset > end {
				end = statement.GetEndPos().Offset
			}
		}
		return FlowScopeKey{File: filename, EndOffset: end, Kind: kind}
	}
	if owner == nil {
		return FlowScopeKey{}
	}
	start, end := owner.GetPos(), owner.GetEndPos()
	if end.Offset <= start.Offset {
		for _, statement := range statements {
			if statement != nil && statement.GetEndPos().Offset > end.Offset {
				end = statement.GetEndPos()
			}
		}
	}
	if start.Offset < 0 || end.Offset <= start.Offset {
		return FlowScopeKey{}
	}
	return FlowScopeKey{File: filename, StartOffset: start.Offset, EndOffset: end.Offset, Kind: kind}
}

func (s *SemanticSnapshot) generateControlFlowGraphs(parsed map[string][]ast.Node) {
	s.flowGraphs = make(map[FlowScopeKey]ControlFlowGraph)
	s.statementReachability = make(map[FlowStatementKey]bool)
	s.ambiguousFlowStatements = make(map[FlowStatementKey]struct{})

	for _, filename := range s.filenames {
		s.addFlowScope(filename, "file", nil, parsed[filename])
	}
}

func (s *SemanticSnapshot) addFlowScope(filename, kind string, owner ast.Node, statements []ast.Node) {
	scope := flowScopeKey(filename, kind, owner, statements)
	if scope.File == "" {
		return
	}
	graph := buildControlFlowGraph(scope, owner, statements)
	if _, duplicate := s.flowGraphs[scope]; duplicate {
		return
	}
	s.flowGraphs[scope] = graph

	reachableBlocks := reachableFlowNodes(graph.blocks)
	for i, statement := range statements {
		reachable := reachableBlocks[FlowNodeID(i+1)]
		key := flowStatementKey(filename, statement)
		if key.File != "" {
			s.recordStatementReachability(key, reachable)
		}
		if !reachable {
			continue
		}
		s.addChildFlowScopes(filename, statement)
	}
}

func (s *SemanticSnapshot) recordStatementReachability(key FlowStatementKey, reachable bool) {
	if _, ambiguous := s.ambiguousFlowStatements[key]; ambiguous {
		return
	}
	if _, exists := s.statementReachability[key]; exists {
		delete(s.statementReachability, key)
		s.ambiguousFlowStatements[key] = struct{}{}
		return
	}
	s.statementReachability[key] = reachable
}

func (s *SemanticSnapshot) addChildFlowScopes(filename string, node ast.Node) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		s.addFlowScope(filename, "function", n, n.Body)
	case *ast.ClassNode:
		for _, method := range n.Methods {
			s.addChildFlowScopes(filename, method)
		}
	case *ast.BlockNode:
		s.addFlowScope(filename, "block", n, n.Statements)
	case *ast.IfNode:
		s.addFlowScope(filename, "if", n, n.Body)
		for _, elseif := range n.ElseIfs {
			s.addFlowScope(filename, "elseif", elseif, elseif.Body)
		}
		if n.Else != nil {
			s.addFlowScope(filename, "else", n.Else, n.Else.Body)
		}
	case *ast.WhileNode:
		s.addFlowScope(filename, "while", n, n.Body)
	case *ast.ForNode:
		s.addFlowScope(filename, "for", n, n.Body)
	case *ast.ForeachNode:
		s.addFlowScope(filename, "foreach", n, n.Body)
	case *ast.DoWhileNode:
		s.addFlowScope(filename, "do", n, n.Body)
	case *ast.NamespaceNode:
		s.addFlowScope(filename, "namespace", n, n.Body)
	}
}

func (s *SemanticSnapshot) StatementReachable(key FlowStatementKey) (bool, bool) {
	if s == nil || key.File == "" {
		return false, false
	}
	if _, ambiguous := s.ambiguousFlowStatements[key]; ambiguous {
		return false, false
	}
	reachable, ok := s.statementReachability[key]
	return reachable, ok
}

func (s *SemanticSnapshot) ScopeMayFallThrough(key FlowScopeKey) (bool, bool) {
	graph, ok := s.ControlFlowGraph(key)
	if !ok {
		return false, false
	}
	return graph.MayFallThrough(), true
}

func (s *SemanticSnapshot) ControlFlowGraph(key FlowScopeKey) (ControlFlowGraph, bool) {
	if s == nil {
		return ControlFlowGraph{}, false
	}
	graph, ok := s.flowGraphs[key]
	return graph, ok
}

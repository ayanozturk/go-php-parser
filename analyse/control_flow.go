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

// ControlFlowGraph is a compact statement-level graph for one lexical scope.
// Compound statements are atomic in this first slice; their child scopes have
// separate graphs. Blocks returns defensive copies.
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

func buildControlFlowGraph(scope FlowScopeKey, statements []ast.Node) ControlFlowGraph {
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
		if isTerminatingStatement(statement) {
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
	graph := buildControlFlowGraph(scope, statements)
	if _, duplicate := s.flowGraphs[scope]; duplicate {
		return
	}
	s.flowGraphs[scope] = graph

	reachable := true
	for _, statement := range statements {
		key := flowStatementKey(filename, statement)
		if key.File != "" {
			s.recordStatementReachability(key, reachable)
		}
		if !reachable {
			continue
		}
		s.addChildFlowScopes(filename, statement)
		if isTerminatingStatement(statement) {
			reachable = false
		}
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

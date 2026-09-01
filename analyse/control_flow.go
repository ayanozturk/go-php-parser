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
	blocks         []storedFlowBlock
	reachable      []bool
	mayFallThrough bool
}

// storedFlowBlock keeps the common one- and two-successor cases inline so
// graph construction does not allocate a successor slice per statement.
type storedFlowBlock struct {
	id         FlowNodeID
	statement  storedFlowStatementKey
	successors compactSuccessors
}

// storedFlowStatementKey omits the filename because a graph belongs to one
// file. The public FlowStatementKey is reconstructed at the reader boundary.
type storedFlowStatementKey struct {
	startOffset int
	endOffset   int
}

type compactSuccessors struct {
	a, b FlowNodeID
	n    uint8
	more []FlowNodeID
}

func (g ControlFlowGraph) Scope() FlowScopeKey { return g.scope }

func (g ControlFlowGraph) MayFallThrough() bool { return g.mayFallThrough }

func (g ControlFlowGraph) Blocks() []FlowBlock {
	blocks := make([]FlowBlock, len(g.blocks))
	for i, block := range g.blocks {
		statement := FlowStatementKey{}
		if block.statement.endOffset > block.statement.startOffset {
			statement = FlowStatementKey{File: g.scope.File, StartOffset: block.statement.startOffset, EndOffset: block.statement.endOffset}
		}
		blocks[i] = FlowBlock{
			ID:         block.id,
			Statement:  statement,
			Successors: block.successors.slice(),
		}
	}
	return blocks
}

func (g ControlFlowGraph) blockReachable(id FlowNodeID) bool {
	idx := int(id)
	return idx >= 0 && idx < len(g.reachable) && g.reachable[idx]
}

func oneSuccessor(id FlowNodeID) compactSuccessors {
	return compactSuccessors{a: id, n: 1}
}

func (s compactSuccessors) slice() []FlowNodeID {
	switch s.n {
	case 0:
		return nil
	case 1:
		return []FlowNodeID{s.a}
	case 2:
		return []FlowNodeID{s.a, s.b}
	default:
		out := make([]FlowNodeID, int(s.n))
		out[0], out[1] = s.a, s.b
		copy(out[2:], s.more)
		return out
	}
}

func (s compactSuccessors) appendTo(dst []FlowNodeID) []FlowNodeID {
	switch s.n {
	case 0:
		return dst
	case 1:
		return append(dst, s.a)
	case 2:
		return append(dst, s.a, s.b)
	default:
		dst = append(dst, s.a, s.b)
		return append(dst, s.more...)
	}
}

func appendUniqueCompact(s compactSuccessors, node FlowNodeID) compactSuccessors {
	switch s.n {
	case 0:
		return compactSuccessors{a: node, n: 1}
	case 1:
		if s.a == node {
			return s
		}
		return compactSuccessors{a: s.a, b: node, n: 2}
	case 2:
		if s.a == node || s.b == node {
			return s
		}
		return compactSuccessors{a: s.a, b: s.b, n: 3, more: []FlowNodeID{node}}
	default:
		if s.a == node || s.b == node {
			return s
		}
		for _, existing := range s.more {
			if existing == node {
				return s
			}
		}
		s.more = append(append([]FlowNodeID(nil), s.more...), node)
		s.n++
		return s
	}
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
	blocks := make([]storedFlowBlock, len(statements)+2)
	exitID := FlowNodeID(len(statements) + 1)
	reachable := make([]bool, len(blocks))
	reachable[0] = true
	blocks[0] = storedFlowBlock{id: 0}
	if len(statements) == 0 {
		blocks[0].successors = oneSuccessor(exitID)
		reachable[exitID] = true
		blocks[exitID] = storedFlowBlock{id: exitID}
		return ControlFlowGraph{scope: scope, blocks: blocks, reachable: reachable, mayFallThrough: true}
	}

	blocks[0].successors = oneSuccessor(1)
	mayReachNext := true
	for i, statement := range statements {
		id := FlowNodeID(i + 1)
		blocks[i+1] = storedFlowBlock{id: id, statement: storedFlowStatementKeyForNode(statement)}
		if !mayReachNext {
			continue
		}
		reachable[id] = true
		if statementFlowOutcomes(statement)&flowNormal == 0 {
			mayReachNext = false
			continue
		}
		next := exitID
		if i+1 < len(statements) {
			next = FlowNodeID(i + 2)
		}
		blocks[i+1].successors = oneSuccessor(next)
	}
	blocks[exitID] = storedFlowBlock{id: exitID}
	if mayReachNext {
		reachable[exitID] = true
	}
	return ControlFlowGraph{scope: scope, blocks: blocks, reachable: reachable, mayFallThrough: mayReachNext}
}

func buildLoopControlFlowGraph(scope FlowScopeKey, owner ast.Node, statements []ast.Node) ControlFlowGraph {
	isDo := scope.Kind == "do"
	extraBlocks := 2
	if isDo {
		extraBlocks = 3
	}
	blocks := make([]storedFlowBlock, len(statements)+extraBlocks)
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
		blocks[0] = storedFlowBlock{id: 0, successors: oneSuccessor(firstBodyID)}
		var conditionSuccessors compactSuccessors
		if mayIterate {
			conditionSuccessors = appendUniqueCompact(conditionSuccessors, firstBodyID)
		}
		if mayExit {
			conditionSuccessors = appendUniqueCompact(conditionSuccessors, exitID)
		}
		blocks[conditionID] = storedFlowBlock{id: conditionID, successors: conditionSuccessors}
	} else {
		var headerSuccessors compactSuccessors
		if mayIterate {
			headerSuccessors = appendUniqueCompact(headerSuccessors, firstBodyID)
		}
		if mayExit {
			headerSuccessors = appendUniqueCompact(headerSuccessors, exitID)
		}
		blocks[0] = storedFlowBlock{id: 0, successors: headerSuccessors}
	}

	continueTarget := FlowNodeID(0)
	if isDo {
		continueTarget = conditionID
	}
	for i, statement := range statements {
		id := FlowNodeID(i + 1)
		outcomes := statementFlowOutcomes(statement)
		var successors compactSuccessors
		if outcomes&flowNormal != 0 {
			next := continueTarget
			if i+1 < len(statements) {
				next = FlowNodeID(i + 2)
			}
			successors = appendUniqueCompact(successors, next)
		}
		if outcomes&flowContinue != 0 {
			successors = appendUniqueCompact(successors, continueTarget)
		}
		if outcomes&flowBreak != 0 {
			successors = appendUniqueCompact(successors, exitID)
		}
		blocks[i+1] = storedFlowBlock{id: id, statement: storedFlowStatementKeyForNode(statement), successors: successors}
	}
	blocks[exitID] = storedFlowBlock{id: exitID}
	return newControlFlowGraph(scope, blocks, exitID)
}

func newControlFlowGraph(scope FlowScopeKey, blocks []storedFlowBlock, exitID FlowNodeID) ControlFlowGraph {
	reachable := reachableStoredBlocks(blocks)
	mayFallThrough := int(exitID) < len(reachable) && reachable[exitID]
	return ControlFlowGraph{scope: scope, blocks: blocks, reachable: reachable, mayFallThrough: mayFallThrough}
}

func reachableStoredBlocks(blocks []storedFlowBlock) []bool {
	reachable := make([]bool, len(blocks))
	if len(blocks) == 0 {
		return reachable
	}
	stack := make([]FlowNodeID, 0, len(blocks))
	stack = append(stack, 0)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		idx := int(id)
		if idx >= len(blocks) || reachable[idx] {
			continue
		}
		reachable[idx] = true
		stack = blocks[idx].successors.appendTo(stack)
	}
	return reachable
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
	case *ast.SwitchNode:
		return switchFlowOutcomes(n)
	case *ast.TryNode:
		return tryFlowOutcomes(n)
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

func storedFlowStatementKeyForNode(node ast.Node) storedFlowStatementKey {
	if node == nil {
		return storedFlowStatementKey{}
	}
	start, end := node.GetPos(), node.GetEndPos()
	if start.Offset < 0 || end.Offset <= start.Offset {
		return storedFlowStatementKey{}
	}
	return storedFlowStatementKey{startOffset: start.Offset, endOffset: end.Offset}
}

func switchFlowOutcomes(n *ast.SwitchNode) flowOutcome {
	if len(n.Cases) == 0 {
		return flowNormal
	}
	outcomes := flowNormal
	hasDefault := false
	canFallThrough := true
	for _, caseNode := range n.Cases {
		if caseNode.IsDefault {
			hasDefault = true
		}
		caseOutcomes := statementsFlowOutcomes(caseNode.Body)
		stripBreak := caseOutcomes &^ flowBreak
		if stripBreak|flowNormal == flowNormal {
			stripBreak = flowNormal
		}
		outcomes |= stripBreak
		if caseOutcomes&flowNormal == 0 && caseOutcomes&flowBreak == 0 && len(caseNode.Body) > 0 {
			canFallThrough = false
		}
	}
	if !hasDefault || canFallThrough {
		outcomes |= flowNormal
	} else {
		outcomes &^= flowNormal
	}
	return outcomes
}

func tryFlowOutcomes(n *ast.TryNode) flowOutcome {
	tryOutcomes := statementsFlowOutcomes(n.Body)
	outcomes := flowNormal
	if len(n.Catches) == 0 {
		outcomes = tryOutcomes
	} else {
		catchOutcomes := tryOutcomes
		for _, catchNode := range n.Catches {
			catchOutcomes |= statementsFlowOutcomes(catchNode.Body)
		}
		outcomes = catchOutcomes
	}
	if len(n.Finally) > 0 {
		finallyOutcomes := statementsFlowOutcomes(n.Finally)
		if finallyOutcomes&(flowTerminate|flowEscape|flowBreak|flowContinue) != 0 {
			return finallyOutcomes
		}
		outcomes |= finallyOutcomes
	}
	return outcomes
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

type compactFlowSpan uint64

func makeCompactFlowSpan(start, end int) (compactFlowSpan, bool) {
	if start < 0 || end < 0 || uint64(start) > maxCompactSourceOffset || uint64(end) > maxCompactSourceOffset {
		return 0, false
	}
	return compactFlowSpan(uint64(uint32(start))<<32 | uint64(uint32(end))), true
}

func (s compactFlowSpan) offsets() (int, int) {
	return int(uint32(uint64(s) >> 32)), int(uint32(s))
}

type storedFlowScopeKind uint8

const (
	storedFlowScopeUnknown storedFlowScopeKind = iota
	storedFlowScopeFile
	storedFlowScopeFunction
	storedFlowScopeBlock
	storedFlowScopeIf
	storedFlowScopeElseIf
	storedFlowScopeElse
	storedFlowScopeWhile
	storedFlowScopeFor
	storedFlowScopeForeach
	storedFlowScopeDo
	storedFlowScopeCase
	storedFlowScopeTry
	storedFlowScopeCatch
	storedFlowScopeFinally
	storedFlowScopeNamespace
)

func storeFlowScopeKind(kind string) (storedFlowScopeKind, bool) {
	switch kind {
	case "file":
		return storedFlowScopeFile, true
	case "function":
		return storedFlowScopeFunction, true
	case "block":
		return storedFlowScopeBlock, true
	case "if":
		return storedFlowScopeIf, true
	case "elseif":
		return storedFlowScopeElseIf, true
	case "else":
		return storedFlowScopeElse, true
	case "while":
		return storedFlowScopeWhile, true
	case "for":
		return storedFlowScopeFor, true
	case "foreach":
		return storedFlowScopeForeach, true
	case "do":
		return storedFlowScopeDo, true
	case "case":
		return storedFlowScopeCase, true
	case "try":
		return storedFlowScopeTry, true
	case "catch":
		return storedFlowScopeCatch, true
	case "finally":
		return storedFlowScopeFinally, true
	case "namespace":
		return storedFlowScopeNamespace, true
	default:
		return storedFlowScopeUnknown, false
	}
}

func (k storedFlowScopeKind) String() string {
	switch k {
	case storedFlowScopeFile:
		return "file"
	case storedFlowScopeFunction:
		return "function"
	case storedFlowScopeBlock:
		return "block"
	case storedFlowScopeIf:
		return "if"
	case storedFlowScopeElseIf:
		return "elseif"
	case storedFlowScopeElse:
		return "else"
	case storedFlowScopeWhile:
		return "while"
	case storedFlowScopeFor:
		return "for"
	case storedFlowScopeForeach:
		return "foreach"
	case storedFlowScopeDo:
		return "do"
	case storedFlowScopeCase:
		return "case"
	case storedFlowScopeTry:
		return "try"
	case storedFlowScopeCatch:
		return "catch"
	case storedFlowScopeFinally:
		return "finally"
	case storedFlowScopeNamespace:
		return "namespace"
	default:
		return ""
	}
}

type storedFlowScopeKey struct {
	span compactFlowSpan
	kind storedFlowScopeKind
}

type localFlowScopeKey struct {
	startOffset int
	endOffset   int
	kind        string
}

type localFlowStatementKey struct {
	startOffset int
	endOffset   int
}

type statementReachabilityState uint8

const (
	statementReachabilityUnknown statementReachabilityState = iota
	statementReachabilityFalse
	statementReachabilityTrue
	statementReachabilityAmbiguous
)

type flowFileStore struct {
	firstGraph            ControlFlowGraph
	firstGraphKey         storedFlowScopeKey
	hasFirstGraph         bool
	graphs                []ControlFlowGraph
	graphIndex            map[storedFlowScopeKey]uint32
	overflowGraphIndex    map[localFlowScopeKey]uint32
	statementReachability map[compactFlowSpan]statementReachabilityState
	overflowReachability  map[localFlowStatementKey]statementReachabilityState
	nesting               map[storedFlowScopeKey]storedFlowScopeKey
	overflowNesting       map[localFlowScopeKey]localFlowScopeKey
	foreignNesting        map[localFlowScopeKey]FlowScopeKey
}

func compactStoredFlowScopeKey(key FlowScopeKey) (storedFlowScopeKey, bool) {
	span, compact := makeCompactFlowSpan(key.StartOffset, key.EndOffset)
	kind, known := storeFlowScopeKind(key.Kind)
	return storedFlowScopeKey{span: span, kind: kind}, compact && known
}

func localStoredFlowScopeKey(key FlowScopeKey) localFlowScopeKey {
	return localFlowScopeKey{startOffset: key.StartOffset, endOffset: key.EndOffset, kind: key.Kind}
}

func publicStoredFlowScopeKey(filename string, key storedFlowScopeKey) FlowScopeKey {
	start, end := key.span.offsets()
	return FlowScopeKey{File: filename, StartOffset: start, EndOffset: end, Kind: key.kind.String()}
}

func publicLocalFlowScopeKey(filename string, key localFlowScopeKey) FlowScopeKey {
	return FlowScopeKey{File: filename, StartOffset: key.startOffset, EndOffset: key.endOffset, Kind: key.kind}
}

func (s *SemanticSnapshot) flowFile(filename string, create bool) *flowFileStore {
	if s == nil || filename == "" {
		return nil
	}
	file := s.flow[filename]
	if file == nil && create {
		file = &flowFileStore{}
		s.flow[filename] = file
	}
	return file
}

func (f *flowFileStore) putGraph(key FlowScopeKey, graph ControlFlowGraph) bool {
	if compact, ok := compactStoredFlowScopeKey(key); ok {
		if f.hasFirstGraph {
			if f.firstGraphKey == compact {
				return false
			}
		} else {
			f.firstGraphKey = compact
			f.firstGraph = graph
			f.hasFirstGraph = true
			return true
		}
		if f.graphIndex == nil {
			f.graphIndex = make(map[storedFlowScopeKey]uint32)
		}
		if _, duplicate := f.graphIndex[compact]; duplicate {
			return false
		}
		f.graphIndex[compact] = uint32(len(f.graphs))
		f.graphs = append(f.graphs, graph)
		return true
	}
	local := localStoredFlowScopeKey(key)
	if f.overflowGraphIndex == nil {
		f.overflowGraphIndex = make(map[localFlowScopeKey]uint32)
	}
	if _, duplicate := f.overflowGraphIndex[local]; duplicate {
		return false
	}
	f.overflowGraphIndex[local] = uint32(len(f.graphs))
	f.graphs = append(f.graphs, graph)
	return true
}

func (f *flowFileStore) graph(key FlowScopeKey) (ControlFlowGraph, bool) {
	if compact, ok := compactStoredFlowScopeKey(key); ok {
		if f.hasFirstGraph && f.firstGraphKey == compact {
			return f.firstGraph, true
		}
		index, found := f.graphIndex[compact]
		if !found || int(index) >= len(f.graphs) {
			return ControlFlowGraph{}, false
		}
		return f.graphs[index], true
	}
	index, found := f.overflowGraphIndex[localStoredFlowScopeKey(key)]
	if !found || int(index) >= len(f.graphs) {
		return ControlFlowGraph{}, false
	}
	return f.graphs[index], true
}

func (f *flowFileStore) putNesting(scope, parent FlowScopeKey) {
	if parent.File != scope.File {
		if f.foreignNesting == nil {
			f.foreignNesting = make(map[localFlowScopeKey]FlowScopeKey)
		}
		f.foreignNesting[localStoredFlowScopeKey(scope)] = parent
		return
	}
	childCompact, childOK := compactStoredFlowScopeKey(scope)
	parentCompact, parentOK := compactStoredFlowScopeKey(parent)
	if childOK && parentOK {
		if f.nesting == nil {
			f.nesting = make(map[storedFlowScopeKey]storedFlowScopeKey)
		}
		f.nesting[childCompact] = parentCompact
		return
	}
	if f.overflowNesting == nil {
		f.overflowNesting = make(map[localFlowScopeKey]localFlowScopeKey)
	}
	f.overflowNesting[localStoredFlowScopeKey(scope)] = localStoredFlowScopeKey(parent)
}

func (f *flowFileStore) parent(filename string, scope FlowScopeKey) (FlowScopeKey, bool) {
	if parent, found := f.foreignNesting[localStoredFlowScopeKey(scope)]; found {
		return parent, true
	}
	if compact, ok := compactStoredFlowScopeKey(scope); ok {
		parent, found := f.nesting[compact]
		if found {
			return publicStoredFlowScopeKey(filename, parent), true
		}
	}
	parent, found := f.overflowNesting[localStoredFlowScopeKey(scope)]
	return publicLocalFlowScopeKey(filename, parent), found
}

func (f *flowFileStore) recordStatement(key FlowStatementKey, reachable bool) {
	state := statementReachabilityFalse
	if reachable {
		state = statementReachabilityTrue
	}
	if span, compact := makeCompactFlowSpan(key.StartOffset, key.EndOffset); compact {
		if f.statementReachability == nil {
			f.statementReachability = make(map[compactFlowSpan]statementReachabilityState)
		}
		if existing := f.statementReachability[span]; existing != statementReachabilityUnknown {
			f.statementReachability[span] = statementReachabilityAmbiguous
			return
		}
		f.statementReachability[span] = state
		return
	}
	local := localFlowStatementKey{startOffset: key.StartOffset, endOffset: key.EndOffset}
	if f.overflowReachability == nil {
		f.overflowReachability = make(map[localFlowStatementKey]statementReachabilityState)
	}
	if existing := f.overflowReachability[local]; existing != statementReachabilityUnknown {
		f.overflowReachability[local] = statementReachabilityAmbiguous
		return
	}
	f.overflowReachability[local] = state
}

func (f *flowFileStore) statement(key FlowStatementKey) (bool, bool) {
	state := statementReachabilityUnknown
	if span, compact := makeCompactFlowSpan(key.StartOffset, key.EndOffset); compact {
		state = f.statementReachability[span]
	} else {
		state = f.overflowReachability[localFlowStatementKey{startOffset: key.StartOffset, endOffset: key.EndOffset}]
	}
	switch state {
	case statementReachabilityFalse:
		return false, true
	case statementReachabilityTrue:
		return true, true
	default:
		return false, false
	}
}

func (s *SemanticSnapshot) generateControlFlowGraphs(parsed map[string][]ast.Node) {
	s.flow = make(map[string]*flowFileStore, len(s.filenames))

	for _, filename := range s.filenames {
		s.addFlowScope(filename, "file", nil, parsed[filename], FlowScopeKey{})
	}
}

func (s *SemanticSnapshot) addFlowScope(filename, kind string, owner ast.Node, statements []ast.Node, parent FlowScopeKey) {
	scope := flowScopeKey(filename, kind, owner, statements)
	if scope.File == "" {
		return
	}
	if parent.File != "" {
		s.flowFile(filename, true).putNesting(scope, parent)
	}
	graph := buildControlFlowGraph(scope, owner, statements)
	if !s.flowFile(filename, true).putGraph(scope, graph) {
		return
	}

	for i, statement := range statements {
		reachable := graph.blockReachable(FlowNodeID(i + 1))
		key := flowStatementKey(filename, statement)
		if key.File != "" {
			s.recordStatementReachability(key, reachable)
		}
		if !reachable {
			continue
		}
		s.addChildFlowScopes(filename, statement, scope)
	}
}

func (s *SemanticSnapshot) recordStatementReachability(key FlowStatementKey, reachable bool) {
	if file := s.flowFile(key.File, true); file != nil {
		file.recordStatement(key, reachable)
	}
}

func (s *SemanticSnapshot) addChildFlowScopes(filename string, node ast.Node, parent FlowScopeKey) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		s.addFlowScope(filename, "function", n, n.Body, parent)
	case *ast.ClassNode:
		for _, method := range n.Methods {
			s.addChildFlowScopes(filename, method, parent)
		}
	case *ast.BlockNode:
		s.addFlowScope(filename, "block", n, n.Statements, parent)
	case *ast.IfNode:
		s.addFlowScope(filename, "if", n, n.Body, parent)
		for _, elseif := range n.ElseIfs {
			s.addFlowScope(filename, "elseif", elseif, elseif.Body, parent)
		}
		if n.Else != nil {
			s.addFlowScope(filename, "else", n.Else, n.Else.Body, parent)
		}
	case *ast.WhileNode:
		s.addFlowScope(filename, "while", n, n.Body, parent)
	case *ast.ForNode:
		s.addFlowScope(filename, "for", n, n.Body, parent)
	case *ast.ForeachNode:
		s.addFlowScope(filename, "foreach", n, n.Body, parent)
	case *ast.DoWhileNode:
		s.addFlowScope(filename, "do", n, n.Body, parent)
	case *ast.SwitchNode:
		for _, caseNode := range n.Cases {
			s.addFlowScope(filename, "case", caseNode, caseNode.Body, parent)
		}
	case *ast.TryNode:
		s.addFlowScope(filename, "try", n, n.Body, parent)
		for _, catchNode := range n.Catches {
			s.addFlowScope(filename, "catch", catchNode, catchNode.Body, parent)
		}
		if len(n.Finally) > 0 {
			s.addFlowScope(filename, "finally", n, n.Finally, parent)
		}
	case *ast.NamespaceNode:
		s.addFlowScope(filename, "namespace", n, n.Body, parent)
	}
}

func (s *SemanticSnapshot) resolveScopeAtLevel(scope FlowScopeKey, level int) FlowScopeKey {
	for i := 0; i < level; i++ {
		file := s.flowFile(scope.File, false)
		if file == nil {
			return FlowScopeKey{}
		}
		parent, ok := file.parent(scope.File, scope)
		if !ok {
			return FlowScopeKey{}
		}
		scope = parent
	}
	return scope
}

func (s *SemanticSnapshot) StatementReachable(key FlowStatementKey) (bool, bool) {
	if s == nil || key.File == "" {
		return false, false
	}
	file := s.flowFile(key.File, false)
	if file == nil {
		return false, false
	}
	return file.statement(key)
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
	file := s.flowFile(key.File, false)
	if file == nil {
		return ControlFlowGraph{}, false
	}
	return file.graph(key)
}

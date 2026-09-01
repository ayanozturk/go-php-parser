package analyse

import (
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// VariableDefinedness is the joined state of a variable at one read.
type VariableDefinedness uint8

const (
	VariableUndefined VariableDefinedness = iota
	VariablePossiblyDefined
	VariableDefinitelyDefined
)

// VariableReadKey identifies one variable read at an exact source span.
// Name is included because compact() can reference more than one variable from
// distinct string arguments whose parser spans may otherwise be ambiguous.
type VariableReadKey struct {
	File        string
	StartOffset int
	EndOffset   int
	Name        string
}

// VariableReadFact is an immutable result of joined variable-flow analysis.
type VariableReadFact struct {
	Key     VariableReadKey
	Start   ast.Position
	End     ast.Position
	State   VariableDefinedness
	Compact bool
}

type variableReadFact struct {
	start   ast.Position
	end     ast.Position
	name    string
	state   VariableDefinedness
	compact bool
}

func (f variableReadFact) public(filename string) VariableReadFact {
	return VariableReadFact{
		Key: VariableReadKey{
			File:        filename,
			StartOffset: f.start.Offset,
			EndOffset:   f.end.Offset,
			Name:        f.name,
		},
		Start:   f.start,
		End:     f.end,
		State:   f.state,
		Compact: f.compact,
	}
}

// VariableFlowReader exposes the complete deterministic, immutable variable-read
// fact set. SemanticSnapshot materializes definitely-defined reads lazily because
// ordinary diagnostics only need undefined and possibly-defined reads.
type VariableFlowReader interface {
	VariableReadsForFile(filename string) []VariableReadFact
}

const maxVariableFlowIterations = 8

var predefinedVariables = []string{
	"GLOBALS", "_SERVER", "_GET", "_POST", "_FILES", "_COOKIE",
	"_SESSION", "_REQUEST", "_ENV", "argc", "argv",
}

var predefinedVariableSet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(predefinedVariables))
	for _, name := range predefinedVariables {
		result[name] = struct{}{}
	}
	return result
}()

type variableFlowState struct {
	owner   *variableFlowAnalyzer
	values  []VariableDefinedness
	dynamic *variableFlowDynamicState
	shared  bool
}

type variableFlowDynamicState struct {
	knownStrings       map[int]string
	knownExtractShapes map[int]variableFlowExtractShape
	definition         VariableDefinedness
}

type variableFlowExtractShape struct {
	keys     []string
	complete bool
}

type variableFlowResult struct {
	normal     *variableFlowState
	breaks     []variableFlowTransfer
	continues  []variableFlowTransfer
	terminates []*variableFlowState
}

type variableFlowTransfer struct {
	state *variableFlowState
	level int
}

type variableFlowResumeKind uint8

const (
	variableFlowResumeNormal variableFlowResumeKind = iota
	variableFlowResumeBreak
	variableFlowResumeContinue
	variableFlowResumeTerminate
)

type variableFlowAnalyzer struct {
	filename                 string
	includeDefinitelyDefined bool
	resolver                 SymbolResolver
	typeContext              FileTypeContext
	currentClassName         string
	readIndex                map[any]int
	reads                    []variableReadFact
	variableIDs              map[string]int
	variableNames            []string
	dynamicNameVariables     map[string]struct{}
	extractSourceVariables   map[string]struct{}
}

func buildVariableFlowFacts(filename string, nodes []ast.Node, includeDefinitelyDefined bool, resolver SymbolResolver) []variableReadFact {
	analyzer := &variableFlowAnalyzer{
		filename:                 filename,
		includeDefinitelyDefined: includeDefinitelyDefined,
		resolver:                 resolver,
		typeContext:              CollectFileTypeContext(nodes),
		readIndex:                make(map[any]int),
		variableIDs:              make(map[string]int),
	}
	analyzer.collectDynamicVariableSources(nodes)
	analyzer.statements(nodes, initialVariableFlowState(analyzer))
	facts := analyzer.reads
	sort.Slice(facts, func(i, j int) bool {
		left, right := facts[i], facts[j]
		if left.start.Offset != right.start.Offset {
			return left.start.Offset < right.start.Offset
		}
		if left.end.Offset != right.end.Offset {
			return left.end.Offset < right.end.Offset
		}
		return left.name < right.name
	})
	return facts
}

func (a *variableFlowAnalyzer) collectDynamicVariableSources(nodes []ast.Node) {
	walkAllWithoutTypeContext(nodes, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.VariableVariableNode:
			if variable, ok := n.Expr.(*ast.VariableNode); ok {
				if a.dynamicNameVariables == nil {
					a.dynamicNameVariables = make(map[string]struct{})
				}
				a.dynamicNameVariables[variable.Name] = struct{}{}
			}
		case *ast.FunctionCallNode:
			if !strings.EqualFold(functionCallName(n), "extract") || len(n.Args) == 0 {
				return
			}
			if variable, ok := argumentValue(n.Args[0]).(*ast.VariableNode); ok {
				if a.extractSourceVariables == nil {
					a.extractSourceVariables = make(map[string]struct{})
				}
				a.extractSourceVariables[variable.Name] = struct{}{}
			}
		}
	})
}

func initialVariableFlowState(analyzer *variableFlowAnalyzer) *variableFlowState {
	return &variableFlowState{owner: analyzer}
}

func functionVariableFlowState(analyzer *variableFlowAnalyzer, function *ast.FunctionNode, includeThis bool) *variableFlowState {
	state := initialVariableFlowState(analyzer)
	for _, parameter := range function.Params {
		if param, ok := parameter.(*ast.ParamNode); ok {
			state.set(param.Name, VariableDefinitelyDefined)
		}
	}
	if includeThis && !isStaticMethod(function) {
		state.set("this", VariableDefinitelyDefined)
	}
	return state
}

func cloneVariableFlowState(state *variableFlowState) *variableFlowState {
	if state == nil {
		return nil
	}
	state.shared = true
	return &variableFlowState{
		owner:   state.owner,
		values:  state.values,
		dynamic: state.dynamic,
		shared:  true,
	}
}

func joinedVariableFlowState(states ...*variableFlowState) *variableFlowState {
	var present []*variableFlowState
	for _, state := range states {
		if state != nil {
			present = append(present, state)
		}
	}
	if len(present) == 0 {
		return nil
	}
	if len(present) == 1 {
		return cloneVariableFlowState(present[0])
	}
	allEqual := true
	for _, state := range present[1:] {
		if !equalVariableFlowState(present[0], state) {
			allEqual = false
			break
		}
	}
	if allEqual {
		return cloneVariableFlowState(present[0])
	}
	owner := present[0].owner
	valueCount := 0
	for _, state := range present {
		if len(state.values) > valueCount {
			valueCount = len(state.values)
		}
	}
	joined := &variableFlowState{owner: owner, values: make([]VariableDefinedness, valueCount)}
	dynamicDefinition := present[0].dynamicDefinedness()
	for _, state := range present[1:] {
		dynamicDefinition = joinVariableDefinedness(dynamicDefinition, state.dynamicDefinedness())
	}
	lastDefined := -1
	for id := 0; id < valueCount; id++ {
		value := present[0].explicitDefinednessByID(id)
		for _, state := range present[1:] {
			value = joinVariableDefinedness(value, state.explicitDefinednessByID(id))
		}
		if value != VariableUndefined {
			joined.values[id] = value
			lastDefined = id
		}
	}
	joined.values = joined.values[:lastDefined+1]
	knownStrings := joinedVariableFlowKnownStrings(present)
	knownExtractShapes := joinedVariableFlowExtractShapes(present)
	if dynamicDefinition != VariableUndefined || len(knownStrings) > 0 || len(knownExtractShapes) > 0 {
		joined.dynamic = &variableFlowDynamicState{knownStrings: knownStrings, knownExtractShapes: knownExtractShapes, definition: dynamicDefinition}
	}
	return joined
}

func joinedVariableFlowKnownStrings(states []*variableFlowState) map[int]string {
	if len(states) == 0 || states[0].dynamic == nil || len(states[0].dynamic.knownStrings) == 0 {
		return nil
	}
	var joined map[int]string
	for id, value := range states[0].dynamic.knownStrings {
		matches := true
		for _, state := range states[1:] {
			if state.dynamic == nil || state.dynamic.knownStrings[id] != value {
				matches = false
				break
			}
		}
		if matches {
			if joined == nil {
				joined = make(map[int]string)
			}
			joined[id] = value
		}
	}
	return joined
}

func joinedVariableFlowExtractShapes(states []*variableFlowState) map[int]variableFlowExtractShape {
	if len(states) == 0 || states[0].dynamic == nil || len(states[0].dynamic.knownExtractShapes) == 0 {
		return nil
	}
	var joined map[int]variableFlowExtractShape
	for id, shape := range states[0].dynamic.knownExtractShapes {
		matches := true
		for _, state := range states[1:] {
			if state.dynamic == nil {
				matches = false
				break
			}
			other, ok := state.dynamic.knownExtractShapes[id]
			if !ok || shape.complete != other.complete || !equalStrings(shape.keys, other.keys) {
				matches = false
				break
			}
		}
		if matches {
			if joined == nil {
				joined = make(map[int]variableFlowExtractShape)
			}
			joined[id] = shape
		}
	}
	return joined
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (s *variableFlowState) definedness(name string) VariableDefinedness {
	if s == nil {
		return VariableUndefined
	}
	if id, ok := s.owner.variableIDs[name]; ok {
		return s.definednessByID(id)
	}
	if _, predefined := predefinedVariableSet[name]; predefined {
		return VariableDefinitelyDefined
	}
	return s.dynamicDefinedness()
}

func (s *variableFlowState) definednessByID(id int) VariableDefinedness {
	value := s.explicitDefinednessByID(id)
	if value != VariableUndefined {
		return value
	}
	return s.dynamicDefinedness()
}

func (s *variableFlowState) dynamicDefinedness() VariableDefinedness {
	if s == nil || s.dynamic == nil {
		return VariableUndefined
	}
	return s.dynamic.definition
}

func (s *variableFlowState) explicitDefinednessByID(id int) VariableDefinedness {
	if s == nil {
		return VariableUndefined
	}
	if id < len(s.values) && s.values[id] != VariableUndefined {
		return s.values[id]
	}
	if id < len(s.owner.variableNames) {
		if _, predefined := predefinedVariableSet[s.owner.variableNames[id]]; predefined {
			return VariableDefinitelyDefined
		}
	}
	return VariableUndefined
}

func (s *variableFlowState) set(name string, value VariableDefinedness) {
	id := s.owner.variableID(name)
	s.detach(id + 1)
	s.values[id] = value
	if s.dynamic != nil {
		delete(s.dynamic.knownStrings, id)
		delete(s.dynamic.knownExtractShapes, id)
	}
}

func (s *variableFlowState) setKnownString(name, value string) {
	if s == nil || value == "" {
		return
	}
	id := s.owner.variableID(name)
	s.detach(len(s.values))
	s.ensureDynamic()
	if s.dynamic.knownStrings == nil {
		s.dynamic.knownStrings = make(map[int]string)
	}
	s.dynamic.knownStrings[id] = value
}

func (s *variableFlowState) knownString(node ast.Node) (string, bool) {
	if value, ok := stringLiteralValue(node); ok && value != "" {
		return value, true
	}
	variable, ok := node.(*ast.VariableNode)
	if !ok {
		return "", false
	}
	id, ok := s.owner.variableIDs[variable.Name]
	if !ok {
		return "", false
	}
	if s.dynamic == nil {
		return "", false
	}
	value, ok := s.dynamic.knownStrings[id]
	return value, ok && value != ""
}

func (s *variableFlowState) setKnownExtractShape(name string, shape variableFlowExtractShape) {
	if s == nil {
		return
	}
	id := s.owner.variableID(name)
	s.detach(len(s.values))
	s.ensureDynamic()
	if s.dynamic.knownExtractShapes == nil {
		s.dynamic.knownExtractShapes = make(map[int]variableFlowExtractShape)
	}
	s.dynamic.knownExtractShapes[id] = shape
}

func (s *variableFlowState) unset(name string) {
	if s == nil || s.definedness(name) == VariableUndefined {
		return
	}
	id, ok := s.owner.variableIDs[name]
	if !ok || id >= len(s.values) {
		return
	}
	s.detach(len(s.values))
	s.values[id] = VariableUndefined
	if s.dynamic != nil {
		delete(s.dynamic.knownStrings, id)
		delete(s.dynamic.knownExtractShapes, id)
	}
}

func (s *variableFlowState) detach(capacity int) {
	if s == nil {
		return
	}
	if s.shared {
		values := make([]VariableDefinedness, len(s.values), max(capacity, len(s.values)))
		copy(values, s.values)
		s.values = values
		if s.dynamic != nil {
			s.dynamic = &variableFlowDynamicState{
				knownStrings:       cloneIntStringMap(s.dynamic.knownStrings),
				knownExtractShapes: cloneExtractShapeMap(s.dynamic.knownExtractShapes),
				definition:         s.dynamic.definition,
			}
		}
		s.shared = false
	}
	if len(s.values) < capacity {
		s.values = append(s.values, make([]VariableDefinedness, capacity-len(s.values))...)
	}
}

func (s *variableFlowState) ensureDynamic() {
	if s.dynamic == nil {
		s.dynamic = &variableFlowDynamicState{}
	}
}

func cloneIntStringMap(source map[int]string) map[int]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[int]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneExtractShapeMap(source map[int]variableFlowExtractShape) map[int]variableFlowExtractShape {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[int]variableFlowExtractShape, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func (a *variableFlowAnalyzer) variableID(name string) int {
	if id, ok := a.variableIDs[name]; ok {
		return id
	}
	id := len(a.variableNames)
	a.variableIDs[name] = id
	a.variableNames = append(a.variableNames, name)
	return id
}

func joinVariableDefinedness(left, right VariableDefinedness) VariableDefinedness {
	if left == right {
		return left
	}
	return VariablePossiblyDefined
}

func equalVariableFlowState(left, right *variableFlowState) bool {
	if left == nil || right == nil {
		return left == right
	}
	valueCount := max(len(left.values), len(right.values))
	for id := 0; id < valueCount; id++ {
		if left.definednessByID(id) != right.definednessByID(id) {
			return false
		}
	}
	if left.dynamicDefinedness() != right.dynamicDefinedness() || len(left.knownStringValues()) != len(right.knownStringValues()) || len(left.knownExtractShapeValues()) != len(right.knownExtractShapeValues()) {
		return false
	}
	for id, value := range left.knownStringValues() {
		if right.knownStringValues()[id] != value {
			return false
		}
	}
	for id, shape := range left.knownExtractShapeValues() {
		other, ok := right.knownExtractShapeValues()[id]
		if !ok || shape.complete != other.complete || !equalStrings(shape.keys, other.keys) {
			return false
		}
	}
	return true
}

func (s *variableFlowState) knownStringValues() map[int]string {
	if s == nil || s.dynamic == nil {
		return nil
	}
	return s.dynamic.knownStrings
}

func (s *variableFlowState) knownExtractShapeValues() map[int]variableFlowExtractShape {
	if s == nil || s.dynamic == nil {
		return nil
	}
	return s.dynamic.knownExtractShapes
}

func (a *variableFlowAnalyzer) statements(statements []ast.Node, input *variableFlowState) variableFlowResult {
	state := cloneVariableFlowState(input)
	result := variableFlowResult{}
	for _, statement := range statements {
		if state == nil {
			break
		}
		step := a.statement(statement, state)
		result.breaks = append(result.breaks, step.breaks...)
		result.continues = append(result.continues, step.continues...)
		result.terminates = append(result.terminates, step.terminates...)
		state = step.normal
	}
	result.normal = state
	return result
}

func (a *variableFlowAnalyzer) statement(node ast.Node, state *variableFlowState) variableFlowResult {
	if node == nil {
		return variableFlowResult{normal: state}
	}
	switch n := node.(type) {
	case *ast.NamespaceNode:
		return a.statements(n.Body, state)
	case *ast.BlockNode:
		return a.statements(n.Statements, state)
	case *ast.FunctionNode:
		a.statements(n.Body, functionVariableFlowState(a, n, false))
		return variableFlowResult{normal: state}
	case *ast.ClassNode:
		a.propertyHooks(n.Properties)
		previousClassName := a.currentClassName
		a.currentClassName = a.typeContext.resolveClassLike(n.Name)
		for _, method := range n.Methods {
			if function, ok := method.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(a, function, true))
			}
		}
		a.currentClassName = previousClassName
		return variableFlowResult{normal: state}
	case *ast.TraitNode:
		a.propertyHooks(n.Body)
		previousClassName := a.currentClassName
		if n.Name != nil {
			a.currentClassName = a.typeContext.resolveClassLike(n.Name.Name)
		} else {
			a.currentClassName = ""
		}
		for _, member := range n.Body {
			if function, ok := member.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(a, function, true))
			}
		}
		a.currentClassName = previousClassName
		return variableFlowResult{normal: state}
	case *ast.EnumNode:
		previousClassName := a.currentClassName
		a.currentClassName = a.typeContext.resolveClassLike(n.Name)
		for _, member := range n.Methods {
			if function, ok := member.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(a, function, true))
			}
		}
		a.currentClassName = previousClassName
		return variableFlowResult{normal: state}
	case *ast.StaticVarDeclNode:
		for _, entry := range n.Vars {
			a.expression(entry.Init, state, false)
			state.set(entry.Name, VariableDefinitelyDefined)
		}
		return variableFlowResult{normal: state}
	case *ast.GlobalVarDeclNode:
		for _, entry := range n.Vars {
			state.set(entry.Name, VariableDefinitelyDefined)
		}
		return variableFlowResult{normal: state}
	case *ast.AssignmentNode:
		a.assignment(n, state)
		return variableFlowResult{normal: state}
	case *ast.ExpressionStmt:
		a.expression(n.Expr, state, false)
		return variableFlowResult{normal: state}
	case *ast.ReturnNode:
		a.expression(n.Expr, state, false)
		return variableFlowResult{terminates: []*variableFlowState{cloneVariableFlowState(state)}}
	case *ast.ThrowNode:
		a.expression(n.Expr, state, false)
		return variableFlowResult{terminates: []*variableFlowState{cloneVariableFlowState(state)}}
	case *ast.BreakNode:
		a.expression(n.Level, state, false)
		return variableFlowResult{breaks: []variableFlowTransfer{{state: cloneVariableFlowState(state), level: variableFlowTransferLevel(n.Level)}}}
	case *ast.ContinueNode:
		a.expression(n.Level, state, false)
		return variableFlowResult{continues: []variableFlowTransfer{{state: cloneVariableFlowState(state), level: variableFlowTransferLevel(n.Level)}}}
	case *ast.IfNode:
		return a.ifStatement(n, state)
	case *ast.WhileNode:
		return a.whileStatement(n, state)
	case *ast.ForNode:
		return a.forStatement(n, state)
	case *ast.ForeachNode:
		return a.foreachStatement(n, state)
	case *ast.DoWhileNode:
		return a.doWhileStatement(n, state)
	case *ast.TryNode:
		return a.tryStatement(n, state)
	case *ast.SwitchNode:
		return a.switchStatement(n, state)
	case *ast.DeclareNode:
		for _, value := range n.Directives {
			a.expression(value, state, false)
		}
		if n.Body != nil {
			return a.statement(n.Body, state)
		}
		return variableFlowResult{normal: state}
	default:
		a.expression(node, state, false)
		return variableFlowResult{normal: state}
	}
}

func (a *variableFlowAnalyzer) propertyHooks(members []ast.Node) {
	for _, member := range members {
		property, ok := member.(*ast.PropertyNode)
		if !ok {
			continue
		}
		for _, hook := range property.Hooks {
			state := initialVariableFlowState(a)
			if !property.IsStatic {
				state.set("this", VariableDefinitelyDefined)
			}
			if parameter := propertyHookVariableName(hook.Parameter); parameter != "" {
				state.set(parameter, VariableDefinitelyDefined)
			}
			a.expression(hook.Expr, state, false)
			a.statements(hook.Body, state)
		}
	}
}

func propertyHookVariableName(header string) string {
	header = strings.TrimSpace(header)
	index := strings.LastIndex(header, "$")
	if index < 0 || index+1 >= len(header) {
		return ""
	}
	name := header[index+1:]
	for i, r := range name {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return name[:i]
		}
	}
	return name
}

func (a *variableFlowAnalyzer) ifStatement(node *ast.IfNode, input *variableFlowState) variableFlowResult {
	remaining := cloneVariableFlowState(input)
	a.expression(node.Condition, remaining, false)
	branches := []variableFlowResult{a.statements(node.Body, remaining)}
	for _, elseif := range node.ElseIfs {
		a.expression(elseif.Condition, remaining, false)
		branches = append(branches, a.statements(elseif.Body, remaining))
	}
	if node.Else != nil {
		branches = append(branches, a.statements(node.Else.Body, remaining))
	} else {
		branches = append(branches, variableFlowResult{normal: cloneVariableFlowState(remaining)})
	}
	return joinVariableFlowResults(branches...)
}

func (a *variableFlowAnalyzer) whileStatement(node *ast.WhileNode, input *variableFlowState) variableFlowResult {
	header := cloneVariableFlowState(input)
	var body variableFlowResult
	var conditionState *variableFlowState
	mayIterate, mayExit := loopConditionPaths(node)
	for iteration := 0; iteration < maxVariableFlowIterations; iteration++ {
		conditionState = cloneVariableFlowState(header)
		a.expression(node.Condition, conditionState, false)
		if !mayIterate {
			break
		}
		body = a.statements(node.Body, conditionState)
		localContinues, _ := consumeVariableFlowTransfers(body.continues)
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, localContinues...)...)
		next := joinedVariableFlowState(input, back)
		if equalVariableFlowState(next, header) {
			header = next
			break
		}
		header = next
	}
	localBreaks, outerBreaks := consumeVariableFlowTransfers(body.breaks)
	_, outerContinues := consumeVariableFlowTransfers(body.continues)
	exits := append([]*variableFlowState(nil), localBreaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), breaks: outerBreaks, continues: outerContinues, terminates: body.terminates}
}

func (a *variableFlowAnalyzer) forStatement(node *ast.ForNode, input *variableFlowState) variableFlowResult {
	entry := cloneVariableFlowState(input)
	for _, expression := range node.Init {
		a.expression(expression, entry, false)
	}
	header := cloneVariableFlowState(entry)
	var body variableFlowResult
	var conditionState *variableFlowState
	mayIterate, mayExit := loopConditionPaths(node)
	for iteration := 0; iteration < maxVariableFlowIterations; iteration++ {
		conditionState = cloneVariableFlowState(header)
		for _, condition := range node.Conditions {
			a.expression(condition, conditionState, false)
		}
		if !mayIterate {
			break
		}
		body = a.statements(node.Body, conditionState)
		localContinues, _ := consumeVariableFlowTransfers(body.continues)
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, localContinues...)...)
		if back != nil {
			for _, update := range node.Updates {
				a.expression(update, back, false)
			}
		}
		next := joinedVariableFlowState(entry, back)
		if equalVariableFlowState(next, header) {
			header = next
			break
		}
		header = next
	}
	localBreaks, outerBreaks := consumeVariableFlowTransfers(body.breaks)
	_, outerContinues := consumeVariableFlowTransfers(body.continues)
	exits := append([]*variableFlowState(nil), localBreaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), breaks: outerBreaks, continues: outerContinues, terminates: body.terminates}
}

func (a *variableFlowAnalyzer) foreachStatement(node *ast.ForeachNode, input *variableFlowState) variableFlowResult {
	entry := cloneVariableFlowState(input)
	a.expression(node.Expr, entry, false)
	header := cloneVariableFlowState(entry)
	var body variableFlowResult
	for iteration := 0; iteration < maxVariableFlowIterations; iteration++ {
		bodyInput := cloneVariableFlowState(header)
		defineVariableFlowTarget(node.KeyVar, bodyInput)
		defineVariableFlowTarget(node.ValueVar, bodyInput)
		body = a.statements(node.Body, bodyInput)
		localContinues, _ := consumeVariableFlowTransfers(body.continues)
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, localContinues...)...)
		next := joinedVariableFlowState(entry, back)
		if equalVariableFlowState(next, header) {
			header = next
			break
		}
		header = next
	}
	localBreaks, outerBreaks := consumeVariableFlowTransfers(body.breaks)
	_, outerContinues := consumeVariableFlowTransfers(body.continues)
	exits := append([]*variableFlowState{entry, header}, localBreaks...)
	return variableFlowResult{normal: joinedVariableFlowState(exits...), breaks: outerBreaks, continues: outerContinues, terminates: body.terminates}
}

func (a *variableFlowAnalyzer) doWhileStatement(node *ast.DoWhileNode, input *variableFlowState) variableFlowResult {
	bodyInput := cloneVariableFlowState(input)
	var body variableFlowResult
	var conditionState *variableFlowState
	mayIterate, mayExit := loopConditionPaths(node)
	for iteration := 0; iteration < maxVariableFlowIterations; iteration++ {
		body = a.statements(node.Body, bodyInput)
		localContinues, _ := consumeVariableFlowTransfers(body.continues)
		conditionState = joinedVariableFlowState(append([]*variableFlowState{body.normal}, localContinues...)...)
		if conditionState != nil {
			a.expression(node.Condition, conditionState, false)
		}
		if !mayIterate {
			break
		}
		next := joinedVariableFlowState(input, conditionState)
		if equalVariableFlowState(next, bodyInput) {
			break
		}
		bodyInput = next
	}
	localBreaks, outerBreaks := consumeVariableFlowTransfers(body.breaks)
	_, outerContinues := consumeVariableFlowTransfers(body.continues)
	exits := append([]*variableFlowState(nil), localBreaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), breaks: outerBreaks, continues: outerContinues, terminates: body.terminates}
}

func (a *variableFlowAnalyzer) tryStatement(node *ast.TryNode, input *variableFlowState) variableFlowResult {
	tryResult := a.statements(node.Body, input)
	branches := []variableFlowResult{tryResult}
	for _, catch := range node.Catches {
		catchInput := cloneVariableFlowState(input)
		if catch.Variable != "" {
			catchInput.set(strings.TrimPrefix(catch.Variable, "$"), VariableDefinitelyDefined)
		}
		branches = append(branches, a.statements(catch.Body, catchInput))
	}
	joined := joinVariableFlowResults(branches...)
	if len(node.Finally) == 0 {
		return joined
	}
	return a.applyFinally(node.Finally, joined)
}

func (a *variableFlowAnalyzer) applyFinally(statements []ast.Node, input variableFlowResult) variableFlowResult {
	output := variableFlowResult{}
	resume := func(state *variableFlowState, kind variableFlowResumeKind, level int) {
		if state == nil {
			return
		}
		result := a.statements(statements, state)
		output.breaks = append(output.breaks, result.breaks...)
		output.continues = append(output.continues, result.continues...)
		output.terminates = append(output.terminates, result.terminates...)
		if result.normal == nil {
			return
		}
		switch kind {
		case variableFlowResumeNormal:
			output.normal = joinedVariableFlowState(output.normal, result.normal)
		case variableFlowResumeBreak:
			output.breaks = append(output.breaks, variableFlowTransfer{state: result.normal, level: level})
		case variableFlowResumeContinue:
			output.continues = append(output.continues, variableFlowTransfer{state: result.normal, level: level})
		case variableFlowResumeTerminate:
			output.terminates = append(output.terminates, result.normal)
		}
	}

	resume(input.normal, variableFlowResumeNormal, 0)
	for _, transfer := range input.breaks {
		resume(transfer.state, variableFlowResumeBreak, transfer.level)
	}
	for _, transfer := range input.continues {
		resume(transfer.state, variableFlowResumeContinue, transfer.level)
	}
	for _, state := range input.terminates {
		resume(state, variableFlowResumeTerminate, 0)
	}
	return output
}

func (a *variableFlowAnalyzer) switchStatement(node *ast.SwitchNode, input *variableFlowState) variableFlowResult {
	entry := cloneVariableFlowState(input)
	a.expression(node.Expr, entry, false)
	var branches []variableFlowResult
	var fallthroughState *variableFlowState
	hasDefault := false
	for _, switchCase := range node.Cases {
		caseInput := cloneVariableFlowState(entry)
		if fallthroughState != nil {
			caseInput = joinedVariableFlowState(caseInput, fallthroughState)
		}
		if switchCase.IsDefault {
			hasDefault = true
		} else {
			a.expression(switchCase.Expr, caseInput, false)
		}
		result := a.statements(switchCase.Body, caseInput)
		fallthroughState = result.normal
		branches = append(branches, result)
	}
	joined := joinVariableFlowResults(branches...)
	localBreaks, outerBreaks := consumeVariableFlowTransfers(joined.breaks)
	localContinues, outerContinues := consumeVariableFlowTransfers(joined.continues)
	joined.normal = joinedVariableFlowState(joined.normal, joinedVariableFlowState(localBreaks...), joinedVariableFlowState(localContinues...))
	joined.breaks = outerBreaks
	joined.continues = outerContinues
	if !hasDefault {
		joined.normal = joinedVariableFlowState(joined.normal, entry)
	}
	return joined
}

func variableFlowTransferLevel(level ast.Node) int {
	if level == nil {
		return 1
	}
	var value int64
	switch literal := level.(type) {
	case *ast.IntegerLiteral:
		value = literal.Value
	case *ast.IntegerNode:
		value = literal.Value
	default:
		return 1
	}
	if value <= 1 {
		return 1
	}
	maxInt := int(^uint(0) >> 1)
	if uint64(value) > uint64(maxInt) {
		return maxInt
	}
	return int(value)
}

func consumeVariableFlowTransfers(transfers []variableFlowTransfer) ([]*variableFlowState, []variableFlowTransfer) {
	var local []*variableFlowState
	var outer []variableFlowTransfer
	for _, transfer := range transfers {
		if transfer.state == nil {
			continue
		}
		if transfer.level <= 1 {
			local = append(local, transfer.state)
			continue
		}
		transfer.level--
		outer = append(outer, transfer)
	}
	return local, outer
}

func joinVariableFlowResults(results ...variableFlowResult) variableFlowResult {
	joined := variableFlowResult{}
	var normals []*variableFlowState
	for _, result := range results {
		normals = append(normals, result.normal)
		joined.breaks = append(joined.breaks, result.breaks...)
		joined.continues = append(joined.continues, result.continues...)
		joined.terminates = append(joined.terminates, result.terminates...)
	}
	joined.normal = joinedVariableFlowState(normals...)
	return joined
}

func (a *variableFlowAnalyzer) assignment(node *ast.AssignmentNode, state *variableFlowState) {
	if node.Operator == "=" {
		if reference, ok := node.Right.(*ast.UnaryExpr); ok && reference.Operator == "&" {
			a.assignmentTargetExpressions(reference.Operand, state)
			defineVariableFlowTarget(node.Left, state)
			return
		}
	}
	if node.Operator != "" && node.Operator != "=" {
		a.expression(node.Left, state, false)
	} else {
		a.assignmentTargetExpressions(node.Left, state)
	}
	// PHP evaluates the right-hand side before the assignment becomes visible.
	a.expression(node.Right, state, false)
	defineVariableFlowTarget(node.Left, state)
	if node.Operator != "" && node.Operator != "=" {
		return
	}
	variable, ok := node.Left.(*ast.VariableNode)
	if !ok {
		return
	}
	if _, tracked := a.dynamicNameVariables[variable.Name]; tracked {
		if value, ok := stringLiteralValue(node.Right); ok {
			state.setKnownString(variable.Name, value)
			return
		}
	}
	if _, tracked := a.extractSourceVariables[variable.Name]; !tracked {
		return
	}
	if shape, ok := variableFlowExtractShapeForNode(node.Right, state); ok {
		state.setKnownExtractShape(variable.Name, shape)
	}
}

func (a *variableFlowAnalyzer) assignmentTargetExpressions(node ast.Node, state *variableFlowState) {
	switch n := node.(type) {
	case *ast.ArrayAccessNode:
		a.assignmentTargetExpressions(n.Var, state)
		a.expression(n.Index, state, false)
	case *ast.PropertyFetchNode:
		a.expression(n.Object, state, false)
	case *ast.VariableVariableNode:
		a.expression(n.Expr, state, false)
	}
}

func defineVariableFlowTarget(node ast.Node, state *variableFlowState) {
	switch n := node.(type) {
	case *ast.VariableNode:
		state.set(n.Name, VariableDefinitelyDefined)
	case *ast.ArrayAccessNode:
		defineVariableFlowTarget(n.Var, state)
	case *ast.UnaryExpr:
		defineVariableFlowTarget(n.Operand, state)
	case *ast.ArrayNode:
		for _, element := range n.Elements {
			defineVariableFlowTarget(element, state)
		}
	case *ast.ArrayItemNode:
		defineVariableFlowTarget(n.Value, state)
	case *ast.KeyValueNode:
		defineVariableFlowTarget(n.Value, state)
	}
}

func isVariableFlowTarget(node ast.Node) bool {
	switch node.(type) {
	case *ast.VariableNode, *ast.VariableVariableNode, *ast.ArrayAccessNode, *ast.PropertyFetchNode, *ast.ArrayNode, *ast.ArrayItemNode, *ast.KeyValueNode, *ast.UnaryExpr:
		return true
	default:
		return false
	}
}

func (a *variableFlowAnalyzer) functionCallParams(call *ast.FunctionCallNode) []ResolvedParam {
	if a == nil || a.resolver == nil || call == nil {
		return nil
	}
	name := functionCallName(call)
	if name == "" {
		return nil
	}
	if className, methodName, ok := strings.Cut(name, "::"); ok {
		className = a.resolveCallClassName(className)
		if className == "" {
			return nil
		}
		params, found := resolveMethodReferenceParams(a.resolver, className, methodName)
		if !found {
			return nil
		}
		return params
	}
	resolvedName := resolveFunctionNameForCall(name, a.typeContext, &AnalysisContext{Resolver: a.resolver})
	function, ok := resolveFunctionView(a.resolver, resolvedName)
	if !ok {
		return nil
	}
	return function.Params
}

func (a *variableFlowAnalyzer) resolveCallClassName(name string) string {
	switch asciiLowerIdent(strings.TrimPrefix(name, `\`)) {
	case "self", "static":
		return a.currentClassName
	case "parent":
		if a.currentClassName == "" || a.resolver == nil {
			return ""
		}
		if class, ok := a.resolver.ResolveClass(a.currentClassName); ok && len(class.Extends) > 0 {
			return class.Extends[0]
		}
		return ""
	default:
		return a.typeContext.resolveClassLike(name)
	}
}

func (a *variableFlowAnalyzer) methodCallParams(call *ast.MethodCallNode) []ResolvedParam {
	if a == nil || a.resolver == nil || call == nil || call.Method == "" {
		return nil
	}
	className := ""
	if variable, ok := call.Object.(*ast.VariableNode); ok && variable.Name == "this" {
		className = a.currentClassName
	} else {
		className = methodCallClassName(call.Object, a.typeContext)
	}
	if className == "" {
		return nil
	}
	params, ok := resolveMethodReferenceParams(a.resolver, className, call.Method)
	if !ok {
		return nil
	}
	return params
}

func (a *variableFlowAnalyzer) constructorParams(call *ast.NewNode) []ResolvedParam {
	if a == nil || a.resolver == nil || call == nil {
		return nil
	}
	className := ""
	if call.ClassName != "" {
		className = a.resolveCallClassName(call.ClassName)
	} else if identifier, ok := call.ClassExpr.(*ast.IdentifierNode); ok {
		className = a.resolveCallClassName(identifier.Value)
	}
	if className == "" {
		return nil
	}
	params, _ := resolveMethodReferenceParams(a.resolver, className, "__construct")
	return params
}

func (a *variableFlowAnalyzer) callArguments(arguments []ast.Node, params []ResolvedParam, state *variableFlowState, suppressed bool) {
	for index, argument := range arguments {
		value := argumentValue(argument)
		if param, ok := resolvedArgumentParam(params, argument, index); ok && param.IsByRef && isVariableFlowTarget(value) {
			if !param.IsOut {
				a.expression(value, state, suppressed)
			}
			a.assignmentTargetExpressions(value, state)
			defineVariableFlowTarget(value, state)
			continue
		}
		a.expression(value, state, suppressed)
	}
}

func resolvedArgumentParam(params []ResolvedParam, argument ast.Node, position int) (ResolvedParam, bool) {
	if named, ok := argument.(*ast.NamedArgumentNode); ok {
		for _, param := range params {
			if param.Name == named.Name {
				return param, true
			}
		}
		return ResolvedParam{}, false
	}
	if position < len(params) {
		return params[position], true
	}
	if len(params) > 0 && params[len(params)-1].IsVariadic {
		return params[len(params)-1], true
	}
	return ResolvedParam{}, false
}

func (a *variableFlowAnalyzer) expression(node ast.Node, state *variableFlowState, suppressed bool) {
	if node == nil || state == nil {
		return
	}
	switch n := node.(type) {
	case *ast.VariableNode:
		if !suppressed {
			a.recordRead(n.Name, n, state.definedness(n.Name), false)
		}
	case *ast.VariableVariableNode:
		a.expression(n.Expr, state, suppressed)
		if name, ok := state.knownString(n.Expr); ok {
			a.recordRead(name, n, state.definedness(name), false)
		}
	case *ast.AssignmentNode:
		a.assignment(n, state)
	case *ast.FunctionCallNode:
		name := asciiLowerIdent(functionCallName(n))
		if name == "isset" || name == "empty" {
			return
		}
		if name == "compact" {
			for _, argument := range n.Args {
				arg := argumentValue(argument)
				if variableName, ok := stringLiteralValue(arg); ok {
					a.recordRead(variableName, arg, state.definedness(variableName), true)
				} else {
					a.expression(arg, state, false)
				}
			}
			return
		}
		a.expression(n.Name, state, suppressed)
		a.callArguments(n.Args, a.functionCallParams(n), state, suppressed)
		if name == "extract" {
			a.extractVariables(n.Args, state)
		}
	case *ast.MethodCallNode:
		a.expression(n.Object, state, suppressed)
		a.callArguments(n.Args, a.methodCallParams(n), state, suppressed)
	case *ast.NewNode:
		a.expression(n.ClassExpr, state, suppressed)
		a.callArguments(n.Args, a.constructorParams(n), state, suppressed)
	case *ast.PropertyFetchNode:
		a.expression(n.Object, state, suppressed)
	case *ast.ClassConstFetchNode:
		a.expression(n.ConstExpr, state, suppressed)
	case *ast.ArrayAccessNode:
		a.expression(n.Var, state, suppressed)
		a.expression(n.Index, state, suppressed)
	case *ast.BinaryExpr:
		a.expression(n.Left, state, suppressed)
		if n.Operator == "&&" || n.Operator == "||" || n.Operator == "and" || n.Operator == "or" || n.Operator == "??" {
			skipped := cloneVariableFlowState(state)
			evaluated := cloneVariableFlowState(state)
			a.expression(n.Right, evaluated, suppressed)
			replaceVariableFlowState(state, joinedVariableFlowState(skipped, evaluated))
		} else {
			a.expression(n.Right, state, suppressed)
		}
	case *ast.UnaryExpr:
		a.expression(n.Operand, state, suppressed)
		if n.Operator == "++" || n.Operator == "--" {
			defineVariableFlowTarget(n.Operand, state)
		}
	case *ast.TypeCastNode:
		a.expression(n.Expr, state, suppressed)
	case *ast.TernaryExpr:
		a.expression(n.Condition, state, suppressed)
		ifState := cloneVariableFlowState(state)
		elseState := cloneVariableFlowState(state)
		a.expression(n.IfTrue, ifState, suppressed)
		a.expression(n.IfFalse, elseState, suppressed)
		replaceVariableFlowState(state, joinedVariableFlowState(ifState, elseState))
	case *ast.ArrayNode:
		for _, element := range n.Elements {
			a.expression(element, state, suppressed)
		}
	case *ast.ArrayItemNode:
		a.expression(n.Key, state, suppressed)
		a.expression(n.Value, state, suppressed)
	case *ast.KeyValueNode:
		a.expression(n.Key, state, suppressed)
		a.expression(n.Value, state, suppressed)
	case *ast.NamedArgumentNode:
		a.expression(n.Value, state, suppressed)
	case *ast.UnpackedArgumentNode:
		a.expression(n.Expr, state, suppressed)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			a.expression(part, state, suppressed)
		}
	case *ast.HeredocNode:
		for _, part := range n.Parts {
			a.expression(part, state, suppressed)
		}
	case *ast.YieldNode:
		a.expression(n.Key, state, suppressed)
		a.expression(n.Value, state, suppressed)
	case *ast.MatchNode:
		a.expression(n.Condition, state, suppressed)
		var arms []*variableFlowState
		for _, arm := range n.Arms {
			armState := cloneVariableFlowState(state)
			for _, condition := range arm.Conditions {
				a.expression(condition, armState, suppressed)
			}
			a.expression(arm.Body, armState, suppressed)
			arms = append(arms, armState)
		}
		if len(arms) > 0 {
			replaceVariableFlowState(state, joinedVariableFlowState(arms...))
		}
	case *ast.ArrowFunctionNode:
		closureState := cloneVariableFlowState(state)
		for _, parameter := range n.Params {
			if param, ok := parameter.(*ast.ParamNode); ok {
				closureState.set(param.Name, VariableDefinitelyDefined)
			}
		}
		a.expression(n.Expr, closureState, suppressed)
	case *ast.FunctionNode:
		if n.Name == "" {
			closureState := initialVariableFlowState(a)
			if !isStaticMethod(n) && state.definedness("this") != VariableUndefined {
				closureState.set("this", VariableDefinitelyDefined)
			}
			for index := range n.Uses {
				capture := &n.Uses[index]
				if capture.ByRef {
					state.set(capture.Name, VariableDefinitelyDefined)
				} else {
					a.recordReadAt(capture, capture.Name, capture.Pos, capture.EndPos, state.definedness(capture.Name), false)
				}
				closureState.set(capture.Name, VariableDefinitelyDefined)
			}
			for _, parameter := range n.Params {
				if param, ok := parameter.(*ast.ParamNode); ok {
					closureState.set(param.Name, VariableDefinitelyDefined)
				}
			}
			a.statements(n.Body, closureState)
		}
	case *ast.FirstClassCallableNode:
		a.expression(n.Target, state, suppressed)
	case *ast.AttributeNode:
		for _, argument := range n.Arguments {
			a.expression(argument, state, suppressed)
		}
	}
}

func (a *variableFlowAnalyzer) extractVariables(arguments []ast.Node, state *variableFlowState) {
	if len(arguments) == 0 || state == nil {
		return
	}
	shape, known := variableFlowExtractShapeForNode(argumentValue(arguments[0]), state)
	if !known {
		state.markPossibleDynamicDefinition()
		return
	}
	for _, name := range shape.keys {
		state.set(name, VariableDefinitelyDefined)
	}
	if !shape.complete {
		state.markPossibleDynamicDefinition()
	}
}

func (s *variableFlowState) markPossibleDynamicDefinition() {
	if s == nil {
		return
	}
	s.detach(len(s.values))
	s.ensureDynamic()
	s.dynamic.definition = joinVariableDefinedness(s.dynamic.definition, VariablePossiblyDefined)
}

func variableFlowExtractShapeForNode(node ast.Node, state *variableFlowState) (variableFlowExtractShape, bool) {
	if array, ok := node.(*ast.ArrayNode); ok {
		shape := variableFlowExtractShape{complete: true}
		for _, element := range array.Elements {
			item, ok := element.(*ast.ArrayItemNode)
			if !ok {
				shape.complete = false
				continue
			}
			if item.Unpack {
				shape.complete = false
				continue
			}
			if item.Key == nil {
				continue
			}
			name, ok := stringLiteralValue(item.Key)
			if !ok {
				shape.complete = false
				continue
			}
			if validExtractVariableName(name) {
				shape.keys = append(shape.keys, name)
			}
		}
		return shape, true
	}
	variable, ok := node.(*ast.VariableNode)
	if !ok || state == nil {
		return variableFlowExtractShape{}, false
	}
	id, ok := state.owner.variableIDs[variable.Name]
	if !ok {
		return variableFlowExtractShape{}, false
	}
	shape, ok := state.knownExtractShapeValues()[id]
	return shape, ok
}

func validExtractVariableName(name string) bool {
	if name == "" {
		return false
	}
	for index, r := range name {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || index > 0 && r >= '0' && r <= '9' || r >= 0x80 {
			continue
		}
		return false
	}
	return true
}

func replaceVariableFlowState(destination, source *variableFlowState) {
	if destination == nil || source == nil {
		return
	}
	source.shared = true
	destination.values = source.values
	destination.dynamic = source.dynamic
	destination.shared = true
}

func (a *variableFlowAnalyzer) recordRead(name string, node ast.Node, state VariableDefinedness, compact bool) {
	if node == nil || name == "" {
		return
	}
	a.recordReadAt(node, name, node.GetPos(), node.GetEndPos(), state, compact)
}

func (a *variableFlowAnalyzer) recordReadAt(identity any, name string, start, end ast.Position, state VariableDefinedness, compact bool) {
	if a.filename == "" || start.Offset < 0 || end.Offset <= start.Offset {
		return
	}
	fact := variableReadFact{start: start, end: end, name: name, state: state, compact: compact}
	if index, ok := a.readIndex[identity]; ok {
		a.reads[index].state = joinVariableDefinedness(a.reads[index].state, state)
		return
	}
	if state == VariableDefinitelyDefined && !a.includeDefinitelyDefined {
		return
	}
	a.readIndex[identity] = len(a.reads)
	a.reads = append(a.reads, fact)
}

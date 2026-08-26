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
	owner  *variableFlowAnalyzer
	values []VariableDefinedness
	shared bool
}

type variableFlowResult struct {
	normal     *variableFlowState
	breaks     []*variableFlowState
	continues  []*variableFlowState
	terminates []*variableFlowState
}

type variableFlowAnalyzer struct {
	filename                 string
	includeDefinitelyDefined bool
	resolver                 SymbolResolver
	typeContext              fileTypeContext
	currentClassName         string
	readIndex                map[any]int
	reads                    []variableReadFact
	variableIDs              map[string]int
	variableNames            []string
}

func buildVariableFlowFacts(filename string, nodes []ast.Node, includeDefinitelyDefined bool, resolver SymbolResolver) []variableReadFact {
	analyzer := &variableFlowAnalyzer{
		filename:                 filename,
		includeDefinitelyDefined: includeDefinitelyDefined,
		resolver:                 resolver,
		typeContext:              collectFileTypeContext(nodes),
		readIndex:                make(map[any]int),
		variableIDs:              make(map[string]int),
	}
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
	return &variableFlowState{owner: state.owner, values: state.values, shared: true}
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
	lastDefined := -1
	for id := 0; id < valueCount; id++ {
		value := present[0].definednessByID(id)
		for _, state := range present[1:] {
			value = joinVariableDefinedness(value, state.definednessByID(id))
		}
		if value != VariableUndefined {
			joined.values[id] = value
			lastDefined = id
		}
	}
	joined.values = joined.values[:lastDefined+1]
	return joined
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
	return VariableUndefined
}

func (s *variableFlowState) definednessByID(id int) VariableDefinedness {
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
}

func (s *variableFlowState) detach(capacity int) {
	if s == nil {
		return
	}
	if s.shared {
		values := make([]VariableDefinedness, len(s.values), max(capacity, len(s.values)))
		copy(values, s.values)
		s.values = values
		s.shared = false
	}
	if len(s.values) < capacity {
		s.values = append(s.values, make([]VariableDefinedness, capacity-len(s.values))...)
	}
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
	return true
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
		return variableFlowResult{breaks: []*variableFlowState{cloneVariableFlowState(state)}}
	case *ast.ContinueNode:
		a.expression(n.Level, state, false)
		return variableFlowResult{continues: []*variableFlowState{cloneVariableFlowState(state)}}
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
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, body.continues...)...)
		next := joinedVariableFlowState(input, back)
		if equalVariableFlowState(next, header) {
			header = next
			break
		}
		header = next
	}
	exits := append([]*variableFlowState(nil), body.breaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), terminates: body.terminates}
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
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, body.continues...)...)
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
	exits := append([]*variableFlowState(nil), body.breaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), terminates: body.terminates}
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
		back := joinedVariableFlowState(append([]*variableFlowState{body.normal}, body.continues...)...)
		next := joinedVariableFlowState(entry, back)
		if equalVariableFlowState(next, header) {
			header = next
			break
		}
		header = next
	}
	exits := append([]*variableFlowState{entry, header}, body.breaks...)
	return variableFlowResult{normal: joinedVariableFlowState(exits...), terminates: body.terminates}
}

func (a *variableFlowAnalyzer) doWhileStatement(node *ast.DoWhileNode, input *variableFlowState) variableFlowResult {
	bodyInput := cloneVariableFlowState(input)
	var body variableFlowResult
	var conditionState *variableFlowState
	mayIterate, mayExit := loopConditionPaths(node)
	for iteration := 0; iteration < maxVariableFlowIterations; iteration++ {
		body = a.statements(node.Body, bodyInput)
		conditionState = joinedVariableFlowState(append([]*variableFlowState{body.normal}, body.continues...)...)
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
	exits := append([]*variableFlowState(nil), body.breaks...)
	if mayExit {
		exits = append(exits, conditionState)
	}
	return variableFlowResult{normal: joinedVariableFlowState(exits...), terminates: body.terminates}
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
	allIncoming := []*variableFlowState{joined.normal}
	allIncoming = append(allIncoming, joined.breaks...)
	allIncoming = append(allIncoming, joined.continues...)
	allIncoming = append(allIncoming, joined.terminates...)
	// Analyse the finally body against every incoming path so its reads receive
	// the conservative joined state even when the try body cannot continue.
	a.statements(node.Finally, joinedVariableFlowState(allIncoming...))
	if joined.normal == nil {
		return variableFlowResult{breaks: joined.breaks, continues: joined.continues, terminates: joined.terminates}
	}
	normalFinally := a.statements(node.Finally, joined.normal)
	normalFinally.breaks = append(normalFinally.breaks, joined.breaks...)
	normalFinally.continues = append(normalFinally.continues, joined.continues...)
	normalFinally.terminates = append(normalFinally.terminates, joined.terminates...)
	return normalFinally
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
	joined.normal = joinedVariableFlowState(joined.normal, joinedVariableFlowState(joined.breaks...))
	joined.breaks = nil
	if !hasDefault {
		joined.normal = joinedVariableFlowState(joined.normal, entry)
	}
	return joined
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
	function, ok := a.resolver.ResolveFunction(resolvedName)
	if !ok {
		return nil
	}
	return function.Params
}

func (a *variableFlowAnalyzer) resolveCallClassName(name string) string {
	switch strings.ToLower(strings.TrimPrefix(name, `\`)) {
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
	case *ast.AssignmentNode:
		a.assignment(n, state)
	case *ast.FunctionCallNode:
		name := strings.ToLower(functionCallName(n))
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

func replaceVariableFlowState(destination, source *variableFlowState) {
	if destination == nil || source == nil {
		return
	}
	source.shared = true
	destination.values = source.values
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

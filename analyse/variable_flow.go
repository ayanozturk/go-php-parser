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

// VariableFlowReader exposes deterministic, immutable variable-read facts.
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
	values map[string]VariableDefinedness
	shared bool
}

type variableFlowResult struct {
	normal     *variableFlowState
	breaks     []*variableFlowState
	continues  []*variableFlowState
	terminates []*variableFlowState
}

type variableFlowAnalyzer struct {
	filename  string
	readIndex map[ast.Node]int
	reads     []variableReadFact
}

func buildVariableFlowFacts(filename string, nodes []ast.Node) []variableReadFact {
	analyzer := &variableFlowAnalyzer{filename: filename, readIndex: make(map[ast.Node]int)}
	analyzer.statements(nodes, initialVariableFlowState())
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

func initialVariableFlowState() *variableFlowState {
	return &variableFlowState{}
}

func functionVariableFlowState(function *ast.FunctionNode, includeThis bool) *variableFlowState {
	state := initialVariableFlowState()
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
	return &variableFlowState{values: state.values, shared: true}
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
	joined := &variableFlowState{values: make(map[string]VariableDefinedness)}
	names := make(map[string]struct{})
	for _, state := range present {
		for name := range state.values {
			names[name] = struct{}{}
		}
	}
	for name := range names {
		value := present[0].definedness(name)
		for _, state := range present[1:] {
			value = joinVariableDefinedness(value, state.definedness(name))
		}
		if value != VariableUndefined {
			joined.values[name] = value
		}
	}
	return joined
}

func (s *variableFlowState) definedness(name string) VariableDefinedness {
	if s == nil {
		return VariableUndefined
	}
	if value, ok := s.values[name]; ok {
		return value
	}
	if _, predefined := predefinedVariableSet[name]; predefined {
		return VariableDefinitelyDefined
	}
	return VariableUndefined
}

func (s *variableFlowState) set(name string, value VariableDefinedness) {
	s.detach(len(s.values) + 1)
	s.values[name] = value
}

func (s *variableFlowState) unset(name string) {
	if s == nil || s.definedness(name) == VariableUndefined {
		return
	}
	s.detach(len(s.values))
	delete(s.values, name)
}

func (s *variableFlowState) detach(capacity int) {
	if s == nil {
		return
	}
	if s.shared {
		values := make(map[string]VariableDefinedness, capacity)
		for existingName, existingValue := range s.values {
			values[existingName] = existingValue
		}
		s.values = values
		s.shared = false
	}
	if s.values == nil {
		s.values = make(map[string]VariableDefinedness, capacity)
	}
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
	if len(left.values) != len(right.values) {
		return false
	}
	for name, value := range left.values {
		if right.values[name] != value {
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
		a.statements(n.Body, functionVariableFlowState(n, false))
		return variableFlowResult{normal: state}
	case *ast.ClassNode:
		a.propertyHooks(n.Properties)
		for _, method := range n.Methods {
			if function, ok := method.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(function, true))
			}
		}
		return variableFlowResult{normal: state}
	case *ast.TraitNode:
		a.propertyHooks(n.Body)
		for _, member := range n.Body {
			if function, ok := member.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(function, true))
			}
		}
		return variableFlowResult{normal: state}
	case *ast.EnumNode:
		for _, member := range n.Methods {
			if function, ok := member.(*ast.FunctionNode); ok {
				a.statements(function.Body, functionVariableFlowState(function, true))
			}
		}
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
			state := initialVariableFlowState()
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
		for _, argument := range n.Args {
			a.expression(argumentValue(argument), state, suppressed)
		}
	case *ast.MethodCallNode:
		a.expression(n.Object, state, suppressed)
		for _, argument := range n.Args {
			a.expression(argumentValue(argument), state, suppressed)
		}
	case *ast.NewNode:
		a.expression(n.ClassExpr, state, suppressed)
		for _, argument := range n.Args {
			a.expression(argumentValue(argument), state, suppressed)
		}
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
		// The current AST does not retain an anonymous closure's explicit use
		// list. Preserve outer definedness conservatively, then isolate all
		// writes inside the closure from the enclosing state.
		if n.Name == "" {
			closureState := cloneVariableFlowState(state)
			if isStaticMethod(n) {
				closureState.unset("this")
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
	start, end := node.GetPos(), node.GetEndPos()
	if a.filename == "" || start.Offset < 0 || end.Offset <= start.Offset {
		return
	}
	fact := variableReadFact{start: start, end: end, name: name, state: state, compact: compact}
	if index, ok := a.readIndex[node]; ok {
		a.reads[index].state = joinVariableDefinedness(a.reads[index].state, state)
		return
	}
	a.readIndex[node] = len(a.reads)
	a.reads = append(a.reads, fact)
}

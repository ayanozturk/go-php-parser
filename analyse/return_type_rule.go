package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"sort"
	"strings"
)

type ReturnTypeRule struct{}

type observedReturn struct {
	Type Type
	Pos  ast.Position
}

type functionScope struct {
	className     string
	typeCtx       fileTypeContext
	propertyDecls map[string]Type
	variables     map[string]Type
	properties    map[string]Type
	methods       map[string]ResolvedMethod
	methodReturns map[string]Type
}

type ReturnTypeError struct {
	FuncName     string
	DeclaredType string
	ActualType   string
	Pos          ast.Position
}

func (e *ReturnTypeError) Error() string {
	return fmt.Sprintf("Function %s: return type mismatch, declared: %s, actual: %s at %d:%d", e.FuncName, e.DeclaredType, e.ActualType, e.Pos.Line, e.Pos.Column)
}

func (r *ReturnTypeRule) CheckFunctionReturnType(fn *ast.FunctionNode, class *ast.ClassNode, typeCtx fileTypeContext, ctx *AnalysisContext) []error {
	declaredType := declaredFunctionReturnType(fn, typeCtx)
	if declaredType.IsEmpty() {
		return nil // no declared return type, nothing to check
	}

	// Collect all actual return types
	returnTypes := map[string]int{}
	var firstMismatch *ReturnTypeError
	scope := newFunctionScope(class, fn, typeCtx)
	for _, ret := range collectObservedReturns(fn.Body, scope, ctx) {
		actualType := ret.Type
		actualLabel := actualType.String()
		if actualLabel == "" {
			actualLabel = "mixed"
		}
		returnTypes[actualLabel]++
		if firstMismatch == nil && !declaredType.AcceptsWithContext(actualType, scope, ctx) {
			firstMismatch = &ReturnTypeError{
				FuncName:     fn.Name,
				DeclaredType: declaredType.String(),
				ActualType:   actualLabel,
				Pos:          ret.Pos,
			}
		}
	}
	declaredLabel := declaredType.String()
	if len(returnTypes) > 0 && returnTypes[declaredLabel] == 0 {
		// If none of the observed actual types are exactly the declared type,
		// check if any observed type is still compatible (e.g., mixed, float vs int, void vs null).
		hasCompatible := false
		var foundTypes []string
		for t := range returnTypes {
			foundTypes = append(foundTypes, t)
			if declaredType.AcceptsWithContext(ParseType(t), scope, ctx) {
				hasCompatible = true
			}
		}
		sort.Strings(foundTypes)
		if !hasCompatible {
			return []error{&ReturnTypeError{
				FuncName:     fn.Name,
				DeclaredType: declaredLabel,
				ActualType:   fmt.Sprintf("%v", foundTypes),
				Pos:          fn.GetPos(),
			}}
		}
	}
	if len(returnTypes) > 1 {
		// If all observed types are compatible with the declared type, do not report.
		allCompat := true
		var foundTypes []string
		for t := range returnTypes {
			foundTypes = append(foundTypes, t)
			if !declaredType.AcceptsWithContext(ParseType(t), scope, ctx) {
				allCompat = false
			}
		}
		sort.Strings(foundTypes)
		if !allCompat {
			return []error{&ReturnTypeError{
				FuncName:     fn.Name,
				DeclaredType: declaredLabel,
				ActualType:   fmt.Sprintf("multiple: %v", foundTypes),
				Pos:          fn.GetPos(),
			}}
		}
	}
	if firstMismatch != nil {
		return []error{firstMismatch}
	}
	return nil
}

func (r *ReturnTypeRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	fileCtx := collectFileTypeContext(nodes)
	var checkFunc func(n ast.Node, class *ast.ClassNode)
	checkFunc = func(n ast.Node, class *ast.ClassNode) {
		switch fn := n.(type) {
		case *ast.FunctionNode:
			errs := r.CheckFunctionReturnType(fn, class, fileCtx, ctx)
			for _, err := range errs {
				pos := fn.GetPos()
				if typedErr, ok := err.(*ReturnTypeError); ok {
					pos = typedErr.Pos
				}
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     pos.Line,
					Column:   pos.Column,
					Code:     "A.RETURN.TYPE",
					Message:  err.Error(),
				})
			}
		case *ast.ClassNode:
			for _, m := range fn.Methods {
				checkFunc(m, fn)
			}
		}
	}
	for _, n := range nodes {
		checkFunc(n, nil)
	}
	return issues
}

func collectObservedReturns(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext) []observedReturn {
	var returns []observedReturn
	for _, n := range nodes {
		switch n := n.(type) {
		case *ast.ExpressionStmt:
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.ReturnNode:
			returns = append(returns, observedReturn{Type: inferType(n.Expr, scope, ctx), Pos: n.GetPos()})
		case *ast.AssignmentNode:
			applyAssignmentScope(scope, n, ctx)
		case *ast.IfNode:
			returns = append(returns, collectObservedReturns(n.Body, scope.clone(), ctx)...)
			for _, elseif := range n.ElseIfs {
				returns = append(returns, collectObservedReturns(elseif.Body, scope.clone(), ctx)...)
			}
			if n.Else != nil {
				returns = append(returns, collectObservedReturns(n.Else.Body, scope.clone(), ctx)...)
			}
		case *ast.BlockNode:
			returns = append(returns, collectObservedReturns(n.Statements, scope.clone(), ctx)...)
		case *ast.WhileNode:
			returns = append(returns, collectObservedReturns(n.Body, scope.clone(), ctx)...)
		case *ast.ForeachNode:
			returns = append(returns, collectObservedReturns(n.Body, scope.clone(), ctx)...)
		}
	}
	return returns
}

// inferType determines a simple type label for a given AST node.
// Kept lean; detailed cases are delegated to helpers to reduce complexity.
func inferType(expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type {
	switch n := expr.(type) {
	case *ast.FunctionCallNode:
		return inferFunctionCallType(n)
	case *ast.MethodCallNode:
		return inferMethodCallType(n, scope, ctx)
	case *ast.NewNode:
		return inferNewTypeWithScope(n, scope)
	case *ast.PropertyFetchNode:
		return inferPropertyFetchType(n, scope, ctx)
	case *ast.ConcatNode:
		return ParseType("string")
	case *ast.ExpressionStmt:
		return inferType(n.Expr, scope, ctx)
	case *ast.TypeCastNode:
		return ParseType(n.Type)
	case *ast.VariableNode:
		if scope != nil {
			if t, ok := scope.variables[n.Name]; ok {
				return t
			}
		}
		return MixedType()
	case nil:
		return ParseType("void")
	default:
		if t := inferNodeKindType(n); t != "" {
			return ParseType(t)
		}
		return ParseType(inferFallbackType(n))
	}
}

// inferFunctionCallType handles known function-call return types.
func inferFunctionCallType(n *ast.FunctionCallNode) Type {
	if n == nil || n.Name == nil {
		return MixedType()
	}
	if id, ok := n.Name.(*ast.IdentifierNode); ok {
		// Normalize FQFN: trim leading backslashes and namespaces
		name := strings.TrimLeft(id.Value, "\\")
		if idx := strings.LastIndex(name, "\\"); idx != -1 {
			name = name[idx+1:]
		}
		name = strings.ToLower(name)
		switch name {
		case "implode", "join", "sprintf", "json_encode", "strval":
			return ParseType("string")
		}
	}
	return MixedType()
}

// inferNodeKindType maps concrete node kinds to simple types. Returns "" if unknown.
func inferNodeKindType(n ast.Node) string {
	switch n.(type) {
	case *ast.IntegerLiteral, *ast.IntegerNode:
		return "int"
	case *ast.FloatLiteral, *ast.FloatNode:
		return "float"
	case *ast.StringLiteral, *ast.InterpolatedStringLiteral, *ast.StringNode:
		return "string"
	case *ast.BooleanLiteral, *ast.BooleanNode:
		return "bool"
	case *ast.NullLiteral, *ast.NullNode:
		return "null"
	case *ast.VariableNode:
		return "mixed"
	}
	return ""
}

// inferFallbackType tries value-based inference as a last resort, defaulting to mixed.
func inferFallbackType(n ast.Node) string {
	if lit, ok := n.(interface{ GetValue() interface{} }); ok {
		switch lit.GetValue().(type) {
		case int, int64:
			return "int"
		case float32, float64:
			return "float"
		case string:
			return "string"
		case bool:
			return "bool"
		case nil:
			return "null"
		}
	}
	return "mixed"
}

func declaredFunctionReturnType(fn *ast.FunctionNode, typeCtx fileTypeContext) Type {
	if fn == nil {
		return EmptyType()
	}
	if fn.ReturnType != "" {
		return ParseType(normalizeTypeWithContext(fn.ReturnType, typeCtx))
	}
	if fn.PHPDoc != nil && fn.PHPDoc.ReturnType != "" {
		return ParseType(normalizeTypeWithContext(fn.PHPDoc.ReturnType, typeCtx))
	}
	return EmptyType()
}

func newFunctionScope(class *ast.ClassNode, fn *ast.FunctionNode, typeCtx fileTypeContext) *functionScope {
	scope := &functionScope{
		typeCtx:       typeCtx,
		propertyDecls: make(map[string]Type),
		variables:     make(map[string]Type),
		properties:    make(map[string]Type),
		methods:       make(map[string]ResolvedMethod),
		methodReturns: make(map[string]Type),
	}

	if class != nil {
		scope.className = typeCtx.resolveClassLike(class.Name)
		for _, propertyNode := range class.Properties {
			property, ok := propertyNode.(*ast.PropertyNode)
			if !ok {
				continue
			}
			propertyType := ParseType(normalizeTypeWithContext(property.TypeHint, typeCtx))
			if propertyType.IsEmpty() && property.DefaultValue != nil {
				propertyType = inferType(property.DefaultValue, scope, nil)
			}
			if !propertyType.IsEmpty() {
				scope.propertyDecls[property.Name] = propertyType
				scope.properties[property.Name] = propertyType
			}
		}
		for _, promoted := range promotedClassProperties(class, typeCtx, scope) {
			scope.propertyDecls[promoted.name] = promoted.typ
			scope.properties[promoted.name] = promoted.typ
		}
		for _, methodNode := range class.Methods {
			method, ok := methodNode.(*ast.FunctionNode)
			if !ok {
				continue
			}
			methodType := declaredFunctionReturnType(method, typeCtx)
			resolved := ResolvedMethod{
				Name:       method.Name,
				ReturnType: methodType.String(),
				Params:     make([]ResolvedParam, 0, len(method.Params)),
			}
			for _, paramNode := range method.Params {
				param, ok := paramNode.(*ast.ParamNode)
				if !ok {
					continue
				}
				paramType := ParseType(normalizeTypeWithContext(param.TypeHint, typeCtx))
				if paramType.IsEmpty() && param.UnionType != nil {
					paramType = ParseType(normalizeTypeWithContext(param.UnionType.TokenLiteral(), typeCtx))
				}
				if paramType.IsEmpty() && method.PHPDoc != nil {
					paramType = ParseType(normalizeTypeWithContext(method.PHPDoc.GetParamTypeFromPHPDoc(param.Name), typeCtx))
				}
				resolved.Params = append(resolved.Params, ResolvedParam{Name: param.Name, Type: paramType.String()})
			}
			scope.methods[strings.ToLower(method.Name)] = resolved
			if !methodType.IsEmpty() {
				scope.methodReturns[strings.ToLower(method.Name)] = methodType
			}
		}
	}

	for _, paramNode := range fn.Params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok {
			continue
		}
		paramType := ParseType(normalizeTypeWithContext(param.TypeHint, typeCtx))
		if paramType.IsEmpty() && param.UnionType != nil {
			paramType = ParseType(normalizeTypeWithContext(param.UnionType.TokenLiteral(), typeCtx))
		}
		if paramType.IsEmpty() && fn.PHPDoc != nil {
			paramType = ParseType(normalizeTypeWithContext(fn.PHPDoc.GetParamTypeFromPHPDoc(param.Name), typeCtx))
		}
		if paramType.IsEmpty() && param.DefaultValue != nil {
			paramType = inferType(param.DefaultValue, scope, nil)
		}
		if !paramType.IsEmpty() {
			scope.variables[param.Name] = paramType
		}
	}

	return scope
}

type promotedProperty struct {
	name string
	typ  Type
}

func promotedClassProperties(class *ast.ClassNode, typeCtx fileTypeContext, scope *functionScope) []promotedProperty {
	if class == nil {
		return nil
	}

	var properties []promotedProperty
	for _, methodNode := range class.Methods {
		method, ok := methodNode.(*ast.FunctionNode)
		if !ok || method == nil || !strings.EqualFold(method.Name, "__construct") {
			continue
		}
		for _, paramNode := range method.Params {
			param, ok := paramNode.(*ast.ParamNode)
			if !ok || !param.IsPromoted {
				continue
			}
			paramType := ParseType(normalizeTypeWithContext(param.TypeHint, typeCtx))
			if paramType.IsEmpty() && param.UnionType != nil {
				paramType = ParseType(normalizeTypeWithContext(param.UnionType.TokenLiteral(), typeCtx))
			}
			if paramType.IsEmpty() && method.PHPDoc != nil {
				paramType = ParseType(normalizeTypeWithContext(method.PHPDoc.GetParamTypeFromPHPDoc(param.Name), typeCtx))
			}
			if paramType.IsEmpty() && param.DefaultValue != nil {
				paramType = inferType(param.DefaultValue, scope, nil)
			}
			if paramType.IsEmpty() {
				continue
			}
			properties = append(properties, promotedProperty{name: param.Name, typ: paramType})
		}
	}
	return properties
}

func (s *functionScope) clone() *functionScope {
	if s == nil {
		return nil
	}
	clone := &functionScope{
		className:     s.className,
		typeCtx:       s.typeCtx,
		propertyDecls: make(map[string]Type, len(s.propertyDecls)),
		variables:     make(map[string]Type, len(s.variables)),
		properties:    make(map[string]Type, len(s.properties)),
		methods:       make(map[string]ResolvedMethod, len(s.methods)),
		methodReturns: make(map[string]Type, len(s.methodReturns)),
	}
	for name, typ := range s.propertyDecls {
		clone.propertyDecls[name] = typ
	}
	for name, typ := range s.variables {
		clone.variables[name] = typ
	}
	for name, typ := range s.properties {
		clone.properties[name] = typ
	}
	for name, method := range s.methods {
		clone.methods[name] = method
	}
	for name, typ := range s.methodReturns {
		clone.methodReturns[name] = typ
	}
	return clone
}

func applyExpressionScope(scope *functionScope, expr ast.Node, ctx *AnalysisContext) {
	assignment, ok := expr.(*ast.AssignmentNode)
	if !ok {
		return
	}
	applyAssignmentScope(scope, assignment, ctx)
}

func applyAssignmentScope(scope *functionScope, assignment *ast.AssignmentNode, ctx *AnalysisContext) {
	if scope == nil || assignment == nil {
		return
	}

	assignedType := inferType(assignment.Right, scope, ctx)
	switch left := assignment.Left.(type) {
	case *ast.VariableNode:
		scope.variables[left.Name] = assignedType
	case *ast.PropertyFetchNode:
		if object, ok := left.Object.(*ast.VariableNode); ok && object.Name == "this" {
			scope.properties[left.Property] = assignedType
		}
	}
}

func inferNewType(node *ast.NewNode) Type {
	return inferNewTypeWithScope(node, nil)
}

func inferNewTypeWithScope(node *ast.NewNode, scope *functionScope) Type {
	if node == nil {
		return MixedType()
	}
	className := node.ClassName
	if className == "" {
		if ident, ok := node.ClassExpr.(*ast.IdentifierNode); ok {
			className = ident.Value
		}
	}
	if className == "" {
		return MixedType()
	}
	if scope != nil {
		className = scope.typeCtx.resolveClassLike(className)
	}
	return ClassType(className)
}

func inferPropertyFetchType(node *ast.PropertyFetchNode, scope *functionScope, ctx *AnalysisContext) Type {
	if node == nil {
		return MixedType()
	}
	if object, ok := node.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if propertyType, ok := resolveSameClassPropertyType(scope, node.Property); ok {
			return propertyType
		}
		if propertyType, ok := scope.properties[node.Property]; ok {
			return propertyType
		}
	}

	objectType := inferType(node.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return MixedType()
	}
	if scope != nil && strings.EqualFold(className, scope.className) {
		if propertyType, ok := resolveSameClassPropertyType(scope, node.Property); ok {
			return propertyType
		}
		if propertyType, ok := scope.properties[node.Property]; ok {
			return propertyType
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		if property, ok := ctx.Resolver.ResolveProperty(className, node.Property); ok {
			return ParseType(property.Type)
		}
	}
	return MixedType()
}

func inferMethodCallType(node *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext) Type {
	if node == nil {
		return MixedType()
	}
	if object, ok := node.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if method, ok := resolveSameClassMethod(scope, node.Method); ok {
			return ParseType(method.ReturnType)
		}
	}

	objectType := inferType(node.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return MixedType()
	}
	if scope != nil && strings.EqualFold(className, scope.className) {
		if method, ok := resolveSameClassMethod(scope, node.Method); ok {
			return ParseType(method.ReturnType)
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		if method, ok := ctx.Resolver.ResolveMethod(className, node.Method); ok {
			return ParseType(method.ReturnType)
		}
	}
	return MixedType()
}

func resolveSameClassMethod(scope *functionScope, methodName string) (ResolvedMethod, bool) {
	if scope == nil {
		return ResolvedMethod{}, false
	}
	method, ok := scope.methods[strings.ToLower(methodName)]
	return method, ok
}

func resolveSameClassPropertyType(scope *functionScope, propertyName string) (Type, bool) {
	if scope == nil {
		return EmptyType(), false
	}
	propertyType, ok := scope.propertyDecls[propertyName]
	return propertyType, ok
}

func init() {
	RegisterAnalysisRuleWithContext("A.RETURN.TYPE", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ReturnTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}

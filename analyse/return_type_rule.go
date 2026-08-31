package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"sort"
	"strings"
)

type ReturnTypeRule struct{}

type observedReturn struct {
	Type Type
	Pos  ast.Position
	Expr ast.Node
}

type expressionTypeInferer func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type

type functionScope struct {
	className        string
	typeCtx          FileTypeContext
	propertyDecls    map[string]Type
	variables        map[string]Type
	properties       map[string]Type
	variablesShared  bool
	propertiesShared bool
	methods          map[string]ResolvedMethod
	methodReturns    map[string]Type
	callableReturns  map[string]Type
	callablesShared  bool
	// genericContext maps variable names to their generic class instantiations
	// e.g., "$coll" → (className: "Collection", typeArguments: ["User"])
	genericContext map[string]GenericInstance
}

type classScopeData struct {
	className     string
	propertyDecls map[string]Type
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

func (r *ReturnTypeRule) CheckFunctionReturnType(fn *ast.FunctionNode, class *ast.ClassNode, typeCtx FileTypeContext, ctx *AnalysisContext) []error {
	return r.checkFunctionReturnType("", fn, class, typeCtx, ctx)
}

func (r *ReturnTypeRule) checkFunctionReturnType(filename string, fn *ast.FunctionNode, class *ast.ClassNode, typeCtx FileTypeContext, ctx *AnalysisContext) []error {
	declaredType := declaredFunctionReturnType(fn, typeCtx)
	if declaredType.IsEmpty() {
		return nil // no declared return type, nothing to check
	}

	// Collect all actual return types
	returnTypes := map[string]int{}
	var firstMismatch *ReturnTypeError
	scope := analysisFunctionScope(ctx, class, fn, typeCtx)
	for _, ret := range collectObservedReturns(filename, fn.Body, scope, ctx) {
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
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, _ *ast.FunctionNode, fileCtx FileTypeContext) {
		fn, ok := node.(*ast.FunctionNode)
		if !ok {
			return
		}
		for _, err := range r.checkFunctionReturnType(filename, fn, class, fileCtx, ctx) {
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
		if missingReturnValue(fn, filename, fileCtx, ctx) {
			name := fn.Name
			if name == "" {
				name = "closure"
			}
			declared := declaredFunctionReturnType(fn, fileCtx).String()
			issues = append(issues, issueSpan(filename, fn, "A.RETURN.TYPE", fmt.Sprintf(
				"Function %s: declared return type %s but not all paths return a value", name, declared,
			)))
		}
	})
	return issues
}

func missingReturnValue(fn *ast.FunctionNode, filename string, typeCtx FileTypeContext, ctx *AnalysisContext) bool {
	declared := declaredFunctionReturnType(fn, typeCtx)
	if fn == nil || declared.IsEmpty() || declared.String() == "void" || hasModifier(fn.Modifiers, "abstract") || functionContainsYield(fn) {
		return false
	}

	if ctx != nil && ctx.Flow != nil {
		if scope, ok := FlowScopeKeyForNode(filename, "function", fn, fn.Body); ok {
			if mayFallThrough, found := ctx.Flow.ScopeMayFallThrough(scope); found {
				return mayFallThrough
			}
		}
	}
	return !statementsTerminate(fn.Body)
}

func functionContainsYield(fn *ast.FunctionNode) bool {
	containsYield := false
	walkAll([]ast.Node{fn}, func(node ast.Node, _ *ast.ClassNode, currentFn *ast.FunctionNode, _ FileTypeContext) {
		if _, ok := node.(*ast.YieldNode); ok && currentFn == fn {
			containsYield = true
		}
	})
	return containsYield
}

func collectObservedReturns(filename string, nodes []ast.Node, scope *functionScope, ctx *AnalysisContext) []observedReturn {
	return collectObservedReturnsUsing(filename, nodes, scope, ctx, inferTypeWithFacts)
}

func collectObservedReturnsUsing(filename string, nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, infer expressionTypeInferer) []observedReturn {
	var returns []observedReturn
	for _, n := range nodes {
		switch n := n.(type) {
		case *ast.ExpressionStmt:
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.ReturnNode:
			returns = append(returns, observedReturn{Type: infer(filename, n.Expr, scope, ctx), Pos: n.GetPos(), Expr: n.Expr})
		case *ast.AssignmentNode:
			applyAssignmentScope(scope, n, ctx)
		case *ast.IfNode:
			returns = append(returns, collectObservedReturnsUsing(filename, n.Body, scope.clone(), ctx, infer)...)
			for _, elseif := range n.ElseIfs {
				returns = append(returns, collectObservedReturnsUsing(filename, elseif.Body, scope.clone(), ctx, infer)...)
			}
			if n.Else != nil {
				returns = append(returns, collectObservedReturnsUsing(filename, n.Else.Body, scope.clone(), ctx, infer)...)
			}
			applyLazyInitPropertyScope(scope, n, ctx)
		case *ast.BlockNode:
			returns = append(returns, collectObservedReturnsUsing(filename, n.Statements, scope.clone(), ctx, infer)...)
		case *ast.WhileNode:
			returns = append(returns, collectObservedReturnsUsing(filename, n.Body, scope.clone(), ctx, infer)...)
		case *ast.DoWhileNode:
			returns = append(returns, collectObservedReturnsUsing(filename, n.Body, scope.clone(), ctx, infer)...)
		case *ast.ForNode:
			for _, expression := range n.Init {
				applyExpressionScope(scope, expression, ctx)
			}
			returns = append(returns, collectObservedReturnsUsing(filename, n.Body, scope.clone(), ctx, infer)...)
		case *ast.ForeachNode:
			returns = append(returns, collectObservedReturnsUsing(filename, n.Body, scope.clone(), ctx, infer)...)
		}
	}
	return returns
}

func inferTypeWithFacts(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type {
	if filename != "" && expr != nil && ctx != nil && ctx.Facts != nil {
		start, end := expr.GetPos(), expr.GetEndPos()
		if end.Offset > start.Offset {
			key := SemanticFactKey{
				File:        filename,
				StartOffset: start.Offset,
				EndOffset:   end.Offset,
				Kind:        FactKindInferredType,
			}
			// fact.Type was already fully resolved when the fact was
			// generated (see addGeneratedInferredTypeFact); re-running it
			// through normalizeTypeWithContext here would treat an
			// already-qualified name (e.g. "RGTenantConnection\ExtendedPdo")
			// as relative and incorrectly prepend the current namespace.
			if fact, ok := ctx.Facts.Fact(key); ok && strings.TrimSpace(fact.Type) != "" {
				if inferred := ParseType(fact.Type); !inferred.IsEmpty() {
					return inferred
				}
			}

			// Check for narrowed-type facts (e.g., from instanceof checks)
			narrowedKey := SemanticFactKey{
				File:        filename,
				StartOffset: start.Offset,
				EndOffset:   end.Offset,
				Kind:        FactKindNarrowed,
			}
			if narrowedFact, ok := ctx.Facts.Fact(narrowedKey); ok && strings.TrimSpace(narrowedFact.Type) != "" {
				narrowedType := narrowedFact.Type
				if scope != nil {
					narrowedType = normalizeTypeWithContext(narrowedType, scope.typeCtx)
				}
				if narrowed := ParseType(narrowedType); !narrowed.IsEmpty() {
					return narrowed
				}
			}
		}
	}
	return inferType(expr, scope, ctx)
}

// inferType determines a simple type label for a given AST node.
// Kept lean; detailed cases are delegated to helpers to reduce complexity.
func inferType(expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type {
	switch n := expr.(type) {
	case *ast.FunctionCallNode:
		return inferFunctionCallType(n, scope, ctx)
	case *ast.MethodCallNode:
		return inferMethodCallType(n, scope, ctx)
	case *ast.NewNode:
		return inferNewTypeWithScope(n, scope)
	case *ast.FunctionNode, *ast.ArrowFunctionNode:
		return ParseType("callable")
	case *ast.PropertyFetchNode:
		return inferPropertyFetchType(n, scope, ctx)
	case *ast.ConcatNode:
		return ParseType("string")
	case *ast.ExpressionStmt:
		return inferType(n.Expr, scope, ctx)
	case *ast.TypeCastNode:
		return ParseType(n.Type)
	case *ast.TernaryExpr:
		ifTrue := inferType(n.IfTrue, scope, ctx)
		if n.IfTrue == nil {
			ifTrue = inferType(n.Condition, scope, ctx)
		}
		return unionInferredTypes(ifTrue, inferType(n.IfFalse, scope, ctx))
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

// inferFunctionCallType handles indexed and known built-in function return types.
func inferFunctionCallType(n *ast.FunctionCallNode, scope *functionScope, ctx *AnalysisContext) Type {
	if n == nil || n.Name == nil {
		return MixedType()
	}
	if variable, ok := n.Name.(*ast.VariableNode); ok && scope != nil {
		if returnType, found := scope.callableReturns[variable.Name]; found && !returnType.IsEmpty() {
			return returnType
		}
	}
	if name := functionCallName(n); name != "" && ctx != nil && ctx.Resolver != nil {
		var typeCtx FileTypeContext
		if scope != nil {
			typeCtx = scope.typeCtx
		}
		resolvedName := resolveFunctionNameForCall(name, typeCtx, ctx)
		if function, ok := ctx.Resolver.ResolveFunction(resolvedName); ok && strings.TrimSpace(function.ReturnType) != "" {
			return ParseType(function.ReturnType)
		}
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

func unionInferredTypes(types ...Type) Type {
	parts := make([]string, 0, len(types))
	for _, inferred := range types {
		if value := inferred.dnfString(); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return EmptyType()
	}
	return ParseType(strings.Join(parts, "|"))
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
	case *ast.ArrayNode:
		return "array"
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

func declaredFunctionReturnType(fn *ast.FunctionNode, typeCtx FileTypeContext) Type {
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

func newFunctionScope(class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) *functionScope {
	return newFunctionScopeWithContext(nil, class, fn, typeCtx)
}

func newFunctionScopeWithContext(ctx *AnalysisContext, class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) *functionScope {
	scope := &functionScope{
		typeCtx:        typeCtx,
		propertyDecls:  make(map[string]Type),
		variables:      make(map[string]Type),
		properties:     make(map[string]Type),
		methods:        make(map[string]ResolvedMethod),
		methodReturns:  make(map[string]Type),
		genericContext: make(map[string]GenericInstance),
	}

	if class != nil {
		classData := analysisClassScopeData(ctx, class, typeCtx)
		scope.className = classData.className
		scope.propertyDecls = classData.propertyDecls
		scope.properties = classData.properties
		scope.propertiesShared = true
		scope.methods = classData.methods
		scope.methodReturns = classData.methodReturns
	}

	for _, paramNode := range fn.Params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok {
			continue
		}
		documentedType := ""
		if fn.PHPDoc != nil {
			documentedType = fn.PHPDoc.GetParamTypeFromPHPDoc(param.Name)
		}
		paramType := ParseType(normalizeTypeWithContext(param.TypeHint, typeCtx))
		if paramType.IsEmpty() && param.UnionType != nil {
			paramType = ParseType(normalizeTypeWithContext(param.UnionType.TokenLiteral(), typeCtx))
		}
		if paramType.IsEmpty() && documentedType != "" {
			paramType = ParseType(normalizeTypeWithContext(documentedType, typeCtx))
		}
		if paramType.IsEmpty() && param.DefaultValue != nil {
			paramType = inferType(param.DefaultValue, scope, nil)
		}
		if !paramType.IsEmpty() {
			scope.variables[param.Name] = paramType
			if returnType := callableReturnType(documentedType, typeCtx); !returnType.IsEmpty() {
				scope.setCallableReturn(param.Name, returnType)
			}
			if instance, ok := parseExactGenericTypeFromString(documentedType); ok && strings.EqualFold(instance.ClassName, "class-string") && len(instance.TypeArguments) == 1 {
				target := normalizeTypeWithContext(instance.TypeArguments[0], typeCtx)
				if className, single := ParseType(target).SingleClassName(); single {
					scope.genericContext[param.Name] = GenericInstance{ClassName: "class-string", TypeArguments: []string{className}}
				}
			}

			// Check if param type is a generic class with declared type arguments
			if genInst, ok := parseGenericTypeFromString(paramType.String()); ok {
				scope.genericContext[param.Name] = genInst
			} else {
				// Also check if the param type itself is a class that has generic parents
				if className, ok := paramType.SingleClassName(); ok && ctx != nil && ctx.Resolver != nil {
					if resolvedClass, ok := ctx.Resolver.ResolveClass(className); ok && len(resolvedClass.GenericParents) > 0 {
						// For now, just store the first generic parent as the primary generics
						// This handles cases like UserRepository(extends Repository<User>)
						if len(resolvedClass.GenericParents) > 0 {
							gparen := resolvedClass.GenericParents[0]
							if len(gparen.TypeArguments) > 0 {
								scope.genericContext[param.Name] = GenericInstance{
									ClassName:     gparen.Name,
									TypeArguments: gparen.TypeArguments,
								}
							}
						}
					}
				}
			}
		}
	}

	return scope
}

func buildClassScopeData(class *ast.ClassNode, typeCtx FileTypeContext) classScopeData {
	return buildClassScopeDataWithSeen(class, typeCtx, map[string]struct{}{})
}

func buildClassScopeDataWithSeen(class *ast.ClassNode, typeCtx FileTypeContext, seen map[string]struct{}) classScopeData {
	data := classScopeData{
		className:     typeCtx.resolveClassLike(class.Name),
		propertyDecls: make(map[string]Type),
		properties:    make(map[string]Type),
		methods:       make(map[string]ResolvedMethod),
		methodReturns: make(map[string]Type),
	}
	key := strings.ToLower(strings.TrimPrefix(data.className, `\`))
	if _, ok := seen[key]; ok {
		return data
	}
	seen[key] = struct{}{}

	if class.Extends != "" {
		parentName := typeCtx.resolveClassLike(class.Extends)
		if parent, ok := typeCtx.ClassNodes[strings.ToLower(strings.TrimPrefix(parentName, `\`))]; ok {
			parentData := buildClassScopeDataWithSeen(parent, typeCtx, seen)
			mergeClassScopeData(&data, parentData)
		}
	}
	scope := &functionScope{
		className:     data.className,
		typeCtx:       typeCtx,
		propertyDecls: data.propertyDecls,
		variables:     make(map[string]Type),
		properties:    data.properties,
		methods:       data.methods,
		methodReturns: data.methodReturns,
	}

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
			data.propertyDecls[property.Name] = propertyType
			data.properties[property.Name] = propertyType
		}
	}
	for _, promoted := range promotedClassProperties(class, typeCtx, scope) {
		data.propertyDecls[promoted.name] = promoted.typ
		data.properties[promoted.name] = promoted.typ
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
			resolved.Params = append(resolved.Params, ResolvedParam{
				Name:       param.Name,
				Type:       paramType.String(),
				HasDefault: param.DefaultValue != nil,
				IsVariadic: param.IsVariadic,
				IsByRef:    param.IsByRef,
				IsOut:      param.IsByRef,
			})
		}
		data.methods[strings.ToLower(method.Name)] = resolved
		if !methodType.IsEmpty() {
			data.methodReturns[strings.ToLower(method.Name)] = methodType
		}
	}
	return data
}

func mergeClassScopeData(dst *classScopeData, src classScopeData) {
	for name, typ := range src.propertyDecls {
		dst.propertyDecls[name] = typ
	}
	for name, typ := range src.properties {
		dst.properties[name] = typ
	}
	for name, method := range src.methods {
		dst.methods[name] = method
	}
	for name, typ := range src.methodReturns {
		dst.methodReturns[name] = typ
	}
}

func copyTypeMap(src map[string]Type) map[string]Type {
	dst := make(map[string]Type, len(src))
	for name, typ := range src {
		dst[name] = typ
	}
	return dst
}

func copyGenericContext(src map[string]GenericInstance) map[string]GenericInstance {
	if src == nil || len(src) == 0 {
		return make(map[string]GenericInstance)
	}
	dst := make(map[string]GenericInstance, len(src))
	for name, inst := range src {
		dst[name] = inst
	}
	return dst
}

type promotedProperty struct {
	name string
	typ  Type
}

func promotedClassProperties(class *ast.ClassNode, typeCtx FileTypeContext, scope *functionScope) []promotedProperty {
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
	s.variablesShared = true
	s.propertiesShared = true
	s.callablesShared = true
	clone := &functionScope{
		className:        s.className,
		typeCtx:          s.typeCtx,
		propertyDecls:    s.propertyDecls,
		variables:        s.variables,
		properties:       s.properties,
		variablesShared:  true,
		propertiesShared: true,
		methods:          s.methods,
		methodReturns:    s.methodReturns,
		callableReturns:  s.callableReturns,
		callablesShared:  true,
		genericContext:   copyGenericContext(s.genericContext),
	}
	return clone
}

func (s *functionScope) setCallableReturn(name string, typ Type) {
	if s == nil || typ.IsEmpty() {
		return
	}
	if s.callablesShared {
		s.callableReturns = copyTypeMap(s.callableReturns)
		s.callablesShared = false
	}
	if s.callableReturns == nil {
		s.callableReturns = make(map[string]Type)
	}
	s.callableReturns[name] = typ
}

func (s *functionScope) clearCallableReturn(name string) {
	if s == nil || s.callableReturns == nil {
		return
	}
	if s.callablesShared {
		s.callableReturns = copyTypeMap(s.callableReturns)
		s.callablesShared = false
	}
	delete(s.callableReturns, name)
}

func (s *functionScope) setVariable(name string, typ Type) {
	if s == nil {
		return
	}
	if s.variablesShared {
		s.variables = copyTypeMap(s.variables)
		s.variablesShared = false
	}
	if s.variables == nil {
		s.variables = make(map[string]Type)
	}
	s.variables[name] = typ
}

func (s *functionScope) setProperty(name string, typ Type) {
	if s == nil {
		return
	}
	if s.propertiesShared {
		s.properties = copyTypeMap(s.properties)
		s.propertiesShared = false
	}
	if s.properties == nil {
		s.properties = make(map[string]Type)
	}
	s.properties[name] = typ
}

func applyExpressionScope(scope *functionScope, expr ast.Node, ctx *AnalysisContext) {
	if condition, ok := assertionCondition(expr); ok {
		applyConditionTrueScope(scope, condition)
		return
	}
	assignment, ok := expr.(*ast.AssignmentNode)
	if !ok {
		return
	}
	applyAssignmentScope(scope, assignment, ctx)
}

func assertionCondition(expr ast.Node) (ast.Node, bool) {
	call, ok := expr.(*ast.FunctionCallNode)
	if !ok || len(call.Args) == 0 {
		return nil, false
	}
	nameNode, ok := call.Name.(*ast.IdentifierNode)
	if !ok {
		return nil, false
	}
	name := strings.TrimLeft(nameNode.Value, "\\")
	if idx := strings.LastIndex(name, "\\"); idx != -1 {
		name = name[idx+1:]
	}
	if !strings.EqualFold(name, "assert") {
		return nil, false
	}
	return argumentValue(call.Args[0]), true
}

func applyAssignmentScope(scope *functionScope, assignment *ast.AssignmentNode, ctx *AnalysisContext) {
	if scope == nil || assignment == nil {
		return
	}

	assignedType := inferType(assignment.Right, scope, ctx)
	switch left := assignment.Left.(type) {
	case *ast.VariableNode:
		scope.clearCallableReturn(left.Name)
		delete(scope.genericContext, left.Name)
		scope.setVariable(left.Name, assignedType)
		if returnType := declaredCallableExpressionReturnType(assignment.Right, scope.typeCtx); !returnType.IsEmpty() {
			scope.setCallableReturn(left.Name, returnType)
		}
		// If the assigned type contains generic arguments, track it
		if genInst, ok := parseGenericTypeFromString(assignedType.String()); ok {
			if scope.genericContext == nil {
				scope.genericContext = make(map[string]GenericInstance)
			}
			scope.genericContext[left.Name] = genInst
		}
	case *ast.PropertyFetchNode:
		if object, ok := left.Object.(*ast.VariableNode); ok && object.Name == "this" {
			scope.setProperty(left.Property, assignedType)
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
	if strings.HasPrefix(className, "$") {
		if target, ok := classStringTarget(scope, strings.TrimPrefix(className, "$")); ok {
			return target
		}
		return MixedType()
	}
	if className == "" {
		if variable, ok := node.ClassExpr.(*ast.VariableNode); ok {
			if target, found := classStringTarget(scope, variable.Name); found {
				return target
			}
			return MixedType()
		}
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

func classStringTarget(scope *functionScope, variable string) (Type, bool) {
	if scope == nil {
		return EmptyType(), false
	}
	instance, ok := scope.genericContext[variable]
	if !ok || !strings.EqualFold(instance.ClassName, "class-string") || len(instance.TypeArguments) != 1 {
		return EmptyType(), false
	}
	target := ParseType(instance.TypeArguments[0])
	_, single := target.SingleClassName()
	return target, single
}

func declaredCallableExpressionReturnType(expr ast.Node, typeCtx FileTypeContext) Type {
	switch closure := expr.(type) {
	case *ast.FunctionNode:
		return declaredFunctionReturnType(closure, typeCtx)
	case *ast.ArrowFunctionNode:
		return ParseType(normalizeTypeWithContext(closure.ReturnType, typeCtx))
	default:
		return EmptyType()
	}
}

func inferPropertyFetchType(node *ast.PropertyFetchNode, scope *functionScope, ctx *AnalysisContext) Type {
	if node == nil {
		return MixedType()
	}
	if object, ok := node.Object.(*ast.VariableNode); ok && object.Name == "this" && scope != nil {
		// Narrowed/assigned type takes precedence over the declared type so
		// that flow-based updates (e.g. lazy-init null strips) are respected.
		if propertyType, ok := scope.properties[node.Property]; ok {
			return propertyType
		}
		if propertyType, ok := resolveSameClassPropertyType(scope, node.Property); ok {
			return propertyType
		}
	}

	objectType := inferType(node.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return MixedType()
	}
	if scope != nil && strings.EqualFold(className, scope.className) {
		if propertyType, ok := scope.properties[node.Property]; ok {
			return propertyType
		}
		if propertyType, ok := resolveSameClassPropertyType(scope, node.Property); ok {
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
		if scope != nil && ctx != nil && ctx.Resolver != nil {
			if method, ok := ctx.Resolver.ResolveMethod(scope.className, node.Method); ok {
				return ParseType(method.ReturnType)
			}
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
	if scope != nil {
		if classData, ok := analysisClassScopeDataByName(ctx, className, scope.typeCtx); ok {
			if method, ok := classData.methods[strings.ToLower(node.Method)]; ok {
				return ParseType(method.ReturnType)
			}
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
	RegisterAnalysisRuleWithLevel("A.RETURN.TYPE", 10, "phpstan.types", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ReturnTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}

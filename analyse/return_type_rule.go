package analyse

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

type ReturnTypeRule struct{}

type observedReturn struct {
	Type Type
	Pos  ast.Position
	Expr ast.Node
}

type expressionTypeInferer func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type

// scopeTypeLayer is a persistent type map used by branch-sensitive scopes.
// A clone shares its immutable layer chain; the first write adds a small
// writable delta instead of copying every type visible at that branch.
type scopeTypeLayer struct {
	values map[string]Type
	parent *scopeTypeLayer
	depth  int
	name   string
	typ    Type
	hasOne bool
}

const maxScopeTypeLayerDepth = 16

func rootScopeTypeLayer(values map[string]Type) *scopeTypeLayer {
	if values == nil {
		values = make(map[string]Type)
	}
	return &scopeTypeLayer{values: values, depth: 1}
}

func deltaScopeTypeLayer(parent *scopeTypeLayer) *scopeTypeLayer {
	if parent == nil {
		return rootScopeTypeLayer(nil)
	}
	if parent.depth < maxScopeTypeLayerDepth {
		return &scopeTypeLayer{parent: parent, depth: parent.depth + 1}
	}

	capacity := 0
	for current := parent; current != nil; current = current.parent {
		capacity += len(current.values)
		if current.hasOne {
			capacity++
		}
	}
	values := make(map[string]Type, capacity)
	layers := make([]*scopeTypeLayer, 0, parent.depth)
	for current := parent; current != nil; current = current.parent {
		layers = append(layers, current)
	}
	for i := len(layers) - 1; i >= 0; i-- {
		for name, typ := range layers[i].values {
			values[name] = typ
		}
		if layers[i].hasOne {
			values[layers[i].name] = layers[i].typ
		}
	}
	return rootScopeTypeLayer(values)
}

func (l *scopeTypeLayer) get(name string) (Type, bool) {
	for current := l; current != nil; current = current.parent {
		if current.hasOne && current.name == name {
			return current.typ, true
		}
		if typ, ok := current.values[name]; ok {
			return typ, true
		}
	}
	return EmptyType(), false
}

func (l *scopeTypeLayer) set(name string, typ Type) {
	if l.hasOne && l.name == name {
		l.typ = typ
		return
	}
	if !l.hasOne && len(l.values) == 0 {
		l.name = name
		l.typ = typ
		l.hasOne = true
		return
	}
	if l.values == nil {
		l.values = make(map[string]Type)
	}
	l.values[name] = typ
}

type functionScope struct {
	className               string
	typeCtx                 FileTypeContext
	propertyDecls           map[string]Type
	variables               *scopeTypeLayer
	properties              *scopeTypeLayer
	variablesOwned          bool
	propertiesOwned         bool
	methods                 map[string]ResolvedMethod
	methodReturns           map[string]Type
	propertyCallableReturns map[string]Type
	callableReturns         map[string]Type
	callablesShared         bool
	arrayShapeCallables     map[string]map[string]arrayShapeField
	arrayShapesShared       bool
	arrayIndexKeys          map[string][]string
	arrayIndexKeysShared    bool
	classConstantValues     map[string]string
	propertyArrayShapes     map[string]map[string]arrayShapeField
	methodArrayShapes       map[string]map[string]arrayShapeField
	// genericContext maps variable names to their generic class instantiations
	// e.g., "$coll" → (className: "Collection", typeArguments: ["User"])
	genericContext       map[string]GenericInstance
	genericContextShared bool
}

type classScopeData struct {
	className               string
	propertyDecls           map[string]Type
	properties              map[string]Type
	methods                 map[string]ResolvedMethod
	methodReturns           map[string]Type
	propertyCallableReturns map[string]Type
	propertyArrayShapes     map[string]map[string]arrayShapeField
	methodArrayShapes       map[string]map[string]arrayShapeField
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
	return returnTypeIssuesForFile(filename, nodes, ctx)
}

func returnTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureStructuralIssues(filename, nodes, ctx)
	if !ctx.hasReturnTypeIssues {
		collectReturnTypeIssues(filename, nodes, ctx)
	}
	return ctx.returnTypeIssues
}

func collectReturnTypeIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) {
	if ctx.hasReturnTypeIssues {
		return
	}
	fileCtx := analysisFileTypeContext(ctx, nodes)
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, _ *ast.FunctionNode, fileCtx FileTypeContext) {
		appendReturnTypeOnNode(filename, node, class, fileCtx, ctx, &ctx.returnTypeIssues)
	})
	ctx.hasReturnTypeIssues = true
}

func appendReturnTypeOnNode(filename string, node ast.Node, class *ast.ClassNode, fileCtx FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	fn, ok := node.(*ast.FunctionNode)
	if !ok {
		return
	}
	rule := ReturnTypeRule{}
	for _, err := range rule.checkFunctionReturnType(filename, fn, class, fileCtx, ctx) {
		pos := fn.GetPos()
		if typedErr, ok := err.(*ReturnTypeError); ok {
			pos = typedErr.Pos
		}
		*issues = append(*issues, AnalysisIssue{
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
		*issues = append(*issues, issueSpan(filename, fn, "A.RETURN.TYPE", fmt.Sprintf(
			"Function %s: declared return type %s but not all paths return a value", name, declared,
		)))
	}
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
	case *ast.ArrayAccessNode:
		if field := lookupArrayShapeField(arrayShapeFieldsOf(n.Var, scope, ctx), n.Index, scope); !field.typ.IsEmpty() {
			return field.typ
		}
		return MixedType()
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
	case *ast.UnaryExpr:
		if n.Operator == "clone" {
			return inferType(n.Operand, scope, ctx)
		}
		if t := inferNodeKindType(n); t != "" {
			return ParseType(t)
		}
		return ParseType(inferFallbackType(n))
	case *ast.BinaryExpr:
		if n.Operator == "??" {
			return unionInferredTypes(inferType(n.Left, scope, ctx).withoutBuiltin("null"), inferType(n.Right, scope, ctx))
		}
		if t := inferNodeKindType(n); t != "" {
			return ParseType(t)
		}
		return ParseType(inferFallbackType(n))
	case *ast.MatchNode:
		armTypes := make([]Type, 0, len(n.Arms))
		for _, arm := range n.Arms {
			armTypes = append(armTypes, inferType(arm.Body, scope, ctx))
		}
		return unionInferredTypes(armTypes...)
	case *ast.VariableNode:
		if scope != nil {
			if t, ok := scope.variable(n.Name); ok {
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
	if returnType := inferCallableInvocationReturn(n.Name, scope, ctx); !returnType.IsEmpty() {
		return returnType
	}
	if name := functionCallName(n); name != "" && ctx != nil && ctx.Resolver != nil {
		var typeCtx FileTypeContext
		if scope != nil {
			typeCtx = scope.typeCtx
		}
		resolvedName := resolveFunctionNameForCall(name, typeCtx, ctx)
		if function, ok := resolveFunctionView(ctx.Resolver, resolvedName); ok && strings.TrimSpace(function.ReturnType) != "" {
			return ParseType(function.ReturnType)
		}
	}
	if id, ok := n.Name.(*ast.IdentifierNode); ok {
		// Normalize FQFN: trim leading backslashes and namespaces
		name := strings.TrimLeft(id.Value, "\\")
		if idx := strings.LastIndex(name, "\\"); idx != -1 {
			name = name[idx+1:]
		}
		name = asciiLowerIdent(name)
		switch name {
		case "implode", "join", "sprintf", "json_encode", "strval":
			return ParseType("string")
		}
	}
	return MixedType()
}

func inferCallableInvocationReturn(expr ast.Node, scope *functionScope, ctx *AnalysisContext) Type {
	switch callable := expr.(type) {
	case *ast.VariableNode:
		if scope != nil {
			return scope.callableReturns[callable.Name]
		}
	case *ast.ArrayAccessNode:
		return inferArrayShapeCallableReturn(callable, scope, ctx)
	case *ast.PropertyFetchNode:
		if object, ok := callable.Object.(*ast.VariableNode); ok && object.Name == "this" && scope != nil {
			if returnType := scope.propertyCallableReturns[callable.Property]; !returnType.IsEmpty() {
				return returnType
			}
		}
		objectType := inferType(callable.Object, scope, ctx)
		if className, ok := objectType.SingleClassName(); ok && ctx != nil && ctx.Resolver != nil {
			if property, found := ctx.Resolver.ResolveProperty(className, callable.Property); found {
				return ParseType(property.CallableReturnType)
			}
		}
	case *ast.FunctionCallNode:
		name := functionCallName(callable)
		if name != "" && ctx != nil && ctx.Resolver != nil {
			var typeCtx FileTypeContext
			if scope != nil {
				typeCtx = scope.typeCtx
			}
			if function, ok := resolveFunctionView(ctx.Resolver, resolveFunctionNameForCall(name, typeCtx, ctx)); ok {
				return ParseType(function.CallableReturnType)
			}
		}
	case *ast.MethodCallNode:
		if object, ok := callable.Object.(*ast.VariableNode); ok && object.Name == "this" && scope != nil {
			if method, found := resolveSameClassMethod(scope, callable.Method); found {
				return ParseType(method.CallableReturnType)
			}
		}
		objectType := inferType(callable.Object, scope, ctx)
		if className, ok := objectType.SingleClassName(); ok && ctx != nil && ctx.Resolver != nil {
			if method, found := ctx.Resolver.ResolveMethod(className, callable.Method); found {
				return ParseType(method.CallableReturnType)
			}
		}
	case *ast.FunctionNode, *ast.ArrowFunctionNode:
		var typeCtx FileTypeContext
		if scope != nil {
			typeCtx = scope.typeCtx
		}
		return declaredCallableExpressionReturnType(callable, typeCtx)
	}
	return EmptyType()
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

func methodReturnTypeAnnotation(fn *ast.FunctionNode) string {
	if fn == nil {
		return ""
	}
	if fn.PHPDoc != nil && fn.PHPDoc.ReturnType != "" {
		return fn.PHPDoc.ReturnType
	}
	return fn.ReturnType
}

func newFunctionScope(class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) *functionScope {
	return newFunctionScopeWithContext(nil, class, fn, typeCtx)
}

func newFunctionScopeWithContext(ctx *AnalysisContext, class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) *functionScope {
	scope := &functionScope{
		typeCtx:                 typeCtx,
		propertyDecls:           make(map[string]Type),
		variables:               rootScopeTypeLayer(nil),
		properties:              rootScopeTypeLayer(nil),
		variablesOwned:          true,
		propertiesOwned:         true,
		methods:                 make(map[string]ResolvedMethod),
		methodReturns:           make(map[string]Type),
		propertyCallableReturns: make(map[string]Type),
	}

	if class != nil {
		scope.classConstantValues = classConstantLiteralValues(class)
		classData := analysisClassScopeData(ctx, class, typeCtx)
		scope.className = classData.className
		scope.propertyDecls = classData.propertyDecls
		scope.properties = rootScopeTypeLayer(classData.properties)
		scope.propertiesOwned = false
		scope.methods = classData.methods
		scope.methodReturns = classData.methodReturns
		scope.propertyCallableReturns = classData.propertyCallableReturns
		scope.propertyArrayShapes = classData.propertyArrayShapes
		scope.methodArrayShapes = classData.methodArrayShapes
	}
	templateBounds := localTemplateBounds(class, fn, typeCtx)

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
			scope.setVariable(param.Name, paramType)
			if returnType := callableReturnType(documentedType, typeCtx); !returnType.IsEmpty() {
				scope.setCallableReturn(param.Name, returnType)
			}
			if fields := parseArrayShapeFields(documentedType, typeCtx); len(fields) > 0 {
				scope.setArrayShapeCallables(param.Name, fields)
			}
			if instance, ok := parseExactGenericTypeFromString(documentedType); ok && strings.EqualFold(instance.ClassName, "class-string") && len(instance.TypeArguments) == 1 {
				targetName := instance.TypeArguments[0]
				target := ""
				if bound, ok := templateBounds[asciiLowerIdent(strings.TrimSpace(targetName))]; ok {
					target = bound
				} else {
					target = normalizeTypeWithContext(targetName, typeCtx)
				}
				if className, single := ParseType(target).SingleClassName(); single {
					scope.setGenericContext(param.Name, GenericInstance{ClassName: "class-string", TypeArguments: []string{className}})
				}
			}

			// Check if param type is a generic class with declared type arguments
			if genInst, ok := parseGenericTypeFromString(paramType.String()); ok {
				scope.setGenericContext(param.Name, genInst)
			} else {
				// Also check if the param type itself is a class that has generic parents
				if className, ok := paramType.SingleClassName(); ok && ctx != nil && ctx.Resolver != nil {
					if resolvedClass, ok := ctx.Resolver.ResolveClass(className); ok && len(resolvedClass.GenericParents) > 0 {
						// For now, just store the first generic parent as the primary generics
						// This handles cases like UserRepository(extends Repository<User>)
						if len(resolvedClass.GenericParents) > 0 {
							gparen := resolvedClass.GenericParents[0]
							if len(gparen.TypeArguments) > 0 {
								scope.setGenericContext(param.Name, GenericInstance{
									ClassName:     gparen.Name,
									TypeArguments: gparen.TypeArguments,
								})
							}
						}
					}
				}
			}
		}
	}

	return scope
}

func localTemplateBounds(class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) map[string]string {
	bounds := make(map[string]string)
	add := func(doc *ast.PHPDocNode) {
		if doc == nil {
			return
		}
		for _, template := range doc.Templates {
			bound := normalizeTypeWithContext(template.Bound, typeCtx)
			if _, ok := ParseType(bound).SingleClassName(); ok {
				bounds[asciiLowerIdent(template.Name)] = bound
			} else {
				bounds[asciiLowerIdent(template.Name)] = ""
			}
		}
	}
	if class != nil {
		add(class.PHPDoc)
	}
	if fn != nil {
		add(fn.PHPDoc)
	}
	return bounds
}

func buildClassScopeData(class *ast.ClassNode, typeCtx FileTypeContext) classScopeData {
	return buildClassScopeDataWithSeen(class, typeCtx, map[string]struct{}{})
}

func buildClassScopeDataWithSeen(class *ast.ClassNode, typeCtx FileTypeContext, seen map[string]struct{}) classScopeData {
	data := classScopeData{
		className:               typeCtx.resolveClassLike(class.Name),
		propertyDecls:           make(map[string]Type),
		properties:              make(map[string]Type),
		methods:                 make(map[string]ResolvedMethod),
		methodReturns:           make(map[string]Type),
		propertyCallableReturns: make(map[string]Type),
		propertyArrayShapes:     make(map[string]map[string]arrayShapeField),
		methodArrayShapes:       make(map[string]map[string]arrayShapeField),
	}
	key := asciiLowerIdent(strings.TrimPrefix(data.className, `\`))
	if _, ok := seen[key]; ok {
		return data
	}
	seen[key] = struct{}{}

	if class.Extends != "" {
		parentName := typeCtx.resolveClassLike(class.Extends)
		if parent, ok := typeCtx.ClassNodes[asciiLowerIdent(strings.TrimPrefix(parentName, `\`))]; ok {
			parentData := buildClassScopeDataWithSeen(parent, typeCtx, seen)
			mergeClassScopeData(&data, parentData)
		}
	}
	scope := &functionScope{
		className:               data.className,
		typeCtx:                 typeCtx,
		propertyDecls:           data.propertyDecls,
		variables:               rootScopeTypeLayer(nil),
		properties:              rootScopeTypeLayer(data.properties),
		variablesOwned:          true,
		propertiesOwned:         true,
		methods:                 data.methods,
		methodReturns:           data.methodReturns,
		propertyCallableReturns: data.propertyCallableReturns,
	}

	for _, propertyNode := range class.Properties {
		property, ok := propertyNode.(*ast.PropertyNode)
		if !ok {
			continue
		}
		propertyType := ParseType(normalizeTypeWithContext(property.TypeHint, typeCtx))
		if property.PHPDoc != nil && property.PHPDoc.VarType != "" {
			if fields := parseArrayShapeFields(property.PHPDoc.VarType, typeCtx); len(fields) > 0 {
				data.propertyArrayShapes[property.Name] = fields
			}
			if callableReturn := callableReturnType(property.PHPDoc.VarType, typeCtx); !callableReturn.IsEmpty() {
				propertyType = ParseType("callable")
				data.propertyCallableReturns[property.Name] = callableReturn
			} else if propertyType.IsEmpty() {
				propertyType = ParseType(normalizeTypeWithContext(property.PHPDoc.VarType, typeCtx))
			}
		}
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
			Name:               method.Name,
			ReturnType:         methodType.String(),
			CallableReturnType: callableReturnType(methodReturnTypeAnnotation(method), typeCtx).dnfString(),
			Params:             make([]ResolvedParam, 0, len(method.Params)),
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
		data.methods[asciiLowerIdent(method.Name)] = resolved
		if !methodType.IsEmpty() {
			data.methodReturns[asciiLowerIdent(method.Name)] = methodType
		}
		if fields := parseArrayShapeFields(methodReturnTypeAnnotation(method), typeCtx); len(fields) > 0 {
			data.methodArrayShapes[asciiLowerIdent(method.Name)] = fields
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
	for name, typ := range src.propertyCallableReturns {
		dst.propertyCallableReturns[name] = typ
	}
	for name, fields := range src.propertyArrayShapes {
		dst.propertyArrayShapes[name] = fields
	}
	for name, fields := range src.methodArrayShapes {
		dst.methodArrayShapes[name] = fields
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
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]GenericInstance, len(src))
	for name, inst := range src {
		dst[name] = inst
	}
	return dst
}

func copyArrayShapeCallables(src map[string]map[string]arrayShapeField) map[string]map[string]arrayShapeField {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]map[string]arrayShapeField, len(src))
	for name, fields := range src {
		dst[name] = fields
	}
	return dst
}

func arrayShapeFieldsOf(expr ast.Node, scope *functionScope, ctx *AnalysisContext) map[string]arrayShapeField {
	if expr == nil || scope == nil {
		return nil
	}
	switch node := expr.(type) {
	case *ast.VariableNode:
		return scope.arrayShapeCallables[node.Name]
	case *ast.ArrayAccessNode:
		return lookupArrayShapeField(arrayShapeFieldsOf(node.Var, scope, ctx), node.Index, scope).nested
	case *ast.PropertyFetchNode:
		return propertyArrayShapesOf(node, scope, ctx)
	case *ast.MethodCallNode:
		return methodArrayShapesOf(node, scope, ctx)
	case *ast.ClassConstFetchNode:
		return staticPropertyArrayShapesOf(node, scope, ctx)
	default:
		return nil
	}
}

func inferArrayShapeCallableReturn(node *ast.ArrayAccessNode, scope *functionScope, ctx *AnalysisContext) Type {
	if node == nil {
		return EmptyType()
	}
	return lookupArrayShapeField(arrayShapeFieldsOf(node.Var, scope, ctx), node.Index, scope).callable
}

func propertyArrayShapesOf(node *ast.PropertyFetchNode, scope *functionScope, ctx *AnalysisContext) map[string]arrayShapeField {
	if node == nil || scope == nil {
		return nil
	}
	if object, ok := node.Object.(*ast.VariableNode); ok && object.Name == "this" {
		return scope.propertyArrayShapes[node.Property]
	}
	objectType := inferType(node.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return nil
	}
	if strings.EqualFold(className, scope.className) {
		return scope.propertyArrayShapes[node.Property]
	}
	data, ok := analysisClassScopeDataByName(ctx, className, scope.typeCtx)
	if !ok {
		return nil
	}
	return data.propertyArrayShapes[node.Property]
}

func methodArrayShapesOf(node *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext) map[string]arrayShapeField {
	if node == nil || scope == nil {
		return nil
	}
	key := asciiLowerIdent(node.Method)
	if object, ok := node.Object.(*ast.VariableNode); ok && object.Name == "this" {
		return scope.methodArrayShapes[key]
	}
	objectType := inferType(node.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return nil
	}
	if strings.EqualFold(className, scope.className) {
		return scope.methodArrayShapes[key]
	}
	data, ok := analysisClassScopeDataByName(ctx, className, scope.typeCtx)
	if !ok {
		return nil
	}
	return data.methodArrayShapes[key]
}

func staticPropertyArrayShapesOf(node *ast.ClassConstFetchNode, scope *functionScope, ctx *AnalysisContext) map[string]arrayShapeField {
	if node == nil || scope == nil || node.ConstExpr != nil || !strings.HasPrefix(node.Const, "$") {
		return nil
	}
	property := strings.TrimPrefix(node.Const, "$")
	className := strings.TrimLeft(node.Class, "\\")
	if className == "" || strings.EqualFold(className, "self") || strings.EqualFold(className, "static") {
		return scope.propertyArrayShapes[property]
	}
	data, ok := analysisClassScopeDataByName(ctx, className, scope.typeCtx)
	if !ok {
		return nil
	}
	return data.propertyArrayShapes[property]
}

type arrayIndexLookup struct {
	keys    []string
	all     bool
	numeric bool
}

func (lookup arrayIndexLookup) empty() bool {
	return !lookup.all && !lookup.numeric && len(lookup.keys) == 0
}

func lookupArrayShapeField(fields map[string]arrayShapeField, index ast.Node, scope *functionScope) arrayShapeField {
	if len(fields) == 0 {
		return arrayShapeField{}
	}
	lookup := resolveArrayIndex(index, scope)
	if lookup.empty() {
		return arrayShapeField{}
	}
	var result arrayShapeField
	for key, field := range fields {
		if !arrayIndexIncludes(lookup, key) {
			continue
		}
		result = mergeArrayShapeField(result, field)
	}
	return result
}

func arrayIndexIncludes(lookup arrayIndexLookup, key string) bool {
	if lookup.all {
		return true
	}
	if lookup.numeric {
		return isNumericArrayKey(key)
	}
	for _, candidate := range lookup.keys {
		if candidate == key {
			return true
		}
	}
	return false
}

func mergeArrayShapeField(left, right arrayShapeField) arrayShapeField {
	nested := copyArrayShapeFieldMap(left.nested)
	for key, field := range right.nested {
		if nested == nil {
			nested = make(map[string]arrayShapeField)
		}
		nested[key] = mergeArrayShapeField(nested[key], field)
	}
	return arrayShapeField{
		callable: unionInferredTypes(left.callable, right.callable),
		nested:   nested,
		typ:      unionInferredTypes(left.typ, right.typ),
	}
}

func copyArrayShapeFieldMap(src map[string]arrayShapeField) map[string]arrayShapeField {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]arrayShapeField, len(src))
	for key, field := range src {
		dst[key] = field
	}
	return dst
}

func resolveArrayIndex(node ast.Node, scope *functionScope) arrayIndexLookup {
	if node == nil {
		return arrayIndexLookup{}
	}
	if key, ok := literalArrayKey(node); ok {
		return arrayIndexLookup{keys: []string{key}}
	}
	switch value := node.(type) {
	case *ast.ConcatNode:
		parts := make([]string, 0, len(value.Parts))
		for _, part := range value.Parts {
			partLookup := resolveArrayIndex(part, scope)
			if partLookup.all || partLookup.numeric || len(partLookup.keys) != 1 {
				return arrayIndexLookup{}
			}
			parts = append(parts, partLookup.keys[0])
		}
		if len(parts) == 0 {
			return arrayIndexLookup{}
		}
		return arrayIndexLookup{keys: []string{strings.Join(parts, "")}}
	case *ast.BinaryExpr:
		if value.Operator == "." {
			return resolveArrayIndex(&ast.ConcatNode{Parts: []ast.Node{value.Left, value.Right}}, scope)
		}
	case *ast.TernaryExpr:
		ifTrue := value.IfTrue
		if ifTrue == nil {
			ifTrue = value.Condition
		}
		return mergeArrayIndexLookups(resolveArrayIndex(ifTrue, scope), resolveArrayIndex(value.IfFalse, scope))
	case *ast.MatchNode:
		if len(value.Arms) == 0 {
			return arrayIndexLookup{}
		}
		lookup := resolveArrayIndex(value.Arms[0].Body, scope)
		for _, arm := range value.Arms[1:] {
			lookup = mergeArrayIndexLookups(lookup, resolveArrayIndex(arm.Body, scope))
		}
		return lookup
	case *ast.IdentifierNode:
		if scope != nil {
			if key, ok := scope.typeCtx.Constants[value.Value]; ok {
				return arrayIndexLookup{keys: []string{key}}
			}
		}
		return arrayIndexLookup{}
	case *ast.VariableNode:
		if scope != nil {
			if keys := scope.arrayIndexKeys[value.Name]; len(keys) > 0 {
				return arrayIndexLookup{keys: append([]string(nil), keys...)}
			}
			if typ, ok := scope.variable(value.Name); ok {
				return arrayIndexLookupFromType(typ)
			}
		}
		return arrayIndexLookup{}
	case *ast.ClassConstFetchNode:
		if key, ok := classConstantArrayKey(value, scope); ok {
			return arrayIndexLookup{keys: []string{key}}
		}
	}
	return arrayIndexLookup{}
}

func mergeArrayIndexLookups(left, right arrayIndexLookup) arrayIndexLookup {
	if left.empty() && right.empty() {
		return arrayIndexLookup{}
	}
	if left.all || right.all || left.empty() || right.empty() {
		return arrayIndexLookup{all: true}
	}
	if left.numeric && right.numeric {
		return arrayIndexLookup{numeric: true}
	}
	if left.numeric || right.numeric {
		return arrayIndexLookup{all: true}
	}
	return arrayIndexLookup{keys: uniqueStrings(append(append([]string{}, left.keys...), right.keys...))}
}

func arrayIndexLookupFromType(typ Type) arrayIndexLookup {
	if typ.IsEmpty() {
		return arrayIndexLookup{}
	}
	if typ.hasBuiltin("mixed") || typ.hasBuiltin("string") {
		return arrayIndexLookup{all: true}
	}
	if typ.hasBuiltin("int") || typ.hasBuiltin("float") {
		return arrayIndexLookup{numeric: true}
	}
	return arrayIndexLookup{}
}

func classConstantArrayKey(node *ast.ClassConstFetchNode, scope *functionScope) (string, bool) {
	if node == nil || node.ConstExpr != nil || node.Const == "" || strings.HasPrefix(node.Const, "$") || scope == nil {
		return "", false
	}
	values := classConstantValuesFor(node.Class, scope)
	key, ok := values[node.Const]
	return key, ok
}

func classConstantValuesFor(className string, scope *functionScope) map[string]string {
	if scope == nil {
		return nil
	}
	className = strings.TrimLeft(className, "\\")
	if className == "" || strings.EqualFold(className, "self") || strings.EqualFold(className, "static") || (scope.className != "" && strings.EqualFold(className, scope.className)) {
		return scope.classConstantValues
	}
	class, ok := scope.typeCtx.ClassNodes[asciiLowerIdent(className)]
	if !ok {
		return nil
	}
	return classConstantLiteralValues(class)
}

func classConstantLiteralValues(class *ast.ClassNode) map[string]string {
	if class == nil {
		return nil
	}
	values := make(map[string]string)
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		switch value := node.(type) {
		case *ast.ConstantNode:
			if key, ok := literalArrayKey(value.Value); ok {
				values[value.Name] = key
			}
		case *ast.BlockNode:
			for _, child := range value.Statements {
				visit(child)
			}
		}
	}
	for _, node := range class.Constants {
		visit(node)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func isNumericArrayKey(key string) bool {
	_, err := strconv.ParseInt(key, 10, 64)
	return err == nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func copyArrayIndexKeys(src map[string][]string) map[string][]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string][]string, len(src))
	for name, keys := range src {
		dst[name] = keys
	}
	return dst
}

func literalArrayKey(node ast.Node) (string, bool) {
	switch value := node.(type) {
	case *ast.StringNode:
		return value.Value, true
	case *ast.StringLiteral:
		return value.Value, true
	case *ast.IntegerNode:
		return strconv.FormatInt(value.Value, 10), true
	case *ast.IntegerLiteral:
		return strconv.FormatInt(value.Value, 10), true
	default:
		return "", false
	}
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
	s.variablesOwned = false
	s.propertiesOwned = false
	s.callablesShared = true
	s.arrayShapesShared = true
	s.arrayIndexKeysShared = true
	s.genericContextShared = true
	clone := &functionScope{
		className:               s.className,
		typeCtx:                 s.typeCtx,
		propertyDecls:           s.propertyDecls,
		variables:               s.variables,
		properties:              s.properties,
		variablesOwned:          false,
		propertiesOwned:         false,
		methods:                 s.methods,
		methodReturns:           s.methodReturns,
		propertyCallableReturns: s.propertyCallableReturns,
		callableReturns:         s.callableReturns,
		callablesShared:         true,
		arrayShapeCallables:     s.arrayShapeCallables,
		arrayShapesShared:       true,
		arrayIndexKeys:          s.arrayIndexKeys,
		arrayIndexKeysShared:    true,
		classConstantValues:     s.classConstantValues,
		propertyArrayShapes:     s.propertyArrayShapes,
		methodArrayShapes:       s.methodArrayShapes,
		genericContext:          s.genericContext,
		genericContextShared:    true,
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
	if _, exists := s.callableReturns[name]; !exists {
		return
	}
	if s.callablesShared {
		s.callableReturns = copyTypeMap(s.callableReturns)
		s.callablesShared = false
	}
	delete(s.callableReturns, name)
}

func (s *functionScope) setArrayShapeCallables(name string, fields map[string]arrayShapeField) {
	if s == nil || name == "" || len(fields) == 0 {
		return
	}
	if s.arrayShapesShared {
		s.arrayShapeCallables = copyArrayShapeCallables(s.arrayShapeCallables)
		s.arrayShapesShared = false
	}
	if s.arrayShapeCallables == nil {
		s.arrayShapeCallables = make(map[string]map[string]arrayShapeField)
	}
	s.arrayShapeCallables[name] = fields
}

func (s *functionScope) clearArrayShapeCallables(name string) {
	if s == nil || s.arrayShapeCallables == nil {
		return
	}
	if _, exists := s.arrayShapeCallables[name]; !exists {
		return
	}
	if s.arrayShapesShared {
		s.arrayShapeCallables = copyArrayShapeCallables(s.arrayShapeCallables)
		s.arrayShapesShared = false
	}
	delete(s.arrayShapeCallables, name)
}

func (s *functionScope) setArrayIndexKeys(name string, keys []string) {
	if s == nil || name == "" || len(keys) == 0 {
		return
	}
	if s.arrayIndexKeysShared {
		s.arrayIndexKeys = copyArrayIndexKeys(s.arrayIndexKeys)
		s.arrayIndexKeysShared = false
	}
	if s.arrayIndexKeys == nil {
		s.arrayIndexKeys = make(map[string][]string)
	}
	s.arrayIndexKeys[name] = append([]string(nil), keys...)
}

func (s *functionScope) clearArrayIndexKeys(name string) {
	if s == nil || s.arrayIndexKeys == nil {
		return
	}
	if _, exists := s.arrayIndexKeys[name]; !exists {
		return
	}
	if s.arrayIndexKeysShared {
		s.arrayIndexKeys = copyArrayIndexKeys(s.arrayIndexKeys)
		s.arrayIndexKeysShared = false
	}
	delete(s.arrayIndexKeys, name)
}

func (s *functionScope) setGenericContext(name string, instance GenericInstance) {
	if s == nil || name == "" {
		return
	}
	if s.genericContextShared {
		s.genericContext = copyGenericContext(s.genericContext)
		s.genericContextShared = false
	}
	if s.genericContext == nil {
		s.genericContext = make(map[string]GenericInstance)
	}
	instance.TypeArguments = append([]string(nil), instance.TypeArguments...)
	s.genericContext[name] = instance
}

func (s *functionScope) clearGenericContext(name string) {
	if s == nil || s.genericContext == nil {
		return
	}
	if _, exists := s.genericContext[name]; !exists {
		return
	}
	if s.genericContextShared {
		s.genericContext = copyGenericContext(s.genericContext)
		s.genericContextShared = false
	}
	delete(s.genericContext, name)
}

func definiteArrayIndexKeys(expr ast.Node, scope *functionScope) []string {
	lookup := resolveArrayIndex(expr, scope)
	if lookup.all || lookup.numeric || len(lookup.keys) == 0 {
		return nil
	}
	return lookup.keys
}

func (s *functionScope) setVariable(name string, typ Type) {
	if s == nil {
		return
	}
	if s.variables == nil {
		s.variables = rootScopeTypeLayer(nil)
		s.variablesOwned = true
	} else if !s.variablesOwned {
		s.variables = deltaScopeTypeLayer(s.variables)
		s.variablesOwned = true
	}
	s.variables.set(name, typ)
}

func (s *functionScope) setProperty(name string, typ Type) {
	if s == nil {
		return
	}
	if s.properties == nil {
		s.properties = rootScopeTypeLayer(nil)
		s.propertiesOwned = true
	} else if !s.propertiesOwned {
		s.properties = deltaScopeTypeLayer(s.properties)
		s.propertiesOwned = true
	}
	s.properties.set(name, typ)
}

func (s *functionScope) variable(name string) (Type, bool) {
	if s == nil {
		return EmptyType(), false
	}
	return s.variables.get(name)
}

func (s *functionScope) property(name string) (Type, bool) {
	if s == nil {
		return EmptyType(), false
	}
	return s.properties.get(name)
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
		scope.clearArrayShapeCallables(left.Name)
		scope.clearArrayIndexKeys(left.Name)
		scope.clearGenericContext(left.Name)
		scope.setVariable(left.Name, assignedType)
		if keys := definiteArrayIndexKeys(assignment.Right, scope); len(keys) > 0 {
			scope.setArrayIndexKeys(left.Name, keys)
		}
		if returnType := inferCallableInvocationReturn(assignment.Right, scope, ctx); !returnType.IsEmpty() {
			scope.setCallableReturn(left.Name, returnType)
		}
		if source, ok := assignment.Right.(*ast.VariableNode); ok {
			if fields := scope.arrayShapeCallables[source.Name]; len(fields) > 0 {
				scope.setArrayShapeCallables(left.Name, fields)
			}
		}
		if nested := arrayShapeFieldsOf(assignment.Right, scope, ctx); len(nested) > 0 {
			scope.setArrayShapeCallables(left.Name, nested)
		}
		// If the assigned type contains generic arguments, track it
		if genInst, ok := parseGenericTypeFromString(assignedType.String()); ok {
			scope.setGenericContext(left.Name, genInst)
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
		if propertyType, ok := scope.property(node.Property); ok {
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
		if propertyType, ok := scope.property(node.Property); ok {
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
			if method, ok := classData.methods[asciiLowerIdent(node.Method)]; ok {
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
	method, ok := scope.methods[asciiLowerIdent(methodName)]
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
	RegisterAnalysisRuleWithLevel("A.RETURN.TYPE", 10, "types", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ReturnTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}

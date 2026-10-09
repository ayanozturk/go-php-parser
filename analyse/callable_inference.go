package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const callableInferenceDepthLimit = 16

// parseCallableContract retains arity flags that the PHPDoc type-reference
// walker intentionally erases. Unsupported reference parameters stay unknown.
func parseCallableContract(raw string, ft FileTypeContext, templates map[string]struct{}) (callableSignature, bool) {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	if len(splitTopLevelTypes(raw, '|')) > 1 || len(splitTopLevelTypes(raw, '&')) > 1 {
		return callableSignature{}, false
	}
	types, result, ok := phpDocCallableSignature(raw)
	if !ok {
		return callableSignature{}, false
	}
	open := strings.Index(raw, "(")
	depth, close := 0, -1
	for i, r := range raw[open:] {
		if r == '(' {
			depth++
		}
		if r == ')' {
			depth--
			if depth == 0 {
				close = open + i
				break
			}
		}
	}
	if close < 0 {
		return callableSignature{}, false
	}
	parts := splitTopLevelTypes(raw[open+1:close], ',')
	if strings.TrimSpace(raw[open+1:close]) == "" {
		parts = nil
	}
	if len(parts) != len(types) {
		return callableSignature{}, false
	}
	params := make([]ResolvedParam, len(types))
	for i, typ := range types {
		part := strings.TrimSpace(parts[i])
		// An intersection in a type is not a by-reference parameter marker.
		byRef := strings.HasPrefix(part, "&") || strings.Contains(part, "&$") || strings.Contains(part, "& $") || strings.HasSuffix(part, "&")
		if byRef {
			return callableSignature{}, false
		}
		params[i] = ResolvedParam{Name: fmt.Sprintf("arg%d", i+1), Type: normalizeTemplateAwareType(typ, ft, templates), HasDefault: strings.HasSuffix(part, "="), IsVariadic: strings.Contains(part, "...")}
	}
	return callableSignature{params: params, returnType: ParseType(normalizeTemplateAwareType(result, ft, templates))}, true
}

func normalizedCallableContract(raw string, ft FileTypeContext, templates map[string]struct{}) string {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	sig, ok := parseCallableContract(raw, ft, templates)
	if !ok {
		return ""
	}
	parts := make([]string, len(sig.params))
	for i, p := range sig.params {
		parts[i] = p.Type
		if p.IsVariadic {
			parts[i] += "..."
		} else if p.HasDefault {
			parts[i] += "="
		}
	}
	base := "callable"
	if strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(raw[:strings.Index(raw, "(")]), `\`), "Closure") {
		base = "Closure"
	}
	return base + "(" + strings.Join(parts, ", ") + "): " + sig.returnType.dnfString()
}

func callableExpressionScope(expr ast.Node, outer *functionScope, ctx *AnalysisContext) *functionScope {
	var fn *ast.FunctionNode
	arrow := false
	switch n := expr.(type) {
	case *ast.FunctionNode:
		fn = n
	case *ast.ArrowFunctionNode:
		fn = &ast.FunctionNode{Params: n.Params, ReturnType: n.ReturnType}
		arrow = true
	default:
		return nil
	}
	var ft FileTypeContext
	if outer != nil {
		ft = outer.typeCtx
	}
	local := newFunctionScopeWithContext(ctx, nil, fn, ft)
	for _, node := range fn.Params {
		if param, ok := node.(*ast.ParamNode); ok && param.IsVariadic {
			local.setVariable(param.Name, ParseType("array"))
			if element := TypeFromAST(param.TypeHint, ft); !element.IsEmpty() {
				local.setGenericContext(param.Name, GenericInstance{ClassName: "array", TypeArguments: []string{element.dnfString()}})
			}
		}
	}
	if outer == nil {
		return local
	}
	shared := outer.clone()
	local.functionScopeContext = shared.functionScopeContext
	local.properties, local.propertiesOwned = shared.properties, false
	local.callableInferenceDepth = outer.callableInferenceDepth + 1
	if arrow {
		// Arrow functions capture the outer scope by value, then parameters shadow it.
		shared.callableInferenceDepth = local.callableInferenceDepth
		for _, node := range fn.Params {
			if param, ok := node.(*ast.ParamNode); ok {
				typ, found := local.variable(param.Name)
				if !found {
					typ = MixedType()
				}
				shared.setVariable(param.Name, typ)
				shared.clearCallableReturn(param.Name)
				shared.clearCallableParams(param.Name)
				shared.clearArrayShapeCallables(param.Name)
				shared.clearGenericContext(param.Name)
			}
		}
		return shared
	}
	for _, capture := range fn.Uses {
		if typ, ok := outer.variable(capture.Name); ok {
			local.setVariable(capture.Name, typ)
		}
		if sig, ok := outer.callableSignatures[capture.Name]; ok {
			local.setCallableReturn(capture.Name, sig.returnType)
			local.setCallableParams(capture.Name, sig.params)
		}
		if fields := outer.arrayShapeCallables[capture.Name]; len(fields) > 0 {
			local.setArrayShapeCallables(capture.Name, fields)
		}
		if instance, ok := outer.genericContext[capture.Name]; ok {
			local.setGenericContext(capture.Name, instance)
		}
	}
	return local
}

func inferCallableExpressionSignature(expr ast.Node, outer *functionScope, ctx *AnalysisContext) (callableSignature, bool) {
	var nodes []ast.Node
	switch n := expr.(type) {
	case *ast.VariableNode:
		if outer != nil {
			sig, ok := outer.callableSignatures[n.Name]
			return sig, ok
		}
		return callableSignature{}, false
	case *ast.FunctionNode:
		nodes = n.Params
	case *ast.ArrowFunctionNode:
		nodes = n.Params
	default:
		return callableSignature{}, false
	}
	if outer != nil && outer.callableInferenceDepth >= callableInferenceDepthLimit {
		return callableSignature{}, false
	}
	local := callableExpressionScope(expr, outer, ctx)
	sig := callableSignature{params: paramsFromNodesWithPHPDoc(nodes, nil, local.typeCtx, nil, nil)}
	for i, p := range sig.params {
		if p.Type == "" {
			sig.params[i].Type = "mixed"
		}
		if outer != nil && outer.className != "" {
			sig.params[i].Type = bindCalleeSignatureType(sig.params[i].Type, outer.className, outer.className, ctx).dnfString()
		}
		if p.IsByRef {
			return callableSignature{}, false
		}
	}
	sig.returnType = declaredCallableExpressionReturnType(expr, local.typeCtx)
	if outer != nil && outer.className != "" {
		sig.returnType = bindCalleeSignatureType(sig.returnType.dnfString(), outer.className, outer.className, ctx)
	}
	if !sig.returnType.IsEmpty() {
		return sig, true
	}
	switch n := expr.(type) {
	case *ast.ArrowFunctionNode:
		sig.returnType = inferReturnTypeWithFacts("", n.Expr, local, ctx)
	case *ast.FunctionNode:
		if functionContainsYield(n) || !simpleCallableBody(n.Body) {
			return sig, false
		}
		returns := collectObservedReturns("", n.Body, local, ctx)
		hasValue := false
		for _, r := range returns {
			if r.Expr != nil {
				hasValue = true
			}
		}
		if len(returns) > 0 && !hasValue {
			sig.returnType = ParseType("void")
			return sig, true
		}
		var values []Type
		for _, r := range returns {
			if r.Expr == nil {
				values = append(values, ParseType("null"))
			} else {
				values = append(values, r.Type)
			}
		}
		if len(values) == 0 && !statementsTerminate(n.Body) {
			sig.returnType = ParseType("void")
			return sig, true
		}
		if !statementsTerminate(n.Body) {
			values = append(values, ParseType("null"))
		}
		if len(values) == 0 {
			sig.returnType = ParseType("never")
		} else {
			sig.returnType = unionInferredTypes(values...)
		}
	}
	if sig.returnType.IsEmpty() || sig.returnType.hasBuiltin("mixed") {
		return sig, false
	}
	return sig, true
}

// Keep body inference conservative where the shared return collector does not
// model all exits. Nested declarations are not returns from this callable.
func simpleCallableBody(nodes []ast.Node) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt, *ast.AssignmentNode, *ast.ReturnNode, *ast.ThrowNode, *ast.FunctionNode:
		case *ast.IfNode:
			if !simpleCallableBody(n.Body) {
				return false
			}
			for _, branch := range n.ElseIfs {
				if !simpleCallableBody(branch.Body) {
					return false
				}
			}
			if n.Else != nil && !simpleCallableBody(n.Else.Body) {
				return false
			}
		case *ast.BlockNode:
			if !simpleCallableBody(n.Statements) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func callableContractMismatch(expected, actual callableSignature, scope *functionScope, ctx *AnalysisContext) bool {
	// Callback returns are covariant, including the declared void contract.
	if !phpDocTypeIsSubtype(actual.returnType.dnfString(), expected.returnType.dnfString(), FileTypeContext{}, ctx) {
		return true
	}
	for i, param := range actual.params {
		if i >= len(expected.params) {
			if !param.HasDefault && !param.IsVariadic {
				return true
			}
			continue
		}
		want := expected.params[i]
		if (want.HasDefault || want.IsVariadic) && !param.HasDefault && !param.IsVariadic {
			return true
		}
		if !phpDocTypeIsSubtype(want.Type, param.Type, FileTypeContext{}, ctx) {
			return true
		}
		if param.IsVariadic {
			for _, remaining := range expected.params[i+1:] {
				if !phpDocTypeIsSubtype(remaining.Type, param.Type, FileTypeContext{}, ctx) {
					return true
				}
			}
			break
		}
	}
	// Extra supplied arguments may be ignored by a callback with fewer params.
	return false
}

func callableContractDisplay(sig callableSignature, base string) string {
	parts := make([]string, len(sig.params))
	for i, p := range sig.params {
		parts[i] = p.Type
		if p.IsVariadic {
			parts[i] += "..."
		} else if p.HasDefault {
			parts[i] += "="
		}
	}
	return base + "(" + strings.Join(parts, ", ") + "): " + sig.returnType.dnfString()
}

func callableContractUsesOpenTemplates(raw string, method ResolvedMethod, scope *functionScope, ctx *AnalysisContext) bool {
	names := templateNames(method.TemplateParams)
	if scope != nil {
		for name := range scope.templateBounds {
			names = mergeTemplateNames(names, []string{name})
		}
	}
	if method.DeclaringClass != "" && ctx != nil && ctx.Resolver != nil {
		if class, ok := ctx.Resolver.ResolveClass(method.DeclaringClass); ok {
			names = mergeTemplateNames(names, class.TemplateParams)
		}
	}
	return phpDocUsesTemplate(raw, names)
}

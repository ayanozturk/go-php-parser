package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

func applyPHPUnitAssertionScope(scope *functionScope, expr ast.Node, ctx *AnalysisContext) {
	if scope == nil {
		return
	}
	method, args, ok := phpUnitAssertionCall(expr)
	if !ok {
		return
	}
	switch asciiLowerIdent(method) {
	case "assertinstanceof":
		applyAssertInstanceOfScope(scope, args)
	case "assertnotnull":
		if variable, ok := argumentValue(phpUnitArg(args, 0, "actual")).(*ast.VariableNode); ok {
			if typ, ok := nonNullVariableType(scope, variable.Name); ok {
				scope.setVariable(variable.Name, typ)
			} else {
				scope.setVariable(variable.Name, MixedType())
			}
		}
	case "asserttrue":
		if condition := argumentValue(phpUnitArg(args, 0, "condition")); condition != nil {
			applyConditionTrueScope(scope, condition, ctx)
		}
	}
}

func phpUnitAssertionCall(expr ast.Node) (string, []ast.Node, bool) {
	switch n := expr.(type) {
	case *ast.MethodCallNode:
		receiver, ok := n.Object.(*ast.VariableNode)
		if !ok || receiver.Name != "this" {
			return "", nil, false
		}
		return n.Method, n.Args, true
	case *ast.FunctionCallNode:
		identifier, ok := n.Name.(*ast.IdentifierNode)
		if !ok {
			return "", nil, false
		}
		if !strings.Contains(identifier.Value, "::") {
			return "", nil, false
		}
		return staticCallMethodName(identifier.Value), n.Args, true
	default:
		return "", nil, false
	}
}

func applyAssertInstanceOfScope(scope *functionScope, args []ast.Node) {
	expected := argumentValue(phpUnitArg(args, 0, "expected"))
	actual := argumentValue(phpUnitArg(args, 1, "actual"))
	variable, ok := actual.(*ast.VariableNode)
	if !ok {
		return
	}
	className := classNameFromAssertionType(expected, scope)
	if className == "" {
		return
	}
	scope.setVariable(variable.Name, ClassType(className))
}

func phpUnitArg(args []ast.Node, index int, names ...string) ast.Node {
	for _, arg := range args {
		named, ok := arg.(*ast.NamedArgumentNode)
		if !ok {
			continue
		}
		for _, name := range names {
			if strings.EqualFold(named.Name, name) {
				return named.Value
			}
		}
	}
	positional := 0
	for _, arg := range args {
		if _, ok := arg.(*ast.NamedArgumentNode); ok {
			continue
		}
		if positional == index {
			return arg
		}
		positional++
	}
	return nil
}

func classNameFromAssertionType(node ast.Node, scope *functionScope) string {
	switch n := node.(type) {
	case *ast.ClassConstFetchNode:
		if !strings.EqualFold(n.Const, "class") {
			return ""
		}
		return resolveAssertionClassName(n.Class, scope)
	case *ast.IdentifierNode:
		return resolveAssertionClassName(n.Value, scope)
	case *ast.StringLiteral:
		return resolveAssertionClassName(n.Value, scope)
	default:
		return ""
	}
}

func resolveAssertionClassName(name string, scope *functionScope) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if scope != nil {
		resolved := scope.typeCtx.resolveClassLike(name)
		if resolved != "" {
			return resolved
		}
	}
	return name
}

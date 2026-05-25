package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

func checkCallArguments(filename string, pos ast.Position, target, name string, args []ast.Node, method ResolvedMethod, issues *[]AnalysisIssue) {
	if method.Name == "" && len(method.Params) == 0 {
		return
	}
	actualCount, hasUnpacked := countCallArguments(args)
	if !hasUnpacked {
		required, max, variadic := parameterBounds(method.Params)
		if actualCount < required {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("%s invoked with %d %s, at least %d required.", target, actualCount, pluralizeParameters(actualCount), required)))
		} else if !variadic && actualCount > max {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("%s invoked with %d %s, at most %d allowed.", target, actualCount, pluralizeParameters(actualCount), max)))
		}
	}
	checkNamedArguments(filename, pos, name, args, method.Params, issues)
}

func checkNamedArguments(filename string, pos ast.Position, name string, args []ast.Node, params []ResolvedParam, issues *[]AnalysisIssue) {
	seenNamed := false
	seenUnpacked := false
	used := map[string]struct{}{}
	paramsByName := map[string]ResolvedParam{}
	var variadic bool
	for _, param := range params {
		paramsByName[param.Name] = param
		if param.IsVariadic {
			variadic = true
		}
	}
	for _, arg := range args {
		switch a := arg.(type) {
		case *ast.NamedArgumentNode:
			seenNamed = true
			if _, ok := paramsByName[a.Name]; !ok && !variadic {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, fmt.Sprintf("Unknown parameter $%s in call to %s.", a.Name, name)))
			}
			if _, exists := used[a.Name]; exists {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, fmt.Sprintf("Argument for parameter $%s has already been passed.", a.Name)))
			}
			used[a.Name] = struct{}{}
		case *ast.UnpackedArgumentNode:
			if seenNamed {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, "Named argument cannot be followed by an unpacked (...) argument."))
			}
			seenUnpacked = true
		default:
			if seenNamed {
				*issues = append(*issues, issue(filename, arg.GetPos(), level0InvocationCode, "Named argument cannot be followed by a positional argument."))
			}
			if seenUnpacked {
				*issues = append(*issues, issue(filename, arg.GetPos(), level0InvocationCode, "Unpacked argument (...) cannot be followed by a non-unpacked argument."))
			}
		}
	}
}

func parameterBounds(params []ResolvedParam) (int, int, bool) {
	required := 0
	max := 0
	variadic := false
	for _, param := range params {
		if param.IsVariadic {
			variadic = true
			continue
		}
		max++
		if !param.HasDefault {
			required++
		}
	}
	return required, max, variadic
}

func constructorFor(className string, ctx *AnalysisContext) ResolvedMethod {
	method, ok := ctx.Resolver.ResolveMethod(className, "__construct")
	if !ok {
		return ResolvedMethod{Name: "__construct"}
	}
	return method
}

func checkMethodVisibility(filename string, pos ast.Position, method ResolvedMethod, className string, currentClass *ast.ClassNode, static bool, issues *[]AnalysisIssue) {
	if method.Visibility == "private" && (currentClass == nil || !strings.EqualFold(currentClass.Name, className)) {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to private method %s() of class %s.", method.Name, className)))
	}
	if method.Visibility == "protected" && currentClass == nil {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to protected method %s() of class %s.", method.Name, className)))
	}
	if static && !method.IsStatic {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Static call to instance method %s::%s().", className, method.Name)))
	}
}

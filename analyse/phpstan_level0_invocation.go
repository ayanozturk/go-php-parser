package analyse

import (
	"fmt"
	"go-phpcs/ast"
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
	if ctx.Project != nil {
		for _, candidate := range ctx.Project.classLineage(className) {
			if method, ok := ctx.Resolver.ResolveMethod(candidate, "__construct"); ok {
				return method
			}
		}
	}
	method, ok := ctx.Resolver.ResolveMethod(className, "__construct")
	if !ok {
		return ResolvedMethod{Name: "__construct"}
	}
	return method
}

func checkMethodVisibility(filename string, pos ast.Position, method ResolvedMethod, className string, currentClass *ast.ClassNode, ft fileTypeContext, project *ProjectIndex, static bool, issues *[]AnalysisIssue) {
	declaringClass := method.DeclaringClass
	if declaringClass == "" {
		declaringClass = className
	}
	caller := callerClassName(currentClass, ft)
	switch method.Visibility {
	case "private":
		if caller == "" || indexKey(caller) != indexKey(declaringClass) {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to private method %s::%s().", declaringClass, method.Name)))
		}
	case "protected":
		if caller == "" || !isSubclassOf(project, caller, declaringClass) {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to protected method %s::%s().", declaringClass, method.Name)))
		}
	}
	if static && !method.IsStatic {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Static call to instance method %s::%s().", className, method.Name)))
	}
}

func checkInstanceStaticMethodCall(filename string, pos ast.Position, method ResolvedMethod, className string, issues *[]AnalysisIssue) {
	if method.IsStatic {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to static method %s::%s() on instance.", className, method.Name)))
	}
}

func checkConstructorAccess(filename string, pos ast.Position, targetClass string, currentClass *ast.ClassNode, ft fileTypeContext, project *ProjectIndex, constructor ResolvedMethod, issues *[]AnalysisIssue) {
	if constructor.Name != "__construct" || constructor.Visibility == "public" || constructor.Visibility == "" {
		return
	}
	caller := callerClassName(currentClass, ft)
	switch constructor.Visibility {
	case "private":
		if caller == "" || indexKey(caller) != indexKey(targetClass) {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Cannot instantiate class %s via private constructor.", targetClass)))
		}
	case "protected":
		if caller == "" || !isSubclassOf(project, caller, targetClass) {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Cannot instantiate class %s via protected constructor.", targetClass)))
		}
	}
}

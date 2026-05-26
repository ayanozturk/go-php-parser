package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

func (r *PHPStanLevel0Rule) checkSymbolsAndCalls(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	guards := collectReflectionGuards(nodes, fileCtx)
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft fileTypeContext) {
		switch n := node.(type) {
		case *ast.NewNode:
			className := resolveNewClassName(n, ft)
			if className == "" || isSpecialClassName(className) {
				return
			}
			resolved, ok := ctx.Resolver.ResolveClass(className)
			if !ok {
				if guards.hasClass(className) {
					return
				}
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Instantiated class %s not found.", className)))
				return
			}
			switch resolved.Kind {
			case "interface", "trait", "enum":
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Cannot instantiate %s %s.", resolved.Kind, resolved.Name)))
			}
			if resolved.Abstract {
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Instantiated class %s is abstract.", resolved.Name)))
			}
			constructor := constructorFor(resolved.Name, ctx)
			checkConstructorAccess(filename, n.GetPos(), resolved.Name, class, ft, ctx.Project, constructor, &issues)
			checkCallArguments(filename, n.GetPos(), "Class "+resolved.Name+" constructor", "__construct", n.Args, constructor, &issues)
		case *ast.FunctionCallNode:
			name := functionCallName(n)
			if name == "" {
				return
			}
			if className, methodName, ok := strings.Cut(name, "::"); ok {
				if strings.HasPrefix(className, "$") {
					return
				}
				resolvedClass := resolveClassLikeForCall(className, class, ft, ctx)
				if isSpecialClassName(resolvedClass) {
					return
				}
				if _, ok := ctx.Resolver.ResolveClass(resolvedClass); !ok {
					if guards.hasClass(resolvedClass) {
						return
					}
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to static method %s() on an unknown class %s.", methodName, resolvedClass)))
					return
				}
				method, ok := ctx.Resolver.ResolveMethod(resolvedClass, methodName)
				if !ok {
					if guards.hasMethod(resolvedClass, methodName) {
						return
					}
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to an undefined static method %s::%s().", resolvedClass, methodName)))
					return
				}
				checkMethodVisibility(filename, n.GetPos(), method, resolvedClass, class, ft, ctx.Project, true, &issues)
				checkCallArguments(filename, n.GetPos(), "Static method "+resolvedClass+"::"+method.Name+"()", method.Name, n.Args, method, &issues)
				return
			}
			resolvedName := resolveFunctionNameForCall(name, ft, ctx)
			if !ctx.Resolver.FunctionExists(resolvedName) {
				if guards.hasFunction(name) || guards.hasFunction(resolvedName) {
					return
				}
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Function %s not found.", name)))
				return
			}
			if fn, ok := ctx.Resolver.ResolveFunction(resolvedName); ok {
				checkCallArguments(filename, n.GetPos(), "Function "+fn.Name, fn.Name, n.Args, ResolvedMethod{Name: fn.Name, Params: fn.Params}, &issues)
			}
		case *ast.MethodCallNode:
			if receiver, ok := n.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
				if isStaticMethod(currentFn) {
					methodName := "method"
					if currentFn != nil && currentFn.Name != "" {
						methodName = currentFn.Name
					}
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Using $this inside static method %s::%s().", currentClassName(class, ft), methodName)))
					return
				}
				className := currentClassName(class, ft)
				if className == "" {
					return
				}
				method, ok := ctx.Resolver.ResolveMethod(className, n.Method)
				if !ok {
					if resolvedClass, classOK := ctx.Resolver.ResolveClass(className); classOK && resolvedClass.Kind == "trait" {
						return
					}
					if guards.hasMethod(className, n.Method) {
						return
					}
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to an undefined method %s::%s().", className, n.Method)))
					return
				}
				checkMethodVisibility(filename, n.GetPos(), method, className, class, ft, ctx.Project, false, &issues)
				checkCallArguments(filename, n.GetPos(), "Method "+className+"::"+method.Name+"()", method.Name, n.Args, method, &issues)
				return
			}
			className := methodCallClassName(n.Object, ft)
			if className == "" {
				return
			}
			method, ok := ctx.Resolver.ResolveMethod(className, n.Method)
			if !ok {
				return
			}
			checkMethodVisibility(filename, n.GetPos(), method, className, class, ft, ctx.Project, false, &issues)
			checkInstanceStaticMethodCall(filename, n.GetPos(), method, className, &issues)
			checkCallArguments(filename, n.GetPos(), "Method "+className+"::"+method.Name+"()", method.Name, n.Args, method, &issues)
		case *ast.ClassConstFetchNode:
			if strings.HasPrefix(n.Class, "$") {
				return
			}
			className := resolveClassLikeForCall(n.Class, class, ft, ctx)
			if isSpecialClassName(className) || strings.HasPrefix(className, "$") {
				return
			}
			resolvedClass, ok := ctx.Resolver.ResolveClass(className)
			if !ok {
				if n.Const == "class" {
					return
				}
				if guards.hasClass(className) || (n.Const == "class" && guards.hasClass(className)) {
					return
				}
				if n.Const != "class" && guards.hasConstant(className+"::"+n.Const) {
					return
				}
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to constant %s::%s on an unknown class %s.", className, n.Const, className)))
				return
			}
			if strings.HasPrefix(n.Const, "$") {
				propertyName := strings.TrimPrefix(n.Const, "$")
				property, ok := ctx.Resolver.ResolveProperty(className, propertyName)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to undefined static property %s::$%s.", className, propertyName)))
					return
				}
				if !property.IsStatic {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Static access to instance property %s::$%s.", resolvedClass.Name, property.Name)))
				}
				return
			}
			if n.Const != "class" {
				constantName := className + "::" + n.Const
				if constant, ok := resolveClassConstant(ctx.Project, className, n.Const); ok {
					checkConstantVisibility(filename, n.GetPos(), constant, className, class, ft, ctx.Project, &issues)
				} else if !ctx.Resolver.ConstantExists(constantName) {
					if guards.hasConstant(constantName) {
						return
					}
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to undefined constant %s::%s.", className, n.Const)))
				}
			}
		case *ast.PropertyFetchNode:
			receiver, ok := n.Object.(*ast.VariableNode)
			if !ok || receiver.Name != "this" {
				return
			}
			if isStaticMethod(currentFn) {
				methodName := "method"
				if currentFn != nil && currentFn.Name != "" {
					methodName = currentFn.Name
				}
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Using $this inside static method %s::%s().", currentClassName(class, ft), methodName)))
				return
			}
			className := currentClassName(class, ft)
			if className == "" {
				return
			}
			if _, ok := ctx.Resolver.ResolveProperty(className, n.Property); !ok {
				if resolvedClass, classOK := ctx.Resolver.ResolveClass(className); classOK && resolvedClass.Kind == "trait" {
					return
				}
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to an undefined property %s::$%s.", className, n.Property)))
			}
		}
	})
	return issues
}

func resolveClassConstant(project *ProjectIndex, className, constName string) (ResolvedConstant, bool) {
	if project == nil {
		return ResolvedConstant{}, false
	}
	return project.ResolveConstant(className, constName)
}

func checkConstantVisibility(filename string, pos ast.Position, constant ResolvedConstant, className string, currentClass *ast.ClassNode, ft fileTypeContext, project *ProjectIndex, issues *[]AnalysisIssue) {
	declaringClass := constant.DeclaringClass
	if declaringClass == "" {
		declaringClass = className
	}
	caller := callerClassName(currentClass, ft)
	switch constant.Visibility {
	case "private":
		if caller == "" || indexKey(caller) != indexKey(declaringClass) {
			*issues = append(*issues, issue(filename, pos, level0SymbolsCode, fmt.Sprintf("Access to private constant %s::%s.", declaringClass, constant.Name)))
		}
	case "protected":
		if caller == "" || !isSubclassOf(project, caller, declaringClass) {
			*issues = append(*issues, issue(filename, pos, level0SymbolsCode, fmt.Sprintf("Access to protected constant %s::%s.", declaringClass, constant.Name)))
		}
	}
}

func methodCallClassName(object ast.Node, ft fileTypeContext) string {
	switch receiver := object.(type) {
	case *ast.IdentifierNode:
		return ft.resolveClassLike(receiver.Value)
	case *ast.Identifier:
		return ft.resolveClassLike(receiver.Name)
	case *ast.ClassConstFetchNode:
		if receiver.Const == "class" {
			return ft.resolveClassLike(receiver.Class)
		}
	case *ast.NewNode:
		return resolveNewClassName(receiver, ft)
	}
	return ""
}

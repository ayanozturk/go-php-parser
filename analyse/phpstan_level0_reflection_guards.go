package analyse

import (
	"go-phpcs/ast"
	"strings"
)

type reflectionGuards struct {
	classes   map[string]struct{}
	functions map[string]struct{}
	constants map[string]struct{}
	methods   map[string]struct{}
}

func collectReflectionGuards(nodes []ast.Node, fileCtx fileTypeContext) reflectionGuards {
	guards := reflectionGuards{
		classes:   map[string]struct{}{},
		functions: map[string]struct{}{},
		constants: map[string]struct{}{},
		methods:   map[string]struct{}{},
	}
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		call, ok := node.(*ast.FunctionCallNode)
		if !ok {
			return
		}
		name := strings.ToLower(functionCallName(call))
		switch name {
		case "class_exists", "interface_exists", "trait_exists", "enum_exists":
			if len(call.Args) == 0 {
				return
			}
			if className, ok := classNameGuardValue(argumentValue(call.Args[0]), ft); ok {
				guards.classes[indexKey(className)] = struct{}{}
			}
		case "function_exists":
			if len(call.Args) == 0 {
				return
			}
			if functionName, ok := stringLiteralValue(argumentValue(call.Args[0])); ok {
				guards.functions[indexKey(functionName)] = struct{}{}
				guards.functions[indexKey(ft.resolveClassLike(functionName))] = struct{}{}
			}
		case "defined":
			if len(call.Args) == 0 {
				return
			}
			if constantName, ok := stringLiteralValue(argumentValue(call.Args[0])); ok {
				guards.constants[indexKey(constantName)] = struct{}{}
			}
		case "method_exists":
			if len(call.Args) < 2 {
				return
			}
			methodName, ok := stringLiteralValue(argumentValue(call.Args[1]))
			if !ok {
				return
			}
			if receiver, ok := methodGuardClass(argumentValue(call.Args[0]), class, ft); ok {
				guards.methods[methodKey(receiver, methodName)] = struct{}{}
			}
		}
	})
	return guards
}

func (guards reflectionGuards) hasClass(name string) bool {
	_, ok := guards.classes[indexKey(name)]
	return ok
}

func (guards reflectionGuards) hasFunction(name string) bool {
	_, ok := guards.functions[indexKey(name)]
	return ok
}

func (guards reflectionGuards) hasMethod(className, methodName string) bool {
	_, ok := guards.methods[methodKey(className, methodName)]
	return ok
}

func classNameGuardValue(node ast.Node, ft fileTypeContext) (string, bool) {
	if value, ok := stringLiteralValue(node); ok {
		return ft.resolveClassLike(value), true
	}
	if fetch, ok := node.(*ast.ClassConstFetchNode); ok && fetch.Const == "class" {
		return ft.resolveClassLike(fetch.Class), true
	}
	return "", false
}

func methodGuardClass(node ast.Node, current *ast.ClassNode, ft fileTypeContext) (string, bool) {
	if receiver, ok := node.(*ast.VariableNode); ok && receiver.Name == "this" {
		className := currentClassName(current, ft)
		return className, className != ""
	}
	if className, ok := classNameGuardValue(node, ft); ok {
		return className, true
	}
	return "", false
}

func methodKey(className, methodName string) string {
	return indexKey(className) + "::" + strings.ToLower(strings.TrimSpace(methodName))
}

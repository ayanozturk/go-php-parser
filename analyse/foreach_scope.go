package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

func applyForeachIterationTypes(scope *functionScope, node *ast.ForeachNode, ctx *AnalysisContext, filename string) {
	if scope == nil || node == nil {
		return
	}
	keyType, valueType := foreachIterableTypes(node.Expr, scope, ctx, filename)
	if name := foreachVariableName(node.ValueVar); name != "" && !valueType.IsEmpty() {
		scope.setVariable(name, valueType)
		if inst, ok := genericInstanceFromType(valueType); ok {
			scope.setGenericContext(name, inst)
		}
	}
	if name := foreachVariableName(node.KeyVar); name != "" && !keyType.IsEmpty() {
		scope.setVariable(name, keyType)
	}
}

func foreachVariableName(node ast.Node) string {
	variable, ok := node.(*ast.VariableNode)
	if !ok {
		return ""
	}
	return variable.Name
}

func foreachIterableTypes(expr ast.Node, scope *functionScope, ctx *AnalysisContext, filename string) (Type, Type) {
	if variable, ok := expr.(*ast.VariableNode); ok && scope != nil {
		if inst, found := scope.genericContext[variable.Name]; found {
			if keyType, valueType, ok := iterableTypesFromGeneric(inst); ok {
				return keyType, valueType
			}
		}
	}
	inferred := inferTypeWithFacts(filename, expr, scope, ctx)
	if keyType, valueType, ok := iterableTypesFromInferred(inferred); ok {
		return keyType, valueType
	}
	if inferred.hasBuiltin("array") || inferred.hasBuiltin("iterable") {
		return MixedType(), MixedType()
	}
	return EmptyType(), EmptyType()
}

func genericInstanceFromType(typ Type) (GenericInstance, bool) {
	if typ.IsEmpty() {
		return GenericInstance{}, false
	}
	for _, atom := range typ.atoms {
		if inst, ok := parseGenericTypeFromString(atom.display); ok {
			return inst, true
		}
	}
	return GenericInstance{}, false
}

func iterableTypesFromInferred(inferred Type) (Type, Type, bool) {
	if inst, ok := genericInstanceFromType(inferred); ok {
		if keyType, valueType, ok := iterableTypesFromGeneric(inst); ok {
			return keyType, valueType, true
		}
	}
	for _, atom := range inferred.atoms {
		if keyType, valueType, ok := iterableTypesFromArraySuffix(atom.display); ok {
			return keyType, valueType, true
		}
	}
	return EmptyType(), EmptyType(), false
}

func iterableTypesFromGeneric(inst GenericInstance) (Type, Type, bool) {
	if len(inst.TypeArguments) == 0 {
		return EmptyType(), EmptyType(), false
	}
	base := genericIterableBaseName(inst.ClassName)
	switch base {
	case "list", "non-empty-list":
		return ParseType("int"), ParseType(inst.TypeArguments[0]), true
	case "array", "non-empty-array", "associative-array", "iterable":
		if len(inst.TypeArguments) == 1 {
			return MixedType(), ParseType(inst.TypeArguments[0]), true
		}
		return ParseType(inst.TypeArguments[0]), ParseType(inst.TypeArguments[1]), true
	default:
		if len(inst.TypeArguments) == 1 {
			return MixedType(), ParseType(inst.TypeArguments[0]), true
		}
		return ParseType(inst.TypeArguments[0]), ParseType(inst.TypeArguments[1]), true
	}
}

func genericIterableBaseName(className string) string {
	name := strings.TrimSpace(className)
	if idx := strings.LastIndex(name, "\\"); idx >= 0 {
		name = name[idx+1:]
	}
	return asciiLowerIdent(name)
}

func iterableTypesFromArraySuffix(typeStr string) (Type, Type, bool) {
	typeStr = strings.TrimSpace(typeStr)
	if !strings.HasSuffix(typeStr, "[]") {
		return EmptyType(), EmptyType(), false
	}
	value := strings.TrimSpace(strings.TrimSuffix(typeStr, "[]"))
	if value == "" {
		return EmptyType(), EmptyType(), false
	}
	return ParseType("int"), ParseType(value), true
}

func joinLoopAssignedVariables(outer, loop *functionScope) {
	if outer == nil || loop == nil || loop.variables == nil || !loop.variablesOwned {
		return
	}
	delta := loop.variables
	join := func(name string, loopType Type) {
		if name == "" || loopType.IsEmpty() {
			return
		}
		if outerType, ok := outer.variable(name); ok {
			outer.setVariable(name, unionInferredTypes(outerType, loopType))
			return
		}
		outer.setVariable(name, loopType)
	}
	if delta.hasOne {
		join(delta.name, delta.typ)
	}
	for name, typ := range delta.values {
		join(name, typ)
	}
}

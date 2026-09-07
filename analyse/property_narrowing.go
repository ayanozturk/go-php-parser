package analyse

import "github.com/ayanozturk/go-php-parser/ast"

// applyThisPropertyConditionScope refines direct properties of the current
// receiver and of a named foreign variable. A property on another object must
// not share $this flow state, even when both objects have the same class.
func applyThisPropertyConditionScope(scope *functionScope, condition ast.Node, truth bool, ctx *AnalysisContext) {
	if scope == nil {
		return
	}
	switch n := condition.(type) {
	case *ast.UnaryExpr:
		if n.Operator == "!" {
			applyThisPropertyConditionScope(scope, n.Operand, !truth, ctx)
		}
	case *ast.BinaryExpr:
		switch n.Operator {
		case "&&", "and":
			if truth {
				applyThisPropertyConditionScope(scope, n.Left, true, ctx)
				applyThisPropertyConditionScope(scope, n.Right, true, ctx)
			}
		case "||", "or":
			if !truth {
				applyThisPropertyConditionScope(scope, n.Left, false, ctx)
				applyThisPropertyConditionScope(scope, n.Right, false, ctx)
			}
		case "==", "===", "!=", "!==":
			var property ast.Node
			if isNullLiteral(n.Left) {
				property = n.Right
			} else if isNullLiteral(n.Right) {
				property = n.Left
			}
			equal := n.Operator == "==" || n.Operator == "==="
			if name, ok := directThisPropertyName(property); ok {
				if truth != equal {
					removeNullFromThisProperty(scope, name)
				} else if n.Operator == "===" || n.Operator == "!==" {
					scope.setProperty(name, ParseType("null"))
				}
				return
			}
			fetch, ok := property.(*ast.PropertyFetchNode)
			if !ok {
				return
			}
			if truth != equal {
				stripNullFromForeignProperty(scope, fetch, ctx)
			} else if n.Operator == "===" || n.Operator == "!==" {
				if key, ok := foreignPropertyKey(fetch); ok {
					scope.setProperty(key, ParseType("null"))
				}
			}
		case "instanceof":
			if !truth {
				return
			}
			typ := typeFromInstanceofTarget(n.Right, scope)
			if typ.IsEmpty() {
				return
			}
			property, ok := n.Left.(*ast.PropertyFetchNode)
			if !ok {
				return
			}
			if name, ok := directThisPropertyName(property); ok {
				scope.setProperty(name, typ)
				return
			}
			if key, ok := foreignPropertyKey(property); ok {
				scope.setProperty(key, typ)
			}
		}
	case *ast.PropertyFetchNode:
		if truth {
			if name, ok := directThisPropertyName(n); ok {
				removeNullFromThisProperty(scope, name)
				return
			}
			stripNullFromForeignProperty(scope, n, ctx)
		}
	}
}

func directThisPropertyName(node ast.Node) (string, bool) {
	property, ok := node.(*ast.PropertyFetchNode)
	if !ok || property.Property == "" {
		return "", false
	}
	receiver, ok := property.Object.(*ast.VariableNode)
	return property.Property, ok && receiver.Name == "this"
}

func foreignPropertyKey(node *ast.PropertyFetchNode) (string, bool) {
	if node == nil || node.Property == "" {
		return "", false
	}
	receiver, ok := node.Object.(*ast.VariableNode)
	if !ok || receiver.Name == "" || receiver.Name == "this" {
		return "", false
	}
	return receiver.Name + "\x1f" + node.Property, true
}

func removeNullFromThisProperty(scope *functionScope, name string) {
	current, ok := scope.property(name)
	if !ok {
		current, ok = resolveSameClassPropertyType(scope, name)
	}
	if !ok || !current.hasBuiltin("null") {
		return
	}
	if refined := current.withoutBuiltin("null"); !refined.IsEmpty() {
		scope.setProperty(name, refined)
	}
}

func stripNullFromForeignProperty(scope *functionScope, property *ast.PropertyFetchNode, ctx *AnalysisContext) {
	key, ok := foreignPropertyKey(property)
	if !ok {
		return
	}
	current := inferPropertyFetchType(property, scope, ctx)
	if !current.hasBuiltin("null") {
		return
	}
	if refined := current.withoutBuiltin("null"); !refined.IsEmpty() {
		scope.setProperty(key, refined)
	}
}

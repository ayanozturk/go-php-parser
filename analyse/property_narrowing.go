package analyse

import "github.com/ayanozturk/go-php-parser/ast"

// applyThisPropertyConditionScope refines only direct properties of the current
// receiver. A property on another object must not share this flow state, even
// when both objects have the same class.
func applyThisPropertyConditionScope(scope *functionScope, condition ast.Node, truth bool) {
	if scope == nil {
		return
	}
	switch n := condition.(type) {
	case *ast.UnaryExpr:
		if n.Operator == "!" {
			applyThisPropertyConditionScope(scope, n.Operand, !truth)
		}
	case *ast.BinaryExpr:
		switch n.Operator {
		case "&&", "and":
			if truth {
				applyThisPropertyConditionScope(scope, n.Left, true)
				applyThisPropertyConditionScope(scope, n.Right, true)
			}
		case "||", "or":
			if !truth {
				applyThisPropertyConditionScope(scope, n.Left, false)
				applyThisPropertyConditionScope(scope, n.Right, false)
			}
		case "==", "===", "!=", "!==":
			var property ast.Node
			if isNullLiteral(n.Left) {
				property = n.Right
			} else if isNullLiteral(n.Right) {
				property = n.Left
			}
			name, ok := directThisPropertyName(property)
			if !ok {
				return
			}
			equal := n.Operator == "==" || n.Operator == "==="
			if truth != equal {
				removeNullFromThisProperty(scope, name)
			} else if n.Operator == "===" || n.Operator == "!==" {
				scope.setProperty(name, ParseType("null"))
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
			}
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

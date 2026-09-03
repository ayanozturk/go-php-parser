package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level0PropertyCallableTypeCode = "Level0.PropertyCallableType"

func appendPropertyCallableTypeIssue(filename string, node ast.Node, issues *[]AnalysisIssue) {
	var name, raw string
	switch n := node.(type) {
	case *ast.PropertyNode:
		name, raw = n.Name, n.TypeHint
	case *ast.ParamNode:
		if !n.IsPromoted {
			return
		}
		name, raw = n.Name, paramTypeName(n)
	default:
		return
	}
	if !typeContainsNativeCallable(raw) {
		return
	}
	*issues = append(*issues, issueSpan(filename, node, level0PropertyCallableTypeCode, fmt.Sprintf("Property $%s cannot have callable in its type declaration.", name)))
}

func typeContainsNativeCallable(raw string) bool {
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "?"))
	for _, part := range splitTopLevelTypes(raw, '|') {
		if asciiLowerIdent(strings.TrimSpace(part)) == "callable" {
			return true
		}
	}
	return false
}

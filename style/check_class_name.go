package style

import (
	"fmt"
	"go-phpcs/ast"
)

// StyleChecker interface is defined in style_checker.go
// ClassNameChecker implements StyleChecker

type ClassNameChecker struct{}

func (c *ClassNameChecker) Check(nodes []ast.Node) {
	for _, node := range nodes {
		if fn, ok := node.(*ast.FunctionNode); ok {
			if fn.Name != camelCase(fn.Name) {
				fmt.Printf("Function '%s' should be camelCase\n", fn.Name)
			}
		}
	}
}

// camelCase returns the camelCase version of the input name.
// If the name is already camelCase, it returns it unchanged.
func camelCase(name string) string {
	if name == "" {
		return name
	}
	// Remove underscores and capitalize following letter
	result := ""
	capitalizeNext := false
	for i, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if i == 0 {
			result += string(toLower(r))
			continue
		}
		if capitalizeNext {
			result += string(toUpper(r))
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	return result
}

func toLower(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func toUpper(r rune) rune {
	if 'a' <= r && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}



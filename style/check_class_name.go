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
		if cls, ok := node.(*ast.ClassNode); ok {
			if cls.Name != pascalCase(cls.Name) {
				fmt.Printf("Class '%s' should be PascalCase\n", cls.Name)
			}
		}
	}
}

// pascalCase returns the PascalCase version of the input name.
// If the name is already PascalCase, it returns it unchanged.
func pascalCase(name string) string {
	if name == "" {
		return name
	}
	result := ""
	capitalizeNext := true
	for _, r := range name {
		if r == '_' {
			capitalizeNext = true
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

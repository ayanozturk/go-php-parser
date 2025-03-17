package style

import (
	"fmt"

	"go-phpcs/ast"
)

func Check(nodes []ast.Node) {
	for _, node := range nodes {
		if fn, ok := node.(*ast.FunctionNode); ok {
			if fn.Name != camelCase(fn.Name) {
				fmt.Printf("Function '%s' should be camelCase\n", fn.Name)
			}
		}
	}
}

func camelCase(name string) string {
	// dummy stub - real logic needed
	return name
}

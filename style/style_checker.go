package style

import "go-phpcs/ast"

// StyleChecker defines the interface for all style checkers.
type StyleChecker interface {
	Check(nodes []ast.Node)
}

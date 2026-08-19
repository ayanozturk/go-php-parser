package style

import "github.com/ayanozturk/go-php-parser/ast"

// StyleChecker defines the interface for all style checkers.
type StyleChecker interface {
	Check(nodes []ast.Node, filename string)
}

// Package lower is the public facade for declaration-tier CST→AST lowering.
//
// Gaps: attributes, anonymous class, closures/arrows; unsupported stmt/expr
// kinds skip; method bodies stay empty in index mode. See README.md.
package lower

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// File lowers the red file root to classic []ast.Node. Implementation lives in
// package syntax to avoid an import cycle with ParseAST*.
func File(root *syntax.RedNode, file *syntax.File) []ast.Node {
	return syntax.LowerFile(root, file)
}

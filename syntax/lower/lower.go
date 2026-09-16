// Package lower is the public facade for declaration-tier CST→AST lowering.
//
// Gaps (MVP): no PHPDoc, attributes, property hooks, enum/trait/anonymous/
// closures, or full expr/stmt lowering; method bodies stay empty in index mode.
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

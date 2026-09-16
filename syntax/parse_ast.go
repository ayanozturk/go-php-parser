package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

// ParseASTForIndex parses with SkipFunctionBodies and lowers declarations to classic AST.
func ParseASTForIndex(src []byte) ([]ast.Node, []Diagnostic) {
	res := ParseForIndex(src)
	return lowerFromResult(res), res.Diagnostics
}

// ParseAST parses full bodies and lowers KindStatementList method/function
// bodies to classic statements/expressions. KindTokenList bodies stay empty.
func ParseAST(src []byte) ([]ast.Node, []Diagnostic) {
	res := Parse(src)
	return lowerFromResult(res), res.Diagnostics
}

// LowerAST lowers an existing syntax ParseResult to classic AST nodes.
// Share one ParseForIndex/Parse result with BindSyntaxResult / symbol extractors.
func LowerAST(res *ParseResult) []ast.Node {
	return lowerFromResult(res)
}

func lowerFromResult(res *ParseResult) []ast.Node {
	if res == nil || res.File == nil {
		return nil
	}
	return LowerFile(res.File.Root, res.File)
}

package syntax

import (
	"context"

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

// ParseASTForIndexWithContext is the cancellable declaration-tier AST adapter.
func ParseASTForIndexWithContext(ctx context.Context, src []byte) ([]ast.Node, []Diagnostic) {
	res := ParseWithContext(ctx, src, ParseOptions{SkipFunctionBodies: true})
	return lowerFromResult(res), res.Diagnostics
}

// ParseASTWithContext is the cancellable full-body AST adapter.
func ParseASTWithContext(ctx context.Context, src []byte) ([]ast.Node, []Diagnostic) {
	res := ParseWithContext(ctx, src, ParseOptions{})
	return lowerFromResult(res), res.Diagnostics
}

// ParseAndLower returns the ParseResult and lowered AST so callers can share
// one CST across lowering and CST-direct analysis.
func ParseAndLower(src []byte) (nodes []ast.Node, res *ParseResult) {
	res = Parse(src)
	return lowerFromResult(res), res
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

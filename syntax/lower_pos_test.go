package syntax

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func TestNodePosSkipsLeadingTrivia(t *testing.T) {
	src := []byte("<?php\nfunction value(): int\n{\n    return 1;\n    $x = 2; // @phpstan-ignore-line\n}\n")
	res := Parse(src)
	var fnTok *RedNode
	var varTok *RedNode
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if n.Green != nil && n.Green.IsToken() {
			tok, _ := n.Green.Token()
			if tok.Type == token.T_FUNCTION {
				fnTok = n
			}
			if tok.Type == token.T_VARIABLE && tok.Literal == "$x" {
				varTok = n
			}
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	if fnTok == nil || varTok == nil {
		t.Fatalf("tokens not found fn=%v var=%v", fnTok != nil, varTok != nil)
	}
	fp, _ := nodePos(res.File, fnTok)
	t.Logf("function tok span=%v contentStart=%d pos=%d:%d lines=%v", fnTok.Span(), contentStartOffset(fnTok), fp.Line, fp.Column, res.File.Lines)
	if fp.Line != 2 || fp.Column != 1 {
		t.Fatalf("function pos=%d:%d want 2:1", fp.Line, fp.Column)
	}
	vp, _ := nodePos(res.File, varTok)
	t.Logf("var tok span=%v contentStart=%d pos=%d:%d", varTok.Span(), contentStartOffset(varTok), vp.Line, vp.Column)
	if vp.Line != 5 {
		t.Fatalf("var pos=%d:%d want line 5", vp.Line, vp.Column)
	}

	nodes := LowerAST(res)
	fn := nodes[0].(*ast.FunctionNode)
	t.Logf("lowered func pos=%d:%d", fn.Pos.Line, fn.Pos.Column)
	if fn.Pos.Line != 2 {
		t.Fatalf("lowered function pos line=%d want 2", fn.Pos.Line)
	}
}

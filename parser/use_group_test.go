package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseGroupUse(t *testing.T) {
	src := "<?php\nuse Foo\\Bar\\{A, B as C};\n"
	p := New(lexer.NewFileBytes([]byte(src)), false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	var uses []*ast.UseNode
	var walk func([]ast.Node)
	walk = func(list []ast.Node) {
		for _, n := range list {
			switch x := n.(type) {
			case *ast.UseNode:
				uses = append(uses, x)
			case *ast.BlockNode:
				walk(x.Statements)
			}
		}
	}
	walk(nodes)
	if len(uses) != 2 {
		t.Fatalf("uses=%d %#v", len(uses), uses)
	}
	if uses[0].Path != `Foo\Bar\A` || uses[0].Alias != "A" {
		t.Fatalf("use0=%+v", uses[0])
	}
	if uses[1].Path != `Foo\Bar\B` || uses[1].Alias != "C" {
		t.Fatalf("use1=%+v", uses[1])
	}
}

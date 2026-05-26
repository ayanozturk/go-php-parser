package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func checkConstantNode(t *testing.T, node *ast.ConstantNode, wantName, wantValue, wantVisibility, wantType string) {
	t.Helper()
	if node == nil {
		t.Fatalf("parseConstant returned nil")
	}
	if node.Name != wantName {
		t.Errorf("expected name %s, got %s", wantName, node.Name)
	}
	if node.Value == nil || node.Value.TokenLiteral() != wantValue {
		t.Errorf("expected value %s, got %v", wantValue, node.Value)
	}
	if node.Visibility != wantVisibility {
		t.Errorf("expected visibility %s, got %s", wantVisibility, node.Visibility)
	}
	if node.Type != wantType {
		t.Errorf("expected type %s, got %s", wantType, node.Type)
	}
}

func parseConstantFromSource(src string, seekToken token.TokenType) *ast.ConstantNode {
	lex := lexer.New(src)
	p := New(lex, false)
	for p.tok.Type != seekToken && p.tok.Type != token.T_EOF {
		p.nextToken()
	}
	if p.tok.Type == token.T_EOF {
		return nil
	}
	return p.parseConstant()
}

func TestParseConstant(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		src := "<?php\nconst FOO = 123;"
		node := parseConstantFromSource(src, token.T_CONST)
		checkConstantNode(t, node, "FOO", "123", "", "")
	})

	t.Run("with visibility and type", func(t *testing.T) {
		src := "<?php\npublic const BAR: int = 42;"
		node := parseConstantFromSource(src, token.T_PUBLIC)
		checkConstantNode(t, node, "BAR", "42", "public", "int")
	})

	t.Run("with protected visibility", func(t *testing.T) {
		src := "<?php\nprotected const BAZ = 'hi';"
		node := parseConstantFromSource(src, token.T_PROTECTED)
		checkConstantNode(t, node, "BAZ", "hi", "protected", "")
	})

	t.Run("with array type before name", func(t *testing.T) {
		src := "<?php\nprivate const array NAMES = ['a'];"
		node := parseConstantFromSource(src, token.T_PRIVATE)
		checkConstantNode(t, node, "NAMES", "array", "private", "array")
	})
}

func TestParseClassConstantKeepsModifierVisibility(t *testing.T) {
	src := `<?php
class Demo {
    private const SECRET = 1;
    protected const TOKEN = 2;
    public const NAME = 'demo';
    final public const LOCKED = true;
}`
	lex := lexer.New(src)
	p := New(lex, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Constants) != 4 {
		t.Fatalf("expected 4 constants, got %d", len(classNode.Constants))
	}
	for i, want := range []string{"private", "protected", "public"} {
		constant, ok := classNode.Constants[i].(*ast.ConstantNode)
		if !ok {
			t.Fatalf("expected ConstantNode, got %T", classNode.Constants[i])
		}
		if constant.Visibility != want {
			t.Fatalf("constant %d visibility: got %q, want %q", i, constant.Visibility, want)
		}
	}
	constant, ok := classNode.Constants[3].(*ast.ConstantNode)
	if !ok {
		t.Fatalf("expected ConstantNode, got %T", classNode.Constants[3])
	}
	if constant.Visibility != "public" || len(constant.Modifiers) != 2 || constant.Modifiers[0] != "final" || constant.Modifiers[1] != "public" {
		t.Fatalf("final public constant modifiers not preserved: %#v", constant)
	}
}

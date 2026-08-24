package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseGlobalVariablesInFunction(t *testing.T) {
	input := `<?php
function foo() {
    global $wpdb, $post;
    echo $wpdb;
}`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed")
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected FunctionNode, got %T", nodes[0])
	}
	if len(fn.Body) < 1 {
		t.Fatalf("Expected function body statements, got %d", len(fn.Body))
	}
	decl, ok := fn.Body[0].(*ast.GlobalVarDeclNode)
	if !ok {
		t.Fatalf("Expected first statement to be GlobalVarDeclNode, got %T", fn.Body[0])
	}
	if len(decl.Vars) != 2 {
		t.Fatalf("Expected 2 global vars, got %d", len(decl.Vars))
	}
	if decl.Vars[0].Name != "wpdb" || decl.Vars[1].Name != "post" {
		t.Fatalf("Expected global var names wpdb,post got %q,%q", decl.Vars[0].Name, decl.Vars[1].Name)
	}
}

func TestParseGlobalVariableAtTopLevel(t *testing.T) {
	// WordPress-style admin template files declare `global` at the top
	// level (not inside a function) before switching to inline HTML.
	input := `<?php
global $hook_suffix;
?>
<div class="wrap"></div>
`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes (global decl + inline html), got %d: %+v", len(nodes), nodes)
	}
	if _, ok := nodes[0].(*ast.GlobalVarDeclNode); !ok {
		t.Fatalf("Expected first node to be GlobalVarDeclNode, got %T", nodes[0])
	}
	html, ok := nodes[1].(*ast.InlineHTMLNode)
	if !ok {
		t.Fatalf("Expected second node to be InlineHTMLNode, got %T", nodes[1])
	}
	if html.Value == "" {
		t.Fatalf("Expected non-empty inline HTML content")
	}
}

func TestParseInlineHTMLBetweenPHPBlocks(t *testing.T) {
	input := `<?php
echo "before";
?>
<div>middle</div>
<?php
echo "after";
`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	var htmlSeen bool
	for _, n := range nodes {
		if html, ok := n.(*ast.InlineHTMLNode); ok {
			htmlSeen = true
			if html.Value == "" {
				t.Fatalf("expected non-empty inline HTML")
			}
		}
	}
	if !htmlSeen {
		t.Fatalf("Expected an InlineHTMLNode among parsed nodes, got %+v", nodes)
	}
}

func TestParseShortEchoTag(t *testing.T) {
	input := `<?php $name = "World"; ?>
<p><?= $name ?></p>
`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed")
	}
}

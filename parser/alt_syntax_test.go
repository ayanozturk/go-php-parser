package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

// TestParseAlternativeIfSyntax covers the WordPress-style template pattern
// of if/elseif/else using ':' ... 'endif;' instead of braces, interleaved
// with inline HTML between PHP blocks.
func TestParseAlternativeIfSyntax(t *testing.T) {
	input := `<?php if ($x): ?>
<div>yes</div>
<?php elseif ($y): ?>
<div>maybe</div>
<?php else: ?>
<div>no</div>
<?php endif; ?>
<p><?= $name ?></p>`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed")
	}
	ifNode, ok := nodes[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Expected IfNode, got %T", nodes[0])
	}
	if len(ifNode.Body) != 1 {
		t.Fatalf("Expected 1 statement in if body, got %d", len(ifNode.Body))
	}
	if len(ifNode.ElseIfs) != 1 {
		t.Fatalf("Expected 1 elseif clause, got %d", len(ifNode.ElseIfs))
	}
	if ifNode.Else == nil {
		t.Fatal("Expected an else clause")
	}
}

func TestParseAlternativeForeachSyntax(t *testing.T) {
	input := `<?php foreach ($items as $key => $item): ?>
  <li><?= $key ?>: <?= $item ?></li>
<?php endforeach; ?>`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if _, ok := nodes[0].(*ast.ForeachNode); !ok {
		t.Fatalf("Expected ForeachNode, got %T", nodes[0])
	}
}

func TestParseAlternativeForSyntax(t *testing.T) {
	input := `<?php for ($i = 0; $i < 10; $i++): ?>
  <span><?= $i ?></span>
<?php endfor; ?>`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if _, ok := nodes[0].(*ast.BlockNode); !ok {
		t.Fatalf("Expected BlockNode, got %T", nodes[0])
	}
}

func TestParseAlternativeWhileSyntax(t *testing.T) {
	input := `<?php while ($row = next_row()): ?>
  <tr><?= $row ?></tr>
<?php endwhile; ?>`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if _, ok := nodes[0].(*ast.WhileNode); !ok {
		t.Fatalf("Expected WhileNode, got %T", nodes[0])
	}
}

func TestParseAlternativeSwitchSyntax(t *testing.T) {
	input := `<?php switch ($x):
  case 1:
    echo "one";
    break;
  case 2:
    echo "two";
    break;
  default:
    echo "other";
endswitch; ?>`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	sw, ok := nodes[0].(*ast.SwitchNode)
	if !ok {
		t.Fatalf("Expected SwitchNode, got %T", nodes[0])
	}
	if len(sw.Cases) != 3 {
		t.Fatalf("Expected 3 cases, got %d", len(sw.Cases))
	}
}

// TestParseDoWhileHasNoAlternativeSyntax ensures 'do...while' (which has no
// PHP alternative/colon syntax) is unaffected.
func TestParseDoWhileHasNoAlternativeSyntax(t *testing.T) {
	input := `<?php do { echo $x; } while ($x < 10);`
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

// TestParseCommentBeforeCloseTag ensures a single-line comment on the same
// line as "?>" does not swallow the closing tag (a real PHP lexer rule).
func TestParseCommentBeforeCloseTag(t *testing.T) {
	input := `<?php
function f() {
	?>
	<div>content</div>
	<?php // a comment ?>
<p>after</p>
<?php
}`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) < 1 {
		t.Fatalf("Expected at least 1 top-level node (function), got %d", len(nodes))
	}
}

// TestParseEchoCommaSeparatedArgs covers PHP's "echo $a, $b, $c;" form,
// distinct from 'print' which only accepts a single expression.
func TestParseEchoCommaSeparatedArgs(t *testing.T) {
	input := `<?php echo "PUT > ", $cmd, CRLF;`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	block, ok := nodes[0].(*ast.BlockNode)
	if !ok {
		t.Fatalf("Expected BlockNode wrapping multiple echo expressions, got %T", nodes[0])
	}
	if len(block.Statements) != 3 {
		t.Fatalf("Expected 3 echo expressions, got %d", len(block.Statements))
	}
}

// TestParseSingleEchoStillExpressionStmt ensures the common single-expression
// echo case keeps its original AST shape (a bare ExpressionStmt).
func TestParseSingleEchoStillExpressionStmt(t *testing.T) {
	input := `<?php echo $x;`
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if _, ok := nodes[0].(*ast.ExpressionStmt); !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[0])
	}
}

// TestParseNamespacedBooleanConstant covers "\false"/"\true"/"\null" — the
// fully-qualified (leading backslash) form of these builtin constants,
// commonly used in library code (e.g. SimplePie) to avoid relying on
// unqualified name resolution.
func TestParseNamespacedBooleanConstant(t *testing.T) {
	input := `<?php if (\false) { echo 1; }`
	l := lexer.New(input)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
}

// TestParseReservedWordPropertyAccess covers PHP's rule that (almost) all
// keywords, including visibility modifiers like 'public', are valid
// property/method names when accessed via '->'.
func TestParseReservedWordPropertyAccess(t *testing.T) {
	input := `<?php
class Foo {
	public $public;
	function bar() {
		return $this->public;
	}
}`
	l := lexer.New(input)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
}

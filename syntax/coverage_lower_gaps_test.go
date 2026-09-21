package syntax

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestLowerRemainingExprAndStmtHelpers(t *testing.T) {
	src := []byte(`<?php
global $g;
static $s = 1;
$$v = 1;
$f = strlen(...);
$h = <<<EOT
hi $name
EOT;
include_once 'a.php';
require 'b.php';
throw new \RuntimeException('x');
function outer($a) {
    if ($a) {
        echo 1;
    } elseif ($a > 1) {
        echo 2;
    } else {
        echo 3;
    }
    for ($i = 0; $i < 1; $i++) {
        echo $i;
    }
}
foo(bar: 1, ...$rest);
`)
	res := Parse(src)
	file := res.File
	nodes := LowerAST(res)
	if len(nodes) == 0 {
		t.Fatal("expected lowered nodes")
	}

	for _, kind := range []Kind{KindGlobalStmt, KindStaticVarStmt, KindIfStmt, KindForStmt} {
		n := firstNodeOfKind(file.Root, kind)
		if n == nil {
			t.Fatalf("missing kind %s", kind)
		}
		if LowerStmtNode(n, file) == nil {
			t.Fatalf("LowerStmtNode(%s) nil", kind)
		}
	}

	var vv *RedNode
	Walk(file.Root, func(n *RedNode) bool {
		if n.Kind() == KindVariableVariableExpr {
			vv = copyRed(n)
			return false
		}
		return true
	})
	if vv != nil && LowerExprNode(vv, file) == nil {
		t.Fatal("LowerExprNode variable-variable")
	}

	fcc := firstNodeOfKind(file.Root, KindFirstClassCallableExpr)
	if fcc != nil && LowerExprNode(fcc, file) == nil {
		t.Fatal("LowerExprNode first-class callable")
	}

	var heredocNode *RedNode
	Walk(file.Root, func(n *RedNode) bool {
		switch n.Kind() {
		case KindHeredoc, KindNowdoc:
			heredocNode = copyRed(n)
			return false
		}
		return true
	})
	if heredocNode != nil && LowerExprNode(heredocNode, file) == nil {
		t.Fatal("LowerExprNode heredoc")
	}

	throw := firstNodeOfKind(file.Root, KindThrowExpr)
	if throw != nil && LowerExprNode(throw, file) == nil {
		t.Fatal("LowerExprNode throw")
	}

	call := firstNodeOfKind(file.Root, KindCallExpr)
	if call != nil && LowerExprNode(call, file) == nil {
		t.Fatal("LowerExprNode call with named/unpacked args")
	}

	inc := firstNodeOfKind(file.Root, KindIncludeExpr)
	if inc != nil && LowerExprNode(inc, file) == nil {
		t.Fatal("LowerExprNode include")
	}

	_ = nodes
	_ = (*ast.FunctionNode)(nil)
}

func TestParseNameAndTypeFragments(t *testing.T) {
	nameSrc := []byte(`Foo\bar`)
	p := &Parser{
		tokens: lexer.LexAll(nameSrc),
		intern: NewInterner(nameSrc),
		src:    nameSrc,
	}
	if p.ParseName() == nil {
		t.Fatal("ParseName")
	}

	typeSrc := []byte(`?int|string`)
	p2 := &Parser{
		tokens: lexer.LexAll(typeSrc),
		intern: NewInterner(typeSrc),
		src:    typeSrc,
	}
	if p2.ParseType() == nil {
		t.Fatal("ParseType")
	}
}

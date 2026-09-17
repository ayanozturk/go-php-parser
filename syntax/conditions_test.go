package syntax

import (
	"strings"
	"testing"
)

func firstNodeOfKind(root *RedNode, k Kind) *RedNode {
	var found *RedNode
	Walk(root, func(n *RedNode) bool {
		if found != nil {
			return false
		}
		if n.Kind() == k {
			found = n
			return false
		}
		return true
	})
	return found
}

func TestIfConditionExtractsCondition(t *testing.T) {
	res := Parse([]byte("<?php\nif ($a > 1) {\n    echo 1;\n} elseif ($b) {\n    echo 2;\n}\n"))
	ifStmt := firstNodeOfKind(res.File.Root, KindIfStmt)
	if ifStmt == nil {
		t.Fatal("expected an IfStmt node")
	}
	cond := IfCondition(ifStmt)
	if cond == nil || strings.TrimSpace(cond.Text()) != "$a > 1" {
		t.Fatalf("expected condition %q, got %q", "$a > 1", cond.Text())
	}
	elseIf := firstNodeOfKind(res.File.Root, KindElseIfClause)
	if elseIf == nil {
		t.Fatal("expected an ElseIfClause node")
	}
	elseIfCond := IfCondition(elseIf)
	if elseIfCond == nil || strings.TrimSpace(elseIfCond.Text()) != "$b" {
		t.Fatalf("expected elseif condition %q, got %q", "$b", elseIfCond.Text())
	}
}

func TestWhileCondition(t *testing.T) {
	res := Parse([]byte("<?php\nwhile ($x < 10) {\n    $x++;\n}\n"))
	while := firstNodeOfKind(res.File.Root, KindWhileStmt)
	if while == nil {
		t.Fatal("expected a WhileStmt node")
	}
	cond := WhileCondition(while)
	if cond == nil || strings.TrimSpace(cond.Text()) != "$x < 10" {
		t.Fatalf("expected condition %q, got %q", "$x < 10", cond.Text())
	}
}

func TestDoWhileCondition(t *testing.T) {
	res := Parse([]byte("<?php\ndo {\n    $x++;\n} while ($x < 10);\n"))
	doWhile := firstNodeOfKind(res.File.Root, KindDoWhileStmt)
	if doWhile == nil {
		t.Fatal("expected a DoWhileStmt node")
	}
	cond := DoWhileCondition(doWhile)
	if cond == nil || strings.TrimSpace(cond.Text()) != "$x < 10" {
		t.Fatalf("expected condition %q, got %q", "$x < 10", cond.Text())
	}
}

func TestForConditions(t *testing.T) {
	res := Parse([]byte("<?php\nfor ($i = 0; $i < 10, $j > 0; $i++) {\n}\n"))
	forStmt := firstNodeOfKind(res.File.Root, KindForStmt)
	if forStmt == nil {
		t.Fatal("expected a ForStmt node")
	}
	conds := ForConditions(forStmt)
	if len(conds) != 2 {
		t.Fatalf("expected 2 for-conditions, got %d", len(conds))
	}
	if strings.TrimSpace(conds[0].Text()) != "$i < 10" {
		t.Fatalf("expected first condition %q, got %q", "$i < 10", conds[0].Text())
	}
	if strings.TrimSpace(conds[1].Text()) != "$j > 0" {
		t.Fatalf("expected second condition %q, got %q", "$j > 0", conds[1].Text())
	}
}

func TestForConditionsEmptyClause(t *testing.T) {
	res := Parse([]byte("<?php\nfor (;;) {\n}\n"))
	forStmt := firstNodeOfKind(res.File.Root, KindForStmt)
	if forStmt == nil {
		t.Fatal("expected a ForStmt node")
	}
	if conds := ForConditions(forStmt); len(conds) != 0 {
		t.Fatalf("expected no conditions for empty clause, got %d", len(conds))
	}
}

func TestMatchCondition(t *testing.T) {
	res := Parse([]byte("<?php\n$r = match ($x) {\n    1 => 'a',\n    default => 'b',\n};\n"))
	match := firstNodeOfKind(res.File.Root, KindMatchExpr)
	if match == nil {
		t.Fatal("expected a MatchExpr node")
	}
	cond := MatchCondition(match)
	if cond == nil || strings.TrimSpace(cond.Text()) != "$x" {
		t.Fatalf("expected condition %q, got %q", "$x", cond.Text())
	}
}

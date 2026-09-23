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
			found = &RedNode{File: n.File, Green: n.Green, Offset: n.Offset}
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

func TestMatchArmConditions(t *testing.T) {
	res := Parse([]byte("<?php\n$r = match ($x) {\n    1, 2 => 'a',\n    default => 'b',\n};\n"))
	match := firstNodeOfKind(res.File.Root, KindMatchExpr)
	if match == nil {
		t.Fatal("expected a MatchExpr node")
	}
	conds := MatchArmConditions(match)
	if len(conds) != 2 {
		t.Fatalf("expected 2 arm conditions (default excluded), got %d", len(conds))
	}
	if strings.TrimSpace(conds[0].Text()) != "1" || strings.TrimSpace(conds[1].Text()) != "2" {
		t.Fatalf("unexpected arm condition texts: %q, %q", conds[0].Text(), conds[1].Text())
	}
}

func TestIfBodyAndElseAccessors(t *testing.T) {
	res := Parse([]byte("<?php\nif ($a) {\n    echo 1;\n} elseif ($b) {\n    echo 2;\n} else {\n    echo 3;\n}\n"))
	ifStmt := firstNodeOfKind(res.File.Root, KindIfStmt)
	if ifStmt == nil {
		t.Fatal("expected an IfStmt node")
	}
	body := IfBody(ifStmt)
	if body == nil || len(StatementBodyList(body)) != 1 {
		t.Fatalf("expected 1 statement in if-body, got %v", body)
	}
	elseIfs := IfElseIfs(ifStmt)
	if len(elseIfs) != 1 {
		t.Fatalf("expected 1 elseif clause, got %d", len(elseIfs))
	}
	elseIfBody := ElseIfBody(elseIfs[0])
	if elseIfBody == nil || len(StatementBodyList(elseIfBody)) != 1 {
		t.Fatalf("expected 1 statement in elseif-body, got %v", elseIfBody)
	}
	els := IfElse(ifStmt)
	if els == nil {
		t.Fatal("expected an else clause")
	}
	elseBody := ElseBody(els)
	if elseBody == nil || len(StatementBodyList(elseBody)) != 1 {
		t.Fatalf("expected 1 statement in else-body, got %v", elseBody)
	}
}

func TestWhileDoWhileForBodyAccessors(t *testing.T) {
	res := Parse([]byte("<?php\nwhile ($a) {\n    echo 1;\n}\ndo {\n    echo 2;\n} while ($b);\nfor ($i = 0; $i < 10; $i++) {\n    echo 3;\n}\nfor (;;):\n    echo 4;\nendfor;\n"))
	while := firstNodeOfKind(res.File.Root, KindWhileStmt)
	if body := WhileBody(while); body == nil || len(StatementBodyList(body)) != 1 {
		t.Fatalf("expected 1 statement in while-body, got %v", body)
	}
	doWhile := firstNodeOfKind(res.File.Root, KindDoWhileStmt)
	if body := DoWhileBody(doWhile); body == nil || len(StatementBodyList(body)) != 1 {
		t.Fatalf("expected 1 statement in do-while-body, got %v", body)
	}
	forStmt := firstNodeOfKind(res.File.Root, KindForStmt)
	if body := ForBody(forStmt); body == nil || len(StatementBodyList(body)) != 1 {
		t.Fatalf("expected 1 statement in for-body, got %v", body)
	}
	var forCount int
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() == KindForStmt {
			forCount++
			if forCount == 2 {
				if body := ForBody(n); body == nil || len(StatementBodyList(body)) != 1 {
					t.Errorf("expected 1 statement in alternate for-body, got %v", body)
				}
			}
		}
		return true
	})
	if forCount != 2 {
		t.Fatalf("expected two for-statements, got %d", forCount)
	}
}

func TestFunctionBodyAndClassMethods(t *testing.T) {
	res := Parse([]byte("<?php\nclass Foo {\n    public function bar() {\n        echo 1;\n    }\n    public function baz();\n}\n"))
	class := firstNodeOfKind(res.File.Root, KindClassDecl)
	if class == nil {
		t.Fatal("expected a ClassDecl node")
	}
	methods := ClassMethods(class)
	if len(methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(methods))
	}
	if body := FunctionBody(methods[0]); body == nil || len(StatementBodyList(body)) != 1 {
		t.Fatalf("expected 1 statement in bar()'s body, got %v", body)
	}
	if body := FunctionBody(methods[1]); body != nil {
		t.Fatalf("expected nil body for abstract-like method baz(), got %v", body)
	}
}

func TestCallIsMethodLikeAndArgList(t *testing.T) {
	res := Parse([]byte("<?php\nfoo($a);\n$obj->bar($b);\n"))
	var calls []*RedNode
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() == KindCallExpr {
			calls = append(calls, &RedNode{File: n.File, Green: n.Green, Offset: n.Offset})
		}
		return true
	})
	if len(calls) != 2 {
		t.Fatalf("expected 2 call expressions, got %d", len(calls))
	}
	if CallIsMethodLike(calls[0]) {
		t.Fatalf("expected foo(...) to not be method-like")
	}
	if CallArgList(calls[0]) == nil {
		t.Fatal("expected an ArgList for foo(...)")
	}
	if !CallIsMethodLike(calls[1]) {
		t.Fatalf("expected $obj->bar(...) to be method-like")
	}
}

func TestExpressionStmtExpr(t *testing.T) {
	res := Parse([]byte("<?php\n$x = 1;\n"))
	stmt := firstNodeOfKind(res.File.Root, KindExpressionStmt)
	if stmt == nil {
		t.Fatal("expected an ExpressionStmt node")
	}
	expr := ExpressionStmtExpr(stmt)
	if expr == nil || expr.Kind() != KindAssignExpr {
		t.Fatalf("expected an AssignExpr, got %v", expr)
	}
	if ExpressionStmtExpr(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestNamespaceBodyBracedAndUnbraced(t *testing.T) {
	braced := Parse([]byte("<?php\nnamespace Foo {\n    class A {}\n    class B {}\n}\n"))
	ns := firstNodeOfKind(braced.File.Root, KindNamespaceDecl)
	if ns == nil {
		t.Fatal("expected a NamespaceDecl node")
	}
	body, consumed := NamespaceBody(ns, braced.File.Root.Children(), 0)
	if consumed != 0 {
		t.Fatalf("expected 0 consumed for braced form, got %d", consumed)
	}
	var classCount int
	for _, n := range body {
		if n.Kind() == KindClassDecl {
			classCount++
		}
	}
	if classCount != 2 {
		t.Fatalf("expected 2 classes in braced namespace body, got %d", classCount)
	}

	unbraced := Parse([]byte("<?php\nnamespace Foo;\n\nclass A {}\nclass B {}\n"))
	children := unbraced.File.Root.Children()
	idx := -1
	for i, c := range children {
		if c.Kind() == KindNamespaceDecl {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatal("expected a NamespaceDecl child")
	}
	ubBody, ubConsumed := NamespaceBody(children[idx], children, idx)
	// NamespaceBody absorbs every following sibling up to the next
	// NamespaceDecl (or end of file), including the trailing EOF token —
	// mirrors lowerNamespace's consumed count exactly, even though the
	// token itself lowers to nothing.
	if ubConsumed != len(children)-idx-1 {
		t.Fatalf("expected %d consumed siblings for unbraced form, got %d", len(children)-idx-1, ubConsumed)
	}
	var ubClassCount int
	for _, n := range ubBody {
		if n.Kind() == KindClassDecl {
			ubClassCount++
		}
	}
	if ubClassCount != 2 {
		t.Fatalf("expected 2 class decls in unbraced namespace body, got %+v", ubBody)
	}
}

func TestConditionAndBodyAccessorsRejectNilAndWrongKinds(t *testing.T) {
	res := Parse([]byte("<?php $value = 1;"))
	wrongKind := firstNodeOfKind(res.File.Root, KindLiteralExpr)
	if wrongKind == nil {
		t.Fatal("expected literal expression for wrong-kind controls")
	}

	checks := []struct {
		name string
		fn   func(*RedNode) bool
	}{
		{"if condition", func(n *RedNode) bool { return IfCondition(n) == nil }},
		{"while condition", func(n *RedNode) bool { return WhileCondition(n) == nil }},
		{"do while condition", func(n *RedNode) bool { return DoWhileCondition(n) == nil }},
		{"match condition", func(n *RedNode) bool { return MatchCondition(n) == nil }},
		{"for conditions", func(n *RedNode) bool { return len(ForConditions(n)) == 0 }},
		{"match arm conditions", func(n *RedNode) bool { return len(MatchArmConditions(n)) == 0 }},
		{"if body", func(n *RedNode) bool { return IfBody(n) == nil }},
		{"if else ifs", func(n *RedNode) bool { return len(IfElseIfs(n)) == 0 }},
		{"if else", func(n *RedNode) bool { return IfElse(n) == nil }},
		{"else if body", func(n *RedNode) bool { return ElseIfBody(n) == nil }},
		{"else body", func(n *RedNode) bool { return ElseBody(n) == nil }},
		{"while body", func(n *RedNode) bool { return WhileBody(n) == nil }},
		{"do while body", func(n *RedNode) bool { return DoWhileBody(n) == nil }},
		{"for body", func(n *RedNode) bool { return ForBody(n) == nil }},
		{"function body", func(n *RedNode) bool { return FunctionBody(n) == nil }},
		{"class methods", func(n *RedNode) bool { return len(ClassMethods(n)) == 0 }},
		{"method-like call", func(n *RedNode) bool { return !CallIsMethodLike(n) }},
		{"call arg list", func(n *RedNode) bool { return CallArgList(n) == nil }},
		{"call callee", func(n *RedNode) bool { return CallCallee(n) == nil }},
		{"call args", func(n *RedNode) bool { return len(CallArgs(n)) == 0 }},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for _, input := range []*RedNode{nil, wrongKind} {
				if !check.fn(input) {
					t.Fatalf("accessor accepted input of kind %v", func() any {
						if input == nil {
							return nil
						}
						return input.Kind()
					}())
				}
			}
		})
	}
	if StatementBodyList(nil) != nil {
		t.Fatal("StatementBodyList(nil) should be nil")
	}
	if body := StatementBodyList(wrongKind); len(body) != 1 || body[0].Kind() != KindLiteralExpr {
		t.Fatalf("single-node statement body should be preserved, got %v", body)
	}
}

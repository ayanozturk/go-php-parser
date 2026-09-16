package syntax_test

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

const bodyFixture = `<?php
class C {
  public function f($x) {
    $y = $x;
    if ($y === null) {
      return 0;
    }
    $this->save($y);
    return $y;
  }
}
`

const bodyWidenFixture = `<?php
class C {
  public function widen($x, $a, $b, $c) {
    $cfg = ['k' => 1];
    $v = $cfg['k'];
    $obj = new Foo($x);
    $arr = [1, 'a' => 2];
    $t = $a ? $b : $c;
    $s = $a ?: $c;
    foreach ($arr as $item) {
      echo $item;
    }
    while ($x) {
      break;
      continue;
      echo $x;
    }
    for ($i = 0; $i < 10; $i++) {
      echo $i;
    }
    do {
      echo $x;
    } while ($x);
    try {
      throw new Exception('err');
    } catch (Exception $e) {
      echo $e;
    } finally {
      echo 'done';
    }
    switch ($a) {
      case 1:
        echo 1;
        break;
      default:
        echo 0;
    }
    self::bar($x);
  }
}
`

const bodyAdvFixture = `<?php
class C {
  public function adv($v, $o, $z, $name, $b) {
    make()->x();
    $f = static function($a) use ($b) { return $a; };
    $g = fn($x): int => $x;
    $m = match ($v) { 1 => 'a', default => 'b' };
    (clone $o)->m();
    $o = new class extends Base { public function t() { return 1; } };
    $u = (unset) $z;
    $s = "hi $name";
  }
}
`

func TestParseASTBodyLowersMethodStatements(t *testing.T) {
	nodes, diags := syntax.ParseAST([]byte(bodyFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	cls, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(cls.Methods) != 1 {
		t.Fatalf("methods=%d", len(cls.Methods))
	}
	fn := cls.Methods[0].(*ast.FunctionNode)
	if len(fn.Body) == 0 {
		t.Fatal("expected non-empty method Body from KindStatementList")
	}

	wantChain := []string{
		"ExpressionStmt",
		"If",
		"ExpressionStmt",
		"Return",
	}
	if len(fn.Body) != len(wantChain) {
		t.Fatalf("body len=%d want %d; types=%v", len(fn.Body), len(wantChain), nodeTypes(fn.Body))
	}
	for i, want := range wantChain {
		if got := fn.Body[i].NodeType(); got != want {
			t.Fatalf("body[%d] NodeType=%q want %q (all=%v)", i, got, want, nodeTypes(fn.Body))
		}
	}

	assignStmt := fn.Body[0].(*ast.ExpressionStmt)
	assign, ok := assignStmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("first stmt expr=%T want AssignmentNode", assignStmt.Expr)
	}
	if assign.Operator != "=" {
		t.Fatalf("assign op=%q", assign.Operator)
	}
	if _, ok := assign.Left.(*ast.VariableNode); !ok {
		t.Fatalf("assign left=%T", assign.Left)
	}
	if _, ok := assign.Right.(*ast.VariableNode); !ok {
		t.Fatalf("assign right=%T", assign.Right)
	}

	iff := fn.Body[1].(*ast.IfNode)
	bin, ok := iff.Condition.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("if cond=%T", iff.Condition)
	}
	if bin.Operator != "===" {
		t.Fatalf("if op=%q", bin.Operator)
	}
	if len(iff.Body) != 1 {
		t.Fatalf("if body len=%d", len(iff.Body))
	}
	if _, ok := iff.Body[0].(*ast.ReturnNode); !ok {
		t.Fatalf("if body[0]=%T", iff.Body[0])
	}

	callStmt := fn.Body[2].(*ast.ExpressionStmt)
	mc, ok := callStmt.Expr.(*ast.MethodCallNode)
	if !ok {
		t.Fatalf("call expr=%T want MethodCallNode", callStmt.Expr)
	}
	if mc.Method != "save" {
		t.Fatalf("method=%q", mc.Method)
	}
	if len(mc.Args) != 1 {
		t.Fatalf("args=%d", len(mc.Args))
	}

	ret := fn.Body[3].(*ast.ReturnNode)
	if _, ok := ret.Expr.(*ast.VariableNode); !ok {
		t.Fatalf("return expr=%T", ret.Expr)
	}
}

func TestParseASTBodyBehaviouralNoPanic(t *testing.T) {
	nodesS, diags := syntax.ParseAST([]byte(bodyFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	parsedS := map[string][]ast.Node{"f.php": nodesS}

	idxS := analyse.BuildProjectIndex(parsedS)
	if idxS == nil {
		t.Fatal("BuildProjectIndex returned nil")
	}
	if _, ok := idxS.ResolveClass("C"); !ok {
		t.Fatal("ResolveClass C failed")
	}

	snapS, err := analyse.NewSemanticSnapshot(parsedS, nil)
	if err != nil {
		t.Fatalf("NewSemanticSnapshot: %v", err)
	}
	if snapS == nil {
		t.Fatal("nil snapshot")
	}

	_ = analyse.RunAnalysisRules("f.php", nodesS)
}

func TestParseASTBodyWidenStructural(t *testing.T) {
	nodes, diags := syntax.ParseAST([]byte(bodyWidenFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	cls, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(cls.Methods) != 1 {
		t.Fatalf("methods=%d", len(cls.Methods))
	}
	fn := cls.Methods[0].(*ast.FunctionNode)
	if fn.Name != "widen" {
		t.Fatalf("method=%q want widen", fn.Name)
	}
	if len(fn.Body) == 0 {
		t.Fatal("expected non-empty method Body from KindStatementList")
	}

	wantTypes := []string{
		"ArrayAccess",
		"New",
		"Array",
		"TernaryExpr",
		"Foreach",
		"While",
		"For",
		"DoWhile",
		"Throw",
		"Break",
		"Continue",
		"Try",
		"Switch",
	}
	counts := subtreeNodeTypeCounts(fn.Body)
	for _, want := range wantTypes {
		if counts[want] == 0 {
			t.Fatalf("missing NodeType %q in method body subtree (counts=%v)", want, counts)
		}
	}
	if !bodyHasStaticMemberCall(fn.Body) {
		t.Fatal("expected static member call lowered to FunctionCall with :: in Name")
	}
}

func TestParseASTBodyWidenBehavioural(t *testing.T) {
	nodesS, diags := syntax.ParseAST([]byte(bodyWidenFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	parsedS := map[string][]ast.Node{"f.php": nodesS}

	idxS := analyse.BuildProjectIndex(parsedS)
	if idxS == nil {
		t.Fatal("BuildProjectIndex returned nil")
	}
	if _, ok := idxS.ResolveClass("C"); !ok {
		t.Fatal("ResolveClass C failed")
	}

	snapS, err := analyse.NewSemanticSnapshot(parsedS, nil)
	if err != nil {
		t.Fatalf("NewSemanticSnapshot: %v", err)
	}
	if snapS == nil {
		t.Fatal("nil snapshot")
	}

	_ = analyse.RunAnalysisRules("f.php", nodesS)
}

func TestParseASTBodyAdvStructural(t *testing.T) {
	nodes, diags := syntax.ParseAST([]byte(bodyAdvFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	cls, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(cls.Methods) != 1 {
		t.Fatalf("methods=%d", len(cls.Methods))
	}
	fn := cls.Methods[0].(*ast.FunctionNode)
	if fn.Name != "adv" {
		t.Fatalf("method=%q want adv", fn.Name)
	}
	if len(fn.Body) == 0 {
		t.Fatal("expected non-empty method Body from KindStatementList")
	}

	exprStmt, ok := fn.Body[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("first stmt=%T want ExpressionStmt (make()->x())", fn.Body[0])
	}
	if _, ok := exprStmt.Expr.(*ast.MethodCallNode); !ok {
		t.Fatalf("first stmt expr=%T want MethodCallNode", exprStmt.Expr)
	}

	wantTypes := []string{
		"Function",
		"ArrowFunction",
		"Match",
		"UnaryExpr",
		"New",
		"Class",
		"TypeCast",
		"InterpolatedString",
	}
	counts := subtreeNodeTypeCounts(fn.Body)
	for _, want := range wantTypes {
		if counts[want] == 0 {
			t.Fatalf("missing NodeType %q in method body subtree (counts=%v)", want, counts)
		}
	}
}

func TestParseASTBodyAdvBehavioural(t *testing.T) {
	nodesS, diags := syntax.ParseAST([]byte(bodyAdvFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	parsedS := map[string][]ast.Node{"f.php": nodesS}

	idxS := analyse.BuildProjectIndex(parsedS)
	if idxS == nil {
		t.Fatal("BuildProjectIndex returned nil")
	}
	if _, ok := idxS.ResolveClass("C"); !ok {
		t.Fatal("ResolveClass C failed")
	}

	snapS, err := analyse.NewSemanticSnapshot(parsedS, nil)
	if err != nil {
		t.Fatalf("NewSemanticSnapshot: %v", err)
	}
	if snapS == nil {
		t.Fatal("nil snapshot")
	}

	_ = analyse.RunAnalysisRules("f.php", nodesS)
}

func TestParseASTForIndexStillEmptyBodies(t *testing.T) {
	nodes, diags := syntax.ParseASTForIndex([]byte(bodyFixture))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	cls := nodes[0].(*ast.ClassNode)
	fn := cls.Methods[0].(*ast.FunctionNode)
	if len(fn.Body) != 0 {
		t.Fatalf("ParseASTForIndex Body should stay empty, got %d stmts: %v", len(fn.Body), nodeTypes(fn.Body))
	}
}

func subtreeNodeTypeCounts(body []ast.Node) map[string]int {
	counts := make(map[string]int)
	for _, n := range body {
		countSubtreeNodeTypes(n, counts)
	}
	return counts
}

func countSubtreeNodeTypes(n ast.Node, counts map[string]int) {
	if n == nil {
		return
	}
	counts[n.NodeType()]++
	switch x := n.(type) {
	case *ast.ExpressionStmt:
		countSubtreeNodeTypes(x.Expr, counts)
	case *ast.AssignmentNode:
		countSubtreeNodeTypes(x.Left, counts)
		countSubtreeNodeTypes(x.Right, counts)
	case *ast.ReturnNode:
		countSubtreeNodeTypes(x.Expr, counts)
	case *ast.IfNode:
		countSubtreeNodeTypes(x.Condition, counts)
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
		for _, ei := range x.ElseIfs {
			countSubtreeNodeTypes(ei.Condition, counts)
			for _, s := range ei.Body {
				countSubtreeNodeTypes(s, counts)
			}
		}
		if x.Else != nil {
			for _, s := range x.Else.Body {
				countSubtreeNodeTypes(s, counts)
			}
		}
	case *ast.WhileNode:
		countSubtreeNodeTypes(x.Condition, counts)
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
	case *ast.DoWhileNode:
		countSubtreeNodeTypes(x.Condition, counts)
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
	case *ast.ForNode:
		for _, s := range x.Init {
			countSubtreeNodeTypes(s, counts)
		}
		for _, s := range x.Conditions {
			countSubtreeNodeTypes(s, counts)
		}
		for _, s := range x.Updates {
			countSubtreeNodeTypes(s, counts)
		}
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
	case *ast.ForeachNode:
		countSubtreeNodeTypes(x.Expr, counts)
		countSubtreeNodeTypes(x.KeyVar, counts)
		countSubtreeNodeTypes(x.ValueVar, counts)
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
	case *ast.ThrowNode:
		countSubtreeNodeTypes(x.Expr, counts)
	case *ast.TryNode:
		for _, s := range x.Body {
			countSubtreeNodeTypes(s, counts)
		}
		for _, c := range x.Catches {
			for _, s := range c.Body {
				countSubtreeNodeTypes(s, counts)
			}
		}
		for _, s := range x.Finally {
			countSubtreeNodeTypes(s, counts)
		}
	case *ast.SwitchNode:
		countSubtreeNodeTypes(x.Expr, counts)
		for _, c := range x.Cases {
			for _, s := range c.Body {
				countSubtreeNodeTypes(s, counts)
			}
		}
	case *ast.ArrayAccessNode:
		countSubtreeNodeTypes(x.Var, counts)
		countSubtreeNodeTypes(x.Index, counts)
	case *ast.NewNode:
		countSubtreeNodeTypes(x.ClassExpr, counts)
		for _, a := range x.Args {
			countSubtreeNodeTypes(a, counts)
		}
	case *ast.ArrayNode:
		for _, e := range x.Elements {
			countSubtreeNodeTypes(e, counts)
		}
	case *ast.ArrayItemNode:
		countSubtreeNodeTypes(x.Value, counts)
	case *ast.KeyValueNode:
		countSubtreeNodeTypes(x.Key, counts)
		countSubtreeNodeTypes(x.Value, counts)
	case *ast.TernaryExpr:
		countSubtreeNodeTypes(x.Condition, counts)
		countSubtreeNodeTypes(x.IfTrue, counts)
		countSubtreeNodeTypes(x.IfFalse, counts)
	case *ast.BinaryExpr:
		countSubtreeNodeTypes(x.Left, counts)
		countSubtreeNodeTypes(x.Right, counts)
	case *ast.UnaryExpr:
		countSubtreeNodeTypes(x.Operand, counts)
	case *ast.FunctionCallNode:
		countSubtreeNodeTypes(x.Name, counts)
		for _, a := range x.Args {
			countSubtreeNodeTypes(a, counts)
		}
	case *ast.MethodCallNode:
		countSubtreeNodeTypes(x.Object, counts)
		for _, a := range x.Args {
			countSubtreeNodeTypes(a, counts)
		}
	case *ast.BlockNode:
		for _, s := range x.Statements {
			countSubtreeNodeTypes(s, counts)
		}
	}
}

func bodyHasStaticMemberCall(body []ast.Node) bool {
	for _, n := range body {
		if staticMemberCallInSubtree(n) {
			return true
		}
	}
	return false
}

func staticMemberCallInSubtree(n ast.Node) bool {
	if n == nil {
		return false
	}
	switch x := n.(type) {
	case *ast.FunctionCallNode:
		if id, ok := x.Name.(*ast.IdentifierNode); ok && strings.Contains(id.Value, "::") {
			return true
		}
	case *ast.ExpressionStmt:
		return staticMemberCallInSubtree(x.Expr)
	case *ast.AssignmentNode:
		return staticMemberCallInSubtree(x.Right)
	case *ast.WhileNode:
		for _, s := range x.Body {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
	case *ast.DoWhileNode:
		for _, s := range x.Body {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
	case *ast.ForeachNode:
		for _, s := range x.Body {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
	case *ast.ForNode:
		for _, s := range x.Body {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
	case *ast.TryNode:
		for _, s := range x.Body {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
		for _, c := range x.Catches {
			for _, s := range c.Body {
				if staticMemberCallInSubtree(s) {
					return true
				}
			}
		}
		for _, s := range x.Finally {
			if staticMemberCallInSubtree(s) {
				return true
			}
		}
	case *ast.SwitchNode:
		for _, c := range x.Cases {
			for _, s := range c.Body {
				if staticMemberCallInSubtree(s) {
					return true
				}
			}
		}
	}
	return false
}

func TestParseASTUnsetAndDeclareStatements(t *testing.T) {
	src := `<?php
unset($x);
declare(strict_types=1);
`
	nodes, diags := syntax.ParseAST([]byte(src))
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 top-level nodes, got %d: %v", len(nodes), nodeTypes(nodes))
	}

	unsetStmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("nodes[0]=%T want ExpressionStmt", nodes[0])
	}
	unsetCall, ok := unsetStmt.Expr.(*ast.FunctionCallNode)
	if !ok {
		t.Fatalf("unset expr=%T want FunctionCallNode", unsetStmt.Expr)
	}
	unsetName, ok := unsetCall.Name.(*ast.IdentifierNode)
	if !ok || unsetName.Value != "unset" {
		t.Fatalf("unset name=%T(%v) want IdentifierNode(unset)", unsetCall.Name, unsetCall.Name)
	}
	if len(unsetCall.Args) != 1 {
		t.Fatalf("unset args=%d want 1", len(unsetCall.Args))
	}
	if _, ok := unsetCall.Args[0].(*ast.VariableNode); !ok {
		t.Fatalf("unset arg=%T want VariableNode", unsetCall.Args[0])
	}

	decl, ok := nodes[1].(*ast.DeclareNode)
	if !ok {
		t.Fatalf("nodes[1]=%T want DeclareNode", nodes[1])
	}
	if decl.NodeType() != "Declare" {
		t.Fatalf("declare NodeType=%q", decl.NodeType())
	}
	val, ok := decl.Directives["strict_types"]
	if !ok {
		t.Fatalf("directives=%v want strict_types", decl.Directives)
	}
	switch lit := val.(type) {
	case *ast.IntegerNode:
		if lit.Value != 1 {
			t.Fatalf("strict_types=%d want 1", lit.Value)
		}
	case *ast.IntegerLiteral:
		if lit.Value != 1 {
			t.Fatalf("strict_types=%d want 1", lit.Value)
		}
	default:
		t.Fatalf("strict_types=%T want integer literal", val)
	}
	if decl.Body != nil {
		t.Fatalf("declare body=%T want nil for semicolon form", decl.Body)
	}
}

func nodeTypes(nodes []ast.Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		if n == nil {
			out[i] = "<nil>"
			continue
		}
		out[i] = n.NodeType()
	}
	return out
}

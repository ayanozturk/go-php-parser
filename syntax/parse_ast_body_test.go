package syntax_test

import (
	"fmt"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
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

	// Classic vs syntax structural smoke: same body NodeType sequence.
	classic := parser.New(lexer.New(bodyFixture), false)
	nodesC := classic.Parse()
	clsC := nodesC[0].(*ast.ClassNode)
	fnC := clsC.Methods[0].(*ast.FunctionNode)
	if got, want := nodeTypes(fn.Body), nodeTypes(fnC.Body); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("body NodeTypes syntax=%v classic=%v", got, want)
	}
}

func TestParseASTBodyBehaviouralNoPanic(t *testing.T) {
	classic := parser.New(lexer.New(bodyFixture), false)
	nodesC := classic.Parse()
	nodesS, _ := syntax.ParseAST([]byte(bodyFixture))

	parsedC := map[string][]ast.Node{"f.php": nodesC}
	parsedS := map[string][]ast.Node{"f.php": nodesS}

	idxC := analyse.BuildProjectIndex(parsedC)
	idxS := analyse.BuildProjectIndex(parsedS)
	if idxC == nil || idxS == nil {
		t.Fatal("BuildProjectIndex returned nil")
	}

	snapC, err := analyse.NewSemanticSnapshot(parsedC, nil)
	if err != nil {
		t.Fatalf("classic NewSemanticSnapshot: %v", err)
	}
	snapS, err := analyse.NewSemanticSnapshot(parsedS, nil)
	if err != nil {
		t.Fatalf("syntax NewSemanticSnapshot: %v", err)
	}
	if snapC == nil || snapS == nil {
		t.Fatal("nil snapshot")
	}

	issuesC := analyse.RunAnalysisRules("f.php", nodesC)
	issuesS := analyse.RunAnalysisRules("f.php", nodesS)
	codesC := issueCodes(issuesC)
	codesS := issueCodes(issuesS)
	if fmt.Sprint(codesC) != fmt.Sprint(codesS) {
		t.Fatalf("issue codes classic=%v syntax=%v", codesC, codesS)
	}
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

func issueCodes(issues []analyse.AnalysisIssue) []string {
	out := make([]string, len(issues))
	for i, iss := range issues {
		out[i] = iss.Code
	}
	return out
}

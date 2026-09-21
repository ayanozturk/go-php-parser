package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestSymbolCallCSTMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"unresolvedFunctionCall": `<?php
missingFunction();
`,
		"classicStaticCall": `<?php
class Real {
    public static function bar(): void {}
}
function run(): void {
    Real::bar();
    Real::missing();
    Missing::bar();
}
`,
		"thisMethodCall": `<?php
class Real {
    public function bar(): void {}
    public function run(): void {
        $this->bar();
        $this->missing();
    }
    public static function staticRun(): void {
        $this->bar();
    }
}
`,
		"classConstFetch": `<?php
class Real {
    const FIELD = 1;
}
function run(): void {
    echo Real::FIELD;
    echo Real::MISSING;
    echo Missing::FIELD;
}
`,
		"namedArgUnknown": `<?php
function takes(int $a): void {}
takes(b: 1);
`,
		"cleanKnown": `<?php
function known(): void {}
known();
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, guards, _ := buildTypeRefTestContext(t, filename, src)
			content := []byte(src)

			EnableSyntaxLowerMemoStats(false)
			cstIssues := sortIssuesForCompare(CheckSymbolIssuesFromCST(filename, content, ctx, guards))

			// Force Lower* path: rebuild ctx and bypass CST by comparing against
			// a walk that always LowerExprNode's KindCallExpr.
			ctx2, _, guards2, _ := buildTypeRefTestContext(t, filename, src)
			lowerIssues := sortIssuesForCompare(symbolCallIssuesViaFullLower(filename, content, ctx2, guards2))

			if len(cstIssues) != len(lowerIssues) {
				t.Fatalf("count mismatch cst=%d lower=%d\ncst=%+v\nlower=%+v",
					len(cstIssues), len(lowerIssues), cstIssues, lowerIssues)
			}
			for i := range cstIssues {
				if cstIssues[i].Line != lowerIssues[i].Line ||
					cstIssues[i].Column != lowerIssues[i].Column ||
					cstIssues[i].Message != lowerIssues[i].Message ||
					cstIssues[i].Code != lowerIssues[i].Code {
					t.Fatalf("issue %d mismatch:\ncst=%+v\nlower=%+v", i, cstIssues[i], lowerIssues[i])
				}
			}
		})
	}
}

func symbolCallIssuesViaFullLower(filename string, content []byte, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	res := sharedParseResult(ctx, content)
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	type redNodeKey struct {
		green  *syntax.GreenNode
		offset int
	}
	keyOf := func(n *syntax.RedNode) redNodeKey {
		return redNodeKey{green: n.Green, offset: n.Offset}
	}
	var issues []AnalysisIssue
	callCallees := map[redNodeKey]bool{}
	suppressed := map[redNodeKey]bool{}
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if suppressed[keyOf(n)] {
			return
		}
		switch n.Kind() {
		case syntax.KindCallExpr:
			n.ForEachChildDesc(func(green *syntax.GreenNode, offset int) bool {
				switch green.Kind() {
				case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
					callCallees[redNodeKey{green: green, offset: offset}] = true
				}
				return true
			})
			node := syntax.LowerExprNode(n, res.File)
			if node == nil {
				syntax.Walk(n, func(d *syntax.RedNode) bool {
					suppressed[keyOf(d)] = true
					return true
				})
				return
			}
			cls := memoLowerClassLike(ctx, class, res.File)
			fn := memoLowerFunctionLike(ctx, currentFn, res.File)
			checkSymbolOnNode(filename, node, cls, fn, ft, ctx, guards, &issues)
			if fc, ok := node.(*ast.FunctionCallNode); ok {
				if cf, ok := fc.Name.(*ast.ClassConstFetchNode); ok {
					checkSymbolOnNode(filename, cf, cls, fn, ft, ctx, guards, &issues)
				}
			}
		case syntax.KindNewExpr:
			if node := syntax.LowerExprNode(n, res.File); node != nil {
				checkSymbolOnNode(filename, node, memoLowerClassLike(ctx, class, res.File), memoLowerFunctionLike(ctx, currentFn, res.File), ft, ctx, guards, &issues)
			}
		case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
			if callCallees[keyOf(n)] {
				return
			}
			if node := syntax.LowerExprNode(n, res.File); node != nil {
				checkSymbolOnNode(filename, node, memoLowerClassLike(ctx, class, res.File), memoLowerFunctionLike(ctx, currentFn, res.File), ft, ctx, guards, &issues)
			}
		}
	})
	return issues
}

func TestTryCSTCallExprForMemoShapes(t *testing.T) {
	src := []byte(`<?php
missingFn(1, named: 2, ...$rest);
Real::bar();
$this->m();
$o->m();
`)
	res := syntax.Parse(src)
	var calls []*syntax.RedNode
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindCallExpr {
			calls = append(calls, &syntax.RedNode{File: n.File, Green: n.Green, Offset: n.Offset})
		}
		return true
	})
	if len(calls) < 4 {
		t.Fatalf("expected >=4 calls, got %d", len(calls))
	}
	fn, ok := tryCSTCallExprForMemo(calls[0], res.File)
	if !ok || fn == nil {
		t.Fatal("expected function call shape")
	}
	fc, ok := fn.(*ast.FunctionCallNode)
	if !ok || functionCallName(fc) != "missingFn" {
		t.Fatalf("function shape: %#v", fn)
	}
	if len(fc.Args) != 3 {
		t.Fatalf("args: %d", len(fc.Args))
	}
	if _, ok := fc.Args[1].(*ast.NamedArgumentNode); !ok {
		t.Fatalf("expected named arg, got %T", fc.Args[1])
	}
	if _, ok := fc.Args[2].(*ast.UnpackedArgumentNode); !ok {
		t.Fatalf("expected unpacked, got %T", fc.Args[2])
	}

	st, ok := tryCSTCallExprForMemo(calls[1], res.File)
	if !ok {
		t.Fatal("static")
	}
	sfc := st.(*ast.FunctionCallNode)
	if functionCallName(sfc) != "Real::bar" {
		t.Fatalf("static name %q", functionCallName(sfc))
	}

	th, ok := tryCSTCallExprForMemo(calls[2], res.File)
	if !ok {
		t.Fatal("this")
	}
	mc := th.(*ast.MethodCallNode)
	if mc.Method != "m" {
		t.Fatalf("method %q", mc.Method)
	}
	if v, ok := mc.Object.(*ast.VariableNode); !ok || v.Name != "this" {
		t.Fatalf("object %#v", mc.Object)
	}
}

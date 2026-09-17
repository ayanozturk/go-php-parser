package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

// runReturnTypeOnASTNodes calls appendReturnTypeOnNode directly via
// walkAllWithFileContext, bypassing ensureStructuralIssues's fused walk so
// the ast.Node comparison path is isolated to just this rule.
func runReturnTypeOnASTNodes(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendReturnTypeOnNode(filename, node, class, ft, ctx, &issues)
	})
	return issues
}

func TestCheckReturnTypeIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"declaredVsActual": `<?php
function f(): int { return "x"; }
`,
		"correctReturn": `<?php
function f(): int { return 1; }
`,
		"voidReturnsValue": `<?php
function f(): void { return 1; }
`,
		"missingReturnPath": `<?php
function partial(bool $c): int { if ($c) { return 1; } }
`,
		"allPathsReturn": `<?php
function all(bool $c): int { if ($c) { return 1; } else { return 2; } }
`,
		"methodInClass": `<?php
class C { public function m(): int { return 1; } }
`,
		"closureMismatch": `<?php
$fn = function(): int { return "x"; };
`,
		"interfaceMethod": `<?php
interface I { public function run($x); }
`,
		"neverFallthrough": `<?php
function n(): never {}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, nodes := buildStructuralTestContext(t, filename, src)
			want := sortIssuesForCompare(runReturnTypeOnASTNodes(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckReturnTypeIssuesFromCST(filename, []byte(src), ctx))
			if len(want) != len(got) {
				t.Fatalf("issue count mismatch: ast=%d cst=%d\nast=%+v\ncst=%+v", len(want), len(got), want, got)
			}
			for i := range want {
				if want[i].Line != got[i].Line || want[i].Column != got[i].Column || want[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nast=%+v\ncst=%+v", i, want[i], got[i])
				}
			}
		})
	}
}

func TestCheckReturnTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckReturnTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

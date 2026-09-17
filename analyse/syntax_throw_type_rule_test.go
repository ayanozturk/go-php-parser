package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

// runThrowTypeOnASTNodes calls appendThrowTypeOnNode directly via
// walkAllWithFileContext, bypassing ensureStructuralIssues's fused walk so
// the ast.Node comparison path is isolated to just this rule.
func runThrowTypeOnASTNodes(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendThrowTypeOnNode(filename, node, ft, ctx, &issues)
	})
	return issues
}

func TestCheckThrowTypeIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"throwsInvalidClass": `<?php
class NotThrowable {}
function run(): void {
    throw new NotThrowable();
}
`,
		"throwsValidException": `<?php
function run(): void {
    throw new \RuntimeException("oops");
}
`,
		"throwsTrait": `<?php
trait T {}
function run(T $t): void {
    throw $t;
}
`,
		"throwsEnum": `<?php
enum E { case A; }
function run(): void {
    throw E::A;
}
`,
		"throwExprForm": `<?php
class NotThrowable {}
function run(?string $x): string {
    return $x ?? throw new NotThrowable();
}
`,
		"throwUnresolvedClass": `<?php
function run(): void {
    throw new MissingException();
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, nodes := buildStructuralTestContext(t, filename, src)
			want := sortIssuesForCompare(runThrowTypeOnASTNodes(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckThrowTypeIssuesFromCST(filename, []byte(src), ctx))
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

func TestCheckThrowTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckThrowTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

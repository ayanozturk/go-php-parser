package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

// runMissingTypesOnASTNodes calls appendMissingTypeIssuesOnNode directly via
// walkAllWithFileContext, bypassing ensureStructuralIssues's fused walk so
// the ast.Node comparison path is isolated to just this rule.
func runMissingTypesOnASTNodes(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendMissingTypeIssuesOnNode(filename, node, class, ft, ctx, &issues)
	})
	return issues
}

func TestCheckMissingTypeIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"missingParamType": `<?php
function run($x): void {}
`,
		"missingReturnType": `<?php
function run(int $x) {
    return $x;
}
`,
		"missingPropertyType": `<?php
class C {
    public $field;
}
`,
		"missingGenericType": `<?php
/**
 * @param array $x
 */
function run(array $x): void {}
`,
		"missingIterableValueType": `<?php
/**
 * @param iterable $x
 */
function run(iterable $x): void {}
`,
		"interfaceMethodMissingType": `<?php
interface I {
    public function run($x);
}
`,
		"constructorExemptFromReturnType": `<?php
class C {
    public function __construct(int $x) {}
}
`,
		"okAllTypesPresent": `<?php
class C {
    public int $field;
    public function run(int $x): int {
        return $x;
    }
}
`,
		"docParamTypeSuppresses": `<?php
/**
 * @param int $x
 */
function run($x): void {}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, nodes := buildStructuralTestContext(t, filename, src)
			want := sortIssuesForCompare(runMissingTypesOnASTNodes(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckMissingTypeIssuesFromCST(filename, []byte(src), ctx))
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

func TestCheckMissingTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckMissingTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

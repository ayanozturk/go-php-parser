package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

// runPHPDocOnASTNodes calls appendPHPDocIssuesOnNode directly via
// walkAllWithFileContext, bypassing ensureStructuralIssues's fused walk so
// the ast.Node comparison path is isolated to just this rule. Mirrors
// ensureStructuralIssues's own ctx.phpDocTypeAliases caching (computed once
// per ctx from the full nodes list, exactly like production).
func runPHPDocOnASTNodes(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	if ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendPHPDocIssuesOnNode(filename, node, class, ft, ctx, &issues)
	})
	return issues
}

func TestCheckPHPDocIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"paramTypeMismatch": `<?php
class C {
    /**
     * @param int $x
     */
    public function run(string $x): void {}
}
`,
		"paramUnknownName": `<?php
class C {
    /**
     * @param int $missing
     */
    public function run(string $x): void {}
}
`,
		"returnTypeMismatch": `<?php
/**
 * @return int
 */
function run(): string {
    return "x";
}
`,
		"propertyTypeMismatch": `<?php
class C {
    /**
     * @var int
     */
    public string $field;
}
`,
		"interfaceMethodTypeMismatch": `<?php
interface I {
    /**
     * @param int $x
     */
    public function run(string $x): void;
}
`,
		"closureParamTypeMismatch": `<?php
function outer(): void {
    /**
     * @param int $x
     */
    $fn = function (string $x): void {};
}
`,
		"templateNameSuppressesMismatch": `<?php
/**
 * @template T
 * @param T $x
 */
function run(string $x): void {}
`,
		"okNoIssues": `<?php
class C {
    /**
     * @param int $x
     * @return int
     */
    public function run(int $x): int {
        return $x;
    }
}
`,
		// Regression: an anonymous class's property nested behind
		// `(new class {...})::class` has its class-part expression
		// discarded by splitStaticMemberAccessParts (only .TokenLiteral()
		// is kept - see syntax_walk.go's staticMemberAccessDynamicClassPart
		// doc comment), so the anonymous class's members (including this
		// property's PHPDoc) are entirely invisible to walkAllConfigured.
		// self::* is a legitimate PHPStan class-constant wildcard, not an
		// unknown class reference - but it's moot here since the property
		// should never be reached by either path.
		"anonClassBehindStaticClassConstUnreachable": `<?php
function run() {
    useIt((new class {
        /** @var self::*|null */
        public $foo;
    })::class);
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, nodes := buildStructuralTestContext(t, filename, src)
			want := sortIssuesForCompare(runPHPDocOnASTNodes(filename, nodes, ctx, fileCtx))

			ctx2, _, _ := buildStructuralTestContext(t, filename, src)
			aliases := collectPHPDocTypeAliases(nodes)
			got := sortIssuesForCompare(CheckPHPDocIssuesFromCST(filename, []byte(src), ctx2, aliases))
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

func TestCheckPHPDocIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckPHPDocIssuesFromCST("empty.php", nil, &AnalysisContext{}, nil); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

func TestCheckPHPDocIssuesFromCSTNilContext(t *testing.T) {
	if got := CheckPHPDocIssuesFromCST("empty.php", []byte("<?php\n"), nil, nil); got != nil {
		t.Fatalf("expected nil issues for nil ctx, got %+v", got)
	}
}

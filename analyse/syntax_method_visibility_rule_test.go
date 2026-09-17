package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// buildStructuralTestContext parses src and builds a single-file
// project-index resolver (mirroring buildTypeRefTestContext, minus
// reflectionGuards which the structural-pass rules don't need).
func buildStructuralTestContext(t *testing.T, filename, src string) (ctx *AnalysisContext, fileCtx FileTypeContext, nodes []ast.Node) {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(src))
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics for %s: %v", filename, diags)
	}
	fileCtx = CollectFileTypeContext(nodes)
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx = &AnalysisContext{Resolver: project}
	return ctx, fileCtx, nodes
}

// runMethodVisibilityOnASTNodes calls appendMethodVisibilityOnNode directly
// via walkAllWithFileContext, bypassing ensureStructuralIssues's fused walk
// (which would also compute throw/phpdoc/missing-type/return-type issues as
// a side effect) so the ast.Node comparison path is isolated to just this
// rule, matching the CST-direct port's own scope.
func runMethodVisibilityOnASTNodes(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendMethodVisibilityOnNode(filename, node, class, currentFn, ft, ctx, &issues)
	})
	return issues
}

func TestCheckMethodVisibilityIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"staticCallToProtectedFromUnrelated": `<?php
class Base {
    protected static function helper(): void {}
}
class Caller {
    public function run(): void {
        Base::helper();
    }
}
`,
		"staticCallToProtectedFromSubclass": `<?php
class Base {
    protected static function helper(): void {}
}
class Sub extends Base {
    public function run(): void {
        Base::helper();
    }
}
`,
		"methodCallOnThisFromStaticMethod": `<?php
class Base {
    protected function helper(): void {}
    public static function run(): void {
        $obj = new Base();
        $obj->helper();
    }
}
`,
		"methodCallOnThisOk": `<?php
class Base {
    protected function helper(): void {}
    public function run(): void {
        $this->helper();
    }
}
`,
		"methodCallOnOtherVarUnrelated": `<?php
class Base {
    protected function helper(): void {}
}
class Caller {
    public function run(Base $b): void {
        $b->helper();
    }
}
`,
		"dynamicClassNameSkipped": `<?php
class Base {
    protected static function helper(): void {}
}
function run(string $cls): void {
    $cls::helper();
}
`,
		"traitUserAllowed": `<?php
trait T {
    protected function helper(): void {}
}
class UsesT {
    use T;
}
class Caller {
    public function run(UsesT $u): void {
        $u->helper();
    }
}
`,
		"chainedMethodCalls": `<?php
class Base {
    protected function helper(): void {}
    public function other(): static {
        return $this;
    }
}
class Caller {
    public function run(Base $b): void {
        $b->other()->helper();
    }
}
`,
		"dynamicMethodCallSkipped": `<?php
class Base {
    protected function helper(): void {}
}
class Caller {
    public function run(Base $b, string $m): void {
        $b->{$m}();
    }
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, nodes := buildStructuralTestContext(t, filename, src)
			want := sortIssuesForCompare(runMethodVisibilityOnASTNodes(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckMethodVisibilityIssuesFromCST(filename, []byte(src), ctx))
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

func TestCheckMethodVisibilityIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckMethodVisibilityIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

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

func TestCheckMethodVisibilityIssuesFromCST(t *testing.T) {
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

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"staticCallToProtectedFromUnrelated": {
			{Message: "Call to protected method Base::helper().", Line: 7, Column: 9},
		},
		"staticCallToProtectedFromSubclass": {},
		"methodCallOnThisFromStaticMethod":  {},
		"methodCallOnThisOk":                {},
		"methodCallOnOtherVarUnrelated":     {},
		"dynamicClassNameSkipped":           {},
		"traitUserAllowed":                  {},
		"chainedMethodCalls":                {},
		"dynamicMethodCallSkipped":          {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, _ := buildStructuralTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckMethodVisibilityIssuesFromCST(filename, []byte(src), ctx))

			wantIssues := want[name]
			if len(wantIssues) != len(got) {
				t.Fatalf("issue count mismatch: want=%d got=%d\nwant=%+v\ngot=%+v", len(wantIssues), len(got), wantIssues, got)
			}
			for i := range wantIssues {
				if wantIssues[i].Line != got[i].Line || wantIssues[i].Column != got[i].Column || wantIssues[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nwant=%+v\ngot=%+v", i, wantIssues[i], got[i])
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

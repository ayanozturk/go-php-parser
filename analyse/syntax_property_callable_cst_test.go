package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestPropertyCallableCSTNativeMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"typedPropertyCallable": `<?php
class C {
    public callable $handler;
    public ?callable $maybeHandler;
    public int $count;
}
`,
		"unionTypeWithCallable": `<?php
class C {
    public callable|string $handler;
}
`,
		"multiPropertyOneCallable": `<?php
class C {
    public callable $a, $b;
}
`,
		"promotedConstructorParam": `<?php
class C {
    public function __construct(public callable $handler, private int $count) {}
}
`,
		"nonPromotedCallableParamIgnored": `<?php
class C {
    public function run(callable $handler) {}
}
`,
		"intersectionNoCallable": `<?php
class C {
    public Foo&Bar $x;
}
`,
		"clean": `<?php
class C {
    public int $count;
    public function __construct(private string $name) {}
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			content := []byte(src)
			res := syntax.Parse(content)

			cst := sortIssuesForCompare(checkPropertyCallableTypeIssuesFromParsed(filename, res))
			lower := sortIssuesForCompare(checkPropertyCallableViaLower(filename, res))
			if len(cst) != len(lower) {
				t.Fatalf("count mismatch cst=%d lower=%d\ncst=%+v\nlower=%+v", len(cst), len(lower), cst, lower)
			}
			for i := range cst {
				if cst[i].Line != lower[i].Line || cst[i].Column != lower[i].Column ||
					cst[i].EndLine != lower[i].EndLine || cst[i].EndColumn != lower[i].EndColumn ||
					cst[i].Message != lower[i].Message || cst[i].Code != lower[i].Code {
					t.Fatalf("issue %d mismatch:\ncst=%+v\nlower=%+v", i, cst[i], lower[i])
				}
			}
		})
	}
}

// checkPropertyCallableViaLower mirrors the pre-CST-native fused path for
// differential parity (LowerPropertyDeclNode / LowerParamNode → OnNode).
func checkPropertyCallableViaLower(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindPropertyDecl:
			for _, prop := range syntax.LowerPropertyDeclNode(n, res.File) {
				appendPropertyCallableTypeIssue(filename, prop, &issues)
			}
		case syntax.KindParam:
			if param := syntax.LowerParamNode(n, res.File); param != nil {
				appendPropertyCallableTypeIssue(filename, param, &issues)
			}
		}
		return true
	})
	return issues
}

func TestSyntaxLowerMemoStatsRecordHits(t *testing.T) {
	EnableSyntaxLowerMemoStats(true)
	defer EnableSyntaxLowerMemoStats(false)
	ResetSyntaxLowerMemoStats()

	src := []byte(`<?php class C { public function a(){ strlen('x'); } }`)
	nodes, _ := syntax.ParseAST(src)
	level := 6
	ctx := &AnalysisContext{Content: src, AnalysisLevel: &level, Resolver: BuildProjectIndex(map[string][]ast.Node{"t.php": nodes})}
	ensureSharedFileDiagnosticsFromCST("t.php", src, nodes, ctx)

	snap := SnapshotSyntaxLowerMemoStats()
	var exprSum uint64
	for _, r := range snap.Rows {
		if r.Bucket == "expr" {
			exprSum = r.Sum
		}
	}
	if exprSum == 0 {
		t.Fatalf("expected expr memoLower traffic, got:\n%s", FormatSyntaxLowerMemoStats(snap))
	}
}

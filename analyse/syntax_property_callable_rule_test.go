package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// astPropertyCallableIssuesReal mirrors how ensureSharedFileDiagnostics
// drives appendPropertyCallableTypeIssue: fed by the full ast.Node walk,
// over every node in the tree (the function itself filters by node type).
func astPropertyCallableIssuesReal(t *testing.T, filename, src string) []AnalysisIssue {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(src))
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics for %s: %v", filename, diags)
	}
	var issues []AnalysisIssue
	walkAllWithoutTypeContext(nodes, func(node ast.Node) {
		appendPropertyCallableTypeIssue(filename, node, &issues)
	})
	return issues
}

func TestCheckPropertyCallableTypeIssuesFromCSTMatchesASTPath(t *testing.T) {
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
		"closureInPropertyDefault": `<?php
class C {
    public $factory = null;
    public function make() {
        $fn = function (callable $cb) { return $cb; };
        return $fn;
    }
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
			want := sortIssuesForCompare(astPropertyCallableIssuesReal(t, filename, src))
			got := sortIssuesForCompare(CheckPropertyCallableTypeIssuesFromCST(filename, []byte(src)))
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

func TestCheckPropertyCallableTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckPropertyCallableTypeIssuesFromCST("empty.php", nil); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

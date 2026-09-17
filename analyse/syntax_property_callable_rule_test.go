package analyse

import (
	"testing"
)

func TestCheckPropertyCallableTypeIssuesFromCST(t *testing.T) {
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

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"typedPropertyCallable": {
			{Message: "Property $handler cannot have callable in its type declaration.", Line: 3, Column: 5},
			{Message: "Property $maybeHandler cannot have callable in its type declaration.", Line: 4, Column: 5},
		},
		"unionTypeWithCallable": {
			{Message: "Property $handler cannot have callable in its type declaration.", Line: 3, Column: 5},
		},
		"multiPropertyOneCallable": {
			{Message: "Property $a cannot have callable in its type declaration.", Line: 3, Column: 20},
			{Message: "Property $b cannot have callable in its type declaration.", Line: 3, Column: 24},
		},
		"promotedConstructorParam": {
			{Message: "Property $handler cannot have callable in its type declaration.", Line: 3, Column: 33},
		},
		"nonPromotedCallableParamIgnored": {},
		"closureInPropertyDefault":        {},
		"clean":                           {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			got := sortIssuesForCompare(CheckPropertyCallableTypeIssuesFromCST(filename, []byte(src)))

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

func TestCheckPropertyCallableTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckPropertyCallableTypeIssuesFromCST("empty.php", nil); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

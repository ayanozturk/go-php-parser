package analyse

import "testing"

func TestLevel2PHPDocClosureSignatures(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   map[string]int
	}{
		{"declarations", `<?php
namespace Example;
use Closure;
class Item {}
/** @param Closure(Item): (Item|null) $factory */
function acceptsClosure(Closure $factory): void {}
class Holder {
    /** @var \Closure(Item): Item */
    public Closure $factory;
}
/** @return Closure(Item): Item */
function returnsClosure(): Closure { throw new \RuntimeException(); }
/** @param callable(Item): (Item|null) $factory */
function acceptsCallable(callable $factory): void {}
`, nil},
		{"unknown nested types", `<?php
/** @param Closure(MissingInput): MissingOutput $factory */
function unknownClosure(callable $factory): void {}
`, map[string]int{level2PHPDocClassCode: 2}},
		{"incompatible native", `<?php
/** @param Closure(): int $factory */
function incompatibleClosure(int $factory): void {}
`, map[string]int{level2PHPDocParamTypeCode: 1}},
		{"nullable closure is not nullable result", `<?php
/** @param (Closure(): int)|null $factory */
function nullableClosure(Closure $factory): void {}
`, map[string]int{level2PHPDocParamTypeCode: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": tc.source}, 2)
			got := make(map[string]int)
			for _, issue := range issues {
				got[issue.Code]++
			}
			if len(got) != len(tc.want) {
				t.Fatalf("issues = %#v, want counts %#v", issues, tc.want)
			}
			for code, count := range tc.want {
				if got[code] != count {
					t.Fatalf("issues = %#v, want counts %#v", issues, tc.want)
				}
			}
		})
	}
}

func TestLevel6PHPDocClosureSignatureMissingIterableDetail(t *testing.T) {
	const source = `<?php
/** @param Closure(array): array $factory */
function missingDetail(Closure $factory): void {}
/** @param Closure(array<int, string>): array<int, string> $factory */
function knownDetail(Closure $factory): void {}
`
	for _, level := range []int{5, 6} {
		issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, level)
		want := 0
		if level == 6 {
			want = 2
		}
		if len(issues) != want {
			t.Fatalf("level %d issues = %#v, want %d", level, issues, want)
		}
		for _, issue := range issues {
			if issue.Code != level6MissingIterableTypeCode {
				t.Fatalf("unexpected issue: %#v", issue)
			}
		}
	}
}

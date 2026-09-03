package analyse

import "testing"

func TestLevel0PropertyCallableTypeCoversNativeAndPromotedDeclarations(t *testing.T) {
	const source = `<?php
final class Example {
    public callable $native;
    public callable|null $union;
    /** @var callable */
    public $documented;
    public Closure $closure;
    public function __construct(public ?callable $promoted) {}
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 0)
	if got := countIssuesWithCode(issues, level0PropertyCallableTypeCode); got != 3 {
		t.Fatalf("property callable issue count = %d, want 3; issues: %#v", got, issues)
	}
}

func countIssuesWithCode(issues []AnalysisIssue, code string) int {
	count := 0
	for _, issue := range issues {
		if issue.Code == code {
			count++
		}
	}
	return count
}

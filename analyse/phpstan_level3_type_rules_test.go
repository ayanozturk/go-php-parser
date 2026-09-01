package analyse

import "testing"

func TestLevel3EnablesReturnAndPropertyTypeRules(t *testing.T) {
	files := map[string]string{
		"types.php": `<?php
final class Meter
{
    public int $reading = 0;

    public function label(): string
    {
        return 7;
    }

    public function update(): void
    {
        $this->reading = 'invalid';
    }
}
`,
	}

	for _, issue := range runAnalysisLevelOnFiles(t, files, 2) {
		if issue.Code == "A.RETURN.TYPE" || issue.Code == "A.PROP.TYPE" {
			t.Fatalf("level 2 should suppress level-3 type rules, got %#v", issue)
		}
	}

	level3Issues := runAnalysisLevelOnFiles(t, files, 3)
	want := map[string]int{"A.RETURN.TYPE": 1, "A.PROP.TYPE": 1}
	for _, issue := range level3Issues {
		if _, ok := want[issue.Code]; ok {
			want[issue.Code]--
		}
	}
	for code, remaining := range want {
		if remaining != 0 {
			t.Fatalf("level 3 should emit one %s issue, got %#v", code, level3Issues)
		}
	}
}

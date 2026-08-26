package analyse

import "testing"

func TestLevel1UndefinedVariableIfBranchesPreserveScopeRules(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		wantUndefined []string
	}{
		{
			name: "sibling isolation",
			source: `<?php
$first = true;
$second = false;
if ($first) {
    $onlyThen = 1;
} elseif ($second) {
    echo $onlyThen;
}
`,
			wantUndefined: []string{"$onlyThen"},
		},
		{
			name: "post-if union",
			source: `<?php
$condition = true;
if ($condition) {
    $fromThen = 1;
} else {
    $fromElse = 2;
}
echo $fromThen;
echo $fromElse;
`,
			wantUndefined: []string{"$fromThen", "$fromElse"},
		},
		{
			name: "nested branches",
			source: `<?php
$outer = true;
$inner = false;
if ($outer) {
    if ($inner) {
        $fromInner = 1;
    } else {
        $fromNestedElse = 2;
    }
    echo $fromInner;
echo $fromNestedElse;
}
`,
			wantUndefined: []string{"$fromInner", "$fromNestedElse"},
		},
		{
			name: "condition assignment",
			source: `<?php
if ($assigned = 1) {
    echo $assigned;
}
echo $assigned;
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := runLevel1OnFiles(t, map[string]string{"test.php": test.source})
			for _, name := range test.wantUndefined {
				if !hasIssueContaining(issues, level1VariablesCode, "Variable "+name+" might not be defined.") {
					t.Fatalf("expected undefined variable issue for %s, got %#v", name, issues)
				}
			}
			for _, issue := range issues {
				if issue.Code != level1VariablesCode || len(test.wantUndefined) == 0 {
					continue
				}
				matched := false
				for _, name := range test.wantUndefined {
					if issue.Message == "Variable "+name+" might not be defined." {
						matched = true
						break
					}
				}
				if !matched {
					t.Fatalf("unexpected undefined variable issue: %#v", issue)
				}
			}
			if len(test.wantUndefined) == 0 {
				for _, issue := range issues {
					if issue.Code == level1VariablesCode {
						t.Fatalf("unexpected undefined variable issue: %#v", issue)
					}
				}
			}
		})
	}
}

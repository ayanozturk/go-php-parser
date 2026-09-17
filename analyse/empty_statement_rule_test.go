package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func runEmptyStatementAnalysis(t *testing.T, php string) []AnalysisIssue {
	t.Helper()
	r := &EmptyStatementRule{}
	return r.CheckIssuesWithSource("test.php", []byte(php), nil)
}

func countEmptyStatementIssues(issues []AnalysisIssue) int {
	cnt := 0
	for _, iss := range issues {
		if iss.Code == "Generic.CodeAnalysis.EmptyStatement" {
			cnt++
		}
	}
	return cnt
}

func hasEmptyStatementIssue(issues []AnalysisIssue) bool {
	return countEmptyStatementIssues(issues) > 0
}

func TestStandaloneSemicolon(t *testing.T) {
	php := "<?php\n;\n$z = 1;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if countEmptyStatementIssues(issues) != 1 {
		t.Fatalf("expected 1 empty statement, got %d (issues=%v)", countEmptyStatementIssues(issues), issues)
	}
}

func TestIfWithSemicolonBody(t *testing.T) {
	php := "<?php\nif ($x) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for if(...);, got %v", issues)
	}
}

func TestIfWithSemicolonBodyOnNextLine(t *testing.T) {
	php := "<?php\nif ($x)\n    ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for multiline if body, got %v", issues)
	}
}

func TestWhileWithSemicolonBody(t *testing.T) {
	php := "<?php\nwhile ($x) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for while(...);, got %v", issues)
	}
}

func TestForWithSemicolonBody(t *testing.T) {
	php := "<?php\nfor($i=0;$i<10;$i++) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for for(...);, got %v", issues)
	}
}

func TestForHeaderSemicolonsIgnored(t *testing.T) {
	php := "<?php\nfor($i=0;$i<10;$i++) { }\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("did not expect empty statement for for header semicolons, got %v", issues)
	}
}

func TestMultipleEmptySemicolons(t *testing.T) {
	php := "<?php\n; ; ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if countEmptyStatementIssues(issues) != 3 {
		t.Fatalf("expected 3 empty statements, got %d (%v)", countEmptyStatementIssues(issues), issues)
	}
}

func TestNoFalsePositiveInStatements(t *testing.T) {
	php := "<?php\n$a = 1;\n$b = 2;\nif ($a) { echo $b; }\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("unexpected empty statement issue: %v", issues)
	}
}

func TestDoWhileConditionSemicolonIsNotEmptyStatement(t *testing.T) {
	php := "<?php\ndo {\n    $token = next_token();\n} while ($token !== null);\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("do-while terminator should not be an empty statement, got %v", issues)
	}
}

func TestEmptyWhileAfterBlockStillReported(t *testing.T) {
	php := "<?php\nif ($ready) {\n    work();\n}\nwhile ($drain) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty while body after a closed block, got %v", issues)
	}
}

func TestEmptyStatementInStringIsNotReported(t *testing.T) {
	php := "<?php\n$s = ';';\necho $s;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("semicolon inside a string is not an empty statement, got %v", issues)
	}
}

func TestEmptyStatementInCommentIsNotReported(t *testing.T) {
	php := "<?php\n// ;\n$a = 1;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("semicolon inside a comment is not an empty statement, got %v", issues)
	}
}

// TestCSTPathMatchesLoweredPath guards the CST-direct fallback in
// CheckIssuesWithSource against drifting from the pre-lowered ast.Node path
// still used by the fused walk in ensureSharedFileDiagnostics.
func TestCSTPathMatchesLoweredPath(t *testing.T) {
	cases := []string{
		"<?php\n;\n$z = 1;\n",
		"<?php\nif ($x) ;\n",
		"<?php\nif ($x)\n    ;\n",
		"<?php\nwhile ($x) ;\n",
		"<?php\nfor($i=0;$i<10;$i++) ;\n",
		"<?php\nfor($i=0;$i<10;$i++) { }\n",
		"<?php\n; ; ;\n",
		"<?php\n$a = 1;\n$b = 2;\nif ($a) { echo $b; }\n",
		"<?php\ndo {\n    $token = next_token();\n} while ($token !== null);\n",
		"<?php\nif ($ready) {\n    work();\n}\nwhile ($drain) ;\n",
		"<?php\n$s = ';';\necho $s;\n",
		"<?php\n// ;\n$a = 1;\n",
	}
	r := &EmptyStatementRule{}
	for _, php := range cases {
		content := []byte(php)
		cstIssues := r.CheckIssuesWithSource("test.php", content, nil)
		nodes, _ := syntax.ParseAST(content)
		loweredIssues := r.CheckIssuesWithSource("test.php", nil, nodes)
		if len(cstIssues) != len(loweredIssues) {
			t.Fatalf("issue count mismatch for %q: cst=%d lowered=%d", php, len(cstIssues), len(loweredIssues))
		}
		for i := range cstIssues {
			if cstIssues[i].Line != loweredIssues[i].Line || cstIssues[i].Column != loweredIssues[i].Column {
				t.Fatalf("position mismatch for %q at issue %d: cst=%+v lowered=%+v", php, i, cstIssues[i], loweredIssues[i])
			}
		}
	}
}

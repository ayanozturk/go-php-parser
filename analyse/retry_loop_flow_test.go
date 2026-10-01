package analyse

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestRetryLoopCarriesGuaranteedCatchAssignmentAfterExit(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class RetryFailure extends Exception {}
class RateLimitFailure extends Exception {}
class FatalFailure extends Exception {}

class Worker {
    private const int LIMIT = 3;

    public function execute(): void {
        $attempt = 0;
        $lastFailure = null;

        while ($attempt < self::LIMIT) {
            try {
                $attempt++;
                return;
            } catch (RetryFailure $failure) {
                $lastFailure = $failure;
            } catch (RateLimitFailure $failure) {
                $lastFailure = $failure;
            } catch (FatalFailure $failure) {
                throw $failure;
            }
        }

		$lastFailure->getMessage();
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") ||
		hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
		t.Fatalf("guaranteed catch assignment should make the post-loop receiver non-null, got %#v", issues)
	}
}

func TestRetryLoopDoesNotNarrowWhenZeroIterationsArePossible(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
function execute(bool $retry): void {
    $lastFailure = null;
    while ($retry) {
        $lastFailure = new Exception();
        break;
    }
    $lastFailure->getMessage();
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	if !hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") &&
		!hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
		t.Fatalf("possibly skipped loop must retain nullable receiver diagnostic, got %#v", issues)
	}
}

func TestRetryLoopDoesNotNarrowWhenAFirstIterationPathSkipsAssignment(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
function execute(bool $record): void {
    $attempt = 0;
    $lastFailure = null;
    while ($attempt < 1) {
        $attempt++;
        if ($record) {
            $lastFailure = new Exception();
        }
    }
    $lastFailure->getMessage();
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	if !hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") &&
		!hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
		t.Fatalf("assignment skipped by one loop path must retain nullable receiver diagnostic, got %#v", issues)
	}
}

func TestRetryLoopDoesNotNarrowWhenMultiLevelControlCanExitWithoutAssignment(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
function execute(bool $escape): void {
    $attempt = 0;
    $lastFailure = null;
    while ($attempt < 1) {
        if ($escape) {
            break 2;
        }
        $lastFailure = new Exception();
    }
    $lastFailure->getMessage();
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	if !hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") &&
		!hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
		t.Fatalf("multi-level break path without an assignment must retain nullable receiver diagnostic, got %#v", issues)
	}
}

func TestRetryLoopFlowKeepsPrecisionBelowPathBudget(t *testing.T) {
	files := map[string]string{"test.php": retryLoopBranchSource(6)}
	for path, issues := range map[string][]AnalysisIssue{
		"AST compatibility": runRetryLoopASTPath(t, files, 8),
		"CST production":    runAnalysisLevelOnFiles(t, files, 8),
	} {
		if hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") || hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
			t.Errorf("%s: loop exit assignment should remain precise below the path budget, got %#v", path, issues)
		}
	}
}

func TestRetryLoopFlowStopsAtPathBudget(t *testing.T) {
	files := map[string]string{"test.php": retryLoopBranchSource(8)}
	for path, issues := range map[string][]AnalysisIssue{
		"AST compatibility": runRetryLoopASTPath(t, files, 8),
		"CST production":    runAnalysisLevelOnFiles(t, files, 8),
	} {
		if !hasIssueContaining(issues, level2MethodNonObjectCode, "getMessage") && !hasIssueContaining(issues, level8MethodNonObjectCode, "getMessage") {
			t.Errorf("%s: loop exit assignment should stay unknown when path analysis exceeds its budget, got %#v", path, issues)
		}
	}
}

func runRetryLoopASTPath(t *testing.T, files map[string]string, level int) []AnalysisIssue {
	t.Helper()
	parsed := make(map[string][]ast.Node, len(files))
	for filename, source := range files {
		parsed[filename] = parsePHPForLevel0(t, source)
	}
	project := BuildProjectIndex(parsed)
	var issues []AnalysisIssue
	for filename, nodes := range parsed {
		ctx := &AnalysisContext{Resolver: project, AnalysisLevel: &level}
		issues = append(issues, RunAnalysisRulesWithContext(filename, nodes, ctx)...)
	}
	return issues
}

func retryLoopBranchSource(branchCount int) string {
	var source strings.Builder
	source.WriteString("<?php\nclass RetryFailure extends Exception {}\nfunction execute(")
	for i := 0; i < branchCount; i++ {
		if i > 0 {
			source.WriteString(", ")
		}
		fmt.Fprintf(&source, "bool $branch%d", i)
	}
	source.WriteString("): void {\n$attempt = 0;\n$lastFailure = null;\nwhile ($attempt < 1) {\n$attempt++;\n")
	for i := 0; i < branchCount; i++ {
		fmt.Fprintf(&source, "if ($branch%d) {}\n", i)
	}
	source.WriteString("$lastFailure = new RetryFailure();\n}\n$lastFailure->getMessage();\n}\n")
	return source.String()
}

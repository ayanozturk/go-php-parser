package analyse

import "testing"

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

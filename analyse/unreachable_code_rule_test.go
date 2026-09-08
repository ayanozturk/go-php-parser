package analyse

import (
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"testing"
)

func analyseUnreachablePHP(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	rule := &UnreachableCodeRule{}
	return rule.CheckIssues(nodes, "test.php")
}

func countUnreachableIssues(issues []AnalysisIssue) int {
	count := 0
	for _, issue := range issues {
		if issue.Code == "Generic.CodeAnalysis.UnreachableCode" {
			count++
		}
	}
	return count
}

func TestUnreachableAfterReturnInFunction(t *testing.T) {
	php := `<?php
function foo(): int {
    return 1;
    $x = 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterThrowInFunction(t *testing.T) {
	php := `<?php
function foo(): int {
    throw $e;
    return 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAcrossBranchesNotReported(t *testing.T) {
	php := `<?php
function foo($x): int {
    if ($x) {
        return 1;
    }

    return 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterReturnWithTrailingCommentNotReported(t *testing.T) {
	php := `<?php
class TaskAuditService
{
    public function __construct(private readonly EntityManagerInterface $entityManager) {}

    public function recordStatusChange(Task $task, object $actor, string $oldStatus, string $newStatus): void
    {
        if ($oldStatus === $newStatus) {
            return; // no change
        }
        $this->persist(new TaskAudit($task, $actor, TaskAuditChangeType::STATUS, $oldStatus, $newStatus));
    }
}
`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues after return with trailing comment, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterReturnStillReportedWhenCommentIntervenes(t *testing.T) {
	php := `<?php
function foo(): void {
    return; // done
    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue after return and comment, got %d (%#v)", got, issues)
	}
}

func TestTrailingCommentsAfterTerminatorsAreNotUnreachable(t *testing.T) {
	cases := []string{
		`<?php function foo(): void { return; // no change
}`,
		`<?php function foo(): void { throw $e; // boom
}`,
		`<?php function foo(): void { exit(); // stop
}`,
		`<?php function foo(): void { die(); // stop
}`,
		`<?php function foo(): void { foreach ([1] as $x) { break; // leave
} }`,
		`<?php function foo(): void { foreach ([1] as $x) { continue; // next
} }`,
	}
	for _, php := range cases {
		issues := analyseUnreachablePHP(t, php)
		if got := countUnreachableIssues(issues); got != 0 {
			t.Fatalf("expected 0 unreachable issues for %q, got %d (%#v)", php, got, issues)
		}
	}
}

func TestReachableAfterUnbracedIfReturnNotReported(t *testing.T) {
	php := `<?php
function isFeatureEnabled($feature): bool {
    $user = $this->getUser();
    if (!$user instanceof User)
        return true;

    return $feature->isEnabledFor($user);
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableInsideIfBranch(t *testing.T) {
	php := `<?php
function foo($x): int {
    if ($x) {
        return 1;
        $a = 1;
    }

    return 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterIfReturnWithNullsafeCallNotReported(t *testing.T) {
	php := `<?php
function getContactEmail(): ?string {
    if ($this->email) {
        return $this->email;
    }

    $admin = $this->getAdministrator();
    return $admin?->getEmail();
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterCarbonIntervalDateIntervalGuardNotReported(t *testing.T) {
	php := `<?php
class CarbonInterval extends DateInterval {
    public function __construct($years = null, $months = null) {
        if ($years instanceof DateInterval) {
            parent::__construct(static::getDateIntervalSpec($years));
            $this->f = $years->f;
            self::copyNegativeUnits($years, $this);

            return;
        }

        $spec = $years;
        $isStringSpec = (\is_string($spec) && !preg_match('/^[\d.]/', $spec));

        if (!$isStringSpec || (float) $years) {
            $spec = static::PERIOD_PREFIX;

            $spec .= $years > 0 ? $years.static::PERIOD_YEARS : '';
            $spec .= $months > 0 ? $months.static::PERIOD_MONTHS : '';
        }
    }
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterExitInFunction(t *testing.T) {
	php := `<?php
function foo(): void {
    exit();
    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterConditionalExitNotReported(t *testing.T) {
	php := `<?php
function foo($redirectURL): void {
    if ($redirectURL) {
        redirect($redirectURL);
        exit();
    }

    $authenticationState = $authService->getState();
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterUnbracedIfElseBothTerminate(t *testing.T) {
	php := `<?php
function foo($x): void {
    if ($x)
        return;
    else
        throw $e;

    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterIfElseBothTerminate(t *testing.T) {
	php := `<?php
function foo($x): void {
    if ($x) {
        exit();
    } else {
        throw $e;
    }

    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterForeachWithAssignmentNotReported(t *testing.T) {
	php := `<?php
class PolicyComplianceService {
    public function getCompliantUsers(): array
    {
        return array_values(array_filter($users, $this->isUserCompliant(...)));
    }

    public function getPoliciesWithLowCompliance(Company $company, float $threshold = 80.0): array
    {
        $policiesById = [];
        foreach ($this->policyRepository->findActiveByCompany($company) as $policy) {
            $policiesById[$policy->getId()] = $policy;
        }

        $lowCompliancePolicies = [];
        foreach ($this->getComplianceByPolicy($company) as $policyStats) {
            if ($policyStats['complianceRate'] >= $threshold) {
                continue;
            }
        }

        return $lowCompliancePolicies;
    }
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues after foreach assignment, got %d (%#v)", got, issues)
	}
}

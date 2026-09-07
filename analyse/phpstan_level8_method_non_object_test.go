package analyse

import "testing"

func TestLevel8ReportsKnownMethodsOnNullableObjectReceivers(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class Service { public function execute(): void {} }
class FirstKnown { public function execute(): void {} }
class SecondKnown { public function execute(): void {} }
class First {}
class Second { public function optional(): void {} }
class Missing {}
interface Executable { public function execute(): void; }
interface Marker {}

function run(
    ?Service $nullable,
    Service|null $union,
    (Executable&Marker)|null $intersection,
    FirstKnown|SecondKnown|null $allHave,
    First|Second|null $partial,
    ?Missing $unknown,
    ?object $object,
    mixed $mixed,
    ?int $nullableInt,
    ?Service $nullsafe
): void {
    $nullable->execute();
    $union->execute();
    $intersection->execute();
    $allHave->execute();
    $partial->optional();
    $unknown->missing();
    $object->execute();
    $mixed->execute();
    $nullableInt->missing();
    $nullsafe?->execute();
}

function ternary(bool $flag): void {
    ($flag ? new Service() : null)->execute();
}
`,
	}

	level7Issues := runAnalysisLevelOnFiles(t, files, 7)
	if hasIssueContaining(level7Issues, level8MethodNonObjectCode, "Cannot call method") {
		t.Fatalf("level seven should exclude level-eight nullable diagnostics, got %#v", level7Issues)
	}
	if hasIssueContaining(level7Issues, level2MethodNonObjectCode, "execute() on") {
		t.Fatalf("known nullable object methods should stay clean at level two, got %#v", level7Issues)
	}

	level8Issues := runAnalysisLevelOnFiles(t, files, 8)
	for _, expected := range []string{
		"execute() on Service|null.",
		"execute() on (Executable&Marker)|null.",
		"execute() on FirstKnown|SecondKnown|null.",
		"execute() on object|null.",
	} {
		if countIssueContaining(level8Issues, level8MethodNonObjectCode, expected) < 1 {
			t.Fatalf("expected %s diagnostic, got %#v", expected, level8Issues)
		}
	}
	if countIssueContaining(level8Issues, level8MethodNonObjectCode, "execute() on Service|null.") != 3 {
		t.Fatalf("expected nullable, union, and ternary Service|null diagnostics, got %#v", level8Issues)
	}
	if hasIssueContaining(level8Issues, level8MethodNonObjectCode, "optional()") ||
		hasIssueContaining(level8Issues, level8MethodNonObjectCode, "missing()") ||
		hasIssueContaining(level8Issues, level8MethodNonObjectCode, "on mixed") ||
		hasIssueContaining(level8Issues, level8MethodNonObjectCode, "on int") {
		t.Fatalf("level eight should not duplicate unknown, partial, mixed, or scalar cases, got %#v", level8Issues)
	}
	if hasIssueContaining(level8Issues, level8MethodNonObjectCode, "nullsafe") {
		t.Fatalf("nullsafe calls should remain clean, got %#v", level8Issues)
	}
}

func TestLevel8NegatedInstanceofOrDoesNotReportNullableMethod(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class User {
    public function getId(): string { return ''; }
}
function run(?User $employee, User $user): void {
    if (!$employee instanceof User || $employee->getId() !== $user->getId()) {
        return;
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	if hasIssueContaining(issues, level8MethodNonObjectCode, "getId") {
		t.Fatalf("negated instanceof || should narrow nullable receiver before getId(), got %#v", issues)
	}
}

func TestLevel8MethodExistsOnPropertyReceivers(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class DataHolder {
    public function toArray(): array { return []; }
}
class Holder {
    /** @var DataHolder|null */
    public $data;
    public function ternary(): array {
        return is_object($this->data) && method_exists($this->data, 'toArray') ? $this->data->toArray() : [];
    }
    public function guardedIf(): void {
        if (method_exists($this->data, 'toArray')) {
            $this->data->toArray();
        }
    }
}
class Rule { public function getX(): int { return 0; } }
class RuleHolder {
    /** @var Rule|null */
    public $rule;
    public function guarded(): void {
        if ($this->rule && method_exists($this->rule, 'getX')) {
            $this->rule->getX();
        }
    }
}
class Service { public function execute(): void {} }
function unguarded(?Service $s): void {
    $s->execute();
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 8)
	for _, unexpected := range []string{"toArray", "getX"} {
		if hasIssueContaining(issues, level8MethodNonObjectCode, unexpected) {
			t.Fatalf("guarded property method %q should stay clean at level eight, got %#v", unexpected, issues)
		}
	}
	if !hasIssueContaining(issues, level8MethodNonObjectCode, "execute") {
		t.Fatalf("unguarded nullable Service call should still report Level8.MethodNonObject, got %#v", issues)
	}
}

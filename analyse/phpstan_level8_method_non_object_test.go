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

	level7Issues := runPHPStanLevelOnFiles(t, files, 7)
	if hasIssueContaining(level7Issues, level8MethodNonObjectCode, "Cannot call method") {
		t.Fatalf("level seven should exclude level-eight nullable diagnostics, got %#v", level7Issues)
	}
	if hasIssueContaining(level7Issues, level2MethodNonObjectCode, "execute() on") {
		t.Fatalf("known nullable object methods should stay clean at level two, got %#v", level7Issues)
	}

	level8Issues := runPHPStanLevelOnFiles(t, files, 8)
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

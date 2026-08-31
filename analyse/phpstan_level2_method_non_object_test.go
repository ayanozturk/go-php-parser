package analyse

import "testing"

func TestLevel2NonObjectMethodReceivers(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class UnionService { public function execute(): void {} }

function run(
    int $int,
    string $string,
    array $array,
    callable $callable,
    bool $bool,
    float $float,
    iterable $iterable,
    object $object,
    mixed $mixed,
    UnionService|string $union,
    int|UnionService $intUnion,
    UnionService|null $nullable,
    int|null $nullableInt,
    false $false
): void {
    $int->missing();
    $string->missing();
    $array->missing();
    $callable->missing();
    $bool->missing();
    $float->missing();
    $iterable->missing();
    $object->missing();
    $mixed->missing();
    $union->execute();
    $union->missing();
    $intUnion->execute();
    $intUnion->missing();
    $nullable->missing();
    $nullableInt->missing();
    $false->missing();
}
`,
	}

	level1Issues := runPHPStanLevelOnFiles(t, files, 1)
	if hasIssueContaining(level1Issues, level2MethodNonObjectCode, "Cannot call method") {
		t.Fatalf("level one should exclude level-two non-object diagnostics, got %#v", level1Issues)
	}

	issues := runPHPStanLevelOnFiles(t, files, 2)
	for _, expected := range []string{
		"on string.",
		"on array.",
		"on callable.",
		"on bool.",
		"on float.",
		"on iterable.",
		"on false.",
	} {
		if countIssueContaining(issues, level2MethodNonObjectCode, expected) != 1 {
			t.Fatalf("expected one non-object diagnostic %q, got %#v", expected, issues)
		}
	}
	if countIssueContaining(issues, level2MethodNonObjectCode, "on int.") != 2 {
		t.Fatalf("expected int and nullable-int non-object diagnostics, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodNonObjectCode, "on UnionService|string.") != 1 &&
		countIssueContaining(issues, level2MethodNonObjectCode, "on string|UnionService.") != 1 {
		t.Fatalf("expected union class/string non-object diagnostic, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodNonObjectCode, "on int|UnionService.") != 1 &&
		countIssueContaining(issues, level2MethodNonObjectCode, "on UnionService|int.") != 1 {
		t.Fatalf("expected int/class union non-object diagnostic, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodNonObjectCode, "on object.") {
		t.Fatalf("object receivers should remain clean at level 2, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodNonObjectCode, "on mixed.") {
		t.Fatalf("mixed receivers should remain clean, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "UnionService::missing()") &&
		countIssueContaining(issues, level2MethodNonObjectCode, "missing() on") < 1 {
		t.Fatalf("class|scalar missing methods should use non-object, not existence, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodNonObjectCode, "execute() on") {
		t.Fatalf("known methods on class|scalar unions should remain clean, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "UnionService::execute()") {
		t.Fatalf("known nullable union methods should remain clean, got %#v", issues)
	}
}

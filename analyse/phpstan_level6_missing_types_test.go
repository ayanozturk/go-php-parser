package analyse

import "testing"

func TestLevel6MissingTypesUsePHPDocPrecedenceAndNestedTypes(t *testing.T) {
	const source = `<?php
/** @template TValue */
class Container {}

/** @param Container $container */
function missingGeneric(Container $container): void {}

function missingNativeArray(array $items): void {}

/** @param array $items */
function missingDocumentedArray(array $items): void {}

/** @return iterable */
function missingIterableReturn(): iterable { return []; }

/** @param array<int, Container> $items */
function missingNestedGeneric(array $items): void {}

/** @param Container<int> $container */
function cleanGeneric(Container $container): void {}

/** @param array<int, string> $items */
function cleanArray(array $items): void {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 6)
	want := map[string]int{
		level6MissingGenericTypeCode:  2,
		level6MissingIterableTypeCode: 3,
	}
	got := make(map[string]int)
	for _, issue := range issues {
		got[issue.Code]++
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("%s count = %d, want %d; issues: %#v", code, got[code], count, issues)
		}
	}
}

func TestLevel6MissingTypesAreNotReportedAtLevel5(t *testing.T) {
	const source = `<?php
/** @template TValue */
class Container {}
function inspect(Container $container, array $items): iterable { return []; }
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 5)
	for _, issue := range issues {
		if issue.Code == level6MissingGenericTypeCode || issue.Code == level6MissingIterableTypeCode {
			t.Fatalf("level-6 issue reported at level 5: %#v", issues)
		}
	}
}

func TestLevel6MissingDeclarationTypesRespectPHPDocAndSpecialMethods(t *testing.T) {
	const source = `<?php
final class State {
    public $missing;
    /** @var string */
    public $documented;

    public function __construct($value) { $this->documented = $value; }
    public function missing($value) { return $value; }
    /**
     * @param string $value
     * @return string
     */
    public function documented($value) { return $value; }
}

function missing($value) { return $value; }
function clean(mixed $value): void {}

$closure = function ($value) { return $value; };
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 6)
	want := map[string]int{
		level6MissingParameterCode: 3,
		level6MissingPropertyCode:  1,
		level6MissingReturnCode:    2,
	}
	got := make(map[string]int)
	for _, issue := range issues {
		got[issue.Code]++
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("%s count = %d, want %d; issues: %#v", code, got[code], count, issues)
		}
	}
}

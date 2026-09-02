package analyse

import "testing"

func TestLevel2PHPDocValidationMatchesSupportedFamilies(t *testing.T) {
	const source = `<?php
class KnownService {}

/** @param MissingParamService $service */
function unknownParam($service): void { echo $service; }

/** @return MissingReturnService */
function unknownReturn() { return new KnownService(); }

/** @param string $value */
function incompatibleParam(int $value): void { echo $value; }

/** @return string */
function incompatibleReturn(): int { return 1; }

/** @param string $missing */
function missingParam(int $value): void { echo $value; }

class PHPDocProperties {
    /** @var MissingPropertyService */
    public $service;

    /** @var string */
    public int $count;
}

/**
 * @param KnownService $service
 * @return KnownService
 */
function clean(KnownService $service): KnownService { return $service; }
`

	if issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 1); len(issues) != 0 {
		t.Fatalf("expected PHPDoc validation to stay disabled at level 1, got %#v", issues)
	}

	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	want := map[string]int{
		level2PHPDocClassCode:        3,
		level2PHPDocParamNameCode:    1,
		level2PHPDocParamTypeCode:    1,
		level2PHPDocPropertyTypeCode: 1,
		level2PHPDocReturnTypeCode:   1,
	}
	got := make(map[string]int)
	for _, issue := range issues {
		got[issue.Code]++
	}
	if len(issues) != 7 {
		t.Fatalf("expected seven supported PHPDoc issues, got %#v", issues)
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("%s count = %d, want %d; issues: %#v", code, got[code], count, issues)
		}
	}
}

func TestLevel2PHPDocValidationSkipsTemplateCompatibility(t *testing.T) {
	const source = `<?php
class Item {}

/**
 * @template T of Item
 * @param T $value
 * @return T
 */
function identity(Item $value): Item { return $value; }
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	for _, issue := range issues {
		switch issue.Code {
		case level2PHPDocClassCode, level2PHPDocParamTypeCode, level2PHPDocReturnTypeCode:
			t.Fatalf("expected bounded templates to remain conservative, got %#v", issues)
		}
	}
}

func TestLevel2PHPDocValidationAcceptsRefinedBuiltinForms(t *testing.T) {
	const source = `<?php
class Service {}

/** @param callable(): Service $factory */
function callableFactory(callable $factory): void { echo $factory(); }

/** @param class-string<Service> $class */
function className(string $class): void { echo $class; }

class Holder {
    /** @var callable(): Service */
    public $factory;
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	for _, issue := range issues {
		switch issue.Code {
		case level2PHPDocClassCode, level2PHPDocParamTypeCode, level2PHPDocPropertyTypeCode:
			t.Fatalf("expected refined builtin PHPDoc forms to satisfy native types, got %#v", issues)
		}
	}
}

func TestLevel2PHPDocValidationChecksNestedAndGenericTypes(t *testing.T) {
	const source = `<?php
/** @template T */
class Box {}

/**
 * @template T
 * @template U
 */
class Pair {}

class Plain {}
class KnownItem {}

/** @param Box<int, string> $box */
function tooMany(Box $box): void { echo get_class($box); }

/** @param Pair<int> $pair */
function tooFew(Pair $pair): void { echo get_class($pair); }

/** @param Plain<int> $plain */
function notGeneric(Plain $plain): void { echo get_class($plain); }

/** @param array<int, MissingNested> $items */
function nestedUnknown(array $items): void { echo count($items); }

/** @param Box<MissingGeneric> $box */
function genericUnknown(Box $box): void { echo get_class($box); }

/** @param Box<KnownItem> $box */
function clean(Box $box): void { echo get_class($box); }
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	want := map[string]int{
		level2PHPDocClassCode:       2,
		level2PHPDocGenericLessCode: 1,
		level2PHPDocGenericMoreCode: 1,
		level2PHPDocNotGenericCode:  1,
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
	if len(issues) != 5 {
		t.Fatalf("expected five nested/generic PHPDoc issues, got %#v", issues)
	}
}

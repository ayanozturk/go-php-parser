package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

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

func TestLevel2PHPDocValidationRecognizesClassTemplatesTypeAliasesAndLiteralTypes(t *testing.T) {
	const source = `<?php
class Item {}

/**
 * @template TValue of object
 */
abstract class Bag {
    /** @var list<TValue> */
    private array $items = [];

    /** @return TValue|null */
    public function first(): ?object { return $this->items[0] ?? null; }
}

/** @phpstan-type ItemList list<Item> */
final class Summary {
    /** @param ItemList $items */
    public function __construct(public array $items) {}
}

/** @return array{array{'low'}, array{'high'}} */
function priorities(): array { return [['low'], ['high']]; }

/** @return int<0, max> */
function nonNegative(): int { return 0; }
`
	parsed := parsePHPForLevel0(t, source)
	summary, ok := parsed[2].(*ast.ClassNode)
	if !ok || summary.PHPDoc == nil {
		t.Fatalf("expected local type alias PHPDoc on class, got %#v", parsed[2])
	}

	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	for _, issue := range issues {
		if issue.Code == level2PHPDocClassCode {
			t.Fatalf("expected templates, local aliases, literals, and integer range bounds not to be classes, got %#v", issues)
		}
	}
}

func TestLevel2PHPDocTypeAliasesExpandNestedImportedAndCallableTypes(t *testing.T) {
	const source = `<?php
namespace Source\Dto {
    class ImportedItem {}
}

namespace App {
    use Source\Dto\ImportedItem as ItemAlias;

    /**
     * @phpstan-type ItemList list<ItemAlias>
     * @phpstan-type ItemIndex array<string, ItemList>
     * @phpstan-type ItemFactory callable(ItemAlias): ItemIndex
     */
    class LocalTypes {
        /** @param ItemIndex $items */
        public function cleanIndex(array $items): void {}

        /** @param ItemFactory $factory */
        public function cleanFactory(callable $factory): void {}

        /** @param ItemList $items */
        public function incompatible(int $items): void {}

        /** @param ItemListSuffix $value */
        public function aliasNameBoundary($value): void {}
    }
}
`
	files := map[string]string{"phpdoc-alias.php": source}
	issues := runAnalysisLevelOnFiles(t, files, 2)
	want := map[string]int{
		level2PHPDocClassCode:     1,
		level2PHPDocParamTypeCode: 1,
	}
	got := make(map[string]int)
	for _, issue := range issues {
		got[issue.Code]++
	}
	if len(issues) != 2 || len(got) != len(want) {
		t.Fatalf("expected only alias suffix and native mismatch issues, got %#v", issues)
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("%s count = %d, want %d; issues: %#v", code, got[code], count, issues)
		}
	}
	if issues := runAnalysisLevelOnFiles(t, files, 1); len(issues) != 0 {
		t.Fatalf("level 2 alias diagnostics should stay disabled at level 1, got %#v", issues)
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

func TestLevel2PHPDocAllowsGenericTraversable(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"collection.php": `<?php
class DocModule {}

/**
 * @return Traversable<string, DocModule>
 */
function iterate(): Traversable {
    return new ArrayIterator([]);
}
`,
	}, 2)
	if hasIssueContaining(issues, level2PHPDocNotGenericCode, "Traversable is not generic") {
		t.Fatalf("Traversable should be a generic stub type, got %#v", issues)
	}
}

func TestLevel2PHPDocValidationChecksShapesCallablesAndTemplateBounds(t *testing.T) {
	const source = `<?php
class Animal {}
class Dog extends Animal {}
class Vehicle {}
class KnownInput {}
class KnownResult {}

/** @template TValue of Animal */
class Crate {}

/** @param array{service: MissingShapeService} $value */
function inspectShape(array $value): void {}

/** @param callable(MissingCallableInput): MissingCallableResult $callback */
function inspectCallable(callable $callback): void {}

/** @param Crate<Vehicle> $crate */
function inspectInvalidBound(Crate $crate): void {}

/** @param Crate<Dog> $crate */
function inspectValidBound(Crate $crate): void {}

/** @param array{callback: callable(KnownInput): KnownResult} $value */
function inspectKnownNested(array $value): void {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	want := map[string]int{
		level2PHPDocClassCode:        3,
		level2PHPDocGenericBoundCode: 1,
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
	if len(issues) != 4 {
		t.Fatalf("expected four nested/bound PHPDoc issues, got %#v", issues)
	}
}

func TestLevel2PHPDocTemplateVarianceChecksDirectCallablePositions(t *testing.T) {
	const source = `<?php
/** @template-covariant T */
class Producer {
    /** @return T|null */
    public function get() {}
    /** @param T|null $value */
    public function invalidSet($value): void {}
    /** @param TEntity $value */
    public function unrelated($value): void {}
    /** @template T @param T $value */
    public function localTemplate($value): void {}
}

/** @template-contravariant T */
class Consumer {
    /** @param T $value */
    public function consume($value): void {}
    /** @return T */
    public function invalidExpose() {}
}
class TEntity {}

class CallableVariance {
    /** @template-covariant T */
    public function invalidTemplateTag() {}
}
`
	files := map[string]string{"variance.php": source}
	issues := runAnalysisLevelOnFiles(t, files, 2)
	var violations []AnalysisIssue
	methodVarianceFound := false
	for _, issue := range issues {
		if issue.Code == level2PHPDocTemplateVarianceCode {
			violations = append(violations, issue)
		}
		methodVarianceFound = methodVarianceFound || issue.Code == level2PHPDocMethodVarianceCode
	}
	if len(issues) != 3 {
		t.Fatalf("expected only the two position violations and one invalid method tag, got %#v", issues)
	}
	if len(violations) != 2 {
		t.Fatalf("expected one covariant parameter and one contravariant return violation, got %#v", issues)
	}
	if !methodVarianceFound {
		t.Fatalf("variance tags on callable templates should be rejected, got %#v", issues)
	}
	if !strings.Contains(violations[0].Message, "covariant") && !strings.Contains(violations[1].Message, "covariant") {
		t.Fatalf("missing covariant parameter violation: %#v", violations)
	}
	if !strings.Contains(violations[0].Message, "contravariant") && !strings.Contains(violations[1].Message, "contravariant") {
		t.Fatalf("missing contravariant return violation: %#v", violations)
	}
	for _, issue := range runAnalysisLevelOnFiles(t, files, 1) {
		if issue.Code == level2PHPDocTemplateVarianceCode || issue.Code == level2PHPDocMethodVarianceCode {
			t.Fatalf("level 2 variance diagnostics should stay disabled at level 1, got %#v", issue)
		}
	}
}

func TestLevel2PHPDocTemplateVarianceComposesNestedGenericAndCallablePositions(t *testing.T) {
	const source = `<?php
/** @template-covariant T */
interface ReadBox {}
/** @template-contravariant T */
interface WriteBox {}
/** @template T */
interface InvariantBox {}

/** @template-covariant T */
class NestedVariance {
    /** @return ReadBox<T> */ public function readThroughCovariant() {}
    /** @param ReadBox<T> $value */ public function writeThroughCovariant($value): void {}
    /** @return WriteBox<T> */ public function invalidReadThroughContravariant() {}
    /** @param WriteBox<T> $value */ public function writeThroughContravariant($value): void {}
    /** @return InvariantBox<T> */ public function invalidInvariantReturn() {}
    /** @return callable(T): void */ public function invalidCallableInput() {}
    /** @param callable(T): void $callback */ public function acceptCallableInput($callback): void {}
    /** @param callable(): T $callback */ public function invalidCallableOutput($callback): void {}
    /** @return callable(): T */ public function callableOutput() {}
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"nested-variance.php": source}, 2)
	var varianceIssues []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == level2PHPDocTemplateVarianceCode {
			varianceIssues = append(varianceIssues, issue)
		}
	}
	if len(issues) != 5 || len(varianceIssues) != 5 {
		t.Fatalf("expected five composed variance violations and no other issues, got %#v", issues)
	}
}

func TestLevel2PHPDocGenericBoundsFollowVarianceAndInheritedSubstitutions(t *testing.T) {
	const source = `<?php
class Animal {}
class Dog extends Animal {}

/** @template-covariant T */
class ReadOnlyBox {}
/** @template-contravariant T */
class WriteOnlyBox {}
/** @template T */
class InvariantBox {}

/** @template T */
interface GenericParent {}
/** @template-covariant T */
interface CovariantParent {}
/** @template T @implements CovariantParent<T> */
class ChildParent implements CovariantParent {}
/** @implements CovariantParent<Dog> */
class DogChildParent implements CovariantParent {}
/** @template T @extends ChildParent<T> */
class GrandchildParent extends ChildParent implements CovariantParent {}

/** @template T of ReadOnlyBox<Animal> */
class ReadOnlyBound {}
/** @template T of WriteOnlyBox<Dog> */
class WriteOnlyBound {}
/** @template T of InvariantBox<Animal> */
class InvariantBound {}
/** @template T of CovariantParent<Animal> */
class ParentBound {}
/** @template T of CovariantParent<Animal> */
class ParentTemplateBound {}

/** @param ReadOnlyBound<ReadOnlyBox<Dog>> $value */
function covariantBoundPasses($value): void {}
/** @param WriteOnlyBound<WriteOnlyBox<Animal>> $value */
function contravariantBoundPasses($value): void {}
/** @param InvariantBound<InvariantBox<Dog>> $value */
function invariantBoundFails($value): void {}
/** @param ReadOnlyBound<mixed> $value */
function mixedBoundFails($value): void {}
/** @param ParentBound<DogChildParent> $value */
function fixedInheritedArgumentPasses($value): void {}
/** @param ParentTemplateBound<ChildParent<Dog>> $value */
function substitutedInheritedArgumentPasses($value): void {}
/** @param ParentTemplateBound<GrandchildParent<Dog>> $value */
function transitivelySubstitutedInheritedArgumentPasses($value): void {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"inherited-generics.php": source}, 2)
	var boundIssues []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == level2PHPDocGenericBoundCode {
			boundIssues = append(boundIssues, issue)
		}
	}
	if len(issues) != 2 || len(boundIssues) != 2 {
		t.Fatalf("expected only the invariant and mixed generic-bound mismatches, got %#v", issues)
	}
}

func TestLevel2PHPDocPropertyTypeDoesNotUsePrecedingConstantPHPDoc(t *testing.T) {
	const source = `<?php
class ShiftAssignment {
    /** @var list<string> */
    public const VALID_STATUSES = ['assigned'];

    #[Column]
    private ?string $assignmentId = null;

    /** @var list<string> */
    public const VALID_OUTCOMES = ['resolved'];

    public function __construct() {}
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": source}, 2)
	for _, issue := range issues {
		if issue.Code == level2PHPDocPropertyTypeCode {
			t.Fatalf("constant @var leaked onto following property: %#v", issues)
		}
	}
}

func TestPHPDocGenericSubtypeRecursionLimitIsConservative(t *testing.T) {
	project := BuildProjectIndex(map[string][]ast.Node{"types.php": parsePHPForProjectIndex(t, `<?php
class BaseType {}
class ChildType extends BaseType {}
/** @template-covariant T */
class NestedType {}
`)})
	wrap := func(base string, depth int) string {
		for i := 0; i < depth; i++ {
			base = "NestedType<" + base + ">"
		}
		return base
	}
	ctx := &AnalysisContext{Resolver: project}
	if !phpDocTypeIsSubtype(wrap("ChildType", phpDocGenericRelationDepthLimit-1), wrap("BaseType", phpDocGenericRelationDepthLimit-1), FileTypeContext{}, ctx) {
		t.Fatal("a nested generic subtype below the recursion limit should resolve")
	}
	if phpDocTypeIsSubtype(wrap("ChildType", phpDocGenericRelationDepthLimit), wrap("BaseType", phpDocGenericRelationDepthLimit), FileTypeContext{}, ctx) {
		t.Fatal("a nested generic subtype beyond the recursion limit should stop conservatively")
	}
}

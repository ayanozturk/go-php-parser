package analyse

import "testing"

func TestLevel2UnknownMethodsOnTypedReceivers(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class ParamService {}
class AssignedService {}
class NewService {}
class ChainService {}
class ChainFactory {
    public function service(): ChainService { return new ChainService(); }
}
class PropertyService {}
class PropertyHolder {
    public PropertyService $service;
}
class KnownService {
    public function execute(): void {}
}
class MagicService {
    public function __call(string $name, array $arguments): mixed { return null; }
}

function run(ParamService $param, PropertyHolder $holder, KnownService $known, MagicService $magic, mixed $mixed): void {
    $param->missing();
    $assigned = new AssignedService();
    $assigned->missing();
    $holder->service->missing();
    $known->execute();
    $magic->dynamicMethod();
    $mixed->unknown();
}

(new NewService())->missing();
(new ChainFactory())->service()->missing();
`,
	}

	level1Issues := runAnalysisLevelOnFiles(t, files, 1)
	if hasIssueContaining(level1Issues, level2MethodExistenceCode, "undefined method") {
		t.Fatalf("level one should exclude level-two method diagnostics, got %#v", level1Issues)
	}

	level2Issues := runAnalysisLevelOnFiles(t, files, 2)
	for _, expected := range []string{
		"ParamService::missing()",
		"AssignedService::missing()",
		"NewService::missing()",
		"ChainService::missing()",
		"PropertyService::missing()",
	} {
		if countIssueContaining(level2Issues, level2MethodExistenceCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, level2Issues)
		}
	}
	for _, unexpected := range []string{"KnownService::execute()", "MagicService::dynamicMethod()"} {
		if hasIssueContaining(level2Issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("unexpected %s diagnostic, got %#v", unexpected, level2Issues)
		}
	}
}

func TestLevel2UnknownMethodsOnFunctionAndConditionalReceivers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class FunctionService {}
class FirstBranch {}
class SecondBranch {}
class KnownFunctionService { public function execute(): void {} }

function makeService(): FunctionService { return new FunctionService(); }
function makeKnownService(): KnownFunctionService { return new KnownFunctionService(); }
function makeMixedService(): mixed { return null; }

function run(bool $flag): void {
    makeService()->missing();
    makeKnownService()->execute();
    makeMixedService()->dynamic();
    ($flag ? new FirstBranch() : new SecondBranch())->missing();
}
`,
	}, 2)

	for _, expected := range []string{
		"FunctionService::missing()",
		"FirstBranch|SecondBranch::missing()",
	} {
		if countIssueContaining(issues, level2MethodExistenceCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, issues)
		}
	}
	for _, unexpected := range []string{"KnownFunctionService::execute()", "dynamic()"} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("unexpected %s diagnostic, got %#v", unexpected, issues)
		}
	}
}

func TestLevel2UnknownMethodsResolveNamespacedFunctionReturns(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"service.php": `<?php
namespace Vendor;
class Service {}
function makeService(): Service { return new Service(); }
`,
		"same-namespace.php": `<?php
namespace Vendor;
makeService()->sameNamespaceMissing();
`,
		"fully-qualified.php": `<?php
namespace Consumer;
\Vendor\makeService()->fullyQualifiedMissing();
`,
	}, 2)

	for _, expected := range []string{
		"Vendor\\Service::sameNamespaceMissing()",
		"Vendor\\Service::fullyQualifiedMissing()",
	} {
		if countIssueContaining(issues, level2MethodExistenceCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, issues)
		}
	}
}

func TestLevel2UnknownMethodsOnCallableClosureAndDynamicReceivers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"vendor.php": `<?php
namespace Vendor;

class CallableService {}
class ClosureService {}
class ArrowService {}
class DynamicService {}
class AliasService {}
class StaleService {}
class StaleDynamicService {}
class KnownCallableService { public function execute(): void {} }
class KnownClosureService { public function execute(): void {} }
class KnownArrowService { public function execute(): void {} }
class KnownDynamicService { public function execute(): void {} }
`,
		"consumer.php": `<?php
namespace Consumer;

use Vendor\AliasService as ImportedCallableService;
use Vendor\ClosureService as ImportedClosureService;
use Vendor\ArrowService as ImportedArrowService;
use Vendor\DynamicService as ImportedDynamicService;
use Vendor\StaleService as ImportedStaleService;
use Vendor\KnownCallableService as ImportedKnownCallableService;
use Vendor\KnownClosureService as ImportedKnownClosureService;
use Vendor\KnownArrowService as ImportedKnownArrowService;
use Vendor\KnownDynamicService as ImportedKnownDynamicService;

/** @param callable(): ImportedCallableService $factory */
function callableReceiver(callable $factory): void {
    $factory()->missing();
}

/** @param callable(): ImportedKnownCallableService $factory */
function knownCallableReceiver(callable $factory): void {
    $factory()->execute();
}

function closureReceiver(): void {
    $factory = static function (): ImportedClosureService { return new ImportedClosureService(); };
    $factory()->missing();
    $known = static function (): ImportedKnownClosureService { return new ImportedKnownClosureService(); };
    $known()->execute();
}

function arrowReceiver(): void {
    $factory = fn(): ImportedArrowService => new ImportedArrowService();
    $factory()->missing();
    $known = fn(): ImportedKnownArrowService => new ImportedKnownArrowService();
    $known()->execute();
}

/** @param class-string<ImportedDynamicService> $class */
function dynamicReceiver(string $class): void {
    $value = new $class();
    $value->missing();
}

/** @param class-string<ImportedKnownDynamicService> $class */
function knownDynamicReceiver(string $class): void {
    $value = new $class();
    $value->execute();
}

/** @param callable(): ImportedStaleService $factory */
function reassignedReceiver(callable $factory): void {
    $factory = null;
    $factory()->missing();
}

/** @param class-string<\Vendor\StaleDynamicService> $class */
function reassignedDynamicReceiver(string $class): void {
    $class = 'runtime-value';
    $value = new $class();
    $value->missing();
}
`,
	}, 2)

	var methodIssues []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == level2MethodExistenceCode {
			methodIssues = append(methodIssues, issue)
		}
	}
	if len(methodIssues) != 4 {
		t.Fatalf("expected four unknown-method diagnostics, got %#v", methodIssues)
	}
	for _, expected := range []string{
		"Vendor\\AliasService::missing()",
		"Vendor\\ClosureService::missing()",
		"Vendor\\ArrowService::missing()",
		"Vendor\\DynamicService::missing()",
	} {
		if countIssueContaining(issues, level2MethodExistenceCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, methodIssues)
		}
	}
	for _, unexpected := range []string{
		"KnownCallableService::execute()",
		"KnownClosureService::execute()",
		"KnownArrowService::execute()",
		"KnownDynamicService::execute()",
		"Vendor\\StaleService::missing()",
		"Vendor\\StaleDynamicService::missing()",
	} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("unexpected known or stale-callable diagnostic containing %q, got %#v", unexpected, methodIssues)
		}
	}
}

func TestLevel2UnknownMethodsOnDirectCallableResultsAndTemplateClassString(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"services.php": `<?php
class Service {}
class KnownService { public function execute(): void {} }

class Holder {
    /** @var callable(): Service */
    public $factory;

    /** @return callable(): Service */
    public function make(): callable {}

    /** @return callable(): KnownService */
    public function knownMake(): callable {}
}

class KnownHolder {
    /** @var callable(): KnownService */
    public $factory;

    /** @return callable(): KnownService */
    public function make(): callable {}
}

/** @return callable(): Service */
function makeFactory(): callable {}

/** @return callable(): KnownService */
function makeKnownFactory(): callable {}
`,
		"consumer.php": `<?php
/**
 * @template T of Service
 * @param class-string<T> $class
 */
function run(Holder $holder, KnownHolder $knownHolder, string $class): void {
    ($holder->factory)()->missing();
    makeFactory()()->missing();
	$holder->make()()->missing();
	$functionFactory = makeFactory();
	$functionFactory()->missing();
	$methodFactory = $holder->make();
	$methodFactory()->missing();
	($knownHolder->factory)()->execute();
	makeKnownFactory()()->execute();
	$knownHolder->make()()->execute();
	$knownFunctionFactory = makeKnownFactory();
	$knownFunctionFactory()->execute();
	$knownMethodFactory = $knownHolder->make();
	$knownMethodFactory()->execute();
    $value = new $class();
    $value->missing();
}
`,
	}, 2)

	var methodIssues []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == level2MethodExistenceCode {
			methodIssues = append(methodIssues, issue)
		}
	}
	if len(methodIssues) != 6 {
		t.Fatalf("expected six unknown-method diagnostics, got %#v", methodIssues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "Service::missing()") != 6 {
		t.Fatalf("expected direct callable and template class-string calls to target Service, got %#v", methodIssues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "KnownService::execute()") {
		t.Fatalf("known direct callable results should remain clean, got %#v", methodIssues)
	}
}

func TestLevel2UnknownMethodsOnArrayShapeCallableResults(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class ShapeService {}
class KnownShapeService { public function execute(): void {} }

/** @param array{service: callable(): ShapeService, known?: callable(): KnownShapeService, 0: callable(): ShapeService} $factories */
function run(array $factories): void {
    $factory = $factories["service"];
    $factory()->missing();
    $factories["service"]()->missing();
    $copy = $factories;
    $copy[0]()->missing();
    $factories["known"]()->execute();
    $factories["missing"]()->dynamic();
}
`,
	}, 2)

	if countIssueContaining(issues, level2MethodExistenceCode, "ShapeService::missing()") != 3 {
		t.Fatalf("expected assigned, direct, and copied array-shape callable diagnostics, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "KnownShapeService::execute()") {
		t.Fatalf("known array-shape callable results should remain clean, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "dynamic()") {
		t.Fatalf("unknown array-shape keys should remain conservative, got %#v", issues)
	}
}

func TestLevel2UnknownMethodsOnNestedShapesAndRemainingReceivers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class NestedService {}
class KnownNestedService { public function execute(): void {} }
class ListService {}
class CloneService {}
class CoalesceService {}
class MatchLeft {}
class MatchRight {}
class NullsafeService {}

/**
 * @param array{inner: array{service: callable(): NestedService, known: callable(): KnownNestedService}} $nested
 * @param list{callable(): ListService} $list
 */
function run(array $nested, array $list, CloneService $clone, ?CoalesceService $coalesce, bool $flag, ?NullsafeService $nullsafe): void {
    $inner = $nested["inner"];
    $inner["service"]()->missing();
    $nested["inner"]["service"]()->missing();
    $nested["inner"]["known"]()->execute();
    $list[0]()->missing();
    (clone $clone)->missing();
    ($coalesce ?? new CoalesceService())->missing();
    (match ($flag) { true => new MatchLeft(), false => new MatchRight() })->missing();
    $nullsafe?->missing();
}
`,
	}, 2)

	if countIssueContaining(issues, level2MethodExistenceCode, "NestedService::missing()") != 2 {
		t.Fatalf("expected nested array-shape callable diagnostics, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "ListService::missing()") != 1 {
		t.Fatalf("expected list callable diagnostic, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "CloneService::missing()") != 1 {
		t.Fatalf("expected clone receiver diagnostic, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "CoalesceService::missing()") != 1 {
		t.Fatalf("expected coalesce receiver diagnostic, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "MatchLeft|MatchRight::missing()") != 1 {
		t.Fatalf("expected match receiver diagnostic, got %#v", issues)
	}
	if countIssueContaining(issues, level2MethodExistenceCode, "NullsafeService::missing()") != 1 {
		t.Fatalf("expected nullsafe receiver diagnostic, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "KnownNestedService::execute()") {
		t.Fatalf("known nested array-shape callable results should remain clean, got %#v", issues)
	}
}

func TestLevel2UnknownMethodsOnDynamicArrayShapeIndexes(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class AssignedService {}
class KnownAssignedService { public function execute(): void {} }
class UnionLeft {}
class UnionRight { public function execute(): void {} }
class ListItem {}
class KnownListItem { public function execute(): void {} }
class ConcatService {}
class NestedDynamic {}
class ConstService {}
class Holder {
    public const KEY = 'service';
    /**
     * @param array{service: callable(): ConstService} $factories
     */
    public function constKey(array $factories): void {
        $factories[self::KEY]()->missing();
        $factories[Holder::KEY]()->missing();
    }
}

/**
 * @param array{service: callable(): AssignedService, known: callable(): KnownAssignedService, extra: callable(): UnionLeft, ready: callable(): UnionRight} $factories
 * @param array{inner: array{service: callable(): NestedDynamic}} $nested
 * @param list{callable(): ListItem, callable(): KnownListItem} $list
 * @param array{service: callable(): AssignedService} $named
 */
function run(array $factories, array $nested, array $list, array $named, string $name, int $i, bool $flag): void {
    $key = "service";
    $factories[$key]()->missing();
    $knownKey = 'known';
    $factories[$knownKey]()->execute();
    $concat = "serv" . "ice";
    $factories[$concat]()->missing();
    $ternary = $flag ? "extra" : "ready";
    $factories[$ternary]()->missing();
    $factories[$name]()->missing();
    $factories[$name]()->execute();
    $innerKey = "inner";
    $nested[$innerKey]["service"]()->missing();
    $list[$i]()->missing();
    $list[$i]()->execute();
    $named[$i]()->missing();
    $factories["missing"]()->dynamic();
}
`,
	}, 2)

	want := map[string]int{
		"Call to an undefined method AssignedService::missing().":                                           2,
		"Call to an undefined method UnionLeft|UnionRight::missing().":                                      1,
		"Call to an undefined method AssignedService|KnownAssignedService|UnionLeft|UnionRight::missing().": 1,
		"Call to an undefined method NestedDynamic::missing().":                                             1,
		"Call to an undefined method KnownListItem|ListItem::missing().":                                    1,
		"Call to an undefined method ConstService::missing().":                                              2,
	}
	got := map[string]int{}
	for _, issue := range issues {
		if issue.Code == level2MethodExistenceCode {
			got[issue.Message]++
		}
	}
	for message, count := range want {
		if got[message] != count {
			t.Fatalf("expected %d %q, got %#v", count, message, issues)
		}
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "KnownAssignedService::execute()") {
		t.Fatalf("known assigned array-shape indexes should remain clean, got %#v", issues)
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "dynamic()") {
		t.Fatalf("unknown literal array-shape keys should remain conservative, got %#v", issues)
	}
	total := 0
	for _, count := range got {
		total += count
	}
	if total != 8 {
		t.Fatalf("expected eight unknown-method diagnostics, got %#v", issues)
	}
}

func TestLevel2UnknownMethodsOnRemainingExpressionReceivers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class ShapeService {}
class KnownShapeService { public function execute(): void {} }
class PropService {}
class MethodShapeService {}
class OtherConstService {}
class ListItem {}
class StaticPropService {}

const KEY = 'service';

class Other {
    public const KEY = 'service';
}

class Holder {
    /** @var array{service: callable(): PropService, known: callable(): KnownShapeService} */
    public array $factories;

    /**
     * @return array{service: callable(): MethodShapeService}
     */
    public function factories(): array {
        return ['service' => static fn (): MethodShapeService => new MethodShapeService()];
    }
}

class StaticHolder {
    /** @var array{service: callable(): StaticPropService} */
    public static array $factories;
}

/**
 * @param array{service: callable(): ShapeService, known: callable(): KnownShapeService} $factories
 * @param array{service: callable(): OtherConstService} $other
 * @param list{ListItem} $objects
 */
function run(array $factories, array $other, array $objects, Holder $holder, bool $flag): void {
    $factories[KEY]()->missing();
    $other[Other::KEY]()->missing();
    $factories[match ($flag) { true => 'service', false => 'known' }]()->missing();
    $holder->factories["service"]()->missing();
    $holder->factories["known"]()->execute();
    $holder->factories()["service"]()->missing();
    StaticHolder::$factories["service"]()->missing();
    $objects[0]->missing();
}
`,
	}, 2)

	want := map[string]int{
		"Call to an undefined method ShapeService::missing().":                   1,
		"Call to an undefined method OtherConstService::missing().":              1,
		"Call to an undefined method KnownShapeService|ShapeService::missing().": 1,
		"Call to an undefined method PropService::missing().":                    1,
		"Call to an undefined method MethodShapeService::missing().":             1,
		"Call to an undefined method StaticPropService::missing().":              1,
		"Call to an undefined method ListItem::missing().":                       1,
	}
	got := map[string]int{}
	for _, issue := range issues {
		if issue.Code == level2MethodExistenceCode {
			got[issue.Message]++
		}
	}
	for message, count := range want {
		if got[message] != count {
			t.Fatalf("expected %d %q, got %#v", count, message, issues)
		}
	}
	if hasIssueContaining(issues, level2MethodExistenceCode, "KnownShapeService::execute()") {
		t.Fatalf("known property array-shape methods should remain clean, got %#v", issues)
	}
}

func TestLevel2UnknownMethodHandlesMultiClassReceiversConservatively(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class FirstChoice {}
class SecondChoice { public function optional(): void {} }
class FirstMissing {}
class SecondMissing {}
interface LeftMissing {}
interface RightMissing {}
interface LeftKnown { public function available(): void; }
interface RightKnown {}
function run(FirstChoice|SecondChoice $choice, FirstMissing|SecondMissing $missing, LeftMissing&RightMissing $intersectionMissing, LeftKnown&RightKnown $intersectionKnown): void {
    $choice->optional();
    $missing->absent();
    $intersectionMissing->absent();
    $intersectionKnown->available();
}
`,
	}, 2)

	for _, expected := range []string{"FirstMissing|SecondMissing::absent()", "LeftMissing&RightMissing::absent()"} {
		if countIssueContaining(issues, level2MethodExistenceCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, issues)
		}
	}
	for _, unexpected := range []string{"optional()", "available()"} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("multi-class receiver should remain clean when one class provides %s, got %#v", unexpected, issues)
		}
	}
}

func TestLevel2UnknownMethodHandlesDNFAndNullableReceivers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
interface HasMethod { public function available(): void; }
interface FirstTag {}
interface SecondTag {}
class MissingAlternative {}
class NullableService {}
class KnownNullableService { public function execute(): void {} }

function run((HasMethod&FirstTag)|MissingAlternative $partiallyAvailable, (HasMethod&FirstTag)|(HasMethod&SecondTag) $availableEverywhere, ?NullableService $nullableMissing, KnownNullableService|null $nullableKnown, bool $flag): void {
    $partiallyAvailable->available();
    $availableEverywhere->available();
    $nullableMissing->missing();
    $nullableKnown->execute();
    ($flag ? new NullableService() : null)->missing();
}
`,
	}, 2)

	if countIssueContaining(issues, level2MethodExistenceCode, "NullableService::missing()") != 2 {
		t.Fatalf("expected nullable parameter and ternary diagnostics, got %#v", issues)
	}
	for _, unexpected := range []string{"partiallyAvailable", "available()", "KnownNullableService::execute()"} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("unexpected DNF/known nullable diagnostic containing %q: %#v", unexpected, issues)
		}
	}
}

func TestMethodReceiverRulesShareOneCachedWalk(t *testing.T) {
	filename := "test.php"
	nodes := parsePHPForLevel0(t, `<?php
class Service {}
function run(Service $service): void {
    $service->missing();
}
`)
	level := 10
	ctx := ensureLevel0Context(filename, nodes, &AnalysisContext{AnalysisLevel: &level})
	ctx = ensureArgCallDiagnostics(filename, nodes, ctx)
	if !ctx.hasMethodReceiverIssues {
		t.Fatal("expected the argument walk to collect method-receiver diagnostics")
	}
	primed := ctx.methodReceiverIssues
	existence := checkLevel2MethodExistence(filename, nodes, ctx)
	if len(existence) == 0 {
		t.Fatal("expected a level-2 unknown-method diagnostic")
	}
	cached := ctx.methodReceiverIssues
	if len(cached) != len(primed) || (len(primed) > 0 && &cached[0] != &primed[0]) {
		t.Fatal("the first method-receiver rule must reuse the argument walk result")
	}
	if len(checkLevel2MethodNonObject(filename, nodes, ctx)) != 0 {
		t.Fatal("typed class receivers should not emit level-2 non-object diagnostics")
	}
	if len(checkLevel7MethodUnion(filename, nodes, ctx)) != 0 {
		t.Fatal("single-class receivers should not emit level-7 partial-union diagnostics")
	}
	if len(checkLevel8MethodNonObject(filename, nodes, ctx)) != 0 {
		t.Fatal("non-nullable class receivers should not emit level-8 nullable diagnostics")
	}
	if len(ctx.methodReceiverIssues) != len(cached) || (len(cached) > 0 && &ctx.methodReceiverIssues[0] != &cached[0]) {
		t.Fatal("later method-receiver rules must reuse the cached walk result")
	}
}

func TestLevel2ResolvesTraitMethodsOnUsingClass(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"created.php": `<?php
namespace App\Entity\Traits;
trait CreatedDateTrait {
    public function getCreatedDate(): mixed { return null; }
}
`,
		"entity.php": `<?php
namespace App\Entity;
use App\Entity\Traits\CreatedDateTrait;
class InvitedUser {
    use CreatedDateTrait;
}
function run(InvitedUser $user): mixed {
    return $user->getCreatedDate();
}
`,
	}, 2)

	if hasIssueContaining(issues, level2MethodExistenceCode, "getCreatedDate") {
		t.Fatalf("trait methods should resolve on the using class, got %#v", issues)
	}
}

func TestLevel2InstanceofEarlyReturnNarrowsMethodReceiver(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
interface UserInterface {}
class User implements UserInterface {
    public function getCompany(): object { return new \stdClass(); }
}
class Controller {
    public function getUser(): ?UserInterface { return null; }

    public function me(): void {
        $user = $this->getUser();
        if (!$user instanceof User) {
            return;
        }
        $user->getCompany();
    }

    public function meParenthesized(): void {
        $user = $this->getUser();
        if (!($user instanceof User)) {
            return;
        }
        $user->getCompany();
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodExistenceCode, "getCompany") {
		t.Fatalf("instanceof early-return should narrow $user to User, got %#v", issues)
	}
}

func TestLevel2NegatedInstanceofOrNarrowsSameVariableMethodCall(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
interface UserInterface {}
class User implements UserInterface {
    public function isAdmin(): bool { return false; }
}
class Controller {
    public function getUser(): ?UserInterface { return null; }
    public function run(): void {
        $user = $this->getUser();
        if (!$user instanceof User || !$user->isAdmin()) {
            return;
        }
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodExistenceCode, "isAdmin") {
		t.Fatalf("negated instanceof || should narrow $user to User before isAdmin(), got %#v", issues)
	}
}

func TestLevel2NegatedInstanceofOrFailNarrowsNullableReceivers(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class DateTime {
    public function getTimestamp(): int { return 0; }
}
class ExampleTest {
    public function fail(string $message): void {}

    public function testWindow(): void {
        $capturedFrom = null;
        $capturedTo = null;
        if (!$capturedFrom instanceof DateTime || !$capturedTo instanceof DateTime) {
            $this->fail('Expected reminder window boundaries to be captured.');
        }
        $diffSeconds = $capturedTo->getTimestamp() - $capturedFrom->getTimestamp();
        echo $diffSeconds;
    }

    public function testWindowReturn(): void {
        $capturedFrom = null;
        $capturedTo = null;
        if (!$capturedFrom instanceof DateTime || !$capturedTo instanceof DateTime) {
            return;
        }
        $capturedTo->getTimestamp();
        $capturedFrom->getTimestamp();
    }

    public function testSingleFail(): void {
        $capturedFrom = null;
        if (!$capturedFrom instanceof DateTime) {
            $this->fail('missing from');
        }
        $capturedFrom->getTimestamp();
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getTimestamp") {
		t.Fatalf("negated instanceof || $this->fail() should narrow DateTime receivers, got %#v", issues)
	}
}

func TestLevel2AssertInstanceOfNarrowsNullableReceiver(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class PolicyUser {
    public function getRemainingAmount(): int { return 0; }
}
class ExampleTest {
    public function assertInstanceOf(string $expected, mixed $actual): void {}
    public function assertNotNull(mixed $actual): void {}
    public function assertTrue(mixed $condition): void {}

    public function testCaptured(): void {
        $capturedPolicyUser = null;
        $this->assertInstanceOf(PolicyUser::class, $capturedPolicyUser);
        $capturedPolicyUser->getRemainingAmount();
    }

    public function testNamed(): void {
        $capturedPolicyUser = null;
        $this->assertInstanceOf(actual: $capturedPolicyUser, expected: PolicyUser::class);
        $capturedPolicyUser->getRemainingAmount();
    }

    public function testStatic(): void {
        $capturedPolicyUser = null;
        self::assertInstanceOf(PolicyUser::class, $capturedPolicyUser);
        $capturedPolicyUser->getRemainingAmount();
    }

    public function testNotNull(?PolicyUser $user): void {
        $this->assertNotNull($user);
        $user->getRemainingAmount();
    }

    public function testAssertTrueInstanceof(): void {
        $capturedPolicyUser = null;
        $this->assertTrue($capturedPolicyUser instanceof PolicyUser);
        $capturedPolicyUser->getRemainingAmount();
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getRemainingAmount") {
		t.Fatalf("PHPUnit assertInstanceOf should narrow the captured variable, got %#v", issues)
	}
}

func TestLevel2OrShortCircuitNarrowsNullableReceiver(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class Manager {
    public function getId(): string { return ""; }
}
class User {
    public function getManager(): ?Manager { return null; }
}
class Controller {
    public function review(User $manager): void {
        $employee = null;
        if (!$employee || $employee->getManager()?->getId() !== $manager->getId()) {
            return;
        }
    }

    public function nullCompare(?User $employee, User $manager): void {
        if ($employee === null || $employee->getManager()?->getId() !== $manager->getId()) {
            return;
        }
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getManager") {
		t.Fatalf("|| short-circuit should treat $employee as non-null, got %#v", issues)
	}
}

func TestLevel2ForeachCollectionValueAndLoopAssignment(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
/**
 * @template TKey
 * @template TValue
 */
class Collection {}
class Manager {
    public function getId(): string { return ""; }
}
class User {
    public function getId(): string { return ""; }
    public function getManager(): ?Manager { return null; }
}

/** @return Collection<string, User> */
function reports(): Collection { return new Collection(); }

/** @return Collection<int, int> */
function ints(): Collection { return new Collection(); }

function assign(User $manager): void {
    $employee = null;
    foreach (reports() as $u) {
        $u->getId();
        $employee = $u;
        break;
    }
    if (!$employee || $employee->getManager()?->getId() !== $manager->getId()) {
        return;
    }
}

function numbers(): void {
    foreach (ints() as $n) {
        $n->getId();
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getManager") {
		t.Fatalf("foreach over Collection<User> should type $u and join $employee, got %#v", issues)
	}
	if !hasIssueContaining(issues, level2MethodNonObjectCode, "getId() on int") {
		t.Fatalf("foreach over Collection<int> should keep int values, got %#v", issues)
	}
}

func TestLevel2PhpdocMockIntersectionPropertyAllowsExpects(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
namespace PHPUnit\Framework\MockObject {
    interface MockObject {
        public function expects(mixed $matcher): MockObject;
        public function method(string $name): MockObject;
    }
}
namespace {
    use PHPUnit\Framework\MockObject\MockObject;
    class UserManagementService {
        public function updateUser(): void {}
    }
    class ExampleTest {
        /** @var UserManagementService&MockObject */
        private UserManagementService $userManagementService;
        public function testIt(): void {
            $this->userManagementService->expects(null)->method('updateUser');
        }
    }
}
`,
	}, 2)

	for _, unexpected := range []string{"expects", "method"} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("phpdoc mock intersection property should allow expects/method chain, got %#v", issues)
		}
	}
}

func TestLevel2MethodExistsGuardAllowsCall(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class Service {}

function run(Service $service): void {
    if (method_exists($service, 'optional')) {
        $service->optional();
    }
    $service->optional();
}
`,
	}, 2)

	if countIssueContaining(issues, level2MethodExistenceCode, "Service::optional()") != 1 {
		t.Fatalf("expected one unguarded optional() diagnostic, got %#v", issues)
	}
}

func TestLevel2AndMethodExistsGuardAllowsCall(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class Node {}

function run(?Node $previous): void {
    if ($previous && method_exists($previous, 'getSQLState')) {
        $previous->getSQLState();
    }
}
`,
	}, 2)

	if hasIssueContaining(issues, level2MethodExistenceCode, "getSQLState") {
		t.Fatalf("method_exists in && condition should allow guarded call, got %#v", issues)
	}
}

func TestLevel2DynamicMethodNameIsClean(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class Provider {}

function run(Provider $provider, string $method, mixed ...$args): void {
    $provider->$method(...$args);
}
`,
	}, 2)

	if hasIssueContaining(issues, level2MethodExistenceCode, "Provider::") {
		t.Fatalf("dynamic method name should stay silent, got %#v", issues)
	}
}

func TestLevel2InstanceofKeepsMockIntersectionMethods(t *testing.T) {
	files := map[string]string{
		"mock.php": `<?php
namespace PHPUnit\Framework\MockObject {
    interface MockObject {
        public function method(string $name): MockObject;
    }
}
namespace PHPUnit\Framework {
    use PHPUnit\Framework\MockObject\MockObject;
    abstract class TestCase {
        /**
         * @template RealInstanceType of object
         * @param class-string<RealInstanceType> $type
         * @return MockObject&RealInstanceType
         */
        final protected function createMock(string $type): MockObject {
            throw new \RuntimeException('stub');
        }
    }
}
`,
		"test.php": `<?php
class TemplateShift {
    public function getName(): string { return ''; }
}
final class ExampleTest extends \PHPUnit\Framework\TestCase {
    public function testIt(?string $shiftName): void {
        $templateShift = $shiftName !== null ? $this->createMock(TemplateShift::class) : null;
        if ($templateShift instanceof TemplateShift) {
            $templateShift->method('getName');
        }
    }
}
`,
	}

	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodExistenceCode, "method") {
		t.Fatalf("instanceof should keep mock intersection methods, got %#v", issues)
	}
}

func TestLevel2MethodExistsOnThisProperty(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class Invoice {}
class Controller {
    /** @var Invoice|null */
    public $invoice;
    public function run(): void {
        if (method_exists($this->invoice, 'optional')) {
            $this->invoice->optional();
        }
        $this->invoice->optional();
    }
}
`,
	}, 2)

	if countIssueContaining(issues, level2MethodExistenceCode, "optional()") != 1 {
		t.Fatalf("expected one unguarded optional() diagnostic, got %#v", issues)
	}
}

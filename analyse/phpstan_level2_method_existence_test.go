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

	level1Issues := runPHPStanLevelOnFiles(t, files, 1)
	if hasIssueContaining(level1Issues, level2MethodExistenceCode, "undefined method") {
		t.Fatalf("level one should exclude level-two method diagnostics, got %#v", level1Issues)
	}

	level2Issues := runPHPStanLevelOnFiles(t, files, 2)
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
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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

func TestLevel2UnknownMethodHandlesMultiClassReceiversConservatively(t *testing.T) {
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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
	issues := runPHPStanLevelOnFiles(t, map[string]string{
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

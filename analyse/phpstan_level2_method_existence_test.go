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

	for _, expected := range []string{"FirstMissing|SecondMissing::absent()", "LeftMissing|RightMissing::absent()"} {
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

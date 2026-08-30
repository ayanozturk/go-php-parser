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

func TestLevel2UnknownMethodSkipsUnionReceivers(t *testing.T) {
	issues := runPHPStanLevelOnFiles(t, map[string]string{
		"test.php": `<?php
class FirstChoice {}
class SecondChoice { public function optional(): void {} }
function run(FirstChoice|SecondChoice $choice): void {
    $choice->optional();
}
`,
	}, 2)

	if hasIssueContaining(issues, level2MethodExistenceCode, "optional") {
		t.Fatalf("union receivers remain conservative, got %#v", issues)
	}
}

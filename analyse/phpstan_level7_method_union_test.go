package analyse

import "testing"

func TestLevel7ReportsPartialUnionAndDNFMethods(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class FirstChoice {}
class SecondChoice {
    public function optional(): void {}
}
interface AvailableContract {
    public function execute(): void;
}
interface MarkerContract {}
class AlternativeChoice {}
interface HasMethod { public function available(): void; }
interface FirstTag {}
interface SecondTag {}
class MissingAlternative {}
class MagicService {
    public function __call(string $name, array $arguments): mixed { return null; }
}
class PlainService {}

function run(
    FirstChoice|SecondChoice $choice,
    (AvailableContract&MarkerContract)|AlternativeChoice $dnf,
    (HasMethod&FirstTag)|MissingAlternative $partiallyAvailable,
    (HasMethod&FirstTag)|(HasMethod&SecondTag) $availableEverywhere,
    HasMethod&FirstTag $intersection,
    MagicService|PlainService $magicUnion,
    mixed $mixed
): void {
    $choice->optional();
    $dnf->execute();
    $partiallyAvailable->available();
    $availableEverywhere->available();
    $intersection->available();
    $intersection->missing();
    $magicUnion->dynamic();
    $mixed->unknown();
}
`,
	}

	level2Issues := runPHPStanLevelOnFiles(t, files, 2)
	for _, unexpected := range []string{"optional()", "execute()", "available()", "dynamic()", "unknown()"} {
		if hasIssueContaining(level2Issues, level7MethodUnionCode, unexpected) {
			t.Fatalf("level two should exclude level-seven union diagnostics containing %q, got %#v", unexpected, level2Issues)
		}
	}
	if hasIssueContaining(level2Issues, level2MethodExistenceCode, "optional()") ||
		hasIssueContaining(level2Issues, level2MethodExistenceCode, "execute()") ||
		hasIssueContaining(level2Issues, level2MethodExistenceCode, "available()") {
		t.Fatalf("level two should stay silent for partial unions, got %#v", level2Issues)
	}
	if countIssueContaining(level2Issues, level2MethodExistenceCode, "FirstTag&HasMethod::missing()") != 1 {
		t.Fatalf("all-missing intersections remain a level-two diagnostic, got %#v", level2Issues)
	}

	level7Issues := runPHPStanLevelOnFiles(t, files, 7)
	for _, expected := range []string{
		"FirstChoice|SecondChoice::optional()",
		"(AvailableContract&MarkerContract)|AlternativeChoice::execute()",
		"(FirstTag&HasMethod)|MissingAlternative::available()",
		"MagicService|PlainService::dynamic()",
	} {
		if countIssueContaining(level7Issues, level7MethodUnionCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, level7Issues)
		}
	}
	if hasIssueContaining(level7Issues, level7MethodUnionCode, "(FirstTag&HasMethod)|(HasMethod&SecondTag)::available()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "FirstTag&HasMethod::available()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "FirstTag&HasMethod::missing()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "unknown()") {
		t.Fatalf("level seven should not duplicate all-missing, known, or mixed receivers, got %#v", level7Issues)
	}
}

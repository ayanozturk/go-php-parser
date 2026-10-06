package analyse

import "testing"

func TestLevel1CoreAcceptsMagicConstantsAndMatchDefault(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": `<?php
namespace Demo;

function magicConstants(): string {
    return __DIR__ . __FILE__ . __LINE__ . __NAMESPACE__ . __FUNCTION__;
}

function matchDefault(): string {
    return match (true) {
        default => 'fallback',
    };
}

class MagicConstantContext {
    public function values(): string { return __CLASS__ . __METHOD__; }
}

trait MagicConstantTrait {
    public function traitName(): string { return __TRAIT__; }
}
`}, 1)
	if hasIssueContaining(issues, level1CoreCode, "Constant") {
		t.Fatalf("PHP magic constants and a match default arm must not be treated as undefined constants, got %#v", issues)
	}
}

func TestLevel1ExtraArgumentsAndMagicMemberLevelGates(t *testing.T) {
	files := map[string]string{"test.php": `<?php
function takesOne(int $value): void {}
takesOne(1, 2);
class MagicMembers {
    public function __call(string $name, array $arguments): mixed { return null; }
    public function __get(string $name): mixed { return null; }
    public function run(): void { $this->missingMethod(); $value = $this->missingProperty; }
}
class NoConstructor {}
new NoConstructor(1);
`}
	levelZero := runAnalysisLevelOnFiles(t, files, 0)
	if hasIssueContaining(levelZero, level0InvocationCode, "at most") {
		t.Fatalf("level 0 should allow extra positional arguments, got %#v", levelZero)
	}
	if !hasIssueContaining(levelZero, level0InvocationCode, "does not have a constructor") {
		t.Fatalf("level 0 should report arguments to a class with no constructor, got %#v", levelZero)
	}
	if hasIssueContaining(levelZero, level0SymbolsCode, "undefined method MagicMembers::missingMethod") || hasIssueContaining(levelZero, level0SymbolsCode, "undefined property MagicMembers::$missingProperty") {
		t.Fatalf("level 0 should honor magic member fallback, got %#v", levelZero)
	}
	levelOne := runAnalysisLevelOnFiles(t, files, 1)
	if !hasIssueContaining(levelOne, level0InvocationCode, "at most") {
		t.Fatalf("level 1 should report extra arguments, got %#v", levelOne)
	}
	if !hasIssueContaining(levelOne, level0InvocationCode, "does not have a constructor") {
		t.Fatalf("level 1 should retain the no-constructor diagnostic, got %#v", levelOne)
	}
	if !hasIssueContaining(levelOne, level0SymbolsCode, "undefined method MagicMembers::missingMethod") || !hasIssueContaining(levelOne, level0SymbolsCode, "undefined property MagicMembers::$missingProperty") {
		t.Fatalf("level 1 should report unknown magic members, got %#v", levelOne)
	}
}

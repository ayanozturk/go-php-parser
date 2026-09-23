package analyse

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestArgCallCSTBranchJoinMatchesASTCompatibility(t *testing.T) {
	const source = `<?php
class Foo {}
function accept(Foo $value): void {}
function run(?Foo $value, bool $branch): void {
    if ($branch) {
        $value = new Foo();
    } else {
        $value = 'wrong';
    }
    accept($value);
}
`
	assertArgCallCSTParity(t, source, 9)
}

func TestArgCallCSTGuaranteedWhileExitMatchesASTCompatibility(t *testing.T) {
	const source = `<?php
class Worker {
    private const int LIMIT = 3;
    public function execute(): void {
        $attempt = 0;
        $lastFailure = null;
        while ($attempt < self::LIMIT) {
            try {
                $attempt++;
                return;
            } catch (Exception $failure) {
                $lastFailure = $failure;
            }
        }
        $lastFailure->getMessage();
    }
}
`
	assertArgCallCSTParity(t, source, 9)
}

func TestArgCallCSTWholeRuleFamilyMatchesASTCompatibility(t *testing.T) {
	const source = `<?php
class Service {
    public int $count;
    /** @deprecated use replacement() */
    public function old(int $value): void {}
    public function replacement(): void {}
}
function needsInt(int $value): void {}
function run(Service $service, int $value): void {
    $service->count = 'bad';
    $service->count += 'bad';
    1 + 'bad';
    $service->old();
    needsInt();
    $value->missing();
}
`
	assertArgCallCSTParity(t, source, 9)
}

func TestArgCallCSTNestedWhileAndGuardRefinementsMatchASTCompatibility(t *testing.T) {
	const source = `<?php
class Foo { public function run(): void {} }
class Worker {
    private ?string $Value = null;
    public function execute(bool $enabled, Foo|\stdClass|null $value): void {
        if ($enabled) {
            $attempt = 0;
            $lastFailure = null;
            while ($attempt < 1) {
                try {
                    $attempt++;
                    return;
                } catch (Exception $failure) {
                    $lastFailure = $failure;
                }
            }
            $lastFailure->getMessage();
        }
        if (null === $this->value) {
            $this->VALUE = 'ready';
        }
        if (!$value instanceof Foo) {
            return;
        }
        $value->run();
    }
}
`
	assertArgCallCSTParity(t, source, 9)
}

func assertArgCallCSTParity(t *testing.T, source string, level int) {
	t.Helper()
	nodes, diagnostics := syntax.ParseAST([]byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %v", diagnostics)
	}
	resolver := BuildProjectIndex(map[string][]ast.Node{"test.php": nodes})
	base := &AnalysisContext{Resolver: resolver, AnalysisLevel: &level}
	want := argCallFamilyIssues(RunAnalysisRulesWithContext("test.php", nodes, base))
	parsed := syntax.Parse([]byte(source))
	production := &AnalysisContext{Resolver: resolver, AnalysisLevel: &level, Content: []byte(source), Parsed: parsed}
	got := argCallFamilyIssues(RunAnalysisRulesWithContext("test.php", nodes, production))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CST production diagnostics differ from AST compatibility path\n got: %#v\nwant: %#v", got, want)
	}
}

func argCallFamilyIssues(issues []AnalysisIssue) []AnalysisIssue {
	var out []AnalysisIssue
	for _, issue := range issues {
		switch issue.Code {
		case "A.ARG.COUNT", "A.ARG.TYPE", "A.DEPRECATED.CALL", "A.ASSIGN.OP.INVALID", "A.BINARY.OP.INVALID", "A.PROP.TYPE":
			out = append(out, issue)
		default:
			if issue.Code == level2MethodNonObjectCode || issue.Code == level8MethodNonObjectCode || issue.Code == level7MethodUnionCode || issue.Code == level2MethodExistenceCode {
				out = append(out, issue)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Column != out[j].Column {
			return out[i].Column < out[j].Column
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}

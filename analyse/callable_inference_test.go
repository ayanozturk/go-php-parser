package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
	"testing"
)

func TestCallableContractsRespectVarianceAndArity(t *testing.T) {
	cases := []struct {
		expected, actual string
		mismatch         bool
	}{
		{"callable(int): string", "callable(mixed): non-empty-string", false},
		{"callable(mixed): string", "callable(int): string", true},
		{"callable(int): int", "callable(int): string", true},
		{"callable(int): void", "callable(): string", true},
		{"callable(): int", "callable(int): int", true},
		{"callable(): int", "callable(int=): int", false},
		{"callable(int=): int", "callable(int): int", true},
		{"callable(int, string): int", "callable(int...): int", true},
		{"callable(int, int): int", "callable(int...): int", false},
		{"callable(int): int", "callable(): int", false},
	}
	for _, c := range cases {
		t.Run(c.expected+" <- "+c.actual, func(t *testing.T) {
			e, ok := parseCallableContract(c.expected, FileTypeContext{}, nil)
			if !ok {
				t.Fatal("expected signature did not parse")
			}
			a, ok := parseCallableContract(c.actual, FileTypeContext{}, nil)
			if !ok {
				t.Fatal("actual signature did not parse")
			}
			if got := callableContractMismatch(e, a, nil, nil); got != c.mismatch {
				t.Fatalf("mismatch=%v, want %v", got, c.mismatch)
			}
		})
	}
}

func TestUnannotatedCallableCaptureAndParameterScopes(t *testing.T) {
	const source = `<?php
namespace Example;
/**
 * @template Value
 * @param (callable(): Value) $factory
 * @return Value
 */
function evaluate(callable $factory) { return $factory(); }
function acceptInt(int $value): void {}
function acceptString(string $value): void {}
/** @param callable(int,int):array $collect */
function collect(callable $collect): void {}
collect(fn(int ...$values) => $values);
function clean(int $number): int {
    $label = 'ready';
    $callback = function () use ($number) { return $number; };
    acceptInt(evaluate($callback));
    acceptString(evaluate(fn() => $label));
    $shadow = fn(int $label) => acceptInt($label);
    $modified = function () use ($label) { $label = 3; return $label; };
    acceptInt(evaluate($modified));
    acceptString($label);
    return evaluate(fn() => $number);
}
function wrong(): void {
    acceptInt(evaluate(fn() => 'wrong'));
    acceptInt(evaluate(function () { return 'wrong'; }));
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"scopes.php": source}, 5)
	var mismatches []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == "A.ARG.TYPE" || issue.Code == "A.RETURN.TYPE" {
			mismatches = append(mismatches, issue)
		}
	}
	if len(mismatches) != 2 {
		t.Fatalf("want only two string-return callback mismatches, got %#v", mismatches)
	}
	for _, issue := range mismatches {
		if issue.Line < 26 {
			t.Errorf("unexpected mismatch in clean scope: %#v", issue)
		}
	}
}

func TestCallableInferenceConservativeBoundaries(t *testing.T) {
	outer := newFunctionScope(nil, &ast.FunctionNode{}, FileTypeContext{})
	outer.setVariable("value", ParseType("string"))
	local := callableExpressionScope(&ast.FunctionNode{Params: []ast.Node{&ast.ParamNode{Name: "value"}}}, outer, nil)
	if typ, found := local.variable("value"); found && !typ.hasBuiltin("mixed") {
		t.Fatalf("uncaptured outer parameter leaked: %s", typ.String())
	}
	if _, known := inferCallableExpressionSignature(&ast.FunctionNode{Body: []ast.Node{&ast.WhileNode{}}}, outer, nil); known {
		t.Fatal("unsupported loop must remain unknown")
	}
	outer.callableInferenceDepth = callableInferenceDepthLimit
	if _, known := inferCallableExpressionSignature(&ast.ArrowFunctionNode{Expr: &ast.IntegerLiteral{Value: 1}}, outer, nil); known {
		t.Fatal("nesting limit must stop inference")
	}
	for _, raw := range []string{"callable(int& $value): int", "callable(&int): int", "not-a-signature", "callable(int)"} {
		if _, ok := parseCallableContract(raw, FileTypeContext{}, nil); ok {
			t.Errorf("unsupported contract parsed: %s", raw)
		}
	}
	for _, raw := range []string{`Closure(int): string`, `\Closure(int): string`} {
		if got := normalizeTypeWithContext(raw, FileTypeContext{Namespace: "Example"}); got != "Closure" {
			t.Errorf("Closure was namespaced: %s", got)
		}
	}
	if got := normalizedCallableContract(`Closure(int $value=, string...): string`, FileTypeContext{Namespace: "Example"}, nil); !strings.HasPrefix(got, "Closure(") {
		t.Fatal(got)
	}
}

func TestCallableInheritedRelativeTypesAndReturnAssertions(t *testing.T) {
	const source = `<?php
namespace Example;
class ParentFactory {
    /** @param \Closure(self): self $factory */
    public function accept(\Closure $factory): void {}
}
class ChildFactory extends ParentFactory {}
function acceptInt(int $value): void {}
function clean(ChildFactory $factory): void {
    $factory->accept(fn(ParentFactory $value): ParentFactory => $value);
}
function wrong(ChildFactory $factory): void {
    $factory->accept(fn(ChildFactory $value): ChildFactory => $value);
}
/** @template Value of int|string */
class ValueBox {
    public function asserted(mixed $value): int|string {
        /** @var Value */
        return $value;
    }
}
function rangeAssertion(mixed $value): int {
    /** @var int<0, max> */
    return $value;
}
function otherVariable(int $value): int {
    /** @var string $other */
    return $value;
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"inherited.php": source}, 5)
	var mismatches []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == "A.ARG.TYPE" || issue.Code == "A.RETURN.TYPE" {
			mismatches = append(mismatches, issue)
		}
	}
	if len(mismatches) != 1 || mismatches[0].Line != 13 {
		t.Fatalf("want only narrowing callback parameter mismatch, got %#v", mismatches)
	}
}

func TestCallableNormalizationPreservesGlobalNames(t *testing.T) {
	ft := FileTypeContext{Namespace: "Example"}
	for raw, want := range map[string]string{
		`\Closure|null`:            "Closure|null",
		`callable(int): bool|bool`: "callable|bool",
		`LocalItem`:                `Example\LocalItem`,
	} {
		if got := normalizeTypeWithContext(raw, ft); got != want {
			t.Errorf("normalize %q = %q, want %q", raw, got, want)
		}
		if got := normalizeTemplateAwareType(raw, ft, map[string]struct{}{"Value": {}}); got != want {
			t.Errorf("template normalize %q = %q, want %q", raw, got, want)
		}
	}
}

func TestInitializerCallbacksDoNotBindMethodResultTemplates(t *testing.T) {
	const source = `<?php
class Item { public function id(): int { return 1; } }
class Factory {
    /**
     * @template T of object
     * @param class-string<T> $class
     * @param Closure(T): void $initialize
     * @return T
     */
    public function create(string $class, Closure $initialize): object { return new $class; }
    /** @param Closure(self): self $transform */
    public function accept(Closure $transform): void {}
    public function local(): void { $this->accept(fn(self $value): self => $value); }
}
function useFactory(Factory $factory): int {
    return $factory->create(Item::class, function ($item) {})->id();
}
/** @param Closure():void|null $callback */
function nullable(?Closure $callback): void { $callback = $callback ?? function () {}; }
/** @param callable(int):bool|bool $callback */
function optionalCallback($callback): void {}
optionalCallback(true);
optionalCallback(fn(int $value): bool => true);
/**
 * @template Key of array-key
 * @param Key $key
 * @param callable(Key):int $convert
 */
function convert($key, callable $convert): int { return $convert($key); }
`
	for _, issue := range runAnalysisLevelOnFiles(t, map[string]string{"initializer.php": source}, 8) {
		if issue.Code == "A.ARG.TYPE" || issue.Code == "A.RETURN.TYPE" || issue.Code == "Level1.Core" || strings.Contains(issue.Code, "MethodNonObject") {
			t.Errorf("unexpected initializer/nullable/union callback mismatch: %#v", issue)
		}
	}
}

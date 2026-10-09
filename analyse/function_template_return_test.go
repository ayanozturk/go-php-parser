package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"testing"
)

func TestFunctionTemplateReturnBindsDirectArguments(t *testing.T) {
	const declarations = `<?php
namespace Helpers;
/**
 * @template Value
 * @param Value $value
 * @return Value
 */
function identity($value) { return $value; }
`
	const calls = `<?php
namespace Consumer;
use function Helpers\identity as preserve;
function acceptInt(int $value): void {}
function clean(int $value): int {
    acceptInt(preserve(value: $value));
    return preserve($value);
}
function wrong(): int {
    acceptInt(preserve('wrong'));
    return preserve('wrong');
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"helpers.php": declarations, "consumer.php": calls}, 5)
	var args, returns int
	for _, issue := range issues {
		if issue.Filename != "consumer.php" {
			continue
		}
		if issue.Code == "A.ARG.TYPE" {
			args++
			if issue.Line != 10 {
				t.Errorf("unexpected argument mismatch: %#v", issue)
			}
		}
		if issue.Code == "A.RETURN.TYPE" {
			returns++
			if issue.Line != 9 {
				t.Errorf("unexpected return mismatch: %#v", issue)
			}
		}
	}
	if args != 1 || returns != 1 {
		t.Fatalf("want one wrong argument and one wrong return, got %#v", issues)
	}
}

func TestFunctionTemplateReturnBindingBoundaries(t *testing.T) {
	function := ResolvedFunction{TemplateParams: []string{"Value"}, TemplateBounds: []string{"int|string"}, Params: []ResolvedParam{{Name: "values", Type: "Value", IsVariadic: true}}}
	scope := &functionScope{}
	scope.setVariable("number", ParseType("int"))
	scope.setVariable("text", ParseType("string"))
	bindings := bindCallSiteFunctionTemplates(function, []ast.Node{&ast.VariableNode{Name: "number"}, &ast.VariableNode{Name: "text"}}, scope, nil)
	if got := ParseType(bindings["Value"]); !got.Accepts(ParseType("int")) || !got.Accepts(ParseType("string")) || got.Accepts(ParseType("bool")) {
		t.Fatalf("repeated template arguments must join, got %q", bindings["Value"])
	}
	if got := bindCallSiteFunctionTemplates(function, nil, nil, nil)["Value"]; got != "int|string" {
		t.Fatalf("unbound template must retain its bound, got %q", got)
	}
	function.TemplateBounds = nil
	if got := bindCallSiteFunctionTemplates(function, []ast.Node{&ast.UnpackedArgumentNode{}}, nil, nil)["Value"]; got != "mixed" {
		t.Fatalf("unresolved unpacked template must stay unknown, got %q", got)
	}
	function.Params[0].Type = "array<Value>"
	if got := bindCallSiteFunctionTemplates(function, []ast.Node{&ast.VariableNode{Name: "number"}}, scope, nil)["Value"]; got != "mixed" {
		t.Fatalf("structured parameter must not bind to the whole argument, got %q", got)
	}
	function.TemplateParams = nil
	if got := bindCallSiteFunctionTemplates(function, nil, nil, nil); got != nil {
		t.Fatalf("non-generic function must not infer bindings: %#v", got)
	}
}

func TestFunctionTemplateReturnBindsCallableVariable(t *testing.T) {
	const source = `<?php
/**
 * @template Result
 * @param callable(): Result $factory
 * @return Result
 */
function evaluate(callable $factory) { return $factory(); }
function acceptInt(int $value): void {}
/** @param callable(): int $factory */
function clean(callable $factory): int {
    acceptInt(evaluate($factory));
    return evaluate($factory);
}
/** @param callable(): string $factory */
function wrong(callable $factory): int {
    acceptInt(evaluate($factory));
    return evaluate($factory);
}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"callable.php": source}, 5)
	var args, returns int
	for _, issue := range issues {
		if issue.Code == "A.ARG.TYPE" {
			args++
			if issue.Line != 16 {
				t.Errorf("unexpected argument mismatch: %#v", issue)
			}
		}
		if issue.Code == "A.RETURN.TYPE" {
			returns++
			if issue.Line != 15 {
				t.Errorf("unexpected return mismatch: %#v", issue)
			}
		}
	}
	if args != 1 || returns != 1 {
		t.Fatalf("want only the string-returning callable mismatches, got %#v", issues)
	}
}

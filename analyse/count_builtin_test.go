package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestLevel0BareCountWithTraitClassAlias(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
use Predis\Command\Traits\Count;

function run(array $xs): int
{
    return count($xs);
}
`,
	})
	if hasIssueContaining(issues, level0InvocationCode, "at most 0 allowed") {
		t.Fatalf("expected builtin count arity, got %#v", issues)
	}
	if hasIssueContaining(issues, level0SymbolsCode, "Function count not found") {
		t.Fatalf("expected builtin count to resolve, got %#v", issues)
	}
}

func TestResolveFunctionCountKeepsBuiltinParamsWithTraitClassAlias(t *testing.T) {
	idx := BuildProjectIndex(map[string][]ast.Node{
		"test.php": parsePHPForProjectIndex(t, `<?php
use Predis\Command\Traits\Count;

function run(array $xs): int
{
    return count($xs);
}
`),
	})
	fn, ok := idx.ResolveFunction("count")
	if !ok {
		t.Fatal("expected builtin count to resolve")
	}
	if len(fn.Params) == 0 {
		t.Fatalf("expected builtin count params, got %#v", fn)
	}
	if fn.Params[0].Name != "value" {
		t.Fatalf("unexpected first param: %#v", fn.Params[0])
	}
}

func TestProjectIndexFunctionDoesNotRegisterViaClassAlias(t *testing.T) {
	idx := BuildProjectIndex(map[string][]ast.Node{
		"test.php": parsePHPForProjectIndex(t, `<?php
use Predis\Command\Traits\Count;

function count() {}
`),
	})
	if _, ok := idx.ResolveFunction(`Predis\Command\Traits\Count`); ok {
		t.Fatal("user function count() must not be indexed under trait class alias FQN")
	}
	fn, ok := idx.ResolveFunction("count")
	if !ok {
		t.Fatal("expected global count function entry")
	}
	if fn.Declaration.File == "" {
		t.Fatal("expected project function declaration")
	}
	if len(fn.Params) == 0 {
		t.Fatal("expected seeded builtin params to be preserved for arity checks")
	}
}

func TestProjectIndexUserCountDoesNotClobberBuiltinSignature(t *testing.T) {
	idx := BuildProjectIndex(map[string][]ast.Node{
		"test.php": parsePHPForProjectIndex(t, `<?php
function count() {}
`),
	})
	fn, ok := idx.ResolveFunction("count")
	if !ok {
		t.Fatal("expected count function")
	}
	if fn.Declaration.File == "" {
		t.Fatal("expected project function declaration to win")
	}
	if len(fn.Params) == 0 {
		t.Fatal("expected builtin params preserved when user function has no signature")
	}
	if fn.Params[0].Name != "value" {
		t.Fatalf("unexpected param metadata: %#v", fn.Params)
	}

	issues := runLevel0OnFiles(t, map[string]string{
		"caller.php": `<?php
function caller(array $xs): int { return count($xs); }
`,
	})
	if hasIssueContaining(issues, level0InvocationCode, "at most 0 allowed") {
		t.Fatalf("expected count($xs) to stay valid after user count() decl, got %#v", issues)
	}
}

func TestResolveFunctionDeclarationNameSkipsImports(t *testing.T) {
	ctx := FileTypeContext{
		Namespace: "App",
		Aliases: map[string]string{
			"count": `Predis\Command\Traits\Count`,
		},
		FunctionAliases: map[string]string{
			"count": `Vendor\count`,
		},
	}
	got := ctx.resolveFunctionDeclarationName("count")
	if got != `App\count` {
		t.Fatalf("resolveFunctionDeclarationName(count) = %q, want App\\count", got)
	}
	if strings.Contains(got, "Predis") || strings.Contains(got, "Vendor") {
		t.Fatalf("import alias leaked into function declaration name: %q", got)
	}
}

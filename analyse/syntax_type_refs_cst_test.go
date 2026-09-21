package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestTypeRefUseCSTNativeMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"classUseSkipped": `<?php
use Foo\Bar;
use Baz\Qux as Alias;
use Vendor\Pkg\{One, Two as T};
`,
		"useFunctionConst": `<?php
use function Missing\missing_fn;
use const Missing\MISSING_CONST;
use function strlen;
`,
		"groupedMixed": `<?php
use Vendor\Pkg\{ClassA, function missing_fn, const MISSING_C};
`,
		"clean": `<?php
use function strlen;
use Foo\Bar;
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, guards, _ := buildTypeRefTestContext(t, filename, src)
			content := []byte(src)
			ctx.Content = content
			res := syntax.Parse(content)
			ctx.Parsed = res

			cst := sortIssuesForCompare(typeRefUseIssuesFromCST(filename, res, ctx, guards))
			lower := sortIssuesForCompare(typeRefUseIssuesViaLower(filename, res, ctx, guards))
			if len(cst) != len(lower) {
				t.Fatalf("count mismatch cst=%d lower=%d\ncst=%+v\nlower=%+v", len(cst), len(lower), cst, lower)
			}
			for i := range cst {
				if cst[i].Line != lower[i].Line || cst[i].Column != lower[i].Column ||
					cst[i].Message != lower[i].Message || cst[i].Code != lower[i].Code {
					t.Fatalf("issue %d mismatch:\ncst=%+v\nlower=%+v", i, cst[i], lower[i])
				}
			}
		})
	}
}

func typeRefUseIssuesFromCST(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if n.Kind() == syntax.KindUseDecl {
			appendTypeRefUseIssuesFromCST(filename, n, ctx, guards, &issues)
		}
	})
	return issues
}

func typeRefUseIssuesViaLower(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if n.Kind() == syntax.KindUseDecl {
			for _, u := range syntax.LowerUseDeclNode(n, res.File) {
				checkTypeReferenceOnNode(filename, u, ft, ctx, guards, &issues)
			}
		}
	})
	return issues
}

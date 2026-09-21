package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestTypeRefCatchCSTNativeMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"unknownCatch": `<?php
try { throw new Exception(); } catch (MissingCatch $e) { echo $e; }
`,
		"unionCatch": `<?php
try {} catch (MissingA|MissingB $e) {}
`,
		"traitCatch": `<?php
trait T {}
try {} catch (T $e) {}
`,
		"clean": `<?php
try {} catch (Exception $e) { echo $e->getMessage(); }
try {} catch (\Throwable $e) {}
`,
		"heavyBodySkipped": `<?php
try {} catch (MissingHeavy $e) {
    $a = [1,2,3,4,5];
    foreach ($a as $x) { $y = $x * 2; }
    function nested() { return 1; }
}
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

			cst := sortIssuesForCompare(typeRefCatchIssuesFromCST(filename, res, ctx, guards))
			lower := sortIssuesForCompare(typeRefCatchIssuesViaLower(filename, res, ctx, guards))
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

func typeRefCatchIssuesFromCST(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if n.Kind() == syntax.KindCatchClause {
			appendTypeRefCatchIssuesFromCST(filename, n, ft, ctx, guards, &issues)
		}
	})
	return issues
}

func typeRefCatchIssuesViaLower(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if n.Kind() == syntax.KindCatchClause {
			if catch := syntax.LowerCatchClauseNode(n, res.File); catch != nil {
				checkTypeReferenceOnNode(filename, catch, ft, ctx, guards, &issues)
			}
		}
	})
	return issues
}

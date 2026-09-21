package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestLanguageCallCSTNativeMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"invalidRegex": `<?php
preg_match('/(unclosed/', $subject);
`,
		"printfPlaceholderMismatch": `<?php
printf("%s and %s", "only one");
sprintf("%s", "fine");
`,
		"namedArgPrintf": `<?php
printf(format: "%s %s", "a");
`,
		"methodCallSkipped": `<?php
$o->preg_match('/(unclosed/', $x);
Foo::printf("%s %s", "a");
`,
		"variableCallSkipped": `<?php
$fn = 'printf';
$fn("%s %s", "a");
`,
		"clean": `<?php
preg_match('/ok/', $s);
sprintf("%s", "x");
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			content := []byte(src)
			res := syntax.Parse(content)
			cst := sortIssuesForCompare(languageCallIssuesFromCST(filename, res))
			lower := sortIssuesForCompare(languageCallIssuesViaLower(filename, res))
			if len(cst) != len(lower) {
				t.Fatalf("count mismatch cst=%d lower=%d\ncst=%+v\nlower=%+v", len(cst), len(lower), cst, lower)
			}
			for i := range cst {
				if cst[i].Line != lower[i].Line || cst[i].Column != lower[i].Column ||
					cst[i].EndLine != lower[i].EndLine || cst[i].EndColumn != lower[i].EndColumn ||
					cst[i].Message != lower[i].Message || cst[i].Code != lower[i].Code {
					t.Fatalf("issue %d mismatch:\ncst=%+v\nlower=%+v", i, cst[i], lower[i])
				}
			}
		})
	}
}

func languageCallIssuesFromCST(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindCallExpr {
			appendLanguageCallIssuesFromCST(filename, n, &issues)
		}
		return true
	})
	return issues
}

func languageCallIssuesViaLower(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindCallExpr {
			if lowered := syntax.LowerExprNode(n, res.File); lowered != nil {
				checkLanguageOnNode(filename, lowered, FileTypeContext{}, map[string]struct{}{}, nil, &issues)
			}
		}
		return true
	})
	return issues
}

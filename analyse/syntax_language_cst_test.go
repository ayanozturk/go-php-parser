package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestLanguageNonCallCSTNativeMatchesLowerPath(t *testing.T) {
	cases := map[string]string{
		"dupArrayKeys": `<?php
$a = ["x" => 1, "x" => 2];
$b = [1 => "a", 1 => "b"];
$c = ["ok" => 1, "other" => 2];
`,
		"incDecWritable": `<?php
$x++;
$a[0]++;
$o->p--;
(1 + 2)++;
foo()--;
`,
		"includeMissing": `<?php
include __DIR__ . "/nope.php";
require "definitely-missing-xyz.php";
include "definitely-missing-xyz.php";
`,
		"castVoidUnset": `<?php
$a = (void) $x;
$b = (unset) $y;
$c = (int) $z;
`,
		"gotoUndefined": `<?php
goto missing;
here:
goto here;
`,
		"clean": `<?php
$a = ["a" => 1, "b" => 2];
$x++;
$y = (int) $z;
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			content := []byte(src)
			res := syntax.Parse(content)
			cst := sortIssuesForCompare(languageNonCallIssuesFromCST(filename, res))
			lower := sortIssuesForCompare(languageNonCallIssuesViaLower(filename, res))
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

func languageNonCallIssuesFromCST(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []languageCSTGoto
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindSwitchStmt:
			return false
		case syntax.KindLabelStmt, syntax.KindGotoStmt,
			syntax.KindArrayExpr, syntax.KindUnaryExpr, syntax.KindIncludeExpr, syntax.KindCastExpr:
			appendLanguageNonCallIssuesFromCST(filename, n, labels, &gotos, &issues)
		}
		return true
	})
	appendUndefinedGotoIssuesFromCST(filename, labels, gotos, &issues)
	return issues
}

func languageNonCallIssuesViaLower(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []*ast.GotoNode
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindSwitchStmt:
			return false
		case syntax.KindLabelStmt, syntax.KindGotoStmt:
			if lowered := syntax.LowerStmtNode(n, res.File); lowered != nil {
				checkLanguageOnNode(filename, lowered, FileTypeContext{}, labels, &gotos, &issues)
			}
		case syntax.KindArrayExpr, syntax.KindUnaryExpr, syntax.KindIncludeExpr, syntax.KindCastExpr:
			if lowered := syntax.LowerExprNode(n, res.File); lowered != nil {
				checkLanguageOnNode(filename, lowered, FileTypeContext{}, labels, &gotos, &issues)
			}
		}
		return true
	})
	for _, goTo := range gotos {
		if _, ok := labels[goTo.Label]; !ok {
			issues = append(issues, issueSpan(filename, goTo, level0LanguageCode,
				"Goto to undefined label "+goTo.Label+"."))
		}
	}
	return issues
}

package style

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/overrides"
)

func TestFilterIssuesNilAndMatcher(t *testing.T) {
	issues := []StyleIssue{
		{Code: "PSR1.Classes.ClassDeclaration.PascalCase", SubjectKind: "class", SubjectName: "Legacy_Service", Message: "a"},
		{Code: "PSR1.Classes.ClassDeclaration.PascalCase", SubjectKind: "class", SubjectName: "ModernService", Message: "b"},
		{Code: "PSR12.Files.EndFileNewline", SubjectKind: "class", SubjectName: "Legacy_Service", Message: "c"},
	}
	if got := FilterIssues(issues, nil); !reflect.DeepEqual(got, issues) {
		t.Fatalf("nil matcher should pass through")
	}
	matcher, err := overrides.Compile(overrides.RuleOverrides{
		"PSR1.Classes.ClassDeclaration.PascalCase": {Classes: []string{"/^Legacy_/"}},
	})
	if err != nil || matcher == nil {
		t.Fatalf("compile: %v matcher=%v", err, matcher)
	}
	filtered := FilterIssues(issues, matcher)
	if len(filtered) != 2 {
		t.Fatalf("filtered=%#v", filtered)
	}
	for _, iss := range filtered {
		if iss.SubjectName == "Legacy_Service" && iss.Code == "PSR1.Classes.ClassDeclaration.PascalCase" {
			t.Fatalf("expected Legacy_Service PascalCase ignored: %#v", filtered)
		}
	}
}

func TestIssueCollectorWriteAndAppend(t *testing.T) {
	var issues []StyleIssue
	c := &IssueCollector{Issues: &issues}
	n, err := c.Write([]byte("ignored"))
	if err != nil || n != 7 {
		t.Fatalf("Write: n=%d err=%v", n, err)
	}
	c.Append(StyleIssue{Code: "X", Message: "one"})
	if len(issues) != 1 || issues[0].Code != "X" {
		t.Fatalf("Append: %#v", issues)
	}
	nilIssues := &IssueCollector{}
	nilIssues.Append(StyleIssue{Code: "Y"}) // must not panic
}

func TestPrintPHPCSStyleOutputPaths(t *testing.T) {
	var buf bytes.Buffer
	PrintPHPCSStyleOutputToWriter(&buf, nil)
	if !strings.Contains(buf.String(), "No style errors") {
		t.Fatalf("clean: %q", buf.String())
	}

	buf.Reset()
	issues := []StyleIssue{
		{Filename: "b.php", Line: 2, Column: 1, Type: Warning, Message: "warn", Code: "W.CODE", Fixable: false},
		{Filename: "a.php", Line: 1, Column: 2, Type: Error, Message: "err", Code: "E.CODE", Fixable: true},
		{Filename: "a.php", Line: 1, Column: 1, Type: Error, Message: "first", Code: ""},
		{Filename: "b.php", Line: 2, Column: 3, Type: Warning, Message: "warn2"},
	}
	PrintPHPCSStyleOutputToWriter(&buf, issues)
	out := buf.String()
	for _, want := range []string{
		"FILE: a.php",
		"FILE: b.php",
		"[x] err",
		"(E.CODE)",
		"FOUND 2 ERRORS AND 0 WARNING",
		"FOUND 0 ERRORS AND 2 WARNINGS",
		"Run summary: total style errors found: 2",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}

	// PrintPHPCSStyleOutput → stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	PrintPHPCSStyleOutput(nil)
	_ = w.Close()
	os.Stdout = old
	var stdoutBuf bytes.Buffer
	_, _ = stdoutBuf.ReadFrom(r)
	if !strings.Contains(stdoutBuf.String(), "No style errors") {
		t.Fatalf("stdout clean: %q", stdoutBuf.String())
	}

	var single bytes.Buffer
	PrintPHPCSStyleIssueToWriter(&single, StyleIssue{Line: 3, Type: Error, Message: "m", Code: "C", Fixable: true})
	if !strings.Contains(single.String(), "[x] m") || !strings.Contains(single.String(), "(C)") {
		t.Fatalf("single: %q", single.String())
	}
	if plural(1) != "" || plural(0) != "S" || plural(2) != "S" {
		t.Fatalf("plural: %q %q %q", plural(1), plural(0), plural(2))
	}
}

func TestGetFixerAndEndFileNewlineFix(t *testing.T) {
	if GetFixer("no-such-rule") != nil {
		t.Fatal("unknown fixer should be nil")
	}
	fixer := GetFixer("PSR12.Files.EndFileNewline")
	if fixer == nil {
		t.Fatal("EndFileNewline fixer not registered")
	}
	got := fixer.Fix("foo\n\n\n")
	if got != "foo\n" {
		t.Fatalf("Fix: %q", got)
	}
	got = EndFileNewlineFixer{}.Fix("bar\r\n\r\n")
	if got != "bar\n" {
		t.Fatalf("CRLF Fix: %q", got)
	}
}

func TestClassBraceAndClosingBraceFixers(t *testing.T) {
	openFixer := GetFixer("PSR12.Classes.OpenBraceOnOwnLine")
	if openFixer == nil {
		t.Fatal("open brace fixer missing")
	}
	in := "<?php\nclass Foo {\n    public $x;\n}\n"
	out := openFixer.Fix(in)
	if !strings.Contains(out, "class Foo\n{") && !strings.Contains(out, "class Foo\r\n{") {
		// FixClassBraceOnOwnLine inserts brace on own line
		if strings.Contains(out, "class Foo {") {
			t.Fatalf("expected brace moved off declaration line: %q", out)
		}
	}
	fixed := FixClassBraceOnOwnLine("class Bar {\n}")
	if strings.Contains(fixed, "class Bar {") {
		t.Fatalf("same-line brace should move: %q", fixed)
	}

	closeFixer := GetFixer("PSR12.Classes.ClosingBraceOnOwnLine")
	if closeFixer == nil {
		t.Fatal("closing brace fixer missing")
	}
	ugly := "<?php\nclass C\n{\n    public $a; }\n"
	fixedClose := closeFixer.Fix(ugly)
	if !strings.Contains(fixedClose, "}") {
		t.Fatalf("closing fix empty: %q", fixedClose)
	}
	// Code after closing brace on same line should be split
	split := FixClassClosingBraceOnOwnLine("class D\n{\n    public $a; } echo 1;\n")
	if !strings.Contains(split, "}\n") {
		t.Fatalf("expected closing brace on own line: %q", split)
	}
}

func TestRunSelectedRulesDefaultsAllAndTimings(t *testing.T) {
	content := []byte("<?php\nclass Foo {}\n")
	issues := RunSelectedRules("f.php", content, nil, nil)
	_ = issues // default subset may or may not fire

	RegisterRule("TEST.SLOW.RULE", func(string, []byte, []ast.Node) []StyleIssue {
		return []StyleIssue{{Code: "TEST.SLOW.RULE", Message: "slow", Type: Warning}}
	})
	RegisterRule("TEST.FAST.RULE", func(string, []byte, []ast.Node) []StyleIssue {
		return []StyleIssue{{Code: "TEST.FAST.RULE", Message: "fast", Type: Error}}
	})
	got := RunSelectedRules("f.php", content, nil, []string{"TEST.FAST.RULE", "PSR1.Methods.CamelCase", "TEST.SLOW.RULE", "missing.rule"})
	codes := map[string]bool{}
	for _, iss := range got {
		codes[iss.Code] = true
	}
	if !codes["TEST.FAST.RULE"] || !codes["TEST.SLOW.RULE"] {
		t.Fatalf("selected rules missing: %#v", got)
	}

	all := RunSelectedRules("f.php", content, nil, []string{"all"})
	if len(all) == 0 {
		// some rules may return empty on this fixture; timings still recorded
	}
	timings := GetRuleTimings()
	if len(timings) == 0 {
		t.Fatal("expected rule timings after RunSelectedRules")
	}
	if _, ok := timings["TEST.FAST.RULE"]; !ok {
		t.Fatalf("missing timing for TEST.FAST.RULE: %#v", timings)
	}
	_ = time.Duration(0)

	parallel := runRulesParallel([]string{"TEST.FAST.RULE", "missing"}, map[string]RuleFunc{
		"TEST.FAST.RULE": func(string, []byte, []ast.Node) []StyleIssue {
			return []StyleIssue{{Code: "TEST.FAST.RULE"}}
		},
	}, "f.php", content, nil)
	if len(parallel) != 1 {
		t.Fatalf("parallel: %#v", parallel)
	}
	if runRulesParallel(nil, nil, "f.php", content, nil) != nil {
		t.Fatal("empty parallel should be nil")
	}

	batches := createBatches([]string{"a", "b", "c", "d", "e"}, 2)
	if len(batches) != 3 || len(batches[2]) != 1 {
		t.Fatalf("batches=%#v", batches)
	}
	if createBatches(nil, 3) != nil && len(createBatches(nil, 3)) != 0 {
		t.Fatalf("empty batches")
	}
}

func TestDisallowLongArraySyntaxNestedChildren(t *testing.T) {
	inner := &ast.ArrayNode{
		Elements: nil,
		Pos:      ast.Position{Line: 5, Column: 1},
	}
	item := &ast.ArrayItemNode{
		Key:   &ast.StringLiteral{Value: "k", Pos: ast.Position{Line: 3, Column: 1}},
		Value: inner,
		Pos:   ast.Position{Line: 3, Column: 1},
	}
	kv := &ast.KeyValueNode{
		Key:   &ast.StringLiteral{Value: "outer", Pos: ast.Position{Line: 2, Column: 1}},
		Value: &ast.ArrayNode{Pos: ast.Position{Line: 4, Column: 1}},
		Pos:   ast.Position{Line: 2, Column: 1},
	}
	access := &ast.ArrayAccessNode{
		Var:   &ast.VariableNode{Name: "arr", Pos: ast.Position{Line: 6, Column: 1}},
		Index: &ast.ArrayNode{Pos: ast.Position{Line: 6, Column: 5}},
		Pos:   ast.Position{Line: 6, Column: 1},
	}
	root := &ast.ArrayNode{
		Elements: []ast.Node{item, kv},
		Pos:      ast.Position{Line: 1, Column: 1},
	}
	sniff := &DisallowLongArraySyntaxSniff{}
	sniff.Check([]ast.Node{root, access, nil}, "arr.php")
	if len(sniff.Issues) < 3 {
		t.Fatalf("expected nested long-array issues, got %#v", sniff.Issues)
	}
}

func TestNormalizeConstantNameAndClassConstants(t *testing.T) {
	sniff := &ClassConstantNameSniff{}
	got := sniff.normalizeConstantName("fooBar")
	if got == "" {
		t.Fatal("normalize empty")
	}
	got2 := sniff.normalizeConstantName("already_OK")
	_ = got2

	class := &ast.ClassNode{
		Name: "C",
		Constants: []ast.Node{
			&ast.ConstantNode{Name: "badName", Pos: ast.Position{Line: 2, Column: 1}},
			&ast.ConstantNode{Name: "GOOD_NAME", Pos: ast.Position{Line: 3, Column: 1}},
		},
		Methods: []ast.Node{
			&ast.FunctionNode{Name: "m"},
		},
		Pos: ast.Position{Line: 1, Column: 1},
	}
	issues := sniff.CheckIssues([]ast.Node{class, &ast.StringLiteral{Value: "x"}}, "c.php")
	if len(issues) != 1 || issues[0].Message == "" {
		t.Fatalf("expected one bad constant: %#v", issues)
	}
}

func TestBuilderPoolNilSafe(t *testing.T) {
	putBuilder(nil)
	b := getBuilder()
	b.WriteString("x")
	putBuilder(b)
}

func TestSplitLinesCached(t *testing.T) {
	lines := SplitLinesCached([]byte("a\nb\n"))
	if len(lines) < 2 {
		t.Fatalf("lines=%v", lines)
	}
}

func TestFunctionCallArgumentSpacingEdgeCases(t *testing.T) {
	checker := &FunctionCallArgumentSpacingChecker{}
	cases := []struct {
		line string
		want int
		msg  string
	}{
		{`foo(bar(1, 2), ` + "`x,y`" + `, 3);`, 0, "backticks ok"},
		{`foo(1, /* c, */ 2);`, 0, "block comment commas ignored"},
		{`foo(1, // trailing`, 0, "line comment after comma"},
		{`foo(1,# trailing`, 0, "hash comment"},
		{`foo(1, array(2, 3), 4);`, 0, "nested paren with good outer spacing"},
		{`foo(1,array(2, 3), 4);`, 1, "bad spacing before nested call"},
		{`foo(1, [2,3], 4);`, 0, "brackets nested ok spacing outer"},
		{`foo(1, {2,3}, 4);`, 0, "braces nested"},
		{`foo('a\'b', $x);`, 0, "escaped single quote"},
		{`foo("a\"b", $x);`, 0, "escaped double quote"},
		{`/* comment */`, 0, "comment-only line"},
		{`* doc`, 0, "doc star line"},
		{`foo(1,2`, 0, "unclosed paren"},
		{`$obj->method(1,2);`, 1, "method call bad spacing"},
		{`new Foo(1,2);`, 1, "new call"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues([]string{tc.line}, "t.php")
		if len(issues) != tc.want {
			t.Errorf("%s: want %d got %d (%+v) line=%q", tc.msg, tc.want, len(issues), issues, tc.line)
		}
	}

	fixer := FunctionCallArgumentSpacingFixer{}
	fixed := fixer.Fix("foo(1,2); // keep,this\nbar(1 ,  2);\n# ignore\n")
	if !strings.Contains(fixed, "foo(1, 2);") || !strings.Contains(fixed, "bar(1, 2);") {
		t.Fatalf("multi-line fix: %q", fixed)
	}
	// Unclosed paren left unchanged for that call
	if got := fixFunctionCallSpacingInLine("foo(1,2"); got != "foo(1,2" {
		t.Fatalf("unclosed: %q", got)
	}
	bad, _ := hasBadCommaSpacing("a , b")
	if !bad {
		t.Fatal("space before comma")
	}
	bad, _ = hasBadCommaSpacing("a,  b")
	if !bad {
		t.Fatal("double space after comma")
	}
	bad, _ = hasBadCommaSpacing("a,b")
	if !bad {
		t.Fatal("no space after comma")
	}
	bad, _ = hasBadCommaSpacing("a, b")
	if bad {
		t.Fatal("good spacing")
	}
	bad, _ = hasBadCommaSpacing("a, /*x*/ b")
	if bad {
		t.Fatal("block comment after comma")
	}
	if findMatchingParen("foo(1, bar(2))", 3) != 13 {
		t.Fatalf("nested matching paren: %d", findMatchingParen("foo(1, bar(2))", 3))
	}
	if findMatchingParen("foo(1, ')', 2)", 3) < 0 {
		t.Fatal("paren in string")
	}
	if findMatchingParen("foo(1 #", 3) != -1 {
		t.Fatal("hash aborts match")
	}
	parts := splitFunctionArguments("1, ...$rest, 3")
	if len(parts) < 2 {
		t.Fatalf("unpack split: %#v", parts)
	}
}

func TestDisallowMultipleStatementsQuotesCommentsHeredoc(t *testing.T) {
	sniff := &DisallowMultipleStatementsSniff{}
	cases := []struct {
		lines []string
		want  int
		msg   string
	}{
		{[]string{`$a = "x;y"; $b = 1;`}, 1, "semicolon in string then real"},
		{[]string{`$a = 'x;y'; $b = 1;`}, 1, "single-quoted"},
		{[]string{`echo 1; /* ; */ echo 2;`}, 1, "block comment between"},
		{[]string{`/* start`, `echo 1; echo 2;`, `*/`}, 0, "inside block comment"},
		{[]string{`$a = <<<EOT`, `x;y;z`, `EOT;`, `$b = 1; $c = 2;`}, 1, "after heredoc"},
		{[]string{`$a = 1; $b = 2; /* open`}, 1, "code then open block"},
	}
	for _, tc := range cases {
		got := sniff.CheckIssues(tc.lines, "t.php")
		if len(got) != tc.want {
			t.Errorf("%s: want %d got %d (%+v)", tc.msg, tc.want, len(got), got)
		}
	}
}

func TestElseIfDeclarationStringPosition(t *testing.T) {
	c := &ElseIfDeclarationChecker{}
	lines := []string{
		`if ($a) {} else if ($b) {}`,
		`$s = "else if";`,
		`$s = 'else if';`,
		`$s = "else if \"x"; else if ($y) {}`,
	}
	issues := c.CheckIssues(lines, "t.php")
	if len(issues) < 1 {
		t.Fatalf("expected else if issues, got %#v", issues)
	}
	fixed := FixElseIfDeclaration("if ($a) {} else if ($b) {}")
	if !strings.Contains(fixed, "elseif") || strings.Contains(fixed, "else if") {
		t.Fatalf("fix: %q", fixed)
	}
	if fixer := GetFixer(elseIfDeclarationCode); fixer == nil {
		t.Fatal("elseif fixer not registered")
	} else if got := fixer.Fix("else if ($x)"); !strings.Contains(got, "elseif") {
		t.Fatalf("fixer: %q", got)
	}
}

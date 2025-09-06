package style

import (
	"strings"
	"testing"
)

func runClassInstantiation(content string) []StyleIssue {
	return RunSelectedRules("test.php", []byte(content), nil, []string{psr1ClassInstantiationCode})
}

func countCI(issues []StyleIssue) int {
	c := 0
	for _, iss := range issues {
		if iss.Code == psr1ClassInstantiationCode {
			c++
		}
	}
	return c
}

func TestNewWithoutParenthesesIsError(t *testing.T) {
	php := "<?php\n$x = new Foo;\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", countCI(issues), issues)
	}
	if issues[0].Message == "" || !strings.Contains(issues[0].Message, "parentheses") {
		t.Fatalf("unexpected message: %+v", issues[0])
	}
}

func TestNewWithParenthesesIsOk(t *testing.T) {
	php := "<?php\n$x = new Foo();\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", countCI(issues), issues)
	}
}

func TestFQCNInstantiationIsOk(t *testing.T) {
	php := "<?php\n$x = new \\A\\B\\Foo();\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", countCI(issues), issues)
	}
}

func TestAnonymousClassIsIgnored(t *testing.T) {
	php := "<?php\n$x = new class { public function __construct() {} };\n$y = new class() {};\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues for anonymous classes, got %d: %v", countCI(issues), issues)
	}
}

func TestCommentAndWhitespaceBetweenNameAndParen(t *testing.T) {
	php := "<?php\n$x = new Foo /* c */ ();\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", countCI(issues), issues)
	}
}

func TestMultilineParenthesesAllowed(t *testing.T) {
	php := "<?php\n$x = new Foo\n();\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues for multiline parens, got %d: %v", countCI(issues), issues)
	}
}

func TestFalsePositivesAvoidedInCommentsAndStrings(t *testing.T) {
	php := "<?php\n// new Foo;\n/* new Bar; */\n$z = 'new Baz;';\n"
	issues := runClassInstantiation(php)
	if countCI(issues) != 0 {
		t.Fatalf("expected 0 issues inside comments/strings, got %d: %v", countCI(issues), issues)
	}
}

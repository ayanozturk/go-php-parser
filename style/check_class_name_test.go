package style

import (
	"bytes"

	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"os"
	"testing"
)

func TestClassNameCheckerCheckPrintsWarning(t *testing.T) {
	checker := &ClassNameChecker{}
	class := &ast.ClassNode{Name: "not_PascalCase"}
	var buf bytes.Buffer
	saved := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	checker.Check([]ast.Node{class}, "test.php")

	w.Close()
	os.Stdout = saved
	buf.ReadFrom(r)
	output := buf.String()

	expected := "\n\033[1m\033[31mClass Name Style Error\033[0m\n" +
		"  \033[34mFile   :\033[0m test.php\n" +
		"  \033[34mClass  :\033[0m not_PascalCase\n" +
		"  \033[34mLine   :\033[0m 0\n" +
		"  \033[34mColumn :\033[0m 0\n" +
		"  \033[33mReason :\033[0m Class name should be PascalCase\n\n"
	if output != expected {
		t.Errorf("unexpected output: got %q, want %q", output, expected)
	}
}

func TestClassNameCheckerCheckNoWarning(t *testing.T) {
	checker := &ClassNameChecker{}
	class := &ast.ClassNode{Name: "PascalCase"}
	var buf bytes.Buffer
	saved := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	checker.Check([]ast.Node{class}, "test.php")

	w.Close()
	os.Stdout = saved
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("expected no output for PascalCase class, got %q", output)
	}
}

func TestClassNameCheckerCheckIssues(t *testing.T) {
	checker := &ClassNameChecker{}
	badClass := &ast.ClassNode{Name: "not_PascalCase", Pos: ast.Position{Line: 10, Column: 5}}
	goodClass := &ast.ClassNode{Name: "PascalCase", Pos: ast.Position{Line: 20, Column: 2}}
	filename := "test.php"

	issues := checker.CheckIssues([]ast.Node{badClass, goodClass}, filename)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	issue := issues[0]
	if issue.Filename != filename {
		t.Errorf("expected filename %q, got %q", filename, issue.Filename)
	}
	if issue.Line != 10 || issue.Column != 5 {
		t.Errorf("expected line 10, column 5, got line %d, column %d", issue.Line, issue.Column)
	}
	if issue.Message != "Class name should be PascalCase" {
		t.Errorf("unexpected message: %q", issue.Message)
	}
	if issue.Code != "PSR1.Classes.ClassDeclaration.PascalCase" {
		t.Errorf("unexpected code: %q", issue.Code)
	}

	// Should not report issues for correct class name
	issues = checker.CheckIssues([]ast.Node{goodClass}, filename)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for PascalCase class, got %d", len(issues))
	}
}

func TestPascalCase(t *testing.T) {
	cases := map[string]string{
		"my_class":      "MyClass",
		"anotherTest":   "AnotherTest",
		"":              "",
		"snake_case":    "SnakeCase",
		"Pascal":        "Pascal",
		"AlreadyPascal": "AlreadyPascal",
	}
	for input, want := range cases {
		got := helper.PascalCase(input)
		if got != want {
			t.Errorf("PascalCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCamelCase(t *testing.T) {
	cases := map[string]string{
		"My_Class":     "myClass",
		"Another_Test": "anotherTest",
		"":             "",
		"snake_case":   "snakeCase",
		"Camel":        "camel",
		"alreadyCamel": "alreadyCamel",
	}
	for input, want := range cases {
		got := helper.CamelCase(input)
		if got != want {
			t.Errorf("CamelCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestToLowerToUpper(t *testing.T) {
	if helper.ToLower('A') != 'a' || helper.ToLower('Z') != 'z' || helper.ToLower('a') != 'a' {
		t.Error("ToLower failed")
	}
	if helper.ToUpper('a') != 'A' || helper.ToUpper('z') != 'Z' || helper.ToUpper('A') != 'A' {
		t.Error("ToUpper failed")
	}
}

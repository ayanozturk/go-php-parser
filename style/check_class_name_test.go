package style

import (
	"bytes"

	"go-phpcs/ast"
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
		got := pascalCase(input)
		if got != want {
			t.Errorf("pascalCase(%q) = %q, want %q", input, got, want)
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
		got := camelCase(input)
		if got != want {
			t.Errorf("camelCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestToLowerToUpper(t *testing.T) {
	if toLower('A') != 'a' || toLower('Z') != 'z' || toLower('a') != 'a' {
		t.Error("toLower failed")
	}
	if toUpper('a') != 'A' || toUpper('z') != 'Z' || toUpper('A') != 'A' {
		t.Error("toUpper failed")
	}
}

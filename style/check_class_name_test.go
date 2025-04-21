package style

import (
	"bytes"
	"fmt"
	"go-phpcs/ast"
	"os"
	"testing"
)

func TestClassNameChecker_Check_PrintsWarning(t *testing.T) {
	checker := &ClassNameChecker{}
	class := &ast.ClassNode{Name: "not_PascalCase"}
	var buf bytes.Buffer
	saved := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	checker.Check([]ast.Node{class})

	w.Close()
	os.Stdout = saved
	buf.ReadFrom(r)
	output := buf.String()

	expected := fmt.Sprintf("Class '%s' should be PascalCase\n", class.Name)
	if output != expected {
		t.Errorf("unexpected output: got %q, want %q", output, expected)
	}
}

func TestClassNameChecker_Check_NoWarning(t *testing.T) {
	checker := &ClassNameChecker{}
	class := &ast.ClassNode{Name: "PascalCase"}
	var buf bytes.Buffer
	saved := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	checker.Check([]ast.Node{class})

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
		"my_class":    "MyClass",
		"anotherTest": "AnotherTest",
		"":            "",
		"snake_case":  "SnakeCase",
		"Pascal":      "Pascal",
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
		"My_Class":    "myClass",
		"Another_Test": "anotherTest",
		"":            "",
		"snake_case":  "snakeCase",
		"Camel":       "camel",
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

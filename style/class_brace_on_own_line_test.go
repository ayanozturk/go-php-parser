package style

import "testing"

func TestClassBraceOnOwnLineChecker(t *testing.T) {
	checker := &ClassBraceOnOwnLineChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// Correct: brace on its own line
		{[]string{"class Foo", "{"}, 0, "class brace correct"},
		{[]string{"interface Bar", "{"}, 0, "interface brace correct"},
		{[]string{"trait Baz", "{"}, 0, "trait brace correct"},
		{[]string{"enum E", "{"}, 0, "enum brace correct"},
		// Incorrect: brace on same line
		{[]string{"class Foo {"}, 1, "class brace wrong line"},
		{[]string{"interface Bar {"}, 1, "interface brace wrong line"},
		{[]string{"trait Baz {"}, 1, "trait brace wrong line"},
		{[]string{"enum E {"}, 1, "enum brace wrong line"},
		// Incorrect: brace after whitespace
		{[]string{"class Foo", "   {"}, 1, "class brace with leading spaces"},
		// Correct: unrelated lines
		{[]string{"$a = 1;"}, 0, "not a class declaration"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

func TestFixClassBraceOnOwnLine(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "class brace on same line",
			input:    "class Foo {\n    public $a;\n}",
			expected: "class Foo\n{\n    public $a;\n}",
		},
		{
			name:     "interface brace on same line",
			input:    "interface Bar {\n    function baz();\n}",
			expected: "interface Bar\n{\n    function baz();\n}",
		},
		{
			name:     "trait brace on same line",
			input:    "trait Baz {\n    use SomeTrait;\n}",
			expected: "trait Baz\n{\n    use SomeTrait;\n}",
		},
		{
			name:     "enum brace on same line",
			input:    "enum E {\n    CASE_A,\n}",
			expected: "enum E\n{\n    CASE_A,\n}",
		},
		{
			name:     "already correct",
			input:    "class Foo\n{\n    public $a;\n}",
			expected: "class Foo\n{\n    public $a;\n}",
		},
		{
			name:     "unrelated code",
			input:    "$a = 1;\n$b = 2;",
			expected: "$a = 1;\n$b = 2;",
		},
	}
	for _, tc := range cases {
		output := FixClassBraceOnOwnLine(tc.input)
		if output != tc.expected {
			t.Errorf("%s: expected:\n%q\ngot:\n%q", tc.name, tc.expected, output)
		}
	}
}

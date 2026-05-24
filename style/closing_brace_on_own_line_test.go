package style

import (
	"go-phpcs/ast"
	"testing"
)

func TestClosingBraceOnOwnLineChecker(t *testing.T) {
	checker := &ClosingBraceOnOwnLineChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// Correct: class closing brace on its own line, followed by blank or EOF
		{[]string{"class Foo", "{", "    $a = 1;", "}", ""}, 0, "class brace on own line, blank after"},
		// Incorrect: code after class closing brace on same line
		{[]string{"class Foo", "{", "    $a = 1;", "} $b = 2;"}, 1, "code after class closing brace on same line"},
		// Incorrect: code on line after class closing brace
		{[]string{"class Foo", "{", "    $a = 1;", "}", "$b = 2;"}, 1, "code after class closing brace on next line"},
		// Correct: function closing brace on same line as code (should not flag)
		{[]string{"function bar() {", "    return 1;", "} // comment"}, 0, "function brace with comment after (not class)"},
		// Correct: compact single-line method body
		{[]string{"class Foo", "{", "    function bar() { return 1; }", "}", ""}, 0, "single-line method body is allowed"},
		// Correct: compact single-line typed methods
		{[]string{"class Foo", "{", "    public function getFile(): string { return \"\"; }", "", "    public function getLine(): int { return 0; }", "}", ""}, 0, "single-line typed methods are allowed"},
		// Incorrect: method closing brace on same line as previous statement
		{[]string{"class Foo", "{", "    public function bar(): void", "    {", "        $x = 1; }", "}", ""}, 1, "method closing brace must be on its own line"},
		// Correct: closing brace with whitespace
		{[]string{"    }", ""}, 0, "brace with leading spaces, blank after"},
		// Incorrect: class closing brace with trailing code, then blank
		{[]string{"class Foo", "{", "    $a = 1;", "    } $x = 1;", ""}, 1, "class brace with trailing code, blank after"},
		// Correct: inline docblock tags contain braces but are not PHP blocks.
		{[]string{"class Foo", "{", "    /**", "     * {@inheritdoc}", "     */", "    protected static $modules = [", "        'system',", "    ];", "}", ""}, 0, "docblock inline tag before static property"},
		// Correct: braces in comments and strings should not affect class depth.
		{[]string{"class Foo", "{", "    // comment with }", "    protected string $template = '{value}';", "}"}, 0, "comment and string braces ignored"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

func TestClosingBraceOnOwnLineInit(t *testing.T) {
	// This will trigger the init function and ensure the rule is registered
	issues := RunSelectedRules("test.php", []byte("}"), []ast.Node{}, []string{closingBraceOnOwnLineCode})
	if len(issues) == 0 {
		t.Errorf("Expected at least one issue for a lone closing brace, got none")
	}
}

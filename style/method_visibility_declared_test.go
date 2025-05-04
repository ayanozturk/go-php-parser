package style

import "testing"

const functionBar = "function bar() {}"

func TestMethodVisibilityDeclaredChecker(t *testing.T) {
	checker := &MethodVisibilityDeclaredChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// Correct: all methods have visibility
		{[]string{"class Foo", "{", "public function bar() {}", "protected function baz() {}", "private function qux() {}", "}"}, 0, "all methods have visibility"},
		// Incorrect: missing visibility
		{[]string{"class Foo", "{", functionBar, "public function baz() {}", "}"}, 1, "one method missing visibility"},
		{[]string{"class Foo", "{", functionBar, "function baz() {}", "}"}, 2, "two methods missing visibility"},
		// Correct: not a class
		{[]string{functionBar}, 0, "function outside class"},
		// Correct: anonymous function inside class (should not flag)
		{[]string{"class Foo", "{", "$cb = function ($x) { return $x; };", "}"}, 0, "anonymous function inside class should not flag"},
		// Correct: static anonymous function inside class
		{[]string{"class Foo", "{", "$cb = static function ($x) { return $x; };", "}"}, 0, "static anonymous function inside class should not flag"},
	}

	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

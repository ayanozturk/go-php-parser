package style

import (
	"io"
	"os"
	"testing"
)

func TestFunctionCallArgumentSpacingChecker(t *testing.T) {
	checker := &FunctionCallArgumentSpacingChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// OK: correct spacing
		{[]string{"foo(1, 2, 3);"}, 0, "correct spacing"},
		// Bad: multiple spaces after comma
		{[]string{"foo(1,  2, 3);"}, 1, "multiple spaces after comma"},
		// Bad: space before comma
		{[]string{"foo(1 ,2, 3);"}, 1, "space before comma"},
		// Bad: no space after comma
		{[]string{"foo(1,2, 3);"}, 1, "no space after comma"},
		// Bad: multiple errors
		{[]string{"foo( 1,2 ,  3 );"}, 1, "multiple errors in one call"},
		// OK: single argument
		{[]string{"foo($x);"}, 0, "single argument"},
		// OK: nested call
		{[]string{"foo(bar(1, 2), 3);"}, 0, "nested call with correct spacing"},
		// OK: commas inside strings are not argument separators
		{[]string{"foo('a,b', $x);"}, 0, "string literal comma ignored"},
		// OK: regex patterns containing commas should not trigger spacing issues
		{[]string{`preg_match('/^[A-Z0-9\\-,_]+$/i', $_COOKIE[$this->session->getName()]);`}, 0, "preg_match pattern comma ignored"},
		// OK: docblock example should be ignored
		{[]string{" *  {{ \"one,two,three,four,five\"|split(',', 3) }}"}, 0, "docblock example ignored"},
	}

	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

func TestFunctionCallArgumentSpacingFixer(t *testing.T) {
	fixer := FunctionCallArgumentSpacingFixer{}
	cases := []struct {
		input    string
		expected string
		msg      string
	}{
		// No change needed
		{"foo(1, 2, 3);", "foo(1, 2, 3);", "already correct"},
		// Multiple spaces after comma
		{"foo(1,  2, 3);", "foo(1, 2, 3);", "multiple spaces after comma"},
		// Space before comma
		{"foo(1 ,2, 3);", "foo(1, 2, 3);", "space before comma"},
		// No space after comma
		{"foo(1,2, 3);", "foo(1, 2, 3);", "no space after comma"},
		// Multiple errors
		{"foo( 1,2 ,  3 );", "foo( 1, 2, 3 );", "multiple errors in one call"},
		// Nested call
		{"foo(bar(1,2),3);", "foo(bar(1, 2), 3);", "nested call"},
		// String commas should not be treated as argument separators
		{"foo('a,b', $x);", "foo('a,b', $x);", "string literal comma unchanged"},
		// Regex patterns containing commas should not be reformatted
		{`preg_match('/^[A-Z0-9\\-,_]+$/i', $_COOKIE[$this->session->getName()]);`, `preg_match('/^[A-Z0-9\\-,_]+$/i', $_COOKIE[$this->session->getName()]);`, "preg_match pattern unchanged"},
		// Docblock example should not be touched
		{" *  {{ \"one,two,three,four,five\"|split(',', 3) }}", " *  {{ \"one,two,three,four,five\"|split(',', 3) }}", "docblock example unchanged"},
	}
	for _, tc := range cases {
		output := fixer.Fix(tc.input)
		if output != tc.expected {
			t.Errorf("%s: expected '%s', got '%s'", tc.msg, tc.expected, output)
		}
	}
}

func TestFunctionCallArgumentSpacingFixerDoesNotWriteDebugOutput(t *testing.T) {
	fixer := FunctionCallArgumentSpacingFixer{}
	var output string
	stderr := captureStderr(t, func() {
		output = fixer.Fix("foo(1,2,  3);")
	})

	if output != "foo(1, 2, 3);" {
		t.Fatalf("expected fixed output, got %q", output)
	}
	if stderr != "" {
		t.Fatalf("expected fixer to be silent on stderr, got %q", stderr)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = writePipe
	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("failed to close stderr pipe: %v", err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("failed to read stderr pipe: %v", err)
	}
	if err := readPipe.Close(); err != nil {
		t.Fatalf("failed to close stderr read pipe: %v", err)
	}

	return string(output)
}

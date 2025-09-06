package style

import (
	"strings"
	"testing"
)

func TestElseIfDeclarationChecker_CheckIssues(t *testing.T) {
	checker := NewElseIfDeclarationChecker()

	tests := []struct {
		name           string
		code           string
		expectedIssues int
		expectedLines  []int
		description    string
	}{
		{
			name: "valid elseif usage",
			code: `<?php
if ($condition) {
    echo "true";
} elseif ($other) {
    echo "other";
} else {
    echo "false";
}`,
			expectedIssues: 0,
			description:    "Should not report issues for correct elseif usage",
		},
		{
			name: "invalid else if with single space",
			code: `<?php
if ($condition) {
    echo "true";
} else if ($other) {
    echo "other";
}`,
			expectedIssues: 1,
			expectedLines:  []int{4},
			description:    "Should report issue for 'else if' with single space",
		},
		{
			name: "invalid else if with multiple spaces",
			code: `<?php
if ($condition) {
    echo "true";
} else   if ($other) {
    echo "other";
}`,
			expectedIssues: 1,
			expectedLines:  []int{4},
			description:    "Should report issue for 'else if' with multiple spaces",
		},
		{
			name: "invalid else if with tab",
			code: `<?php
if ($condition) {
    echo "true";
} else	if ($other) {
    echo "other";
}`,
			expectedIssues: 1,
			expectedLines:  []int{4},
			description:    "Should report issue for 'else if' with tab",
		},
		{
			name: "multiple else if violations",
			code: `<?php
if ($a) {
    echo "a";
} else if ($b) {
    echo "b";
} else if ($c) {
    echo "c";
} else {
    echo "default";
}`,
			expectedIssues: 2,
			expectedLines:  []int{4, 6},
			description:    "Should report multiple violations",
		},
		{
			name: "mixed valid and invalid",
			code: `<?php
if ($a) {
    echo "a";
} elseif ($b) {
    echo "b";
} else if ($c) {
    echo "c";
} else {
    echo "default";
}`,
			expectedIssues: 1,
			expectedLines:  []int{6},
			description:    "Should only report invalid usage, not valid elseif",
		},
		{
			name: "else if in comments should be ignored",
			code: `<?php
// This is a comment about else if
/* Another comment with else if */
if ($condition) {
    echo "true";
} elseif ($other) {
    echo "other";
}`,
			expectedIssues: 0,
			description:    "Should ignore else if in comments",
		},
		{
			name: "else if in string literals should be ignored",
			code: `<?php
$message = "Use else if carefully";
if ($condition) {
    echo "true";
} elseif ($other) {
    echo "other";
}`,
			expectedIssues: 0,
			description:    "Should ignore else if in string literals",
		},
		{
			name: "nested else if violations",
			code: `<?php
if ($outer) {
    if ($inner) {
        echo "inner true";
    } else if ($inner2) {
        echo "inner else if";
    }
} else if ($outer2) {
    echo "outer else if";
}`,
			expectedIssues: 2,
			expectedLines:  []int{5, 8},
			description:    "Should detect nested else if violations",
		},
		{
			name: "else if with complex conditions",
			code: `<?php
if ($a && $b) {
    echo "both";
} else if ($a || $b) {
    echo "either";
} else if (isset($c) && !empty($c)) {
    echo "c is set";
}`,
			expectedIssues: 2,
			expectedLines:  []int{4, 6},
			description:    "Should detect else if with complex conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.code, "\n")
			issues := checker.CheckIssues(lines, "test.php")

			if len(issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d. Issues: %+v", tt.expectedIssues, len(issues), issues)
				return
			}

			// Check that issues are on expected lines
			if len(tt.expectedLines) > 0 {
				for i, expectedLine := range tt.expectedLines {
					if i >= len(issues) {
						t.Errorf("Expected issue on line %d, but only got %d issues", expectedLine, len(issues))
						continue
					}
					if issues[i].Line != expectedLine {
						t.Errorf("Expected issue on line %d, got line %d", expectedLine, issues[i].Line)
					}
					if issues[i].Code != elseIfDeclarationCode {
						t.Errorf("Expected code %s, got %s", elseIfDeclarationCode, issues[i].Code)
					}
					if !issues[i].Fixable {
						t.Errorf("Expected issue to be fixable")
					}
				}
			}
		})
	}
}

func TestElseIfDeclarationChecker_MessageContent(t *testing.T) {
	checker := NewElseIfDeclarationChecker()
	
	testCases := []struct {
		name            string
		code            string
		expectedMessage string
	}{
		{
			name:            "single space",
			code:            "} else if ($condition) {",
			expectedMessage: "Use 'elseif' instead of 'else if'. Found   between 'else' and 'if'",
		},
		{
			name:            "multiple spaces",
			code:            "} else   if ($condition) {",
			expectedMessage: "Use 'elseif' instead of 'else if'. Found     between 'else' and 'if'",
		},
		{
			name:            "tab character",
			code:            "} else\tif ($condition) {",
			expectedMessage: "Use 'elseif' instead of 'else if'. Found \\t between 'else' and 'if'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lines := []string{tc.code}
			issues := checker.CheckIssues(lines, "test.php")
			
			if len(issues) != 1 {
				t.Fatalf("Expected 1 issue, got %d", len(issues))
			}
			
			if issues[0].Message != tc.expectedMessage {
				t.Errorf("Expected message: %q, got: %q", tc.expectedMessage, issues[0].Message)
			}
		})
	}
}

func TestFixElseIfDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single else if with space",
			input: `<?php
if ($condition) {
    echo "true";
} else if ($other) {
    echo "other";
}`,
			expected: `<?php
if ($condition) {
    echo "true";
} elseif ($other) {
    echo "other";
}`,
		},
		{
			name: "multiple else if with various spacing",
			input: `<?php
if ($a) {
    echo "a";
} else if ($b) {
    echo "b";
} else   if ($c) {
    echo "c";
} else	if ($d) {
    echo "d";
}`,
			expected: `<?php
if ($a) {
    echo "a";
} elseif ($b) {
    echo "b";
} elseif ($c) {
    echo "c";
} elseif ($d) {
    echo "d";
}`,
		},
		{
			name: "preserve valid elseif",
			input: `<?php
if ($a) {
    echo "a";
} elseif ($b) {
    echo "b";
} else if ($c) {
    echo "c";
}`,
			expected: `<?php
if ($a) {
    echo "a";
} elseif ($b) {
    echo "b";
} elseif ($c) {
    echo "c";
}`,
		},
		{
			name: "nested structures",
			input: `<?php
if ($outer) {
    if ($inner) {
        echo "inner";
    } else if ($inner2) {
        echo "inner2";
    }
} else if ($outer2) {
    echo "outer2";
}`,
			expected: `<?php
if ($outer) {
    if ($inner) {
        echo "inner";
    } elseif ($inner2) {
        echo "inner2";
    }
} elseif ($outer2) {
    echo "outer2";
}`,
		},
		{
			name:     "no changes needed",
			input:    `<?php\nif ($a) {\n    echo "a";\n} elseif ($b) {\n    echo "b";\n}`,
			expected: `<?php\nif ($a) {\n    echo "a";\n} elseif ($b) {\n    echo "b";\n}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixElseIfDeclaration(tt.input)
			if result != tt.expected {
				t.Errorf("FixElseIfDeclaration() failed\nInput:\n%s\nExpected:\n%s\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestElseIfDeclarationFixer(t *testing.T) {
	fixer := ElseIfDeclarationFixer{}
	
	if fixer.Code() != elseIfDeclarationCode {
		t.Errorf("Expected code %s, got %s", elseIfDeclarationCode, fixer.Code())
	}
	
	input := "if ($a) { } else if ($b) { }"
	expected := "if ($a) { } elseif ($b) { }"
	result := fixer.Fix(input)
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestElseIfDeclarationEdgeCases(t *testing.T) {
	checker := NewElseIfDeclarationChecker()

	tests := []struct {
		name           string
		code           string
		expectedIssues int
		description    string
	}{
		{
			name:           "empty line",
			code:           "",
			expectedIssues: 0,
			description:    "Should handle empty lines",
		},
		{
			name:           "only whitespace",
			code:           "   \t  ",
			expectedIssues: 0,
			description:    "Should handle whitespace-only lines",
		},
		{
			name:           "else without if",
			code:           "} else {",
			expectedIssues: 0,
			description:    "Should not match standalone else",
		},
		{
			name:           "if without else",
			code:           "if ($condition) {",
			expectedIssues: 0,
			description:    "Should not match standalone if",
		},
		{
			name:           "elseif as part of larger word",
			code:           "$elseifVariable = true;",
			expectedIssues: 0,
			description:    "Should not match elseif as part of variable name",
		},
		{
			name:           "else if as part of larger words",
			code:           "$elseVariable = $ifVariable;",
			expectedIssues: 0,
			description:    "Should not match when else/if are part of larger words",
		},
		{
			name:           "line comment with else if",
			code:           "// Use else if carefully",
			expectedIssues: 0,
			description:    "Should ignore else if in line comments",
		},
		{
			name:           "block comment with else if",
			code:           "/* This is about else if */",
			expectedIssues: 0,
			description:    "Should ignore else if in block comments",
		},
		{
			name:           "asterisk comment with else if",
			code:           "* Use else if in this case",
			expectedIssues: 0,
			description:    "Should ignore else if in asterisk comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := []string{tt.code}
			issues := checker.CheckIssues(lines, "test.php")

			if len(issues) != tt.expectedIssues {
				t.Errorf("%s: Expected %d issues, got %d. Issues: %+v", tt.description, tt.expectedIssues, len(issues), issues)
			}
		})
	}
}
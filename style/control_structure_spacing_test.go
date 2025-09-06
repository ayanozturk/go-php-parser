package style

import (
	"strings"
	"testing"
)

func TestControlStructureSpacingChecker_CheckIssues(t *testing.T) {
	checker := NewControlStructureSpacingChecker()
	
	tests := []struct {
		name           string
		code           string
		expectedIssues int
		expectedCodes  []string
	}{
		{
			name: "correct spacing",
			code: `<?php
if ($condition) {
    echo "test";
}
for ($i = 0; $i < 10; $i++) {
    echo $i;
}
while ($condition) {
    echo "loop";
}
myFunction($param);`,
			expectedIssues: 0,
			expectedCodes:  []string{},
		},
		{
			name: "missing space after if",
			code: `<?php
if($condition) {
    echo "test";
}`,
			expectedIssues: 1,
			expectedCodes:  []string{controlStructureSpacingCode},
		},
		{
			name: "missing space after for",
			code: `<?php
for($i = 0; $i < 10; $i++) {
    echo $i;
}`,
			expectedIssues: 1,
			expectedCodes:  []string{controlStructureSpacingCode},
		},
		{
			name: "missing space after while",
			code: `<?php
while($condition) {
    echo "loop";
}`,
			expectedIssues: 1,
			expectedCodes:  []string{controlStructureSpacingCode},
		},
		{
			name: "multiple spaces after control keyword",
			code: `<?php
if  ($condition) {
    echo "test";
}
for   ($i = 0; $i < 10; $i++) {
    echo $i;
}`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "space before function parenthesis",
			code: `<?php
myFunction ($param);
anotherFunc  ($param1, $param2);`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "missing space before brace",
			code: `<?php
if ($condition){
    echo "test";
}
for ($i = 0; $i < 10; $i++){
    echo $i;
}`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "multiple spaces before brace",
			code: `<?php
if ($condition)  {
    echo "test";
}
while ($condition)   {
    echo "loop";
}`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "mixed issues",
			code: `<?php
if($condition){
    echo "test";
}
for  ($i = 0; $i < 10; $i++)  {
    echo $i;
}
myFunction ($param);`,
			expectedIssues: 5,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode, controlStructureSpacingCode, controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "else and elseif",
			code: `<?php
if ($condition) {
    echo "if";
} else{
    echo "else";
}
if ($condition) {
    echo "if";
} elseif($other) {
    echo "elseif";
}`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "try catch finally",
			code: `<?php
try{
    riskyOperation();
} catch($e) {
    handleError($e);
} finally{
    cleanup();
}`,
			expectedIssues: 3,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "switch statement",
			code: `<?php
switch($value) {
    case 1:
        echo "one";
        break;
    default:
        echo "other";
}`,
			expectedIssues: 1,
			expectedCodes:  []string{controlStructureSpacingCode},
		},
		{
			name: "foreach loop",
			code: `<?php
foreach($array as $item) {
    echo $item;
}
foreach ($array as $key => $value)  {
    echo $key . ': ' . $value;
}`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "do while loop",
			code: `<?php
do{
    echo "loop";
} while($condition);`,
			expectedIssues: 2,
			expectedCodes:  []string{controlStructureSpacingCode, controlStructureSpacingCode},
		},
		{
			name: "comments should be ignored",
			code: `<?php
// if($condition) - this is a comment
/* for($i = 0; $i < 10; $i++) - this is also a comment */
if ($condition) {
    echo "test";
}`,
			expectedIssues: 0,
			expectedCodes:  []string{},
		},
		{
			name: "control keywords in strings should be ignored",
			code: `<?php
echo "if($condition) this is in a string";
$text = 'for($i = 0; $i < 10; $i++)';`,
			expectedIssues: 0,
			expectedCodes:  []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.code, "\n")
			issues := checker.CheckIssues(lines, "test.php")
			
			if len(issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectedIssues, len(issues))
				for i, issue := range issues {
					t.Errorf("Issue %d: Line %d, Column %d, Message: %s", i+1, issue.Line, issue.Column, issue.Message)
				}
			}
			
			for i, issue := range issues {
				if i < len(tt.expectedCodes) && issue.Code != tt.expectedCodes[i] {
					t.Errorf("Expected issue %d to have code %s, got %s", i, tt.expectedCodes[i], issue.Code)
				}
			}
		})
	}
}

func TestFixControlStructureSpacing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "fix missing space after control keywords",
			input: `<?php
if($condition) {
    echo "test";
}
for($i = 0; $i < 10; $i++) {
    echo $i;
}`,
			expected: `<?php
if ($condition) {
    echo "test";
}
for ($i = 0; $i < 10; $i++) {
    echo $i;
}`,
		},
		{
			name: "fix multiple spaces after control keywords",
			input: `<?php
if  ($condition) {
    echo "test";
}
while   ($condition) {
    echo "loop";
}`,
			expected: `<?php
if ($condition) {
    echo "test";
}
while ($condition) {
    echo "loop";
}`,
		},
		{
			name: "fix function call spacing",
			input: `<?php
myFunction ($param);
anotherFunc  ($param1, $param2);`,
			expected: `<?php
myFunction($param);
anotherFunc($param1, $param2);`,
		},
		{
			name: "fix brace spacing",
			input: `<?php
if ($condition){
    echo "test";
}
while ($condition)  {
    echo "loop";
}`,
			expected: `<?php
if ($condition) {
    echo "test";
}
while ($condition) {
    echo "loop";
}`,
		},
		{
			name: "mixed fixes",
			input: `<?php
if($condition){
    echo "test";
}
for  ($i = 0; $i < 10; $i++)  {
    myFunction ($i);
}`,
			expected: `<?php
if ($condition) {
    echo "test";
}
for ($i = 0; $i < 10; $i++) {
    myFunction($i);
}`,
		},
		{
			name: "preserve control keywords in function fixes",
			input: `<?php
if ($condition) {
    myFunction ($param);
}
for ($i = 0; $i < 10; $i++) {
    echo $i;
}`,
			expected: `<?php
if ($condition) {
    myFunction($param);
}
for ($i = 0; $i < 10; $i++) {
    echo $i;
}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixControlStructureSpacing(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestControlStructureSpacingFixer(t *testing.T) {
	fixer := ControlStructureSpacingFixer{}
	
	if fixer.Code() != controlStructureSpacingCode {
		t.Errorf("Expected fixer code %s, got %s", controlStructureSpacingCode, fixer.Code())
	}
	
	input := `<?php
if($condition){
    echo "test";
}`
	
	expected := `<?php
if ($condition) {
    echo "test";
}`
	
	result := fixer.Fix(input)
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestControlStructureSpacingRegistration(t *testing.T) {
	// Test that the rule is registered
	codes := ListRegisteredRuleCodes()
	found := false
	for _, code := range codes {
		if code == controlStructureSpacingCode {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Rule %s not found in registered rules", controlStructureSpacingCode)
	}
}

func TestNewControlStructureSpacingChecker(t *testing.T) {
	checker := NewControlStructureSpacingChecker()
	
	if checker == nil {
		t.Error("NewControlStructureSpacingChecker returned nil")
	}
	
	if len(checker.controlKeywords) == 0 {
		t.Error("Control keywords not initialized")
	}
	
	if checker.keywordRegex == nil {
		t.Error("Keyword regex not initialized")
	}
	
	if checker.functionRegex == nil {
		t.Error("Function regex not initialized")
	}
	
	// Test that expected keywords are present
	expectedKeywords := []string{"if", "else", "elseif", "for", "foreach", "while", "do", "switch", "try", "catch", "finally"}
	for _, expected := range expectedKeywords {
		found := false
		for _, actual := range checker.controlKeywords {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected keyword %s not found in control keywords", expected)
		}
	}
}
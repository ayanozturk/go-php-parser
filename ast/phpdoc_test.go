package ast

import (
	"testing"
)

func TestParsePHPDoc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PHPDocNode
	}{
		{
			name: "Simple PHPDoc with param and return",
			input: `/**
 * This is a test function
 * @param string $name The name parameter
 * @return int The return value
 */`,
			expected: PHPDocNode{
				Description: "This is a test function",
				Params: []PHPDocParam{
					{Name: "name", Type: "string", Description: "The name parameter"},
				},
				ReturnType: "int",
			},
		},
		{
			name: "PHPDoc with multiple params",
			input: `/**
 * Test function with multiple parameters
 * @param string $first First parameter
 * @param int $second Second parameter
 * @return bool
 */`,
			expected: PHPDocNode{
				Description: "Test function with multiple parameters",
				Params: []PHPDocParam{
					{Name: "first", Type: "string", Description: "First parameter"},
					{Name: "second", Type: "int", Description: "Second parameter"},
				},
				ReturnType: "bool",
			},
		},
		{
			name: "@var tag",
			input: `/**
 * @var string This is a string variable
 */`,
			expected: PHPDocNode{
				VarType: "string",
			},
		},
		{
			name:     "Empty PHPDoc",
			input:    `/** */`,
			expected: PHPDocNode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePHPDoc(tt.input)

			if result.Description != tt.expected.Description {
				t.Errorf("Description: got %q, want %q", result.Description, tt.expected.Description)
			}

			if result.ReturnType != tt.expected.ReturnType {
				t.Errorf("ReturnType: got %q, want %q", result.ReturnType, tt.expected.ReturnType)
			}

			if result.VarType != tt.expected.VarType {
				t.Errorf("VarType: got %q, want %q", result.VarType, tt.expected.VarType)
			}

			if len(result.Params) != len(tt.expected.Params) {
				t.Errorf("Params length: got %d, want %d", len(result.Params), len(tt.expected.Params))
			}

			for i, expectedParam := range tt.expected.Params {
				if i >= len(result.Params) {
					t.Errorf("Missing param at index %d", i)
					continue
				}
				actualParam := result.Params[i]
				if actualParam.Name != expectedParam.Name {
					t.Errorf("Param[%d].Name: got %q, want %q", i, actualParam.Name, expectedParam.Name)
				}
				if actualParam.Type != expectedParam.Type {
					t.Errorf("Param[%d].Type: got %q, want %q", i, actualParam.Type, expectedParam.Type)
				}
				if actualParam.Description != expectedParam.Description {
					t.Errorf("Param[%d].Description: got %q, want %q", i, actualParam.Description, expectedParam.Description)
				}
			}
		})
	}
}

func TestExtractPHPDocFromComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isPHPDoc bool
	}{
		{
			name:     "Valid PHPDoc",
			input:    "/** @return int */",
			isPHPDoc: true,
		},
		{
			name:     "Regular comment",
			input:    "// This is a comment",
			isPHPDoc: false,
		},
		{
			name:     "Block comment",
			input:    "/* This is a block comment */",
			isPHPDoc: false,
		},
		{
			name:     "PHPDoc without proper closing",
			input:    "/** @return int",
			isPHPDoc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPHPDocFromComment(tt.input)
			if tt.isPHPDoc && result == nil {
				t.Error("Expected PHPDoc but got nil")
			}
			if !tt.isPHPDoc && result != nil {
				t.Error("Expected nil but got PHPDoc")
			}
		})
	}
}

func TestGetParamTypeFromPHPDoc(t *testing.T) {
	phpdoc := &PHPDocNode{
		Params: []PHPDocParam{
			{Name: "userId", Type: "int"},
			{Name: "name", Type: "string"},
		},
	}

	tests := []struct {
		paramName string
		expected  string
	}{
		{"userId", "int"},
		{"name", "string"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.paramName, func(t *testing.T) {
			result := phpdoc.GetParamTypeFromPHPDoc(tt.paramName)
			if result != tt.expected {
				t.Errorf("GetParamTypeFromPHPDoc(%q): got %q, want %q", tt.paramName, result, tt.expected)
			}
		})
	}
}

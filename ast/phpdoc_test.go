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

func TestParsePHPDocPreservesWhitespaceInsideGenericTypes(t *testing.T) {
	doc := ParsePHPDoc(`/**
 * @param Map<string, Item> $items Items to process
 * @return Sequence<string, Item> Processed items
 * @var Bucket<int, Item>
 */`)

	if doc.ReturnType != "Sequence<string, Item>" {
		t.Fatalf("expected complete generic return type, got %q", doc.ReturnType)
	}
	if doc.VarType != "Bucket<int, Item>" {
		t.Fatalf("expected complete generic var type, got %q", doc.VarType)
	}
	if len(doc.Params) != 1 || doc.Params[0].Type != "Map<string, Item>" || doc.Params[0].Name != "items" {
		t.Fatalf("expected complete generic param type, got %#v", doc.Params)
	}
}

func TestParsePHPDocPreservesCallableParamReturnType(t *testing.T) {
	doc := ParsePHPDoc(`/**
 * @param callable(): A $factory Factory callback
 */`)

	if len(doc.Params) != 1 {
		t.Fatalf("expected one callable parameter, got %#v", doc.Params)
	}
	param := doc.Params[0]
	if param.Type != "callable(): A" {
		t.Fatalf("callable parameter type = %q, want %q", param.Type, "callable(): A")
	}
	if param.Name != "factory" || param.Description != "Factory callback" {
		t.Fatalf("unexpected callable parameter metadata: %#v", param)
	}
}

func TestParsePHPDocPreservesCallableReturnAndVarTypes(t *testing.T) {
	doc := ParsePHPDoc(`/**
 * @return callable(): Result Factory callback
 * @var callable(): Result
 */`)

	if doc.ReturnType != "callable(): Result" {
		t.Fatalf("callable return type = %q, want %q", doc.ReturnType, "callable(): Result")
	}
	if doc.VarType != "callable(): Result" {
		t.Fatalf("callable var type = %q, want %q", doc.VarType, "callable(): Result")
	}
}

func TestParsePHPDocPreservesArrayShapeCallableParamType(t *testing.T) {
	doc := ParsePHPDoc(`/**
 * @param array{service: callable(): ShapeService} $factories Factory map
 */`)

	if len(doc.Params) != 1 {
		t.Fatalf("expected one array-shape parameter, got %#v", doc.Params)
	}
	param := doc.Params[0]
	if param.Type != "array{service: callable(): ShapeService}" {
		t.Fatalf("array-shape parameter type = %q, want preserved shape", param.Type)
	}
	if param.Name != "factories" || param.Description != "Factory map" {
		t.Fatalf("unexpected array-shape parameter metadata: %#v", param)
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

func TestParsePHPDocGenerics(t *testing.T) {
	doc := ParsePHPDoc(`/**
 * @template T of EntityInterface
 * @template-covariant TValue
 * @template-extends BaseRepository<T>
 * @implements IteratorAggregate<string, list<TValue>>
 */`)

	if len(doc.Templates) != 2 || doc.Templates[0].Name != "T" || doc.Templates[0].Bound != "EntityInterface" || doc.Templates[1].Name != "TValue" {
		t.Fatalf("unexpected templates: %#v", doc.Templates)
	}
	if len(doc.Extends) != 1 || doc.Extends[0].Name != "BaseRepository" || len(doc.Extends[0].TypeArguments) != 1 || doc.Extends[0].TypeArguments[0] != "T" {
		t.Fatalf("unexpected extends references: %#v", doc.Extends)
	}
	if len(doc.Implements) != 1 || doc.Implements[0].Name != "IteratorAggregate" || len(doc.Implements[0].TypeArguments) != 2 || doc.Implements[0].TypeArguments[1] != "list<TValue>" {
		t.Fatalf("unexpected implements references: %#v", doc.Implements)
	}
}

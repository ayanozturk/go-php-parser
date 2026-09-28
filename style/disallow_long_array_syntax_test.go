package style

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"testing"
)

func TestDisallowLongArraySyntaxSniff(t *testing.T) {
	sniff := &DisallowLongArraySyntaxSniff{}
	// Long array syntax: array(...)
	longArray := &ast.ArrayNode{
		Elements: []ast.Node{
			&ast.StringLiteral{Value: "foo", Pos: ast.Position{Line: 2, Column: 10}},
		},
		Pos: ast.Position{Line: 2, Column: 1},
	}
	// Some other node type (not an array)
	otherNode := &ast.StringLiteral{Value: "bar", Pos: ast.Position{Line: 3, Column: 1}}

	sniff.Check([]ast.Node{longArray, otherNode}, "test.php")

	if len(sniff.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(sniff.Issues))
	}
	issue := sniff.Issues[0]
	if issue.Line != 2 || issue.Column != 1 {
		t.Errorf("Expected issue at 2:1, got %d:%d", issue.Line, issue.Column)
	}
	if issue.EndLine != 2 || issue.EndColumn != 6 {
		t.Errorf("Expected exact array keyword range [1,6), got %d:%d to %d:%d", issue.Line, issue.Column, issue.EndLine, issue.EndColumn)
	}
	if issue.Type != "ERROR" {
		t.Errorf("Expected issue type ERROR, got %s", issue.Type)
	}
	if issue.Code != "Generic.Arrays.DisallowLongArraySyntax" {
		t.Errorf("Unexpected code: %s", issue.Code)
	}
}

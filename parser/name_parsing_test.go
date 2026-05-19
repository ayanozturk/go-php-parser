package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseQualifiedNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "<?php use User;",
			expected: "User",
		},
		{
			name:     "fully qualified name",
			input:    "<?php use \\App\\Models\\User;",
			expected: "\\App\\Models\\User",
		},
		{
			name:     "partially qualified name",
			input:    "<?php use App\\Models\\User;",
			expected: "App\\Models\\User",
		},
		{
			name:     "name with static",
			input:    "<?php use static\\Foo;",
			expected: "static\\Foo",
		},
		{
			name:     "name with self",
			input:    "<?php use self\\Foo;",
			expected: "self\\Foo",
		},
		{
			name:     "name with parent",
			input:    "<?php use parent\\Foo;",
			expected: "parent\\Foo",
		},
		{
			name:     "trailing slash handled safely",
			input:    "<?php use App\\Models\\User\\;",
			expected: "App\\Models\\User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			nodes := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors())
			}
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			useNode, ok := nodes[0].(*ast.UseNode)
			if !ok {
				t.Fatalf("expected UseNode, got %T", nodes[0])
			}
			if useNode.Path != tt.expected {
				t.Errorf("expected path %q, got %q", tt.expected, useNode.Path)
			}
		})
	}
}

func TestParseFQCNValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "fqcn simple name",
			input:    "<?php class Foo extends Bar {}",
			expected: "Bar",
		},
		{
			name:     "fqcn fully qualified name",
			input:    "<?php class Foo extends \\Some\\BaseClass {}",
			expected: "\\Some\\BaseClass",
		},
		{
			name:     "fqcn partially qualified name",
			input:    "<?php class Foo extends Some\\BaseClass {}",
			expected: "Some\\BaseClass",
		},
		{
			name:     "fqcn with static",
			input:    "<?php class Foo extends static {}",
			expected: "static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			nodes := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors())
			}
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			classNode, ok := nodes[0].(*ast.ClassNode)
			if !ok {
				t.Fatalf("expected ClassNode, got %T", nodes[0])
			}
			if classNode.Extends == "" {
				t.Fatalf("expected class to extend a base class")
			}
			if classNode.Extends != tt.expected {
				t.Errorf("expected extends value %q, got %q", tt.expected, classNode.Extends)
			}
		})
	}
}

package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParsePropertyDeclarationErrors(t *testing.T) {
	cases := []struct {
		name  string
		code  string
	}{
		{"missing semicolon", "<?php class Foo { public $bar }"},
		{"invalid property name", "<?php class Foo { public bar; }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			_ = p.Parse()
			if len(p.Errors()) == 0 {
				t.Errorf("Expected errors for case '%s', got none", tc.name)
			}
		})
	}
}

func TestParseBlockStatements(t *testing.T) {
	codes := []string{
		"<?php { }", // empty block
		"<?php { { { } } }", // nested blocks
	}
	for _, code := range codes {
		l := lexer.New(code)
		p := New(l, true)
		_ = p.Parse()
		if len(p.Errors()) > 0 {
			t.Errorf("Unexpected errors for code: %s, errors: %v", code, p.Errors())
		}
	}
}

func TestParseEnumCaseErrors(t *testing.T) {
	cases := []struct {
		name  string
		code  string
	}{
		{"missing case name", "<?php enum E { case ; }"},
		{"missing semicolon", "<?php enum E { case FOO }"},
		{"missing value after =", "<?php enum E { case FOO = ; }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			_ = p.Parse()
			if len(p.Errors()) == 0 {
				t.Errorf("Expected errors for enum case '%s', got none", tc.name)
			}
		})
	}
}

func TestParseClassDeclarationErrors(t *testing.T) {
	cases := []struct {
		name  string
		code  string
	}{
		{"missing class name", "<?php class { }"},
		{"missing opening brace", "<?php class Foo "},
		{"missing closing brace", "<?php class Foo { public $a; "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			_ = p.Parse()
			if len(p.Errors()) == 0 {
				t.Errorf("Expected errors for class '%s', got none", tc.name)
			}
		})
	}
}

func TestParseUnionTypeEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		code  string
	}{
		{"empty union", "<?php function foo(): |int {}"},
		{"invalid union syntax", "<?php function foo(): int| {}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			_ = p.Parse()
			if len(p.Errors()) == 0 {
				t.Errorf("Expected errors for union type '%s', got none", tc.name)
			}
		})
	}
}

package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseMethodCallOnThis(t *testing.T) {
	php := `<?php
class Foo {
    public function bar($a, $b) {}
    public function test($x, $y) {
        $this->bar($x, $y);
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
}

func TestParseChainedMethodCall(t *testing.T) {
	php := `<?php
class Foo {
    public function bar() { return $this; }
    public function baz() { $this->bar()->bar(); }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
}

func TestParseMethodCallWithArrayAccess(t *testing.T) {
	php := `<?php
class Foo {
    public function bar($arr) {
        $this->baz($arr[0]);
    }
    public function baz($x) {}
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
}

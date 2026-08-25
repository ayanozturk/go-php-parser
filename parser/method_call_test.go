package parser

import (
	"github.com/ayanozturk/go-php-parser/lexer"
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

func TestParseMethodChainWithCommentsAfterObjectOperator(t *testing.T) {
	php := `<?php
$result = $source-> // select the next stage
    transform()-> /* finish the pipeline */
    complete();
`
	p := New(lexer.New(php), true)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one statement, got %d", len(nodes))
	}
}

func TestParseNullsafePropertyWithCommentAfterOperator(t *testing.T) {
	php := `<?php $name = $account?-> // account may be absent
    displayName;`
	p := New(lexer.New(php), true)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one statement, got %d", len(nodes))
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

func TestParseNullsafeMethodCall(t *testing.T) {
	php := `<?php
class Foo {
    public function test($admin) {
        return $admin?->getEmail();
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	err := p.Errors()
	if len(err) > 0 {
		t.Fatalf("Unexpected errors: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
}

func TestParseMethodChainOnNewExpressionAcrossLines(t *testing.T) {
	php := `<?php
class Factory {
    public function build(): void {
        try {
            $value = new Factory()
                ->from('enabled')
                ->finish();
        } catch (\Throwable $error) {
        }
    }
}
`
	p := New(lexer.New(php), false)
	nodes := p.Parse()
	if errors := p.Errors(); len(errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", errors)
	}
	if len(nodes) == 0 {
		t.Fatal("expected parsed nodes")
	}
}

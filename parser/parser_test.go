package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParserInfiniteLoopScenarios(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "malformed if statement",
			input: `<?php
				if () {
					echo "test";
				}
			`,
			wantErr: true,
		},
		{
			name: "unclosed class body",
			input: `<?php
				class Test {
					public function test() {
					echo "test";
				// Missing closing braces
			`,
			wantErr: true,
		},
		{
			name: "malformed nested blocks",
			input: `<?php
				if (true) {
					if (false) {
						echo "test";
					// Missing closing brace for inner if
				}
			`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			_ = p.Parse()
			hasErr := len(p.Errors()) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Test '%s': expected error=%v, got error=%v", tt.name, tt.wantErr, hasErr)
			}
		})
	}
}

func TestInstanceOf(t *testing.T) {
	l := lexer.New(`<?php
		if ($a instanceof \Exception) {
			echo "Exception";
		}
	`)
	p := New(l, true)
	_ = p.Parse()
	hasErr := len(p.Errors()) > 0
	if hasErr {
		t.Errorf("Test 'InstanceOf': expected no error, got error=%v", p.Errors())
	}
}

func TestInterfaceMethodParam(t *testing.T) {
    src := `<?php
interface AccessDecisionStrategyInterface
{
    public function decide(\Traversable $results): bool;
}`
    DebugPrintTokens(src)

	l := lexer.New(`<?php
interface AccessDecisionStrategyInterface
{
    public function decide(\Traversable $results): bool;
}
	`)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}
}

func TestParseSimpleExpression(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "integer literal",
			input: `<?php
				$a = 123;
			`,
			wantErr: false,
		},
		{
			name: "string literal",
			input: `<?php
				$a = "test";
			`,
			wantErr: false,
		},
		{
			name: "boolean literal",
			input: `<?php
				$a = true;
			`,
			wantErr: false,
		},
		{
			name: "null literal",
			input: `<?php
				$a = null;
			`,
			wantErr: false,
		},
		{
			name: "variable expression",
			input: `<?php
				$a = $b;
			`,
			wantErr: false,
		},
		{
			name: "new expression",
			input: `<?php
				$a = new Foo();
			`,
			wantErr: false,
		},
		{
			name: "FQCN expression",
			input: `<?php
				$a = \Foo\Bar::class;
			`,
			wantErr: false,
		},
		{
			name: "string interpolation",
			input: `<?php
				$a = "Hello, $name!";
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			_ = p.Parse()
			hasErr := len(p.Errors()) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Test '%s': expected error=%v, got error=%v", tt.name, tt.wantErr, hasErr)
			}
		})
	}
}

func TestParseExpressionWithPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "addition",
			input: `<?php
				$a = 1 + 2;
			`,
			wantErr: false,
		},
		{
			name: "subtraction",
			input: `<?php
				$a = 3 - 1;
			`,
			wantErr: false,
		},
		{
			name: "multiplication",
			input: `<?php
				$a = 2 * 3;
			`,
			wantErr: false,
		},
		{
			name: "division",
			input: `<?php
				$a = 6 / 2;
			`,
			wantErr: false,
		},
		{
			name: "modulo",
			input: `<?php
				$a = 5 % 2;
			`,
			wantErr: false,
		},
		{
			name: "concatenation",
			input: `<?php
				$a = "Hello, " . "world!";
			`,
			wantErr: false,
		},
		{
			name: "coalesce",
			input: `<?php
				$a = $b ?? "default";
			`,
			wantErr: false,
		},
		{
			name: "boolean or",
			input: `<?php
				$a = true || false;
			`,
			wantErr: false,
		},
		{
			name: "boolean and",
			input: `<?php
				$a = true && false;
			`,
			wantErr: false,
		},
		{
			name: "pipe",
			input: `<?php
				$a = $b | $c;
			`,
			wantErr: false,
		},
		{
			name: "assignment",
			input: `<?php
				$a = $b;
			`,
			wantErr: false,
		},
		{
			name: "greater or equal",
			input: `<?php
				$a = $b >= $c;
			`,
			wantErr: false,
		},
		{
			name: "smaller or equal",
			input: `<?php
				$a = $b <= $c;
			`,
			wantErr: false,
		},
		{
			name: "ternary",
			input: `<?php
				$a = $b ? $c : $d;
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			_ = p.Parse()
			hasErr := len(p.Errors()) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Test '%s': expected error=%v, got error=%v", tt.name, tt.wantErr, hasErr)
			}
		})
	}
}

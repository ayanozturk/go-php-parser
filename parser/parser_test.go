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

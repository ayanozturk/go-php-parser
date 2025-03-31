package parser

import (
	"go-phpcs/lexer"
	"testing"
	"time"
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

			// Create a channel to signal completion
			done := make(chan struct{})
			var parseErrors []string

			// Run the parser in a goroutine
			go func() {
				p.Parse()
				parseErrors = p.Errors()
				close(done)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				if !tt.wantErr && len(parseErrors) > 0 {
					t.Errorf("Parse() unexpected errors = %v", parseErrors)
				}
				if tt.wantErr && len(parseErrors) == 0 {
					t.Error("Parse() expected errors but got none")
				}
			case <-time.After(time.Second * 2):
				t.Errorf("Parse() potential infinite loop detected - test timed out")
			}
		})
	}
}

func TestParserValidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid if statement",
			input: `<?php
				if (true) {
					echo "test";
				}
			`,
			wantErr: false,
		},
		{
			name: "valid class declaration",
			input: `<?php
				class Test {
					public function test() {
						echo "test";
					}
				}
			`,
			wantErr: false,
		},
		{
			name: "valid nested blocks",
			input: `<?php
				if (true) {
					if (false) {
						echo "test";
					}
				}
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)

			p.Parse()
			parseErrors := p.Errors()

			if !tt.wantErr && len(parseErrors) > 0 {
				t.Errorf("Parse() unexpected errors = %v", parseErrors)
			}
			if tt.wantErr && len(parseErrors) == 0 {
				t.Error("Parse() expected errors but got none")
			}
		})
	}
}

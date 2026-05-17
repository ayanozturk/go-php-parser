package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"io"
	"os"
	"testing"
)

func TestParseForeachStatement(t *testing.T) {
	cases := []struct {
		name          string
		code          string
		keyExpected   bool
		byRefExpected bool
	}{
		{"simple foreach", `<?php foreach ($arr as $v) { echo $v; }`, false, false},
		{"foreach with key => value", `<?php foreach ($arr as $k => $v) { echo $k; echo $v; }`, true, false},
		{"foreach by reference", `<?php foreach ($arr as &$v) { $v = 1; }`, false, true},
		{"foreach key => &value", `<?php foreach ($arr as $k => &$v) { $v = 2; }`, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			nodes := p.Parse()
			if len(p.Errors()) > 0 {
				t.Errorf("Parser errors: %v", p.Errors())
			}
			if len(nodes) == 0 {
				t.Fatal("No nodes returned from parser")
			}
			// Find the ForeachNode recursively
			var findForeach func(nodes []ast.Node) *ast.ForeachNode
			findForeach = func(nodes []ast.Node) *ast.ForeachNode {
				for _, node := range nodes {
					if f, ok := node.(*ast.ForeachNode); ok {
						return f
					}
					// If node has children, search recursively
					switch n := node.(type) {
					case *ast.BlockNode:
						if res := findForeach(n.Statements); res != nil {
							return res
						}
					}
				}
				return nil
			}

			foreach := findForeach(nodes)
			if foreach == nil {
				t.Fatalf("Expected ForeachNode in AST, got %T", nodes[0])
			}
			if tc.keyExpected && foreach.KeyVar == nil {
				t.Errorf("Expected key variable, got nil")
			}
			if !tc.keyExpected && foreach.KeyVar != nil {
				t.Errorf("Did not expect key variable, but got one")
			}
			if foreach.ByRef != tc.byRefExpected {
				t.Errorf("Expected ByRef=%v, got %v", tc.byRefExpected, foreach.ByRef)
			}
			if foreach.ValueVar == nil {
				t.Errorf("Expected value variable, got nil")
			}
			if len(foreach.Body) == 0 {
				t.Errorf("Expected non-empty foreach body")
			}
		})
	}
}

func TestParseForeachDoesNotWriteDebugOutput(t *testing.T) {
	output := captureStdout(t, func() {
		l := lexer.New(`<?php foreach ($arr as $v) { echo $v; }`)
		p := New(l, true)
		_ = p.Parse()
		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
	})

	if output != "" {
		t.Fatalf("expected foreach parsing to be silent, got %q", output)
	}
}

func TestParseParameterDefaultDoesNotWriteDebugOutput(t *testing.T) {
	output := captureStdout(t, func() {
		l := lexer.New(`<?php function example($value = 'default') {}`)
		p := New(l, true)
		_ = p.Parse()
		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
	})

	if output != "" {
		t.Fatalf("expected parameter parsing to be silent, got %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = writePipe
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("failed to close stdout pipe: %v", err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("failed to read stdout pipe: %v", err)
	}
	if err := readPipe.Close(); err != nil {
		t.Fatalf("failed to close stdout read pipe: %v", err)
	}

	return string(output)
}

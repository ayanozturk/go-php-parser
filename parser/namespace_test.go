package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParsePropertyDeclarationErrors(t *testing.T) {
	cases := []struct {
		name string
		code string
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
		"<?php { }",         // empty block
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
		name string
		code string
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
		name string
		code string
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
		name string
		code string
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

func TestParseArrayWithClassKey(t *testing.T) {
	input := `<?php
	return [
		Doctrine\Abc::class => ['all' => true],
	];`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}

	if len(nodes) > 0 {
		returnNode, ok := nodes[0].(*ast.ReturnNode)
		if !ok {
			t.Errorf("Expected ReturnNode, found %s", nodes[0].NodeType())
			return
		}

		arrayNode, ok := returnNode.Expr.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected ArrayNode, found %s", returnNode.Expr.NodeType())
			return
		}

		if len(arrayNode.Elements) != 1 {
			t.Errorf("Expected ArrayNode to have 1 element, but got %d", len(arrayNode.Elements))
			return
		}
		item, ok := arrayNode.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected ArrayItemNode, got %T", arrayNode.Elements[0])
			return
		}
		// Check key is Doctrine\Abc::class
		key, ok := item.Key.(*ast.ClassConstFetchNode)
		if !ok {
			t.Errorf("Expected key to be ClassConstFetchNode, got %T", item.Key)
			return
		}
		if key.Class != "Doctrine\\Abc" || key.Const != "class" {
			t.Errorf("Expected key Doctrine\\Abc::class, got %s::%s", key.Class, key.Const)
		}
		// Check value is array ['all' => true]
		valArr, ok := item.Value.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected value to be ArrayNode, got %T", item.Value)
			return
		}
		if len(valArr.Elements) != 1 {
			t.Errorf("Expected inner array to have 1 element, got %d", len(valArr.Elements))
			return
		}
		innerItem, ok := valArr.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected inner element to be ArrayItemNode, got %T", valArr.Elements[0])
			return
		}
		keyStr, ok := innerItem.Key.(*ast.StringLiteral)
		if !ok || keyStr.Value != "all" {
			t.Errorf("Expected inner key to be string 'all', got %T (%v)", innerItem.Key, keyStr)
		}
		valBool, ok := innerItem.Value.(*ast.BooleanNode)
		if !ok || !valBool.Value {
			t.Errorf("Expected inner value to be boolean true, got %T (%v)", innerItem.Value, valBool)
		}
	}
}

func TestParseMultilineDocCommentInInterface(t *testing.T) {
	php := `<?php
interface NodeVisitorInterface {
    /**
     * Called before child nodes are visited.
     *
     * @return Node The modified node
     */
    public function enterNode(Node $node, Environment $env): Node;
}`
	l := lexer.New(php)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

func TestParserMathExpression(t *testing.T) {
	tests := []struct {
		input          string
		expectedLeft   int
		expectedRight  int
		operator       string
		expectedResult int
	}{
		{"<?php $age = 20 + 5;", 20, 5, "+", 25},
		{"<?php $age = 10 + 15;", 10, 15, "+", 25},
		{"<?php $age = 30 - 5;", 30, 5, "-", 25},
		{"<?php $age = 5 * 5;", 5, 5, "*", 25},
		{"<?php $age = 50 / 2;", 50, 2, "/", 25},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			l := lexer.New(test.input)
			p := New(l, true)
			nodes := p.Parse()
			if len(p.Errors()) > 0 {
				t.Errorf("Parser returned errors: %v", p.Errors())
			}
			if len(nodes) == 0 {
				t.Error("Expected at least one node, but got none")
			}
			if len(nodes) > 0 {
				if assign, ok := nodes[0].(*ast.AssignmentNode); ok {
					if variable, ok := assign.Left.(*ast.VariableNode); ok {
						if variable.Name != "age" {
							t.Errorf("Expected variable name 'age', but got '%s'", variable.Name)
						}
					} else {
						t.Error("Expected left side of assignment to be a VariableNode")
					}

					if _, ok := assign.Right.(*ast.BinaryExpr); ok {
						operatorIndex := len("<?php $age = ")
						for ; operatorIndex < len(test.input); operatorIndex++ {
							if !('0' <= test.input[operatorIndex] && test.input[operatorIndex] <= '9') && test.input[operatorIndex] != ' ' {
								break
							}
						}
						operator := string(test.input[operatorIndex])
						if operator != test.operator {
							t.Errorf("Expected operator '%s', got '%s'", test.operator, operator)
						}
					} else {
						t.Error("Expected right side of assignment to be a BinaryExpr")
					}
				}
			}
		})
	}
}

func TestParseNamespaceDeclarations(t *testing.T) {
	cases := []struct {
		name     string
		code     string
		wantName string
		wantBody bool
		wantErr  bool
	}{
		{"inline namespace", "<?php\nnamespace Foo\\Bar; class X {}", "Foo\\Bar", false, false},
		{"block namespace", "<?php\nnamespace Foo\\Bar { class X {} }", "Foo\\Bar", true, false},
		{"global namespace inline", "<?php\nnamespace; class X {}", "", false, false},
		{"global namespace block", "<?php\nnamespace { class X {} }", "", true, false},
		{"missing semicolon or brace", "<?php\nnamespace Foo Bar class X {}", "Foo", false, true},
		{"nested namespaces", "<?php\nnamespace A { namespace B { class X {} } }", "A", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			nodes := p.Parse()
			errs := p.Errors()
			if tc.wantErr && len(errs) == 0 {
				t.Errorf("Expected error, got none")
			}
			if !tc.wantErr && len(errs) > 0 {
				t.Errorf("Unexpected errors: %v", errs)
			}
			if len(nodes) == 0 && !tc.wantErr {
				t.Errorf("Expected at least one node, got none")
			}
			if len(nodes) > 0 && !tc.wantErr {
				ns, ok := nodes[0].(*ast.NamespaceNode)
				if !ok {
					t.Errorf("Expected NamespaceNode, got %T", nodes[0])
					return
				}
				if ns.Name != tc.wantName {
					t.Errorf("Expected namespace name '%s', got '%s'", tc.wantName, ns.Name)
				}
				if tc.wantBody && len(ns.Body) == 0 {
					t.Errorf("Expected namespace body, got none")
				}
				if !tc.wantBody && ns.Body != nil && len(ns.Body) > 0 {
					t.Errorf("Expected no namespace body, got one")
				}
			}
		})
	}
}

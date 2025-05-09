package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseFunctionWithUnionType(t *testing.T) {
	php := `<?php
class StringLoaderExtension {
    public static function templateFromString(Environment $env, string|\Stringable $template, ?string $name = null): TemplateWrapper {
        return $env->createTemplate((string) $template, $name);
    }
}`
	l := lexer.New(php)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

func TestParseUnionTypeInInterface(t *testing.T) {
	input := `<?php
interface TestInterface
{
    public function testMethod(?string $param = null): array|string|int|float|bool|null;
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("No nodes were parsed from input")
	}

	interfaceNode, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected first node to be InterfaceNode, got %T", nodes[0])
	}

	if interfaceNode.Name != "TestInterface" {
		t.Errorf("Expected interface name 'TestInterface', got '%s'", interfaceNode.Name)
	}

	// Only count InterfaceMethodNode members
	methodCount := 0
	for _, m := range interfaceNode.Members {
		if _, ok := m.(*ast.InterfaceMethodNode); ok {
			methodCount++
		}
	}
	if methodCount != 1 {
		t.Fatalf("Expected interface to have 1 method, got %d", len(interfaceNode.Members))
	}

	// Find the first InterfaceMethodNode
	var methodNode *ast.InterfaceMethodNode
	for _, m := range interfaceNode.Members {
		if mm, ok := m.(*ast.InterfaceMethodNode); ok {
			methodNode = mm
			break
		}
	}
	ok = methodNode != nil
	if !ok {
		t.Fatalf("Expected method to be InterfaceMethodNode, got %T", interfaceNode.Members[0])
	}

	if methodNode.Name != "testMethod" {
		t.Errorf("Expected method name 'testMethod', got '%s'", methodNode.Name)
	}

	// Check return type
	if methodNode.ReturnType == nil {
		t.Fatal("Expected method to have a return type, got nil")
	}

	unionType, ok := methodNode.ReturnType.(*ast.UnionTypeNode)
	if !ok {
		t.Fatalf("Expected return type to be UnionTypeNode, got %T", methodNode.ReturnType)
	}

	expectedTypes := []string{"array", "string", "int", "float", "bool", "null"}
	if len(unionType.Types) != len(expectedTypes) {
		t.Fatalf("Expected union type to have %d types, got %d", len(expectedTypes), len(unionType.Types))
	}

	for i, typeName := range expectedTypes {
		if unionType.Types[i] != typeName {
			t.Errorf("Expected union type %d to be '%s', got '%s'", i, typeName, unionType.Types[i])
		}
	}
}

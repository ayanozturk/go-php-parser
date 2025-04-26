package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseInterfaceWithCallableParameter(t *testing.T) {
	php := `<?php
interface CallableInterface {
    public function setHandler(callable $handler): void;
}`
	l := lexer.New(php)
	p := New(l, false)
	nodes := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Parser errors: %v", errs)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
	interfaceNode, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected first node to be InterfaceNode, got %T", nodes[0])
	}
	if interfaceNode.Name != "CallableInterface" {
		t.Errorf("Expected interface name to be 'CallableInterface', got '%s'", interfaceNode.Name)
	}
	if len(interfaceNode.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(interfaceNode.Methods))
	}
	method, ok := interfaceNode.Methods[0].(*ast.InterfaceMethodNode)
	if !ok {
		t.Fatalf("Expected method to be InterfaceMethodNode, got %T", interfaceNode.Methods[0])
	}
	if method.Name != "setHandler" {
		t.Errorf("Expected method name to be 'setHandler', got '%s'", method.Name)
	}
	if len(method.Params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(method.Params))
	}
	param, ok := method.Params[0].(*ast.ParamNode)
	if !ok {
		t.Fatalf("Expected parameter to be ParamNode, got %T", method.Params[0])
	}
	if param.TypeHint != "callable" {
		t.Errorf("Expected parameter type hint to be 'callable', got '%s'", param.TypeHint)
	}
	if param.Name != "handler" {
		t.Errorf("Expected parameter name to be 'handler', got '%s'", param.Name)
	}
	retType, ok := method.ReturnType.(*ast.IdentifierNode)
	if !ok {
		t.Fatalf("Expected return type to be IdentifierNode, got %T", method.ReturnType)
	}
	if retType.Value != "void" {
		t.Errorf("Expected return type to be 'void', got '%v'", retType.Value)
	}
}

func TestParseInterfaceWithCallableReturnType(t *testing.T) {
	php := `<?php
interface CallableReturnTypeInterface {
    public function getHandler(): callable;
}`
	l := lexer.New(php)
	p := New(l, false)
	nodes := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Parser errors: %v", errs)
	}
	if len(nodes) == 0 {
		t.Fatal("No AST nodes returned")
	}
	interfaceNode, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected first node to be InterfaceNode, got %T", nodes[0])
	}
	if interfaceNode.Name != "CallableReturnTypeInterface" {
		t.Errorf("Expected interface name to be 'CallableReturnTypeInterface', got '%s'", interfaceNode.Name)
	}
	if len(interfaceNode.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(interfaceNode.Methods))
	}
	method, ok := interfaceNode.Methods[0].(*ast.InterfaceMethodNode)
	if !ok {
		t.Fatalf("Expected method to be InterfaceMethodNode, got %T", interfaceNode.Methods[0])
	}
	if method.Name != "getHandler" {
		t.Errorf("Expected method name to be 'getHandler', got '%s'", method.Name)
	}
	if len(method.Params) != 0 {
		t.Fatalf("Expected 0 parameters, got %d", len(method.Params))
	}
	if method.ReturnType == nil {
		t.Fatal("Expected return type, got nil")
	}
	retType, ok := method.ReturnType.(*ast.IdentifierNode)
	if !ok {
		t.Fatalf("Expected return type to be IdentifierNode, got %T", method.ReturnType)
	}
	if retType.Value != "callable" {
		t.Errorf("Expected return type to be 'callable', got '%v'", retType.Value)
	}
}

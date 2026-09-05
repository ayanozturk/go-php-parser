package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestIterableTypesFromGenericListAndArray(t *testing.T) {
	key, value, ok := iterableTypesFromGeneric(GenericInstance{ClassName: "list", TypeArguments: []string{"string"}})
	if !ok || key.String() != "int" || value.String() != "string" {
		t.Fatalf("list<string> = %s => %s, ok=%v", key.String(), value.String(), ok)
	}
	key, value, ok = iterableTypesFromGeneric(GenericInstance{ClassName: "array", TypeArguments: []string{"int", "User"}})
	if !ok || key.String() != "int" || value.String() != "User" {
		t.Fatalf("array<int,User> = %s => %s, ok=%v", key.String(), value.String(), ok)
	}
}

func TestGenericInstanceFromTypeReadsAtomDisplay(t *testing.T) {
	typ := ParseType("Collection<User>")
	inst, ok := genericInstanceFromType(typ)
	if !ok || !strings.EqualFold(inst.ClassName, "Collection") || len(inst.TypeArguments) != 1 || inst.TypeArguments[0] != "User" {
		t.Fatalf("genericInstanceFromType = %#v ok=%v", inst, ok)
	}
}

func TestForeachIterationTypesBindLoopVariables(t *testing.T) {
	scope := &functionScope{
		variables: rootScopeTypeLayer(map[string]Type{"items": ParseType("list<User>")}),
		genericContext: map[string]GenericInstance{
			"items": {ClassName: "list", TypeArguments: []string{"User"}},
		},
	}
	applyForeachIterationTypes(scope, &ast.ForeachNode{
		Expr:     &ast.VariableNode{Name: "items"},
		ValueVar: &ast.VariableNode{Name: "item"},
		KeyVar:   &ast.VariableNode{Name: "idx"},
	}, nil, "foreach.php")
	got, ok := scope.variable("item")
	if !ok || got.String() != "User" {
		t.Fatalf("item type = %s ok=%v", got.String(), ok)
	}
	got, ok = scope.variable("idx")
	if !ok || got.String() != "int" {
		t.Fatalf("idx type = %s ok=%v", got.String(), ok)
	}
}

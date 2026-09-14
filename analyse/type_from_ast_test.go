package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestTypeFromASTUnionNullable(t *testing.T) {
	ctx := FileTypeContext{Namespace: "App", Aliases: map[string]string{"foo": `App\Foo`}}
	got := TypeFromAST(&ast.UnionTypeNode{
		Types: []ast.Node{
			&ast.IdentifierNode{Value: "Foo"},
			&ast.IdentifierNode{Value: "null"},
		},
	}, ctx)
	want := ParseType(normalizeTypeWithContext(`Foo|null`, ctx))
	if got.dnfString() != want.dnfString() {
		t.Fatalf("got %s, want %s", got.dnfString(), want.dnfString())
	}

	nullable := TypeFromAST(&ast.NullableTypeNode{
		Inner: &ast.IdentifierNode{Value: "int"},
	}, ctx)
	if nullable.dnfString() != ParseType("int|null").dnfString() {
		t.Fatalf("nullable got %s", nullable.dnfString())
	}

	dnf := TypeFromAST(&ast.UnionTypeNode{
		Types: []ast.Node{
			&ast.ParenthesizedTypeNode{Inner: &ast.IntersectionTypeNode{
				Types: []ast.Node{
					&ast.IdentifierNode{Value: "Left"},
					&ast.IdentifierNode{Value: "Right"},
				},
			}},
			&ast.IdentifierNode{Value: "Other"},
		},
	}, FileTypeContext{})
	if dnf.dnfString() != "(Left&Right)|Other" {
		t.Fatalf("dnf got %s", dnf.dnfString())
	}
}

func TestNativeTypeDNF(t *testing.T) {
	ctx := FileTypeContext{Namespace: "App", Aliases: map[string]string{"foo": `App\Foo`}}

	if got := nativeTypeDNF(nil, ctx); got != "" {
		t.Fatalf("nil got %q, want empty", got)
	}

	nullable := nativeTypeDNF(&ast.NullableTypeNode{
		Inner: &ast.IdentifierNode{Value: "int"},
	}, ctx)
	if nullable != "int|null" {
		t.Fatalf("nullable got %s, want int|null", nullable)
	}

	union := nativeTypeDNF(&ast.UnionTypeNode{
		Types: []ast.Node{
			&ast.IdentifierNode{Value: "Foo"},
			&ast.IdentifierNode{Value: "null"},
		},
	}, ctx)
	// Structural only — no alias/namespace resolution (callers normalize once).
	if union != "Foo|null" {
		t.Fatalf("union got %s, want Foo|null", union)
	}

	dnf := nativeTypeDNF(&ast.UnionTypeNode{
		Types: []ast.Node{
			&ast.ParenthesizedTypeNode{Inner: &ast.IntersectionTypeNode{
				Types: []ast.Node{
					&ast.IdentifierNode{Value: "Left"},
					&ast.IdentifierNode{Value: "Right"},
				},
			}},
			&ast.IdentifierNode{Value: "null"},
		},
	}, ctx)
	if dnf != "(Left&Right)|null" {
		t.Fatalf("dnf got %s, want (Left&Right)|null", dnf)
	}
}

func TestTypeFromASTCallable(t *testing.T) {
	got := TypeFromAST(&ast.CallableTypeNode{
		HasSignature: true,
		Params: []*ast.CallableParamNode{{
			TypeHint: &ast.IdentifierNode{Value: "int"},
		}},
		ReturnType: &ast.IdentifierNode{Value: "string"},
	}, FileTypeContext{})
	if got.String() != "callable" {
		t.Fatalf("got %s", got)
	}
}

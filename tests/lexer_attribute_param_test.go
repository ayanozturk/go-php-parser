package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestLexerFunctionParamWithAttribute(t *testing.T) {
	input := `<?php
interface Foo {
    public function bar(Request $request, #[\SensitiveParameter] string $secret);
}`
	lex := lexer.New(input)

	foundAttribute := false
	foundParam := false
	for {
		tok := lex.NextToken()
		if tok.Type == token.T_ATTRIBUTE {
			foundAttribute = true
		}
		if tok.Type == token.T_VARIABLE && tok.Literal == "$secret" {
			foundParam = true
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	if !foundAttribute {
		t.Errorf("Expected to find T_ATTRIBUTE token for parameter attribute")
	}
	if !foundParam {
		t.Errorf("Expected to find T_VARIABLE token for parameter $secret")
	}
}

package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestLexerAmpersandReference(t *testing.T) {
	input := `<?php
interface Foo {
    public function bar(?bool &$asGhostObject = null, ?string $id = null): bool;
}`
	lex := lexer.New(input)

	foundAmp := false
	for {
		tok := lex.NextToken()
		if tok.Type == token.T_AMPERSAND {
			foundAmp = true
			break
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	if !foundAmp {
		t.Errorf("Expected to find T_AMPERSAND token for by-reference parameter")
	}
}

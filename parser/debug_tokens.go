package parser

import (
	"fmt"
	"go-phpcs/lexer"
	"go-phpcs/token"
)

func DebugPrintTokens(php string) {
	l := lexer.New(php)
	for {
		tok := l.NextToken()
		fmt.Printf("%v (%q) at %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
		if tok.Type == token.T_EOF {
			break
		}
	}
}

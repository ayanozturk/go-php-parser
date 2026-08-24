package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseAltBody parses statements in PHP's alternative (colon) control
// structure syntax (e.g. `if (...): ... endif;`) until one of the given
// stop tokens is reached. The stop token itself is left unconsumed so the
// caller can distinguish which clause follows (elseif/else/endif, etc.).
func (p *Parser) parseAltBody(stops ...token.TokenType) []ast.Node {
	var body []ast.Node
	for p.tok.Type != token.T_EOF {
		// Consume open/close PHP tag transitions here, rather than
		// delegating to parseStatement: parseStatement's own retry loop
		// jumps straight past a tag into whatever statement follows it
		// (e.g. "<?php elseif (...):"), which would otherwise skip past
		// this loop's chance to notice the stop keyword before trying to
		// parse it as an expression.
		if p.tok.Type == token.T_OPEN_TAG || p.tok.Type == token.T_CLOSE_TAG {
			p.nextToken()
			continue
		}
		if p.atAnyOf(stops) {
			break
		}
		prevOffset := p.tok.Pos.Offset
		stmt, err := p.parseStatement()
		if err != nil {
			p.addError(err.Error())
			p.nextToken()
			continue
		}
		if stmt != nil {
			body = append(body, stmt)
		}
		// Safety: guarantee forward progress even if parseStatement
		// returned nil without consuming a token.
		if stmt == nil && p.tok.Pos.Offset == prevOffset {
			p.nextToken()
		}
	}
	return body
}

func (p *Parser) atAnyOf(types []token.TokenType) bool {
	for _, t := range types {
		if p.tok.Type == t {
			return true
		}
	}
	return false
}

// consumeAltTerminator consumes the ';' that normally follows an
// end-keyword (endif/endfor/endforeach/endwhile/endswitch). PHP also
// allows the statement to be implicitly terminated by a "?>" close tag or
// EOF, matching the same rule already applied to echo/expression
// statements. Returns false (having already recorded an error) if neither
// form is present.
func (p *Parser) consumeAltTerminator(construct string) bool {
	if p.tok.Type == token.T_SEMICOLON {
		p.nextToken()
		return true
	}
	if p.tok.Type == token.T_CLOSE_TAG || p.tok.Type == token.T_EOF {
		return true
	}
	p.addError("line %d:%d: expected ; after %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, construct, p.tok.Literal)
	return false
}

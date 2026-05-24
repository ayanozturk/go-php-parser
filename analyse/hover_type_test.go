package analyse

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func TestInferHoverTargetAtPositionIgnoresNewKeyword(t *testing.T) {
	php := `<?php
class Builder {}
class BuilderTest {
    public function testArguments(): void
    {
        $configuration = (new Builder)->fromParameters(['command', 'argument']);
    }
}`
	l := lexer.New(php)
	p := parser.New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if target, ok := InferHoverTargetAtPosition(nodes, 5, 27, "new", nil); ok {
		t.Fatalf("expected no hover target for new keyword, got %#v", target)
	}
}

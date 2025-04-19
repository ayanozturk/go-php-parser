package tests

import (
	"testing"
	"go-phpcs/parser"
	"go-phpcs/lexer"
)

func TestParseInterfaceWithFQCNExtends(t *testing.T) {
	php := `<?php
interface TwigCallableInterface extends \Stringable {
    public function getName(): string;
    public function getType(): string;
    public function getDynamicName(): string;
}`
	l := lexer.New(php)
	p := parser.New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

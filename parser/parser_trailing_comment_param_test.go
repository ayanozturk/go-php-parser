package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseInterfaceMethodWithCommentedOutParameter(t *testing.T) {
	php := `<?php
namespace Symfony\UX\Turbo\Twig;
use Twig\Environment;
interface TurboStreamListenRendererInterface {
    /**
     * Render turbo stream attributes.
     */
    public function renderTurboStreamListen(Environment $env, $topic /* , array $eventSourceOptions = [] */): string;
}`
	l := lexer.New(php)
	p := New(l, false)
	ast := p.Parse()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("Parser errors: %v", errs)
	}
	if len(ast) == 0 {
		t.Fatal("No AST nodes returned")
	}
}

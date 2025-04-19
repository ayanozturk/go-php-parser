package tests

import (
	"testing"
	"go-phpcs/parser"
	"go-phpcs/lexer"
)

func TestParseFunctionWithUnionType(t *testing.T) {
	php := `<?php
class StringLoaderExtension {
    public static function templateFromString(Environment $env, string|\Stringable $template, ?string $name = null): TemplateWrapper {
        return $env->createTemplate((string) $template, $name);
    }
}`
	l := lexer.New(php)
	p := parser.New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

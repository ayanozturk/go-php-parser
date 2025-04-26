package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseInterfaceMethodWithAttributeParameter(t *testing.T) {
	php := `<?php
interface RequestConfiguratorInterface {
    public function configure(RemoteEvent $event, #[\\SensitiveParameter] string $secret, HttpOptions $options): void;
}`
	l := lexer.New(php)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

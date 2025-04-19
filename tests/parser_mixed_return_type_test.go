package tests

import (
	"testing"
	"go-phpcs/parser"
	"go-phpcs/lexer"
)

func TestParseInterfaceWithMixedReturnType(t *testing.T) {
	php := `<?php
interface MetadataStoreInterface {
    public function getWorkflowMetadata(): array;
    public function getPlaceMetadata(string $place): array;
    public function getTransitionMetadata(Transition $transition): array;
    public function getMetadata(string $key, string|Transition|null $subject = null): mixed;
}`
	l := lexer.New(php)
	p := parser.New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}

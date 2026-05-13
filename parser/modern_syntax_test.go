package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseFinalReadonlyClass(t *testing.T) {
	php := `<?php
final readonly class HtmlRenderer {}
`

	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if classNode.Modifier != "final readonly" {
		t.Fatalf("expected combined modifier, got %q", classNode.Modifier)
	}
}

func TestParseAttributeAfterDocComment(t *testing.T) {
	php := `<?php
/** command */
#[AsCommand(name: 'server:dump')]
class ServerDumpPlaceholderCommand {}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseTypedClassConstant(t *testing.T) {
	php := `<?php
final readonly class HtmlRenderer {
	private const string PAGE_HEADER = <<<'EOT'
header
EOT;
}
`

	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(classNode.Constants))
	}
	constant, ok := classNode.Constants[0].(*ast.ConstantNode)
	if !ok {
		t.Fatalf("expected ConstantNode, got %T", classNode.Constants[0])
	}
	if constant.Type != "string" {
		t.Fatalf("expected typed constant, got type %q", constant.Type)
	}
}

func TestParseArrayAppendAssignment(t *testing.T) {
	php := `<?php
$parts[] = $className;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseStaticMethodCallChain(t *testing.T) {
	php := `<?php
$classLevelTestDox = MetadataRegistry::parser()->forClass($className)->isTestDox();
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseStaticArrowFunction(t *testing.T) {
	php := `<?php
$variables = array_map(
    static fn (string $variable): string => sprintf('/%s(?=\\b)/', preg_quote($variable, '/')),
    array_keys($providedData),
);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseParenthesizedNewMethodCall(t *testing.T) {
	php := `<?php
$instance = (new \ReflectionClass(DumpServer::class))->newInstanceWithoutConstructor();
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

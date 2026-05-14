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

func TestParseAttributeOnClassMethod(t *testing.T) {
	php := `<?php
class TemplateDirIterator extends \IteratorIterator
{
    #[\ReturnTypeWillChange]
    public function current()
    {
        return parent::current();
    }
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseReservedKeywordPropertyFetch(t *testing.T) {
	php := `<?php
class TwigCallable {
    private $class;
    private $callable;

    public function values()
    {
        return [$this->class, $this->callable];
    }
}
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

func TestParseIfElseWithArrayAppend(t *testing.T) {
	php := `<?php
if ($parts !== []) {
    $parts[] = $className;
} else {
    $fullyQualifiedName = $className;
}
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

func TestParseNegatedAssignmentCondition(t *testing.T) {
	php := `<?php
if (!$parameters = $function->getParameters()) {
    return false;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseCloneExpressionAssignment(t *testing.T) {
	php := `<?php
$new = clone $this;
$new->name = $name;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseArrayLiteralWithCommentedFirstElement(t *testing.T) {
	php := `<?php
return [
    // formatting filters
    new TwigFilter('date', [$this, 'formatDate']),
];
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseBooleanExpressionWithInlineComment(t *testing.T) {
	php := `<?php
$isDumpOutputHtmlSafe = \extension_loaded('xdebug')
    // Xdebug overloads var_dump in develop mode when html_errors is enabled
    && str_contains(\ini_get('xdebug.mode'), 'develop');
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseNewExpressionWithNamedArgument(t *testing.T) {
	php := `<?php
$parser = new UnaryOperatorExpressionParser(SpreadUnary::class, '...', 512, description: 'Spread operator');
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseExponentiationExpression(t *testing.T) {
	php := `<?php
return $method($value * 10 ** $precision) / 10 ** $precision;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseComparisonWithAssignmentOnRight(t *testing.T) {
	php := `<?php
if (0 < $timestamp = $cache->getTimestamp()) {
    return $timestamp;
}
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

func TestParseConcatAssignment(t *testing.T) {
	php := `<?php
$buffer .= sprintf('x');
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseIssetAndContinue(t *testing.T) {
	php := `<?php
if (!isset($result[$name])) {
    continue;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseUnaryAndBooleanCondition(t *testing.T) {
	php := `<?php
if (!$wasNumeric && $isNumeric) {
    $wasNumeric = true;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseVariableClassConstFetch(t *testing.T) {
	php := `<?php
$key = $test::class . '#' . $test->name();
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseArrayDestructuringAssignment(t *testing.T) {
	php := `<?php
[$result, $isCustomized] = $this->processTestDox($test, $testDox, $colorize);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParsePostIncrementInArrayAccess(t *testing.T) {
	php := `<?php
$value = $providedDataValues[$i++] ?? null;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseStringCast(t *testing.T) {
	php := `<?php
return (string) $value->value;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseTryCatch(t *testing.T) {
	php := `<?php
try {
    return [$reflector->invokeArgs(null, array_values($test->providedData())), true];
} catch (Throwable $t) {
    return [];
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseTryFinally(t *testing.T) {
	php := `<?php
try {
    return $this;
} finally {
    $this->didUseEcho = array_pop($this->didUseEchoStack);
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseMinusEqualAssignment(t *testing.T) {
	php := `<?php
$this->indentation -= $step;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseBareReturn(t *testing.T) {
	php := `<?php
function done(): void {
    return;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseSuppressedIncludeOnce(t *testing.T) {
	php := `<?php
if (is_file($key)) {
    @include_once $key;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseBitwiseNotInCallArgument(t *testing.T) {
	php := `<?php
@chmod($key, 0666 & ~umask());
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseDynamicNewVariable(t *testing.T) {
	php := `<?php
$cache = new $cls($this);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseDynamicParenthesizedNew(t *testing.T) {
	php := `<?php
return new ($class)($parser);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseYieldFrom(t *testing.T) {
	php := `<?php
function iter() {
    yield from $parsers;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseUnpackedFunctionCallArgument(t *testing.T) {
	php := `<?php
var_dump(...$vars);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseSwitchStatement(t *testing.T) {
	php := `<?php
switch ($extension) {
    case 'js':
    case 'json':
        return 'js';
    default:
        return 'html';
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseReservedKeywordMethodName(t *testing.T) {
	php := `<?php
class CoreExtension {
    public static function default($value, $default = '')
    {
        return $value;
    }
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseSpaceshipOperator(t *testing.T) {
	php := `<?php
return (string) $a <=> $b;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseAnonymousClassInstantiation(t *testing.T) {
	php := `<?php
return new class($precedence, $value) extends BinaryOperatorExpressionParser {
    public function __construct($precedence, $value) {}
};
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseDynamicMethodCall(t *testing.T) {
	php := `<?php
return $parent->$method(...$args);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseReservedEnumMethodName(t *testing.T) {
	php := `<?php
class CoreExtension {
    public static function enum($enum)
    {
        return $enum;
    }
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseVariableArrayKey(t *testing.T) {
	php := `<?php
return [$name => $template];
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseWhileAssignmentCondition(t *testing.T) {
	php := `<?php
while ($e = $e->getPrevious()) {
    $exceptions[] = $e;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseDoWhile(t *testing.T) {
	php := `<?php
do {
    $line = $e->getline();
} while (false !== $e = $e->getPrevious());
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseTraitUseInClass(t *testing.T) {
	php := `<?php
class MultiSelectPromptRenderer {
    use Concerns\DrawsBoxes;
}
`

	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok || len(classNode.Properties) == 0 {
		t.Fatalf("expected class with trait use node, got %T", nodes[0])
	}
	if _, ok := classNode.Properties[0].(*ast.TraitUseNode); !ok {
		t.Fatalf("expected first class property slot to contain TraitUseNode, got %T", classNode.Properties[0])
	}
}

func TestParseNamedArgument(t *testing.T) {
	php := `<?php
$result = $this->box('label', color: 'red');
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseClosureWithUse(t *testing.T) {
	php := `<?php
$result = array_map(function ($label, $key) use ($prompt) {
    return $label;
}, $options);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseFunctionCallArgumentTrailingComment(t *testing.T) {
	php := `<?php
$result = $this->when(
    $prompt->hint,
    fn () => $this->hint($prompt->hint),
    fn () => $this->newLine() // Space for errors
);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseGroupedStaticClosureInvocation(t *testing.T) {
	php := `<?php
$gen = (static function () use ($loaders): \Generator {
    yield 1;
})();
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseAbstractMethodDeclaration(t *testing.T) {
	php := `<?php
abstract class A {
    abstract public function operator(Compiler $compiler): Compiler;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseTypedTraitProperty(t *testing.T) {
	php := `<?php
trait T {
    private bool $definedTest = false;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseGotoAndLabel(t *testing.T) {
	php := `<?php
goto methodCheck;
methodCheck:
return 1;
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseStaticPropertyFetchAndReservedMethod(t *testing.T) {
	php := `<?php
return self::$colors[$foo] ?? CoreExtension::include($env);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseReservedObjectMethodName(t *testing.T) {
	php := `<?php
foreach ($template->yield($context) as $block) {}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseCommentSeparatedFluentChain(t *testing.T) {
	php := `<?php

$compiler
    // chain comment
    ->write('x')
    ->raw('y');
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseCatchWithoutVariable(t *testing.T) {
	php := `<?php
try {
    x();
} catch (RuntimeError) {
    y();
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseLogicalXor(t *testing.T) {
	php := `<?php
if ($a xor $b) {
	return;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseShiftOperators(t *testing.T) {
	php := `<?php
return 0xD800 | ($u >> 10);
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseMatchArmComment(t *testing.T) {
	php := `<?php
return match ($ord) {
	34 => '&quot;', /* quotation mark */
	38 => '&amp;',
	default => 'x',
};
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseCommentSeparatedElseIfChain(t *testing.T) {
	php := `<?php
if ($node instanceof ForNode) {
	$x = 1;
} elseif (!$loops) {
	return;
}

// the loop variable is referenced for the current loop
elseif ($node instanceof ContextVariable) {
	$y = 1;
}
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

func TestParseCommentAfterConcatenationOperator(t *testing.T) {
	php := `<?php
$x = preg_quote($a, '#')."x". // comment
	'|';
`

	l := lexer.New(php)
	p := New(l, true)
	_ = p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}
}

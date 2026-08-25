package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

// This file covers the second large batch of corpus-driven parser fixes:
// first-class callables on arbitrary expressions, by-ref arrow functions,
// binary integer literals, true/false/null in more type-hint positions,
// dynamic/relative class names in "new", grouped typed constants, foreach
// with property-fetch targets, case-insensitive casts, the "<>" operator,
// "namespace\Foo" relative references, and attributes in more positions.

func TestParseFirstClassCallableOnVariable(t *testing.T) {
	parseNoErrors(t, `<?php $fn = $callback(...);`)
}

func TestParseFirstClassCallableOnArrayCallable(t *testing.T) {
	parseNoErrors(t, `<?php $fn = [$obj, 'method'](...);`)
}

func TestParseByRefArrowFunction(t *testing.T) {
	parseNoErrors(t, `<?php $f = fn &() => $this->value;`)
}

func TestParseBinaryIntegerLiteral(t *testing.T) {
	nodes := parseNoErrors(t, `<?php $x = 0b1010;`)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestParseHexOctalBinaryIntegerValues(t *testing.T) {
	// Regression test: integer literal values must be parsed using their
	// actual base, not always base 10 (which previously silently produced 0
	// for anything but decimal digits).
	tests := []struct {
		src  string
		want int64
	}{
		{`<?php $x = 0x1A;`, 26},
		{`<?php $x = 0o17;`, 15},
		{`<?php $x = 0b1010;`, 10},
	}
	for _, tt := range tests {
		nodes := parseNoErrors(t, tt.src)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node for %q, got %d", tt.src, len(nodes))
		}
		stmt, ok := nodes[0].(*ast.ExpressionStmt)
		if !ok {
			t.Fatalf("expected ExpressionStmt for %q, got %T", tt.src, nodes[0])
		}
		assign, ok := stmt.Expr.(*ast.AssignmentNode)
		if !ok {
			t.Fatalf("expected AssignmentNode for %q, got %T", tt.src, nodes[0])
		}
		intNode, ok := assign.Right.(*ast.IntegerNode)
		if !ok {
			t.Fatalf("expected IntegerNode for %q, got %T", tt.src, assign.Right)
		}
		if intNode.Value != tt.want {
			t.Errorf("%q: expected value %d, got %d", tt.src, tt.want, intNode.Value)
		}
	}
}

func TestParseTrueReturnType(t *testing.T) {
	parseNoErrors(t, `<?php class C { public function method(): true { return true; } }`)
}

func TestParseTrueUnionReturnType(t *testing.T) {
	parseNoErrors(t, `<?php class C { public function dayOfYear(): static|int { return 1; } }`)
}

func TestParseTruePropertyType(t *testing.T) {
	parseNoErrors(t, `<?php class C { public true $flag = true; }`)
}

func TestParseTrueParameterType(t *testing.T) {
	parseNoErrors(t, `<?php class C { public function m(true|string $v) { return $v; } }`)
}

func TestParseNewDynamicDollarBraceClassName(t *testing.T) {
	parseNoErrors(t, `<?php $name = 'Foo'; $obj = new ${$name}();`)
}

func TestParseNewNamespaceRelativeClassName(t *testing.T) {
	parseNoErrors(t, `<?php $obj = new namespace\Bar();`)
}

func TestParseNewWithDocCommentBeforeAnonymousClass(t *testing.T) {
	parseNoErrors(t, "<?php $obj = new\n/** doc */\nclass {};")
}

func TestParseGroupedTypedConstants(t *testing.T) {
	parseNoErrors(t, `<?php class C { public const JPEG = 1, PNG = 2, GIF = 3; }`)
}

func TestParseGroupedConstantsWithTrailingComment(t *testing.T) {
	parseNoErrors(t, "<?php class C { public const\n\t\tA = 1, // comment\n\t\tB = 2;\n}")
}

func TestParseTopLevelGroupedConstants(t *testing.T) {
	nodes := parseNoErrors(t, `<?php const FOO = 1, BAR = 2;`)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node (wrapping block), got %d", len(nodes))
	}
	block, ok := nodes[0].(*ast.BlockNode)
	if !ok {
		t.Fatalf("expected BlockNode, got %T", nodes[0])
	}
	if len(block.Statements) != 2 {
		t.Fatalf("expected 2 grouped constants, got %d", len(block.Statements))
	}
}

func TestParseForeachWithPropertyFetchTarget(t *testing.T) {
	parseNoErrors(t, `<?php foreach ($users as $this->user) { echo 1; }`)
}

func TestParseForeachWithPropertyFetchKeyAndValue(t *testing.T) {
	parseNoErrors(t, `<?php foreach ($items as $stub->class => $stub->position) { echo 1; }`)
}

func TestParseCaseInsensitiveCast(t *testing.T) {
	parseNoErrors(t, `<?php $x = (String) $y;`)
	parseNoErrors(t, `<?php $x = (INT) $y;`)
}

func TestParseNotEqualAngleBracketOperator(t *testing.T) {
	parseNoErrors(t, `<?php if ($a <> $b) { echo 1; }`)
}

func TestParseNamespaceRelativeConstantReference(t *testing.T) {
	parseNoErrors(t, `<?php $x = [namespace\M_PI, 1];`)
}

func TestParseNamespaceRelativeFunctionCallStatement(t *testing.T) {
	parseNoErrors(t, `<?php namespace\prepare_result($input);`)
}

func TestParseMultilineNamespaceRelativeFunctionCallStatement(t *testing.T) {
	parseNoErrors(t, "<?php namespace\\prepare_result(\n    $input,\n    namespace\\fallback(),\n);")
}

func TestParseBareYieldStatement(t *testing.T) {
	parseNoErrors(t, `<?php function empty_item(): iterable { yield; }`)
}

func TestParseUnreachableBareYieldStatement(t *testing.T) {
	parseNoErrors(t, `<?php function force_generator(): iterable { return; yield; }`)
}

func TestParseQualifiedMixedFunctionCall(t *testing.T) {
	parseNoErrors(t, `<?php function schema() { return Types\mixed(); }`)
}

func TestParseNestedQualifiedMixedFunctionCall(t *testing.T) {
	parseNoErrors(t, `<?php inspect(Types\collection(Types\string(), Types\mixed()));`)
}

func TestParseReadonlyAnonymousClass(t *testing.T) {
	nodes := parseNoErrors(t, `<?php new readonly class {};`)
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected expression statement, got %T", nodes[0])
	}
	newExpr, ok := stmt.Expr.(*ast.NewNode)
	if !ok {
		t.Fatalf("expected new expression, got %T", stmt.Expr)
	}
	classExpr, ok := newExpr.ClassExpr.(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected anonymous class, got %T", newExpr.ClassExpr)
	}
	if classExpr.Modifier != "readonly" {
		t.Fatalf("expected readonly modifier, got %q", classExpr.Modifier)
	}
}

func TestParseReadonlyAnonymousClassWithArgumentsAndInterface(t *testing.T) {
	parseNoErrors(t, `<?php $runner = new readonly class($value) implements Runner { public function __construct(private string $value) {} public function run(): string { return $this->value; } };`)
}

func TestParsePrintExpressionInArrowFunction(t *testing.T) {
	parseNoErrors(t, `<?php consume($items, fn(string $item) => print "item: {$item}\n");`)
}

func TestParsePrintExpressionPrecedence(t *testing.T) {
	nodes := parseNoErrors(t, `<?php print $message = 'ready' and $enabled;`)
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected expression statement, got %T", nodes[0])
	}
	logical, ok := stmt.Expr.(*ast.BinaryExpr)
	if !ok || logical.Operator != "and" {
		t.Fatalf("expected outer logical and expression, got %#v", stmt.Expr)
	}
	printed, ok := logical.Left.(*ast.UnaryExpr)
	if !ok || printed.Operator != "print" {
		t.Fatalf("expected print on left of logical and, got %#v", logical.Left)
	}
	if _, ok := printed.Operand.(*ast.AssignmentNode); !ok {
		t.Fatalf("expected assignment inside print, got %T", printed.Operand)
	}
}

func TestParseAttributeInInterfaceBody(t *testing.T) {
	parseNoErrors(t, "<?php interface I { #[Groups(['a'])]\n public function get(); }")
}

func TestParseAttributeOnCallArgument(t *testing.T) {
	parseNoErrors(t, `<?php foo(#[Closure] fn () => 1);`)
}

func TestParseAsAndMixedAsMethodNames(t *testing.T) {
	parseNoErrors(t, `<?php class C { public function as() {} public function mixed() {} }`)
	parseNoErrors(t, `<?php $obj->as(); $obj->mixed();`)
}

func TestParseDynamicStaticMethodBraceAccess(t *testing.T) {
	parseNoErrors(t, `<?php $x = Foo::{$method}();`)
}

func TestParseTraitUsingAnotherTrait(t *testing.T) {
	parseNoErrors(t, `<?php trait T1 {} trait T2 { use T1; }`)
}

func TestParseGroupedUseImports(t *testing.T) {
	parseNoErrors(t, `<?php use function a, b, c;`)
	parseNoErrors(t, `<?php use A, B;`)
}

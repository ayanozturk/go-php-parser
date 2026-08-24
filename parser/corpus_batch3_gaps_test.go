package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
)

// This file covers the third large batch of corpus-driven parser fixes:
// a lexer bug where "::$var" (e.g. "static::$class") was silently
// corrupted by dead "::class"-merging logic, a lexer bug where an
// embedded NUL byte in a string literal was conflated with true
// end-of-input, parenthesized union/intersection property types vs.
// asymmetric-visibility "(set)" modifiers, constructor-promoted
// properties with two stacked visibility modifiers, typed class
// constants with union types, match arms with a trailing comma before
// "=>", attributes prefixing expressions (closures, anonymous classes)
// in more positions, interface and abstract property hooks, and named
// arguments on an arbitrary callable expression call.

func TestParseStaticPropertyAccessNamedClass(t *testing.T) {
	// Regression test for a lexer bug where "::$class" was misparsed as
	// if it were "::class" due to an off-by-one peek in the dead
	// "::class"-merging special case.
	parseNoErrors(t, `<?php class Foo { public static $class; public static function bar() { static::$class = null; } }`)
	parseNoErrors(t, `<?php class Foo { public static $class; public function bar() { self::$class = null; } }`)
}

func TestParseClassConstantStillWorks(t *testing.T) {
	parseNoErrors(t, `<?php $x = Foo::class;`)
	parseNoErrors(t, `<?php $x = self::class;`)
	parseNoErrors(t, `<?php $x = static::class;`)
}

func TestParseNulByteInStringLiteral(t *testing.T) {
	// Regression test for a lexer bug where an embedded NUL byte in a
	// string literal was indistinguishable from the EOF sentinel,
	// silently truncating parsing mid-construct.
	parseNoErrors(t, "<?php $x = ['\x00foo']; class C { public function bar() {} }")
}

func TestParseParenthesizedUnionIntersectionPropertyType(t *testing.T) {
	parseNoErrors(t, `<?php class Foo { public (\Traversable&\Countable)|null $someCollection; }`)
	parseNoErrors(t, `<?php class Foo { protected (A&B)|null $parent; }`)
}

func TestParsePromotedPropertyWithAsymmetricVisibility(t *testing.T) {
	parseNoErrors(t, `<?php class Book { public function __construct(public private(set) string $title) {} }`)
	parseNoErrors(t, `<?php class Book { public function __construct(public protected(set) string $author) {} }`)
}

func TestParseTypedConstantUnionType(t *testing.T) {
	parseNoErrors(t, `<?php class Foo { const string|int BAR = 'bar'; }`)
}

func TestParseMatchArmTrailingCommaBeforeArrow(t *testing.T) {
	parseNoErrors(t, `<?php $x = match ($v) { 'a', 'b', 'c', => 1, default => 2, };`)
}

func TestParseAttributeOnReturnedClosure(t *testing.T) {
	parseNoErrors(t, `<?php function f() { return #[When(env: 'prod')] function () { return 1; }; }`)
}

func TestParseAttributeOnArrayValueClosure(t *testing.T) {
	parseNoErrors(t, `<?php $x = ['foo' => #[\Closure(name: 'bar')] function () { return 1; }];`)
}

func TestParseAttributeOnAnonymousClassExpression(t *testing.T) {
	parseNoErrors(t, `<?php $x = new #[\AllowDynamicProperties] class {};`)
}

func TestParseInterfacePropertyHook(t *testing.T) {
	parseNoErrors(t, `<?php interface I { public string $foo { get; } }`)
	parseNoErrors(t, `<?php interface I { public string $foo { set; } }`)
}

func TestParseAbstractPropertyHook(t *testing.T) {
	parseNoErrors(t, `<?php abstract class C { abstract public string $bar { get; } }`)
}

func TestParseNamedArgumentOnVariableCall(t *testing.T) {
	parseNoErrors(t, `<?php $controller($templateName, headers: ['Content-Type' => 'x'])->headers->get('Content-Type');`)
}

func TestParseFileWithContentBeforeOpenTag(t *testing.T) {
	// Real PHP files may start with arbitrary literal content before the
	// first "<?php" tag (e.g. template/view files, XML declarations).
	// This requires lexer.NewFile (used for real files), not lexer.New
	// (used by bare-snippet tests, which always start in PHP-code mode).
	l := lexer.NewFile("<div>Hello</div>\n<?php echo 1; ?>\nBye")
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 top-level nodes (leading HTML, echo, trailing HTML), got %d", len(nodes))
	}
}

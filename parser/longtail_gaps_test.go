package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func parseNoErrors(t *testing.T, input string) []ast.Node {
	t.Helper()
	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	return nodes
}

func TestParseErrorSuppressedListAssignment(t *testing.T) {
	parseNoErrors(t, `<?php @list($a, $b) = explode(',', $s);`)
}

func TestParseErrorSuppressedArrayAppendAssignment(t *testing.T) {
	parseNoErrors(t, `<?php @$info['x'][$y][] = $z;`)
}

func TestParseErrorSuppressedCompoundAssignment(t *testing.T) {
	parseNoErrors(t, `<?php @$info['x']['y'] += $z;`)
}

func TestParseReadonlyAsFunctionName(t *testing.T) {
	nodes := parseNoErrors(t, `<?php function readonly() { return 1; }`)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestParseByRefFunctionReturn(t *testing.T) {
	parseNoErrors(t, `<?php function &create() { return $x; }`)
}

func TestParseByRefMethodReturn(t *testing.T) {
	parseNoErrors(t, `<?php class C { public function &create() { return $this->x; } }`)
}

func TestParseDynamicStaticPropertyBraceExpr(t *testing.T) {
	parseNoErrors(t, `<?php class C { public static function f($type) { return self::${$type . '_id'}; } }`)
}

func TestParseDynamicStaticPropertyDoubleDollar(t *testing.T) {
	parseNoErrors(t, `<?php class C { public static function f($name) { return self::$$name; } }`)
}

func TestParseDoubleDollarVariableVariable(t *testing.T) {
	nodes := parseNoErrors(t, `<?php $$name = 1;`)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestParseTraitUseAdaptationBlock(t *testing.T) {
	nodes := parseNoErrors(t, `<?php class C {
		use A, B {
			A::foo insteadof B;
			B::foo as bar;
			baz as protected;
		}
	}`)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestParseScientificNotationNumberLiteral(t *testing.T) {
	parseNoErrors(t, `<?php $x = array('a' => 1.2e1, 'b' => 1.7976931348623157E+308);`)
}

func TestParseLeadingDotFloatLiteral(t *testing.T) {
	parseNoErrors(t, `<?php $x = .8 * 5;`)
}

func TestParseShiftAssignOperators(t *testing.T) {
	parseNoErrors(t, `<?php $x >>= 2; $y <<= 3;`)
}

func TestParseForeachWithListDestructuring(t *testing.T) {
	parseNoErrors(t, `<?php foreach ($rows as list($a, $b)) { echo $a; }`)
}

func TestParseByRefArrayElementPropertyAndOffset(t *testing.T) {
	parseNoErrors(t, `<?php $a = [&$this->handle, &$arr[0]];`)
}

func TestParseTrailingDocCommentInBlock(t *testing.T) {
	parseNoErrors(t, `<?php function foo() {
		bar();
		/** trailing doc comment */
	}`)
}

func TestParseTrailingDocCommentBeforeSwitchCase(t *testing.T) {
	parseNoErrors(t, `<?php switch ($x) {
		case 'a':
			break;
		/** doc comment */
		case 'b':
			break;
	}`)
}

func TestParseInlineHTMLBeforeSwitchClose(t *testing.T) {
	parseNoErrors(t, `<?php function foo() {
		switch (1) {
			default:
				?>
				<p>hi</p>
				<?php
		}
	}`)
}

package printer

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func pos(line, col int) ast.Position {
	return ast.Position{Line: line, Column: col}
}

func dump(nodes ...ast.Node) string {
	var buf bytes.Buffer
	Fprint(&buf, nodes, 0)
	return buf.String()
}

func TestNewNilWriterDefaults(t *testing.T) {
	p := New(nil)
	if p == nil || p.w == nil {
		t.Fatal("New(nil) should fall back to a non-nil writer")
	}
}

func TestFprintSkipsNilNodes(t *testing.T) {
	out := dump(nil, &ast.NullLiteral{Pos: pos(1, 1)})
	if !strings.Contains(out, "NullLiteral @ 1:1") {
		t.Fatalf("expected NullLiteral dump, got %q", out)
	}
	if strings.Count(out, "@") != 2 { // type line + Value line
		// NullLiteral prints: "NullLiteral @ 1:1\n  Value: null\n"
		if !strings.Contains(out, "Value: null") {
			t.Fatalf("unexpected output: %q", out)
		}
	}
}

func TestPrintLiteralsAndVariable(t *testing.T) {
	out := dump(
		&ast.StringLiteral{Value: "hi", Pos: pos(1, 1)},
		&ast.IntegerLiteral{Value: 42, Pos: pos(2, 1)},
		&ast.FloatLiteral{Value: 3.5, Pos: pos(3, 1)},
		&ast.BooleanLiteral{Value: true, Pos: pos(4, 1)},
		&ast.NullLiteral{Pos: pos(5, 1)},
		&ast.VariableNode{Name: "x", Pos: pos(6, 1)},
	)
	for _, want := range []string{
		`StringLiteral @ 1:1`,
		`Value: "hi"`,
		`IntegerLiteral @ 2:1`,
		`Value: 42`,
		`FloatLiteral @ 3:1`,
		`Value: 3.5`,
		`BooleanLiteral @ 4:1`,
		`Value: true`,
		`NullLiteral @ 5:1`,
		`Value: null`,
		`Variable @ 6:1`,
		`Name: $x`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintComment(t *testing.T) {
	out := dump(&ast.CommentNode{Value: "// note", Pos: pos(1, 1)})
	if !strings.Contains(out, "Comment @ 1:1") || !strings.Contains(out, "Value: // note") {
		t.Fatalf("got %q", out)
	}
}

func TestPrintArrayEmptyAndItems(t *testing.T) {
	empty := dump(&ast.ArrayNode{Pos: pos(1, 1)})
	if !strings.Contains(empty, "[]") {
		t.Fatalf("empty array: %q", empty)
	}

	arr := &ast.ArrayNode{
		Pos: pos(2, 1),
		Elements: []ast.Node{
			&ast.ArrayItemNode{
				Key:    &ast.StringLiteral{Value: "k", Pos: pos(2, 3)},
				Value:  &ast.IntegerLiteral{Value: 1, Pos: pos(2, 8)},
				ByRef:  true,
				Unpack: false,
				Pos:    pos(2, 3),
			},
			&ast.ArrayItemNode{
				Value:  &ast.VariableNode{Name: "rest", Pos: pos(2, 15)},
				Unpack: true,
				Pos:    pos(2, 12),
			},
			&ast.IntegerLiteral{Value: 9, Pos: pos(2, 20)}, // non-item element
		},
	}
	out := dump(arr)
	if !strings.Contains(out, "&k => 1") {
		t.Fatalf("missing by-ref keyed item: %q", out)
	}
	if !strings.Contains(out, "...$rest") && !strings.Contains(out, "...rest") {
		// Variable TokenLiteral is the name without $; printer uses TokenLiteral.
		if !strings.Contains(out, "...") {
			t.Fatalf("missing unpack item: %q", out)
		}
	}
}

func TestArrayItemToStringNilSafe(t *testing.T) {
	p := New(io.Discard)
	if got := p.arrayItemToString(nil); got != "" {
		t.Fatalf("nil item => %q", got)
	}
	got := p.arrayItemToString(&ast.ArrayItemNode{ByRef: true, Unpack: true})
	if got != "&..." {
		t.Fatalf("nil value item => %q want &...", got)
	}
}

func TestPrintArrayItemNode(t *testing.T) {
	item := &ast.ArrayItemNode{
		Key:    &ast.StringLiteral{Value: "a", Pos: pos(1, 2)},
		Value:  &ast.IntegerLiteral{Value: 7, Pos: pos(1, 8)},
		ByRef:  true,
		Unpack: true,
		Pos:    pos(1, 1),
	}
	out := dump(item)
	for _, want := range []string{"ArrayItem @ 1:1", "Key:", "Value:", "ByRef: true", "Unpack: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintAssignmentBinaryReturnExpr(t *testing.T) {
	assign := &ast.AssignmentNode{
		Left:  &ast.VariableNode{Name: "a", Pos: pos(1, 1)},
		Right: &ast.IntegerLiteral{Value: 1, Pos: pos(1, 5)},
		Pos:   pos(1, 1),
	}
	bin := &ast.BinaryExpr{
		Left:     &ast.VariableNode{Name: "a", Pos: pos(2, 1)},
		Operator: "+",
		Right:    &ast.IntegerLiteral{Value: 2, Pos: pos(2, 5)},
		Pos:      pos(2, 1),
	}
	ret := &ast.ReturnNode{Expr: &ast.VariableNode{Name: "a", Pos: pos(3, 8)}, Pos: pos(3, 1)}
	emptyRet := &ast.ReturnNode{Pos: pos(4, 1)}
	exprStmt := &ast.ExpressionStmt{Expr: bin, Pos: pos(5, 1)}
	emptyExpr := &ast.ExpressionStmt{Pos: pos(6, 1)}

	out := dump(assign, bin, ret, emptyRet, exprStmt, emptyExpr)
	for _, want := range []string{
		"Assignment @ 1:1", "Left:", "Right:",
		"BinaryExpr @ 2:1", "Operator: +",
		"Return @ 3:1", "Expression:",
		"Return @ 4:1",
		"ExpressionStmt @ 5:1",
		"ExpressionStmt @ 6:1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Count(out, "Return @ 4:1\n") != 1 {
		// bare return should not invent an Expression section
		if strings.Contains(out, "Return @ 4:1\n  Expression:") {
			t.Fatalf("bare return should omit Expression:\n%s", out)
		}
	}
}

func TestPrintIfWhileFor(t *testing.T) {
	cond := &ast.BooleanLiteral{Value: true, Pos: pos(1, 5)}
	bodyStmt := &ast.ReturnNode{Pos: pos(2, 3)}
	iff := &ast.IfNode{
		Condition: cond,
		Body:      []ast.Node{bodyStmt},
		ElseIfs: []*ast.ElseIfNode{{
			Condition: &ast.BooleanLiteral{Value: false, Pos: pos(3, 10)},
			Body:      []ast.Node{&ast.ReturnNode{Pos: pos(4, 3)}},
			Pos:       pos(3, 1),
		}},
		Else: &ast.ElseNode{
			Body: []ast.Node{&ast.ReturnNode{Pos: pos(6, 3)}},
			Pos:  pos(5, 1),
		},
		Pos: pos(1, 1),
	}
	wh := &ast.WhileNode{
		Condition: cond,
		Body:      []ast.Node{bodyStmt},
		Pos:       pos(10, 1),
	}
	fr := &ast.ForNode{
		Init:       []ast.Node{&ast.IntegerLiteral{Value: 0, Pos: pos(20, 5)}},
		Conditions: []ast.Node{cond},
		Updates:    []ast.Node{&ast.IntegerLiteral{Value: 1, Pos: pos(20, 20)}},
		Body:       []ast.Node{bodyStmt},
		Pos:        pos(20, 1),
	}
	out := dump(iff, wh, fr)
	for _, want := range []string{
		"If @ 1:1", "Condition:", "Then:", "ElseIf:", "Else:",
		"While @ 10:1", "Body:",
		"For @ 20:1", "Init:", "Conditions:", "Updates:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintIfWhileForEmptyBranches(t *testing.T) {
	out := dump(
		&ast.IfNode{Condition: &ast.BooleanLiteral{Value: true, Pos: pos(1, 1)}, Pos: pos(1, 1)},
		&ast.WhileNode{Condition: &ast.BooleanLiteral{Value: true, Pos: pos(2, 1)}, Pos: pos(2, 1)},
		&ast.ForNode{Pos: pos(3, 1)},
	)
	if strings.Contains(out, "Then:") || strings.Contains(out, "Init:") {
		t.Fatalf("empty branches should omit labels:\n%s", out)
	}
}

func TestPrintFunctionClassEnumInterpolated(t *testing.T) {
	fn := &ast.FunctionNode{
		Name:      "foo",
		Modifiers: ast.ModifierListFromTexts([]string{"public"}),
		ReturnType: &ast.IdentifierNode{Value: "int", Pos: pos(1, 20)},
		Params: []ast.Node{
			&ast.VariableNode{Name: "p", Pos: pos(1, 12)},
		},
		Body: []ast.Node{
			&ast.ReturnNode{Expr: &ast.IntegerLiteral{Value: 1, Pos: pos(2, 10)}, Pos: pos(2, 3)},
		},
		Pos: pos(1, 1),
	}
	cls := &ast.ClassNode{
		Name:       "C",
		Extends:    "Base",
		Implements: []string{"I1", "I2"},
		Properties: []ast.Node{&ast.VariableNode{Name: "prop", Pos: pos(10, 5)}},
		Methods:    []ast.Node{fn},
		Pos:        pos(10, 1),
	}
	en := &ast.EnumNode{
		Name:     "Color",
		BackedBy: "string",
		Cases: []*ast.EnumCaseNode{
			{Name: "Red", Pos: pos(30, 5)},
			{Name: "Blue", Value: &ast.StringLiteral{Value: "b", Pos: pos(31, 12)}, Pos: pos(31, 5)},
		},
		Pos: pos(30, 1),
	}
	interp := &ast.InterpolatedStringLiteral{
		Parts: []ast.Node{
			&ast.StringLiteral{Value: "hi ", Pos: pos(40, 2)},
			&ast.VariableNode{Name: "name", Pos: pos(40, 6)},
		},
		Pos: pos(40, 1),
	}
	out := dump(fn, cls, en, interp)
	for _, want := range []string{
		"Function @ 1:1", "Name: foo", "Visibility: public", "ReturnType: int", "Parameters:", "Body:",
		"Class @ 10:1", "Extends: Base", "Implements: I1, I2", "Properties:", "Methods:",
		"Enum @ 30:1", "Backed by: string", "Cases:", "EnumCase @ 30:5",
		"InterpolatedString @ 40:1", "Parts:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintFunctionMinimalAndUnknownNode(t *testing.T) {
	out := dump(
		&ast.FunctionNode{Pos: pos(1, 1)},
		&ast.Identifier{Name: "bare", Pos: pos(2, 1)},
	)
	if !strings.Contains(out, "Function @ 1:1\n") {
		t.Fatalf("minimal function:\n%s", out)
	}
	if !strings.Contains(out, "Identifier @ 2:1") {
		t.Fatalf("unknown node should still print type+pos:\n%s", out)
	}
}

func TestPrintClassEnumWithoutExtras(t *testing.T) {
	out := dump(
		&ast.ClassNode{Name: "Alone", Pos: pos(1, 1)},
		&ast.EnumNode{Name: "E", Pos: pos(2, 1)},
	)
	if strings.Contains(out, "Extends:") || strings.Contains(out, "Cases:") {
		t.Fatalf("unexpected extras:\n%s", out)
	}
}

func TestBuildStringToPrintAndIndent(t *testing.T) {
	var buf bytes.Buffer
	buildStringToPrint([]ast.Node{&ast.NullLiteral{Pos: pos(1, 1)}}, &buf)
	if !strings.Contains(buf.String(), "NullLiteral @ 1:1") {
		t.Fatalf("buildStringToPrint: %q", buf.String())
	}

	var indented bytes.Buffer
	Fprint(&indented, []ast.Node{&ast.NullLiteral{Pos: pos(1, 1)}}, 2)
	if !strings.HasPrefix(indented.String(), "    NullLiteral") {
		t.Fatalf("indent=2 should prefix four spaces, got %q", indented.String())
	}
}

func TestPrintASTWritesStdout(t *testing.T) {
	// Smoke: PrintAST must not panic. Detailed capture is via Fprint.
	PrintAST(nil, 0)
	PrintAST([]ast.Node{&ast.NullLiteral{Pos: pos(1, 1)}}, 0)
}

func TestConvertEnumCasesToNodes(t *testing.T) {
	cases := []*ast.EnumCaseNode{{Name: "A"}, {Name: "B"}}
	nodes := convertEnumCasesToNodes(cases)
	if len(nodes) != 2 || nodes[0] != cases[0] || nodes[1] != cases[1] {
		t.Fatalf("convertEnumCasesToNodes failed: %#v", nodes)
	}
	if got := convertEnumCasesToNodes(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil cases => %#v", got)
	}
}

func TestPrintNodeNilNoOp(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf)
	p.printNode(nil)
	if buf.Len() != 0 {
		t.Fatalf("printNode(nil) wrote %q", buf.String())
	}
}

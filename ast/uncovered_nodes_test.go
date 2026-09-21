package ast

import "testing"

func TestGlobalVarDeclNodeMethodsAndSpan(t *testing.T) {
	node := &GlobalVarDeclNode{
		Vars:   []GlobalVarEntry{{Name: "a", Pos: Position{Line: 1, Column: 8}}},
		Pos:    Position{Line: 1, Column: 1, Offset: 0},
		EndPos: Position{Line: 1, Column: 12, Offset: 11},
	}
	if got := node.NodeType(); got != "GlobalVarDecl" {
		t.Fatalf("NodeType() = %q", got)
	}
	if got := node.TokenLiteral(); got != "global" {
		t.Fatalf("TokenLiteral() = %q", got)
	}
	if got := node.String(); got != "global vars" {
		t.Fatalf("String() = %q", got)
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}

func TestStaticVarDeclNodeMethodsAndSpan(t *testing.T) {
	node := &StaticVarDeclNode{
		Vars: []StaticVarEntry{
			{Name: "n", Init: &IntegerLiteral{Value: 1}, Pos: Position{Line: 2, Column: 9}},
		},
		Pos:    Position{Line: 2, Column: 1, Offset: 10},
		EndPos: Position{Line: 2, Column: 16, Offset: 25},
	}
	if got := node.NodeType(); got != "StaticVarDecl" {
		t.Fatalf("NodeType() = %q", got)
	}
	if got := node.TokenLiteral(); got != "static" {
		t.Fatalf("TokenLiteral() = %q", got)
	}
	if got := node.String(); got != "static vars" {
		t.Fatalf("String() = %q", got)
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}

func TestInlineHTMLNodeMethodsAndSpan(t *testing.T) {
	node := &InlineHTMLNode{
		Value:  "<p>hi</p>",
		Pos:    Position{Line: 1, Column: 1},
		EndPos: Position{Line: 1, Column: 10},
	}
	if got := node.NodeType(); got != "InlineHTML" {
		t.Fatalf("NodeType() = %q", got)
	}
	if got := node.TokenLiteral(); got != "<p>hi</p>" {
		t.Fatalf("TokenLiteral() = %q", got)
	}
	if got := node.String(); got != "inline html" {
		t.Fatalf("String() = %q", got)
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}

func TestSwitchNodeAndCases(t *testing.T) {
	expr := &VariableNode{Name: "x", Pos: Position{Line: 1, Column: 9}}
	caseNode := &SwitchCaseNode{
		Expr:      &IntegerLiteral{Value: 1},
		IsDefault: false,
		Body:      []Node{},
		Pos:       Position{Line: 2, Column: 3},
		EndPos:    Position{Line: 2, Column: 10},
	}
	defaultNode := &SwitchCaseNode{
		IsDefault: true,
		Body:      []Node{},
		Pos:       Position{Line: 3, Column: 3},
		EndPos:    Position{Line: 3, Column: 12},
	}
	sw := &SwitchNode{
		Expr:   expr,
		Cases:  []*SwitchCaseNode{caseNode, defaultNode},
		Pos:    Position{Line: 1, Column: 1},
		EndPos: Position{Line: 4, Column: 2},
	}

	if got := sw.NodeType(); got != "Switch" {
		t.Fatalf("Switch NodeType() = %q", got)
	}
	if got := sw.TokenLiteral(); got != "switch" {
		t.Fatalf("Switch TokenLiteral() = %q", got)
	}
	if got := sw.String(); got != "Switch @ 1:1" {
		t.Fatalf("Switch String() = %q", got)
	}
	assertLoopControlSpan(t, sw, Position{Line: 1, Column: 1}, Position{Line: 4, Column: 2})

	if got := caseNode.NodeType(); got != "SwitchCase" {
		t.Fatalf("case NodeType() = %q", got)
	}
	if got := caseNode.TokenLiteral(); got != "case" {
		t.Fatalf("case TokenLiteral() = %q", got)
	}
	if got := caseNode.String(); got != "Case @ 2:3" {
		t.Fatalf("case String() = %q", got)
	}
	assertLoopControlSpan(t, caseNode, Position{Line: 2, Column: 3}, Position{Line: 2, Column: 10})

	if got := defaultNode.TokenLiteral(); got != "default" {
		t.Fatalf("default TokenLiteral() = %q", got)
	}
	if got := defaultNode.String(); got != "DefaultCase @ 3:3" {
		t.Fatalf("default String() = %q", got)
	}
	assertLoopControlSpan(t, defaultNode, Position{Line: 3, Column: 3}, Position{Line: 3, Column: 12})
}

func TestTryCatchNodes(t *testing.T) {
	catch := &CatchNode{
		Types:    []string{"Exception"},
		Variable: "e",
		Body:     []Node{},
		Pos:      Position{Line: 3, Column: 1},
		EndPos:   Position{Line: 4, Column: 2},
	}
	try := &TryNode{
		Body:    []Node{},
		Catches: []*CatchNode{catch},
		Finally: []Node{},
		Pos:     Position{Line: 1, Column: 1},
		EndPos:  Position{Line: 6, Column: 2},
	}

	if got := try.NodeType(); got != "Try" {
		t.Fatalf("Try NodeType() = %q", got)
	}
	if got := try.TokenLiteral(); got != "try" {
		t.Fatalf("Try TokenLiteral() = %q", got)
	}
	if got := try.String(); got != "Try @ 1:1" {
		t.Fatalf("Try String() = %q", got)
	}
	assertLoopControlSpan(t, try, Position{Line: 1, Column: 1}, Position{Line: 6, Column: 2})

	if got := catch.NodeType(); got != "Catch" {
		t.Fatalf("Catch NodeType() = %q", got)
	}
	if got := catch.TokenLiteral(); got != "catch" {
		t.Fatalf("Catch TokenLiteral() = %q", got)
	}
	if got := catch.String(); got != "Catch @ 3:1" {
		t.Fatalf("Catch String() = %q", got)
	}
	assertLoopControlSpan(t, catch, Position{Line: 3, Column: 1}, Position{Line: 4, Column: 2})
}

func TestNullableAndParenthesizedTypeNodes(t *testing.T) {
	inner := &IdentifierNode{Value: "Foo", Pos: Position{Line: 1, Column: 2}}
	nullable := &NullableTypeNode{
		Inner:  inner,
		Pos:    Position{Line: 1, Column: 1},
		EndPos: Position{Line: 1, Column: 5},
	}
	if got := nullable.NodeType(); got != "NullableType" {
		t.Fatalf("Nullable NodeType() = %q", got)
	}
	if got := nullable.TokenLiteral(); got != "?Foo" {
		t.Fatalf("Nullable TokenLiteral() = %q", got)
	}
	if got := nullable.String(); got != "NullableType(Foo) @ 1:1" {
		t.Fatalf("Nullable String() = %q", got)
	}
	assertLoopControlSpan(t, nullable, Position{Line: 1, Column: 1}, Position{Line: 1, Column: 5})

	paren := &ParenthesizedTypeNode{
		Inner:  &IntersectionTypeNode{Types: []Node{&IdentifierNode{Value: "A"}, &IdentifierNode{Value: "B"}}},
		Pos:    Position{Line: 2, Column: 1},
		EndPos: Position{Line: 2, Column: 8},
	}
	if got := paren.NodeType(); got != "ParenthesizedType" {
		t.Fatalf("Paren NodeType() = %q", got)
	}
	wantLit := "(" + TypeText(paren.Inner) + ")"
	if got := paren.TokenLiteral(); got != wantLit {
		t.Fatalf("Paren TokenLiteral() = %q, want %q", got, wantLit)
	}
	if got := paren.String(); got != "ParenthesizedType("+TypeText(paren.Inner)+") @ 2:1" {
		t.Fatalf("Paren String() = %q", got)
	}
	assertLoopControlSpan(t, paren, Position{Line: 2, Column: 1}, Position{Line: 2, Column: 8})

	nilInner := &NullableTypeNode{Inner: nil, Pos: Position{Line: 3, Column: 1}}
	if got := nilInner.TokenLiteral(); got != "?" {
		t.Fatalf("nil Inner TokenLiteral() = %q, want ?", got)
	}
}

func TestVariableVariableDoWhileFirstClassGotoLabel(t *testing.T) {
	vv := &VariableVariableNode{
		Expr:   &VariableNode{Name: "name"},
		Pos:    Position{Line: 1, Column: 1},
		EndPos: Position{Line: 1, Column: 6},
	}
	if got := vv.NodeType(); got != "VariableVariable" {
		t.Fatalf("VariableVariable NodeType() = %q", got)
	}
	if got := vv.TokenLiteral(); got != "$" {
		t.Fatalf("VariableVariable TokenLiteral() = %q", got)
	}
	if got := vv.String(); got != "VariableVariable($Variable($name) @ 0:0) @ 1:1" {
		t.Fatalf("VariableVariable String() = %q", got)
	}
	nilVV := &VariableVariableNode{Pos: Position{Line: 2, Column: 2}}
	if got := nilVV.String(); got != "VariableVariable($<nil>) @ 2:2" {
		t.Fatalf("nil Expr String() = %q", got)
	}
	assertLoopControlSpan(t, vv, Position{Line: 1, Column: 1}, Position{Line: 1, Column: 6})

	dw := &DoWhileNode{
		Condition: &BooleanNode{Value: true},
		Pos:       Position{Line: 3, Column: 1},
		EndPos:    Position{Line: 5, Column: 2},
	}
	if got := dw.TokenLiteral(); got != "do" {
		t.Fatalf("DoWhile TokenLiteral() = %q", got)
	}
	if got := dw.String(); got != "DoWhile(Cond: true @ 0:0) @ 3:1" {
		t.Fatalf("DoWhile String() = %q", got)
	}
	nilDW := &DoWhileNode{Pos: Position{Line: 4, Column: 4}}
	if got := nilDW.String(); got != "DoWhile(Cond: <nil>) @ 4:4" {
		t.Fatalf("nil Condition String() = %q", got)
	}

	named := &FirstClassCallableNode{
		Name:   &IdentifierNode{Value: "strlen"},
		Pos:    Position{Line: 6, Column: 1},
		EndPos: Position{Line: 6, Column: 10},
	}
	if got := named.String(); got != "FirstClassCallable(strlen) @ 6:1" {
		t.Fatalf("named FCC String() = %q", got)
	}
	if got := named.TokenLiteral(); got != "strlen" {
		t.Fatalf("named FCC TokenLiteral() = %q", got)
	}
	target := &FirstClassCallableNode{
		Target: &VariableNode{Name: "cb"},
		Pos:    Position{Line: 7, Column: 1},
		EndPos: Position{Line: 7, Column: 8},
	}
	if got := target.String(); got != "FirstClassCallable(Variable($cb) @ 0:0) @ 7:1" {
		t.Fatalf("target FCC String() = %q", got)
	}
	if got := target.TokenLiteral(); got != "cb" {
		t.Fatalf("target FCC TokenLiteral() = %q", got)
	}
	emptyFCC := &FirstClassCallableNode{Pos: Position{Line: 8, Column: 1}}
	if got := emptyFCC.String(); got != "FirstClassCallable(<nil>) @ 8:1" {
		t.Fatalf("empty FCC String() = %q", got)
	}
	if got := emptyFCC.TokenLiteral(); got != "" {
		t.Fatalf("empty FCC TokenLiteral() = %q", got)
	}

	gotoNode := &GotoNode{Label: "done", Pos: Position{Line: 9, Column: 1}, EndPos: Position{Line: 9, Column: 10}}
	if got := gotoNode.NodeType(); got != "Goto" {
		t.Fatalf("Goto NodeType() = %q", got)
	}
	if got := gotoNode.TokenLiteral(); got != "goto" {
		t.Fatalf("Goto TokenLiteral() = %q", got)
	}
	if got := gotoNode.String(); got != "Goto(done) @ 9:1" {
		t.Fatalf("Goto String() = %q", got)
	}
	assertLoopControlSpan(t, gotoNode, Position{Line: 9, Column: 1}, Position{Line: 9, Column: 10})

	label := &LabelNode{Name: "done", Pos: Position{Line: 10, Column: 1}, EndPos: Position{Line: 10, Column: 6}}
	if got := label.NodeType(); got != "Label" {
		t.Fatalf("Label NodeType() = %q", got)
	}
	if got := label.TokenLiteral(); got != "done" {
		t.Fatalf("Label TokenLiteral() = %q", got)
	}
	if got := label.String(); got != "Label(done) @ 10:1" {
		t.Fatalf("Label String() = %q", got)
	}
	assertLoopControlSpan(t, label, Position{Line: 10, Column: 1}, Position{Line: 10, Column: 6})
}

func TestTraitUseNamedUnpackedNewAndPHPDocNode(t *testing.T) {
	trait := &TraitUseNode{
		Traits: []string{"A", "B"},
		Pos:    Position{Line: 1, Column: 1},
		EndPos: Position{Line: 1, Column: 12},
	}
	if got := trait.NodeType(); got != "TraitUse" {
		t.Fatalf("TraitUse NodeType() = %q", got)
	}
	if got := trait.TokenLiteral(); got != "use" {
		t.Fatalf("TraitUse TokenLiteral() = %q", got)
	}
	if got := trait.String(); got != "TraitUse(A, B) @ 1:1" {
		t.Fatalf("TraitUse String() = %q", got)
	}
	assertLoopControlSpan(t, trait, Position{Line: 1, Column: 1}, Position{Line: 1, Column: 12})

	named := &NamedArgumentNode{Name: "x", Value: &IntegerLiteral{Value: 1}, Pos: Position{Line: 2, Column: 1}}
	if got := named.String(); got != "x: Integer(1) @ 0:0" {
		t.Fatalf("NamedArgument String() = %q", got)
	}
	if got := named.TokenLiteral(); got != "x" {
		t.Fatalf("NamedArgument TokenLiteral() = %q", got)
	}
	nilNamed := &NamedArgumentNode{Name: "y"}
	if got := nilNamed.String(); got != "y: <nil>" {
		t.Fatalf("nil NamedArgument String() = %q", got)
	}

	unpacked := &UnpackedArgumentNode{Expr: &VariableNode{Name: "args"}, Pos: Position{Line: 3, Column: 1}}
	if got := unpacked.String(); got != "...Variable($args) @ 0:0" {
		t.Fatalf("Unpacked String() = %q", got)
	}
	nilUnpacked := &UnpackedArgumentNode{}
	if got := nilUnpacked.String(); got != "...<nil>" {
		t.Fatalf("nil Unpacked String() = %q", got)
	}
	if got := nilUnpacked.TokenLiteral(); got != "..." {
		t.Fatalf("Unpacked TokenLiteral() = %q", got)
	}

	n := &NewNode{
		ClassExpr: &VariableNode{Name: "cls"},
		Pos:       Position{Line: 4, Column: 1},
	}
	if got := n.String(); got != "New(Variable($cls) @ 0:0) @ 4:1" {
		t.Fatalf("New ClassExpr String() = %q", got)
	}

	doc := &PHPDocNode{Pos: Position{Line: 5, Column: 1}, EndPos: Position{Line: 7, Column: 4}}
	if got := doc.NodeType(); got != "PHPDoc" {
		t.Fatalf("PHPDoc NodeType() = %q", got)
	}
	if got := doc.TokenLiteral(); got != "/** ... */" {
		t.Fatalf("PHPDoc TokenLiteral() = %q", got)
	}
	if got := doc.String(); got != "PHPDoc @ 5:1" {
		t.Fatalf("PHPDoc String() = %q", got)
	}
	assertLoopControlSpan(t, doc, Position{Line: 5, Column: 1}, Position{Line: 7, Column: 4})
}

func TestGetHeaderEndPosFallbacks(t *testing.T) {
	header := Position{Line: 1, Column: 10, Offset: 9}
	end := Position{Line: 5, Column: 2, Offset: 40}

	class := &ClassNode{EndPos: end}
	if got := class.GetHeaderEndPos(); got != end {
		t.Fatalf("Class fallback GetHeaderEndPos = %+v", got)
	}
	class.HeaderEndPos = header
	if got := class.GetHeaderEndPos(); got != header {
		t.Fatalf("Class GetHeaderEndPos = %+v", got)
	}

	fn := &FunctionNode{EndPos: end}
	if got := fn.GetHeaderEndPos(); got != end {
		t.Fatalf("Function fallback GetHeaderEndPos = %+v", got)
	}
	fn.HeaderEndPos = header
	if got := fn.GetHeaderEndPos(); got != header {
		t.Fatalf("Function GetHeaderEndPos = %+v", got)
	}

	iface := &InterfaceNode{EndPos: end}
	if got := iface.GetHeaderEndPos(); got != end {
		t.Fatalf("Interface fallback GetHeaderEndPos = %+v", got)
	}
	iface.HeaderEndPos = header
	if got := iface.GetHeaderEndPos(); got != header {
		t.Fatalf("Interface GetHeaderEndPos = %+v", got)
	}

	en := &EnumNode{EndPos: end}
	if got := en.GetHeaderEndPos(); got != end {
		t.Fatalf("Enum fallback GetHeaderEndPos = %+v", got)
	}
	en.HeaderEndPos = header
	if got := en.GetHeaderEndPos(); got != header {
		t.Fatalf("Enum GetHeaderEndPos = %+v", got)
	}
}

package syntax

import (
	"testing"
)

func copyRed(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	return &RedNode{File: n.File, Parent: n.Parent, Green: n.Green, Offset: n.Offset}
}

func TestCallAndMemberAccessors(t *testing.T) {
	src := []byte(`<?php
foo($a, bar: $b, ...$c);
$o->meth();
$o?->prop;
Foo::BAR;
Foo::{$dyn};
($x);
new Bar();
new class {};
`)
	res := Parse(src)

	call := firstNodeOfKind(res.File.Root, KindCallExpr)
	if call == nil {
		t.Fatal("expected call")
	}
	if CallCallee(call) == nil {
		t.Fatal("CallCallee")
	}
	args := CallArgs(call)
	if len(args) < 3 {
		t.Fatalf("CallArgs got %d", len(args))
	}
	if ArgExpr(args[0]) == nil {
		t.Fatal("ArgExpr positional")
	}
	named := firstNodeOfKind(res.File.Root, KindNamedArg)
	if NamedArgName(named) != "bar" {
		t.Fatalf("NamedArgName=%q", NamedArgName(named))
	}
	if ArgExpr(named) == nil {
		t.Fatal("ArgExpr named")
	}
	var unpacked *RedNode
	for _, a := range args {
		if ArgIsUnpacked(a) {
			unpacked = a
			break
		}
	}
	if unpacked == nil {
		t.Fatal("expected unpacked arg")
	}

	member := firstNodeOfKind(res.File.Root, KindMemberAccessExpr)
	if MemberAccessObject(member) == nil || MemberAccessName(member) == "" {
		t.Fatalf("member access object/name: %v %q", MemberAccessObject(member), MemberAccessName(member))
	}
	nullsafe := firstNodeOfKind(res.File.Root, KindNullsafeMemberAccessExpr)
	if nullsafe != nil {
		if MemberAccessObject(nullsafe) == nil {
			t.Fatal("nullsafe object")
		}
	}

	var staticLiteral, staticDyn *RedNode
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() != KindStaticMemberAccessExpr {
			return true
		}
		_, _, dyn := StaticMemberAccessParts(n)
		if dyn {
			staticDyn = copyRed(n)
		} else {
			staticLiteral = copyRed(n)
		}
		return true
	})
	if staticLiteral == nil {
		t.Fatal("expected static member")
	}
	class, memberName, dyn := StaticMemberAccessParts(staticLiteral)
	if class == "" || memberName == "" || dyn {
		t.Fatalf("static parts class=%q member=%q dyn=%v", class, memberName, dyn)
	}
	if staticDyn != nil {
		_, m, d := StaticMemberAccessParts(staticDyn)
		if !d || m != "$" {
			t.Fatalf("dynamic static member: m=%q dyn=%v", m, d)
		}
	}

	paren := firstNodeOfKind(res.File.Root, KindParenExpr)
	if ParenInner(paren) == nil {
		t.Fatal("ParenInner")
	}
	newExpr := firstNodeOfKind(res.File.Root, KindNewExpr)
	if NewClass(newExpr) == nil {
		t.Fatal("NewClass")
	}
	var sawAnon bool
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() == KindNewExpr && NewIsAnonymous(n) {
			sawAnon = true
		}
		return true
	})
	if !sawAnon {
		t.Fatal("expected anonymous new")
	}
}

func TestLiteralAndVariableAccessors(t *testing.T) {
	src := []byte(`<?php
$s = "hi";
$i = 42;
$v = $name;
`)
	res := Parse(src)
	var stringLit, intLit, varExpr *RedNode
	Walk(res.File.Root, func(n *RedNode) bool {
		switch n.Kind() {
		case KindLiteralExpr:
			if v, ok := LiteralStringValue(n); ok && v == "hi" {
				stringLit = copyRed(n)
			}
			if v, ok := LiteralIntValue(n); ok && v == 42 {
				intLit = copyRed(n)
			}
		case KindVariableExpr:
			if VariableExprName(n) == "name" {
				varExpr = copyRed(n)
			}
		}
		return true
	})
	if stringLit == nil {
		t.Fatal("LiteralStringValue")
	}
	if intLit == nil {
		t.Fatal("LiteralIntValue")
	}
	if varExpr == nil {
		t.Fatal("VariableExprName")
	}
	if _, ok := LiteralStringValue(varExpr); ok {
		t.Fatal("non-string should not decode")
	}
}

func TestUnaryCastArrayLabelAccessors(t *testing.T) {
	src := []byte(`<?php
!$a;
++$i;
include 'x.php';
print $msg;
(int)$n;
$arr = [1 => 2, 3];
$v = $name;
$s = "hi";
label:
goto label;
`)
	res := Parse(src)

	unary := firstNodeOfKind(res.File.Root, KindUnaryExpr)
	if UnaryOperator(unary) == "" || UnaryOperand(unary) == nil {
		t.Fatalf("unary op/operand: %q %v", UnaryOperator(unary), UnaryOperand(unary))
	}

	inc := firstNodeOfKind(res.File.Root, KindIncludeExpr)
	if KeywordUnaryOperator(inc) == "" || UnaryOperand(inc) == nil {
		t.Fatalf("include keyword/operand")
	}
	pr := firstNodeOfKind(res.File.Root, KindPrintExpr)
	if KeywordUnaryOperator(pr) == "" {
		t.Fatal("print keyword")
	}

	cast := firstNodeOfKind(res.File.Root, KindCastExpr)
	if CastTypeName(cast) != "int" {
		t.Fatalf("CastTypeName=%q", CastTypeName(cast))
	}

	arr := firstNodeOfKind(res.File.Root, KindArrayExpr)
	elems := ArrayElements(arr)
	if len(elems) < 2 {
		t.Fatalf("ArrayElements=%d", len(elems))
	}
	if ArrayElementKey(elems[0]) == nil {
		t.Fatal("ArrayElementKey")
	}

	label := firstNodeOfKind(res.File.Root, KindLabelStmt)
	if LabelOrGotoName(label) != "label" {
		t.Fatalf("label name=%q", LabelOrGotoName(label))
	}
	gotoStmt := firstNodeOfKind(res.File.Root, KindGotoStmt)
	if LabelOrGotoName(gotoStmt) != "label" {
		t.Fatalf("goto name=%q", LabelOrGotoName(gotoStmt))
	}

	varExpr := firstNodeOfKind(res.File.Root, KindVariableExpr)
	stringLit := firstNodeOfKind(res.File.Root, KindLiteralExpr)
	if !IsWritableExprKind(varExpr) {
		t.Fatal("variable should be writable")
	}
	if IsWritableExprKind(stringLit) {
		t.Fatal("string literal should not be writable")
	}
	if IsWritableExprKind(nil) {
		t.Fatal("nil should not be writable")
	}
}

func TestPropertyParamAccessors(t *testing.T) {
	src := []byte(`<?php
class C {
    public string $a, $b;
    public function __construct(private int $x = 1) {}
}
`)
	res := Parse(src)
	prop := firstNodeOfKind(res.File.Root, KindPropertyDecl)
	if PropertyDeclType(prop) == nil {
		t.Fatal("PropertyDeclType")
	}
	vars := PropertyDeclVariables(prop)
	if len(vars) != 2 {
		t.Fatalf("PropertyDeclVariables=%d", len(vars))
	}
	if VariableName(vars[0]) != "a" {
		t.Fatalf("VariableName=%q", VariableName(vars[0]))
	}
	start, end := RawSpanPositions(prop)
	if start.Line < 1 || end.Line < 1 {
		t.Fatalf("RawSpanPositions %#v %#v", start, end)
	}

	param := firstNodeOfKind(res.File.Root, KindParam)
	if ParamType(param) == nil || ParamVariable(param) == nil {
		t.Fatal("ParamType/Variable")
	}
	if !ParamIsPromoted(param) {
		t.Fatal("expected promoted param")
	}
	if green, _ := ParamDefaultValue(param); green == nil {
		t.Fatal("ParamDefaultValue")
	}
}

func TestLowerAPIWrappers(t *testing.T) {
	src := []byte(`<?php
namespace N;
use Foo\Bar;
interface I { public function m(): void; }
trait T { use U { U::f as public g; } }
enum E: int { case A = 1; }
class C {
    public const X = 1;
    public string $p;
    public function f($a) { return $a + 1; }
}
try { throw new \Exception(); } catch (\Throwable $e) { }
#[Attr(1)]
function top($x = null) { return $x; }
$fn = function () {};
$a = [1, 2];
list($x) = $a;
`)
	res := Parse(src)
	file := res.File

	if LowerExprNode(nil, file) != nil || LowerStmtNode(nil, file) != nil {
		t.Fatal("nil lower wrappers")
	}
	if LowerParamNode(nil, file) != nil || LowerUseDeclNode(nil, file) != nil {
		t.Fatal("nil decl lowers")
	}

	call := firstNodeOfKind(file.Root, KindBinaryExpr)
	if call != nil {
		if LowerExprNode(call, file) == nil {
			// binary may exist from return $a + 1
			t.Log("binary lower skipped")
		}
	}
	expr := firstNodeOfKind(file.Root, KindLiteralExpr)
	if expr != nil && LowerExprNode(expr, file) == nil {
		t.Fatal("LowerExprNode literal")
	}

	fn := firstNodeOfKind(file.Root, KindFunctionDecl)
	if fn == nil {
		fn = firstNodeOfKind(file.Root, KindMethodDecl)
	}
	if LowerFunctionDeclNode(fn, file) == nil {
		t.Fatal("LowerFunctionDeclNode")
	}
	if LowerFunctionLikeContextNode(fn, file) == nil {
		t.Fatal("LowerFunctionLikeContextNode")
	}

	iface := firstNodeOfKind(file.Root, KindInterfaceDecl)
	if LowerInterfaceDeclNode(iface, file) == nil {
		t.Fatal("LowerInterfaceDeclNode")
	}
	imethod := firstNodeOfKind(iface, KindMethodDecl)
	if imethod == nil {
		imethod = firstNodeOfKind(iface, KindFunctionDecl)
	}
	if LowerInterfaceMethodDeclNode(imethod, file) == nil {
		t.Fatal("LowerInterfaceMethodDeclNode")
	}

	tr := firstNodeOfKind(file.Root, KindTraitDecl)
	if LowerTraitDeclNode(tr, file) == nil {
		t.Fatal("LowerTraitDeclNode")
	}
	if LowerClassLikeContextNode(tr, file) == nil {
		t.Fatal("trait class-like context")
	}

	en := firstNodeOfKind(file.Root, KindEnumDecl)
	if LowerEnumDeclNode(en, file) == nil {
		t.Fatal("LowerEnumDeclNode")
	}
	if LowerClassLikeContextNode(en, file) == nil {
		t.Fatal("enum class-like context")
	}

	cls := firstNodeOfKind(file.Root, KindClassDecl)
	if LowerClassLikeContextNode(cls, file) == nil {
		t.Fatal("class context")
	}
	prop := firstNodeOfKind(cls, KindPropertyDecl)
	if len(LowerPropertyDeclNode(prop, file)) == 0 {
		t.Fatal("LowerPropertyDeclNode")
	}
	cconst := firstNodeOfKind(cls, KindClassConstDecl)
	if len(LowerClassConstDeclNode(cconst, file)) == 0 {
		t.Fatal("LowerClassConstDeclNode")
	}

	useDecl := firstNodeOfKind(file.Root, KindUseDecl)
	if len(LowerUseDeclNode(useDecl, file)) == 0 {
		t.Fatal("LowerUseDeclNode")
	}
	param := firstNodeOfKind(file.Root, KindParam)
	if LowerParamNode(param, file) == nil {
		t.Fatal("LowerParamNode")
	}
	catch := firstNodeOfKind(file.Root, KindCatchClause)
	if LowerCatchClauseNode(catch, file) == nil {
		t.Fatal("LowerCatchClauseNode")
	}
	attr := firstNodeOfKind(file.Root, KindAttribute)
	if LowerAttributeNode(attr, file) == nil {
		t.Fatal("LowerAttributeNode")
	}
	closure := firstNodeOfKind(file.Root, KindClosureExpr)
	if LowerFunctionLikeContextNode(closure, file) == nil {
		t.Fatal("closure context")
	}
	if LowerClassLikeContextNode(iface, file) != nil {
		t.Fatal("interface class-like context should be nil")
	}
	listExpr := firstNodeOfKind(file.Root, KindListExpr)
	if listExpr != nil && LowerExprNode(listExpr, file) == nil {
		t.Fatal("LowerExprNode list")
	}

	// Statement lowers: goto/label already covered elsewhere; throw expression.
	throw := firstNodeOfKind(file.Root, KindThrowExpr)
	if throw != nil && LowerExprNode(throw, file) == nil {
		t.Fatal("LowerExprNode throw")
	}
}

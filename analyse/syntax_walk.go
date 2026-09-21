package analyse

import "github.com/ayanozturk/go-php-parser/syntax"

// syntaxChildRef is a green child descriptor with its absolute source offset
// (same shape as syntax's internal redChildDesc). Used to walk children
// without allocating a []*syntax.RedNode slice via RedNode.Children().
type syntaxChildRef struct {
	green  *syntax.GreenNode
	offset int
}

func syntaxRedChild(parent *syntax.RedNode, ref syntaxChildRef) *syntax.RedNode {
	return &syntax.RedNode{File: parent.File, Parent: parent, Green: ref.green, Offset: ref.offset}
}

func syntaxNonNilSiblings(parent *syntax.RedNode) []*syntax.RedNode {
	if parent == nil || parent.Green == nil {
		return nil
	}
	greens := parent.Green.Children()
	if len(greens) == 0 {
		return nil
	}
	out := make([]*syntax.RedNode, 0, len(greens))
	off := parent.Offset
	for _, g := range greens {
		if g == nil {
			continue
		}
		out = append(out, syntaxRedChild(parent, syntaxChildRef{green: g, offset: off}))
		off += g.Width()
	}
	return out
}

// walkSyntaxConfigured is the CST-direct analogue of walkAllConfigured
// (phpstan_level0_walk.go): a pre-order traversal over *syntax.RedNode that
// threads the enclosing class/function declaration and a FileTypeContext
// down to fn, mirroring the ast.Node walker's context-propagation rules for
// the boundary kinds it special-cases (class-likes, functions/methods,
// closures, namespace declarations). This is Phase 1 of the CST-direct
// migration (see AGENTS.md "CST-direct migration" and
// /memories/repo/cst-direct-migration.md).
//
// Unlike walkAllConfigured, which needs one type-switch case per concrete
// ast.Node type to know which fields to recurse into, the CST's uniform
// Children() means only two things need special-casing here: which kinds
// are pure grouping containers with no ast.Node equivalent (so fn should
// never be invoked for them), and which kinds open a new class/function/
// namespace scope for context-propagation purposes. Every other kind is
// walked generically.
//
// inStatementBody (the final fn parameter) is true once traversal has
// descended past any KindStatementList (a function/method/closure body, or
// any if/while/for/foreach/switch-case/try-catch-finally body) - i.e.
// wherever lowerStmt/lowerStatements (not lowerTopLevel/lowerClassMembers)
// would lower the corresponding ast.Node. It's sticky (never resets to
// false) except when re-entering an anonymous class's own member list.
// Callers that special-case KindAttribute (attributes are only ever visible
// on the ast side when attached to a top-level/namespace-level declaration
// or a class member/param, never when floating inside a statement body -
// see /memories/repo/cst-direct-migration.md) should skip when this is true.
func walkSyntaxConfigured(root *syntax.RedNode, ft FileTypeContext, fn func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool)) {
	if root == nil {
		return
	}
	nsCache := map[*syntax.RedNode]FileTypeContext{}
	var walk func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool)
	walk = func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if n == nil {
			return
		}
		if !syntaxContainerOnlyKinds[n.Kind()] {
			fn(n, class, currentFn, ft, inStatementBody)
		}
		nextClass, nextFn := class, currentFn
		switch n.Kind() {
		case syntax.KindClassDecl, syntax.KindInterfaceDecl, syntax.KindTraitDecl,
			syntax.KindEnumDecl, syntax.KindAnonymousClass:
			nextClass = n
		case syntax.KindFunctionDecl, syntax.KindMethodDecl, syntax.KindClosureExpr:
			// KindArrowFunctionExpr is deliberately excluded: arrow functions
			// lower to the distinct *ast.ArrowFunctionNode type, and
			// walkAllConfigured's *ast.ArrowFunctionNode case does NOT update
			// currentFn for its body (it passes the old currentFn through
			// unchanged) - unlike plain functions/methods/closures, which all
			// lower to *ast.FunctionNode and DO become the new currentFn via
			// walkAllConfigured's *ast.FunctionNode case. See
			// syntax.LowerFunctionLikeContextNode and
			// /memories/repo/cst-direct-migration.md.
			nextFn = n
		}
		// lowerStmt (syntax/lower_stmt.go) has no case for declaration kinds
		// (KindClassDecl/KindInterfaceDecl/KindTraitDecl/KindEnumDecl/
		// KindFunctionDecl/KindMethodDecl) - its default case returns nil,
		// and lowerStatements silently drops nil results. So a declaration
		// nested inside ANY statement body (if/while/for/foreach/switch-case/
		// try-catch-finally/function-or-closure body, etc. - anywhere a
		// KindStatementList is lowered via lowerStatements rather than
		// lowerTopLevel/lowerClassMembers) is completely invisible to the
		// ast.Node model and every rule built on it, not just a dispatcher
		// gap like the KindSwitchStmt one above - it's baked into the core
		// lowering pipeline itself. KindAnonymousClass is NOT part of this
		// gap: `new class {...}` is an expression, lowered via lowerExpr
		// regardless of statement-body nesting. Mirror the gap by tracking
		// inStatementBody (sticky true once we descend past any
		// KindStatementList) and skipping fn + recursion entirely for these
		// six kinds once inside one. KindAnonymousClass resets the flag back
		// to false for its own subtree: `new class {...}`'s member list is
		// lowered via lowerClassMembers (like a named class), not lowerStmt,
		// so its methods/properties are NOT subject to this gap even when the
		// anonymous class expression itself sits inside a statement body
		// (e.g. `return new class { public function f(Missing $x) {} };`). See
		// /memories/repo/cst-direct-migration.md for the full writeup.
		// Also become sticky-true upon entering ANY statement-kind node
		// (not just a KindStatementList body container): a bare top-level
		// `return #[Attr] function () {};` statement is never wrapped in a
		// KindStatementList (top-level file/namespace statements are direct
		// children of KindFile/KindNamespaceDecl), yet its subtree is still
		// exactly as invisible to lowerStmt-based ast consumers as if it
		// were nested three bodies deep - isStmtKind's node set (via
		// syntax.IsStatementKind) excludes declaration kinds, so this never
		// affects top-level/namespace-level `#[Attr] class Foo {}` siblings.
		nextInStatementBody := inStatementBody || n.Kind() == syntax.KindStatementList || syntax.IsStatementKind(n.Kind())
		if n.Kind() == syntax.KindAnonymousClass {
			nextInStatementBody = false
		}
		// A parameter's own attributes (e.g. `#[Argument] string $name` in a
		// closure/function param list) are read as a direct child by
		// lowerParam unconditionally, regardless of how deeply nested the
		// enclosing function/closure is - walkAllConfigured always recurses
		// into FunctionNode.Params[].Attributes whenever it reaches a
		// FunctionNode at all (checkTypeReferenceOnNode's *ast.FunctionNode
		// case doesn't check them directly, but the generic
		// *ast.AttributeNode case does, once the dispatcher walks into
		// Params). So param attributes are NOT subject to the statement-body
		// gap the way class/enum/attribute-before-declaration attributes
		// are; reset the flag for a KindParam's own subtree the same way as
		// KindAnonymousClass. See /memories/repo/cst-direct-migration.md.
		if n.Kind() == syntax.KindParam {
			nextInStatementBody = false
		}
		var elvisCond *syntax.RedNode
		if n.Kind() == syntax.KindTernaryExpr {
			elvisCond = syntax.TernaryElvisCondition(n)
		}
		var paramDefault *syntax.RedNode
		if n.Kind() == syntax.KindParam {
			paramDefault = syntax.ParamDefaultValue(n)
		}
		var dynamicClassPartGreen *syntax.GreenNode
		var dynamicClassPartOff int
		if n.Kind() == syntax.KindStaticMemberAccessExpr {
			dynamicClassPartGreen, dynamicClassPartOff = staticMemberAccessDynamicClassPartGreen(n)
		}
		if n.Green == nil {
			return
		}
		greens := n.Green.Children()
		childIdx := 0
		off := n.Offset
		for gi := 0; gi < len(greens); {
			g := greens[gi]
			if g == nil {
				gi++
				continue
			}
			c := syntaxRedChild(n, syntaxChildRef{green: g, offset: off})
			if nextInStatementBody {
				switch c.Kind() {
				case syntax.KindClassDecl, syntax.KindInterfaceDecl, syntax.KindTraitDecl,
					syntax.KindEnumDecl, syntax.KindFunctionDecl, syntax.KindMethodDecl:
					goto nextChild
				}
			}
			// walkAllConfigured's *ast.ParamNode case (phpstan_level0_walk.go)
			// only walks n.Attributes, never n.DefaultValue - so a parameter
			// default value expression (e.g. `$mode =
			// \RoundingMode::HalfAwayFromZero`), however deeply nested, is
			// entirely invisible to every rule the dispatcher drives, unlike
			// KindCastExpr/KindSwitchStmt (where the outer node is at least
			// visited once): there is no separate ast.Node wrapper here to
			// visit at all - skip both fn and recursion for this child
			// entirely. See syntax.ParamDefaultValue and
			// /memories/repo/cst-direct-migration.md for the full writeup.
			if paramDefault != nil && c.Green == paramDefault.Green && c.Offset == paramDefault.Offset {
				goto nextChild
			}
			// splitStaticMemberAccessParts (syntax/lower_expr.go) lowers a
			// non-Name class part (e.g. `(new class {...})::class`,
			// `$cls::CONST`) purely to call .TokenLiteral() on it, then
			// discards the resulting ast.Node entirely - ClassConstFetchNode.
			// Class/FunctionCallNode.Name only ever store the resulting
			// *string*. This subtree (including any nested anonymous class
			// declaration) has no ast.Node representation at all, so it must
			// never be visited - same "no wrapper exists" shape as
			// ParamDefaultValue above. See
			// /memories/repo/cst-direct-migration.md for the full writeup.
			if dynamicClassPartGreen != nil && g == dynamicClassPartGreen && off == dynamicClassPartOff {
				goto nextChild
			}
			// walkAllConfigured's *ast.EnumNode case (phpstan_level0_walk.go) only
			// walks n.Methods, never any representation of the enum's `case Foo;`
			// members - so attributes (and anything else) attached to an enum
			// case are completely invisible to every rule it drives. Mirror that
			// by never calling fn for a KindEnumCase subtree at all. See
			// /memories/repo/cst-direct-migration.md for the full writeup.
			if c.Kind() == syntax.KindEnumCase {
				goto nextChild
			}
			// walkAllConfigured (phpstan_level0_walk.go) has no case at all
			// for *ast.PropertyDeclNode, and never references its Hooks
			// field - so a PHP 8.4 property hook's body/expr (e.g. `set {
			// throw \LogicException('...'); }`), however deeply nested, is
			// completely invisible to every rule the Level0 dispatcher
			// drives (unlike e.g. undefined-variable analysis, which visits
			// hook bodies via a separate, non-Level0 code path - see
			// TestParseASTPropertyHookBodies in
			// syntax/parse_ast_body_test.go). Mirror that by never calling
			// fn for a KindPropertyHookList subtree at all, the same way as
			// KindEnumCase above. See /memories/repo/cst-direct-migration.md
			// for the full writeup.
			if c.Kind() == syntax.KindPropertyHookList {
				goto nextChild
			}
			// An enum case's own preceding AttributeList sibling (e.g.
			// `#[TestAttribute] case Beta;`) is a SEPARATE child of the
			// enclosing MemberList, not nested inside the KindEnumCase node
			// itself - so skipping just the KindEnumCase subtree above
			// doesn't hide it. Since walkAllConfigured's *ast.EnumNode case
			// never represents case members at all, this attribute is just
			// as invisible ast-side as the case itself; skip any
			// KindAttributeList (or chain of them) immediately followed by
			// a KindEnumCase sibling. The same applies to a class constant's
			// preceding AttributeList (e.g. `#[Repeatable('a'), Repeatable
			// ('b')] const X = 1;`): walkAllConfigured's *ast.ClassNode case
			// never recurses into n.Constants at all (see the
			// KindClassConstDecl comment in syntax_type_refs_rule.go), so a
			// class constant's attributes are just as invisible. See
			// /memories/repo/cst-direct-migration.md for the full writeup.
			if c.Kind() == syntax.KindAttributeList {
				jGi := gi + 1
				for jGi < len(greens) {
					jg := greens[jGi]
					if jg == nil {
						jGi++
						continue
					}
					if jg.Kind() == syntax.KindAttributeList {
						jGi++
						continue
					}
					nk := jg.Kind()
					if nk == syntax.KindEnumCase || nk == syntax.KindClassConstDecl {
						goto nextChild
					}
					break
				}
			}
			// walkAllConfigured (the ast.Node dispatcher) calls fn on every
			// node unconditionally but then type-switches to decide how to
			// recurse into children - and it has no case for
			// *ast.SwitchNode/*ast.SwitchCaseNode, so switch bodies (case
			// conditions and case bodies alike) are invisible to every rule
			// it drives. Mirror that gap here: fn still fires on the switch
			// statement itself (via the generic path below), but its
			// children are never descended into. See
			// /memories/repo/cst-direct-migration.md for the full writeup
			// (originally discovered/preserved per-rule in
			// syntax_language_rule.go; centralized here since every Phase 3
			// rule driven by this shared walker would otherwise need to
			// repeat the same skip).
			if c.Kind() == syntax.KindSwitchStmt {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				goto nextChild
			}
			// walkAllConfigured's *ast.ClassNode case (phpstan_level0_walk.go)
			// only recurses into n.Properties and n.Methods, never
			// n.Constants - a class constant's VALUE expression (e.g.
			// `Level::Debug->value` inside `private const MAP = [Level::
			// Debug->value => ...]`), however deeply nested, is entirely
			// invisible to every rule the dispatcher drives, the same kind
			// of gap as KindSwitchStmt/KindYieldExpr (already noted for the
			// type-refs port in syntax_type_refs_rule.go, which only skips
			// dispatching on the KindClassConstDecl node itself - this
			// centralizes the deeper fix of also never descending into it).
			// See /memories/repo/cst-direct-migration.md for the full
			// writeup.
			if c.Kind() == syntax.KindClassConstDecl {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				goto nextChild
			}
			// walkAllConfigured also has no case at all for *ast.YieldNode (a
			// `yield`/`yield from` expression, both of which lower to the same
			// ast type - see lowerExpr's KindYieldExpr case) - so a yield's
			// value/key expressions (which can themselves contain arbitrarily
			// complex nested calls, closures, etc., e.g. `yield from (new
			// Finder())->filter(function (Type $x) {...})`) are entirely
			// invisible to every rule the dispatcher drives. Mirror that gap
			// the same way as KindSwitchStmt: fn still fires on the yield
			// expression itself, but its children are never descended into.
			// See /memories/repo/cst-direct-migration.md for the full writeup.
			if c.Kind() == syntax.KindYieldExpr {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				goto nextChild
			}
			// walkAllConfigured also has no case at all for
			// *ast.FirstClassCallableNode (PHP 8.1 `foo(...)`/`$obj->foo(...)`
			// first-class callable syntax) - so its target expression (e.g.
			// the receiver of a chained `(new class {...})->method(...)`) is
			// entirely invisible, including any anonymous class declaration
			// nested inside it. Mirror the gap the same way as KindSwitchStmt
			// / KindYieldExpr: fn still fires on the first-class-callable
			// expression itself, but its children are never descended into.
			// See /memories/repo/cst-direct-migration.md for the full writeup.
			if c.Kind() == syntax.KindFirstClassCallableExpr {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				goto nextChild
			}
			// walkAllConfigured also has no case at all for
			// *ast.TypeCastNode (a `(string) $x`/`(int) foo()`/etc. cast
			// expression) - so its operand (which can itself contain
			// arbitrarily complex nested calls, e.g. `(string)
			// ini_get('disable_functions')`) is entirely invisible to every
			// rule the dispatcher drives. Mirror the gap the same way as
			// KindSwitchStmt / KindYieldExpr / KindFirstClassCallableExpr:
			// fn still fires on the cast expression itself, but its
			// children (the parens/type tokens and the operand) are never
			// descended into. See /memories/repo/cst-direct-migration.md
			// for the full writeup.
			if c.Kind() == syntax.KindCastExpr {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				goto nextChild
			}
			// An Elvis/short ternary (`cond ?: else`) lowers to a single
			// *ast.TernaryExpr whose Condition and IfTrue fields are the
			// SAME ast.Node value (see syntax.TernaryElvisCondition's doc
			// comment) - walkAllConfigured's *ast.TernaryExpr case visits
			// both fields unconditionally, so that shared subtree (and any
			// issue found anywhere within it) is visited/reported twice.
			// Mirror that by walking the condition child an extra time
			// here (its own KindTernaryExpr firing normally via the generic
			// path below is unaffected - this only duplicates the
			// condition's subtree walk). See
			// /memories/repo/cst-direct-migration.md for the full writeup.
			if elvisCond != nil && c.Green == elvisCond.Green && c.Offset == elvisCond.Offset {
				walk(c, nextClass, nextFn, ft, nextInStatementBody)
			}
			// An anonymous class's constructor argument list (`new class(
			// $this->message) {...}`) is a CST child of KindAnonymousClass,
			// but lowerNewExpr/lowerAnonymousClass extracts it as
			// NewNode.Args - a SIBLING field to NewNode.ClassExpr, not part
			// of the ClassNode (cls) itself. walkAllConfigured's
			// *ast.NewNode case walks n.Args using the SAME class/currentFn
			// context as the `new` expression's enclosing scope, not the
			// anonymous class's own scope (n.Args is evaluated in the outer
			// scope at call time, before the anonymous class body ever
			// runs) - e.g. a ctor arg `$this->message` refers to the
			// OUTER $this, not a property of the anonymous class being
			// constructed. nextClass/nextInStatementBody above already
			// switched to the anonymous class's own scope for this call
			// (matching its MemberList body), so a KindArgList child needs
			// the pre-switch (outer) class/currentFn/inStatementBody
			// values instead. See /memories/repo/cst-direct-migration.md
			// for the full writeup.
			if n.Kind() == syntax.KindAnonymousClass && c.Kind() == syntax.KindArgList {
				walk(c, class, currentFn, ft, inStatementBody)
				goto nextChild
			}
			// A namespace declaration re-derives FileTypeContext for its
			// body (braced form: c's own KindStatementList child; unbraced
			// form: the following top-level siblings up to the next
			// KindNamespaceDecl), mirroring walkAllConfigured's
			// *ast.NamespaceNode case + namespaceTypeContext. Namespace
			// declarations only ever appear at file top level in valid PHP,
			// so this never fires at deeper recursion depths.
			if c.Kind() == syntax.KindNamespaceDecl {
				if !syntaxContainerOnlyKinds[c.Kind()] {
					fn(c, nextClass, nextFn, ft, nextInStatementBody)
				}
				siblings := syntaxNonNilSiblings(n)
				body, consumed := syntax.NamespaceBody(c, siblings, childIdx)
				nft := namespaceSyntaxTypeContext(c, body, nsCache)
				for _, b := range body {
					walk(b, nextClass, nextFn, nft, nextInStatementBody)
				}
				off += g.Width()
				gi++
				childIdx++
				for skipped := 0; skipped < consumed; skipped++ {
					for gi < len(greens) && greens[gi] == nil {
						gi++
					}
					if gi >= len(greens) {
						break
					}
					off += greens[gi].Width()
					gi++
					childIdx++
				}
				continue
			}
			walk(c, nextClass, nextFn, ft, nextInStatementBody)
		nextChild:
			off += g.Width()
			gi++
			childIdx++
		}
	}
	walk(root, nil, nil, ft, false)
}

// staticMemberAccessDynamicClassPartGreen returns the class-part green child of a
// KindStaticMemberAccessExpr node when it's a complex expression rather
// than a plain name (e.g. `(new class {...})::class`, `$cls::CONST`).
// Returns nil,nil when the class part is a name (isNameKind), which is safely
// walked normally since a Name node has no meaningful sub-expression
// content to miss.
func staticMemberAccessDynamicClassPartGreen(n *syntax.RedNode) (*syntax.GreenNode, int) {
	if n == nil || n.Green == nil {
		return nil, 0
	}
	off := n.Offset
	for _, g := range n.Green.Children() {
		if g == nil {
			continue
		}
		k := g.Kind()
		if k == syntax.KindToken || k == syntax.KindTokenList {
			off += g.Width()
			continue
		}
		if syntax.IsNameKind(k) {
			return nil, 0
		}
		return g, off
	}
	return nil, 0
}

// syntaxContainerOnlyKinds are CST kinds that exist purely for grouping
// (param lists, member lists, statement lists, clause wrappers, …) and have
// no ast.Node equivalent — walkSyntaxConfigured recurses through them but
// never invokes fn, mirroring how walkAllConfigured never calls its walk
// callback on things like an else-if clause or a raw statement slice, since
// those aren't ast.Node values in the classic model.
var syntaxContainerOnlyKinds = map[syntax.Kind]bool{
	syntax.KindToken:               true,
	syntax.KindTokenList:           true,
	syntax.KindMissing:             true,
	syntax.KindError:               true,
	syntax.KindFile:                true,
	syntax.KindModifierList:        true,
	syntax.KindParamList:           true,
	syntax.KindCallableParamList:   true,
	syntax.KindMemberList:          true,
	syntax.KindArgList:             true,
	syntax.KindNameList:            true,
	syntax.KindStatementList:       true,
	syntax.KindAttributeList:       true,
	syntax.KindExtendsClause:       true,
	syntax.KindImplementsClause:    true,
	syntax.KindUseTraitClause:      true,
	syntax.KindUseGroup:            true,
	syntax.KindElseIfClause:        true,
	syntax.KindElseClause:          true,
	syntax.KindTraitAdaptationList: true,
	syntax.KindPropertyHookList:    true,
	syntax.KindClosureUseClause:    true,
}

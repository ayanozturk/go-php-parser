package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// Diagnostic is a structured parse/analysis error with a source span.
type Diagnostic struct {
	Message string
	Span    Span
}

// ParseResult is the versioned public parse API.
type ParseResult struct {
	File        *File
	Diagnostics []Diagnostic
	PHPVersion  string
}

// NameText returns the reconstructed name text without trivia (parts joined).
func NameText(n *RedNode) string {
	if n == nil {
		return ""
	}
	switch n.Kind() {
	case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName, KindName:
		var b []byte
		file := n.File
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if !green.IsToken() {
				return true
			}
			tok, ok := green.Token()
			if !ok {
				return true
			}
			// isContextualIdent is the parser's own "is this token usable as
			// an identifier here" check (parseIdentName uses it to accept
			// reserved words like `declare`, `list`, `default` as method/
			// function/const names). NameText must recognize every token
			// parseIdentName can produce, or a reserved-word name silently
			// reconstructs as "" here even though the CST correctly holds
			// it - T_NS_SEPARATOR is added back explicitly since it's a
			// separator between name segments, not an identifier itself,
			// but still belongs in the reconstructed text. A non-empty
			// placeholder stands in for classification when a
			// position-independent green node has no literal yet:
			// isContextualIdent only needs literal text to sanity-check
			// the first byte is alphabetic, which holds for every real
			// token type reaching this branch.
			classifyLit := tok.Literal
			if classifyLit == "" {
				classifyLit = "x"
			}
			if tok.Type == token.T_NS_SEPARATOR || isContextualIdent(tok.Type, classifyLit) {
				if tok.Literal != "" {
					b = append(b, tok.Literal...)
				} else {
					// Position-independent green: slice significant token text from source span.
					lead := 0
					for _, tr := range tok.LeadingTrivia {
						w := tr.Width()
						if w == 0 {
							w = len(tr.Literal)
						}
						lead += w
					}
					sig := tok.Width()
					if sig == 0 {
						sig = len(tok.Literal)
					}
					start := offset + lead
					end := start + sig
					if file != nil && start >= 0 && end <= len(file.Source) && start <= end {
						b = append(b, file.Source[start:end]...)
					}
				}
			}
			return true
		})
		return string(b)
	default:
		return n.Text()
	}
}

// IsContextualIdent reports whether a token is usable as a PHP identifier in
// contextual positions - decl names, member names after ->/::, named
// arguments. PHP treats most reserved words as valid here (e.g. a method can
// be named `declare`, `list`, or `default`), and token_get_all still emits
// the keyword's own token kind rather than T_STRING. Exported for callers
// outside this package (e.g. analyse's binder) that need to classify a
// member-access name token the same way the parser does when deciding
// whether to wrap it in an UnqualifiedName.
func IsContextualIdent(tt token.TokenType, lit string) bool {
	return isContextualIdent(tt, lit)
}

// TypeText reconstructs a type node's source without leading/trailing trivia gaps.
func TypeText(n *RedNode) string {
	if n == nil {
		return ""
	}
	return trimOuterWS(n.Text())
}

func trimOuterWS(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

// LowerExprNode lowers a single expression CST node to its classic ast.Node,
// letting CST-native rules reuse existing ast.Node-based helpers (literal
// value extraction, etc.) on isolated subtrees without lowering the whole
// file. Returns nil for unsupported kinds or a nil node.
func LowerExprNode(n *RedNode, file *File) ast.Node {
	return lowerExpr(n, file)
}

// LowerStmtNode lowers a single statement CST node to its classic ast.Node,
// mirroring LowerExprNode for statement-kind nodes (e.g. KindGotoStmt,
// KindLabelStmt). Returns nil for unsupported kinds or a nil node.
func LowerStmtNode(n *RedNode, file *File) ast.Node {
	return lowerStmt(n, file)
}

// LowerPropertyDeclNode lowers a single KindPropertyDecl CST node (a class
// property declaration, possibly with multiple comma-separated names sharing
// one type hint) to its classic []ast.Node ([]*ast.PropertyNode). Returns nil
// for a nil or non-KindPropertyDecl node. Leading PHPDoc from contiguous
// preceding sibling KindAttributeList nodes is preserved.
func LowerPropertyDeclNode(n *RedNode, file *File) []ast.Node {
	if n == nil || n.Kind() != KindPropertyDecl {
		return nil
	}
	props := lowerProperties(n, file)
	if doc := leadingAttributeDoc(n); doc != nil {
		for _, node := range props {
			if prop, ok := node.(*ast.PropertyNode); ok && prop.PHPDoc == nil {
				prop.PHPDoc = doc
			}
		}
	}
	return props
}

func leadingAttributeDoc(n *RedNode) *ast.PHPDocNode {
	if n == nil || n.Parent == nil {
		return nil
	}
	children := n.Parent.Children()
	for i, child := range children {
		if child.Green != n.Green || child.Offset != n.Offset {
			continue
		}
		for j := i - 1; j >= 0 && children[j].Kind() == KindAttributeList; j-- {
			if doc := leadingDocFromNode(children[j]); doc != nil {
				return doc
			}
		}
		return nil
	}
	return nil
}

// PrecedingAttributePHPDoc returns the PHPDoc comment attached to contiguous
// attribute-list siblings immediately before declaration. classLike is the
// enclosing class-like node; it provides sibling context for CST walkers that
// visit declaration nodes without retaining parent pointers.
func PrecedingAttributePHPDoc(classLike, declaration *RedNode) *ast.PHPDocNode {
	if classLike == nil || declaration == nil {
		return nil
	}
	for _, child := range classLike.Children() {
		if child.Kind() != KindMemberList {
			continue
		}
		siblings := child.Children()
		for i, sibling := range siblings {
			if sibling.Green != declaration.Green || sibling.Offset != declaration.Offset {
				continue
			}
			for j := i - 1; j >= 0 && siblings[j].Kind() == KindAttributeList; j-- {
				if doc := leadingDocFromNode(siblings[j]); doc != nil {
					return doc
				}
			}
			return nil
		}
	}
	return nil
}

// LowerParamNode lowers a single KindParam CST node to its classic
// *ast.ParamNode. Returns nil for a nil or non-KindParam node.
func LowerParamNode(n *RedNode, file *File) *ast.ParamNode {
	return lowerParam(n, file)
}

// LowerUseDeclNode lowers a single KindUseDecl CST node (a top-level `use
// ...;` import declaration, possibly with multiple comma-separated/grouped
// clauses) to its classic []ast.Node ([]*ast.UseNode). Returns nil for a nil
// or non-KindUseDecl node.
func LowerUseDeclNode(n *RedNode, file *File) []ast.Node {
	if n == nil || n.Kind() != KindUseDecl {
		return nil
	}
	return lowerUseDecl(n, file)
}

// LowerFunctionDeclNode lowers a single KindFunctionDecl or KindMethodDecl
// CST node to its classic *ast.FunctionNode. Returns nil for a nil or
// unsupported-kind node.
func LowerFunctionDeclNode(n *RedNode, file *File) *ast.FunctionNode {
	if n == nil || (n.Kind() != KindFunctionDecl && n.Kind() != KindMethodDecl) {
		return nil
	}
	return lowerFunction(n, file)
}

// LowerInterfaceMethodDeclNode lowers a single KindFunctionDecl or
// KindMethodDecl CST node representing an interface method signature to its
// classic *ast.InterfaceMethodNode. Interface methods reuse the same CST
// kinds as class methods (distinguished only by which declaration kind
// contains them) — callers must decide which of LowerFunctionDeclNode /
// LowerInterfaceMethodDeclNode to use based on the enclosing class-like's
// kind. Returns nil for a nil or unsupported-kind node.
func LowerInterfaceMethodDeclNode(n *RedNode, file *File) *ast.InterfaceMethodNode {
	if n == nil || (n.Kind() != KindFunctionDecl && n.Kind() != KindMethodDecl) {
		return nil
	}
	return lowerInterfaceMethod(n, file)
}

// LowerInterfaceDeclNode lowers a single KindInterfaceDecl CST node to its
// classic *ast.InterfaceNode (including Members). Unlike
// LowerClassLikeContextNode's nil result for KindInterfaceDecl (interfaces
// never contribute a "class" context parameter, since they can't nest
// inside another class-like), this returns the real lowered node for rules
// that need to inspect an interface declaration itself. Returns nil for a
// nil or non-KindInterfaceDecl node.
func LowerInterfaceDeclNode(n *RedNode, file *File) *ast.InterfaceNode {
	if n == nil || n.Kind() != KindInterfaceDecl {
		return nil
	}
	return lowerInterface(n, file)
}

// LowerEnumDeclNode lowers a single KindEnumDecl CST node to its classic
// *ast.EnumNode (including Methods/Cases/Implements). Unlike
// LowerClassLikeContextNode's synthetic Name-only *ast.ClassNode for
// KindEnumDecl (walkAllConfigured's "class" context parameter for an enum's
// body only ever needs Name), this returns the real lowered node for rules
// that need to inspect the enum declaration itself. Returns nil for a nil
// or non-KindEnumDecl node.
func LowerEnumDeclNode(n *RedNode, file *File) *ast.EnumNode {
	if n == nil || n.Kind() != KindEnumDecl {
		return nil
	}
	return lowerEnum(n, file)
}

// LowerTraitDeclNode lowers a single KindTraitDecl CST node to its classic
// *ast.TraitNode (including Body). Unlike LowerClassLikeContextNode's
// synthetic Name-only *ast.ClassNode for KindTraitDecl, this returns the
// real lowered node — needed to inspect a trait's own Body for nested
// *ast.TraitUseNode entries (a trait using another trait), since
// appendClassModelOnNode has no case for *ast.TraitNode itself. Returns nil
// for a nil or non-KindTraitDecl node.
func LowerTraitDeclNode(n *RedNode, file *File) *ast.TraitNode {
	if n == nil || n.Kind() != KindTraitDecl {
		return nil
	}
	return lowerTrait(n, file)
}

// LowerClassConstDeclNode lowers a single KindClassConstDecl or KindConstDecl
// CST node (possibly with multiple comma-separated const names) to its
// classic []ast.Node ([]*ast.ConstantNode). KindConstDecl (a global,
// non-class-member `const` declaration) is lowered the same way (see
// lowerTopLevel's KindConstDecl case). Returns nil for a nil or
// unsupported-kind node.
func LowerClassConstDeclNode(n *RedNode, file *File) []ast.Node {
	if n == nil || (n.Kind() != KindClassConstDecl && n.Kind() != KindConstDecl) {
		return nil
	}
	return lowerClassConsts(n, file)
}

// LowerCatchClauseNode lowers a single KindCatchClause CST node to its
// classic *ast.CatchNode (including its body). Returns nil for a nil or
// non-KindCatchClause node.
func LowerCatchClauseNode(n *RedNode, file *File) *ast.CatchNode {
	if n == nil || n.Kind() != KindCatchClause {
		return nil
	}
	return lowerCatchClause(n, file)
}

// LowerAttributeNode lowers a single KindAttribute CST node (one entry of an
// attribute group, e.g. the `Foo(1)` in `#[Foo(1), Bar]`) to its classic
// ast.Node (*ast.AttributeNode). Returns nil for a nil or non-KindAttribute
// node.
func LowerAttributeNode(n *RedNode, file *File) ast.Node {
	if n == nil || n.Kind() != KindAttribute {
		return nil
	}
	return lowerAttribute(n, file)
}

// LowerClassLikeContextNode lowers a class-like declaration CST node to the
// *ast.ClassNode value that walkAllConfigured would thread down as its
// "class" context parameter for that declaration's body — mirroring
// phpstan_level0_walk.go's per-declaration-kind class-context rules exactly
// (this is not a general "lower any class-like to ast.ClassNode" utility):
//   - KindClassDecl/KindAnonymousClass: the real lowered *ast.ClassNode
//     (walkAllConfigured's *ast.ClassNode case sets class = n directly).
//   - KindTraitDecl/KindEnumDecl: a synthetic *ast.ClassNode with only Name
//     populated (walkAllConfigured's *ast.TraitNode/*ast.EnumNode cases
//     build a throwaway *ast.ClassNode{Name: ...} rather than reusing the
//     real *ast.TraitNode/*ast.EnumNode, since callers of the "class"
//     context parameter only ever read class.Name).
//   - KindInterfaceDecl (and any other kind): nil (walkAllConfigured's
//     *ast.InterfaceNode case leaves the "class" parameter unchanged, which
//     is always nil in valid PHP since interfaces can't nest inside another
//     class-like).
//
// Returns nil for a nil node.
func LowerClassLikeContextNode(n *RedNode, file *File) *ast.ClassNode {
	if n == nil {
		return nil
	}
	switch n.Kind() {
	case KindClassDecl:
		return lowerClass(n, file)
	case KindAnonymousClass:
		cls, _ := lowerAnonymousClass(n, file)
		if c, ok := cls.(*ast.ClassNode); ok {
			return c
		}
		return nil
	case KindTraitDecl:
		t := lowerTrait(n, file)
		if t == nil || t.Name == nil {
			return nil
		}
		return &ast.ClassNode{Name: t.Name.Name}
	case KindEnumDecl:
		e := lowerEnum(n, file)
		if e == nil {
			return nil
		}
		return &ast.ClassNode{Name: e.Name}
	default:
		return nil
	}
}

// LowerFunctionLikeContextNode lowers a function/method/closure CST node to
// the *ast.FunctionNode value that walkAllConfigured would thread down as
// its "currentFn" context parameter, mirroring phpstan_level0_walk.go's
// *ast.FunctionNode case (which applies uniformly to plain functions,
// methods, and closures — all three lower to *ast.FunctionNode). Arrow
// functions (KindArrowFunctionExpr) are deliberately NOT handled here: they
// lower to the distinct *ast.ArrowFunctionNode type, and walkAllConfigured's
// *ast.ArrowFunctionNode case does not update currentFn for its body — the
// enclosing function's currentFn continues to apply inside an arrow
// function. Returns nil for a nil or unsupported-kind node.
func LowerFunctionLikeContextNode(n *RedNode, file *File) *ast.FunctionNode {
	if n == nil {
		return nil
	}
	switch n.Kind() {
	case KindFunctionDecl, KindMethodDecl:
		return lowerFunction(n, file)
	case KindClosureExpr:
		if fn, ok := lowerClosureExpr(n, file).(*ast.FunctionNode); ok {
			return fn
		}
		return nil
	default:
		return nil
	}
}

// TernaryElvisCondition reports whether n is a KindTernaryExpr in "Elvis"
// (short ternary, `cond ?: else`) form - i.e. the `?` is present but there is
// no explicit "then" branch before the `:` - and if so returns its condition
// child. lowerTernaryExpr (syntax/lower_expr.go) reuses the exact same
// lowered ast.Node value for both the resulting *ast.TernaryExpr's
// Condition and IfTrue fields in this form (rather than lowering two
// separate expressions), so walkAllConfigured's *ast.TernaryExpr case -
// which does walk(n.Condition, ...) then walk(n.IfTrue, ...) unconditionally
// - ends up visiting that single shared subtree twice, and any rule that
// fires per-visit (e.g. checkSymbolOnNode) reports every issue found in it
// twice. Exported so CST-direct ports can replicate that double-visit
// (see analyse/syntax_walk.go and /memories/repo/cst-direct-migration.md)
// instead of only visiting the condition once, which would under-count
// relative to the ast.Node path for this specific construct.
func TernaryElvisCondition(n *RedNode) (*GreenNode, int) {
	if n == nil || n.Kind() != KindTernaryExpr {
		return nil, 0
	}
	var condGreen *GreenNode
	var condOff int
	seenQ := false
	seenColon := false
	var thenGreen *GreenNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			switch green.TokenType() {
			case token.T_QUESTION:
				seenQ = true
			case token.T_COLON:
				seenColon = true
			}
			return true
		}
		if !(isExprKind(k) || isNameKind(k)) {
			return true
		}
		if condGreen == nil {
			condGreen = green
			condOff = offset
			return true
		}
		if seenQ && !seenColon && thenGreen == nil {
			thenGreen = green
		}
		return true
	})
	if condGreen == nil || thenGreen != nil {
		return nil, 0
	}
	return condGreen, condOff
}

// ParamDefaultValue returns n's default-value expression child (the
// expr/name-kind child following the `=` token), if n is a KindParam with
// one, mirroring lowerParam's own classification (syntax/lower_param.go).
// walkAllConfigured's *ast.ParamNode case (phpstan_level0_walk.go) only
// walks n.Attributes, never n.DefaultValue - so a parameter default value
// (e.g. `$mode = \RoundingMode::HalfAwayFromZero`), however deeply nested,
// is entirely invisible to every rule the dispatcher drives, the same kind
// of gap as KindCastExpr/KindSwitchStmt/etc. Exported so CST-direct ports
// can skip it the same way. See analyse/syntax_walk.go and
// /memories/repo/cst-direct-migration.md.
func ParamDefaultValue(n *RedNode) (*GreenNode, int) {
	if n == nil || n.Kind() != KindParam {
		return nil, 0
	}
	seenAssign := false
	var defaultGreen *GreenNode
	var defaultOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken && green.TokenType() == token.T_ASSIGN {
			seenAssign = true
			return true
		}
		k := green.Kind()
		if seenAssign && (isExprKind(k) || isNameKind(k)) {
			defaultGreen = green
			defaultOff = offset
			return false
		}
		return true
	})
	return defaultGreen, defaultOff
}

// IsStatementKind reports whether k is one of the statement kinds lowered
// via lowerStmt (as opposed to a declaration, expression, or grouping/
// container kind). Exported for CST-direct rule ports that need to detect
// "are we inside a statement's own subtree" (e.g. to decide whether a
// KindAttribute is reachable from the ast.Node side at all - see
// walkSyntaxConfigured's inStatementBody and
// /memories/repo/cst-direct-migration.md).
func IsStatementKind(k Kind) bool {
	return isStmtKind(k)
}

// Walk performs a pre-order traversal of the CST rooted at root, calling fn
// for each node. If fn returns false, that node's children are skipped
// (unlike the lowered ast.Node walkers, this supports early subtree exit).
//
// fn receives a single reused scratch *RedNode (Parent is always nil). The
// pointer MUST NOT be retained after fn returns — store a copy by value
// (RedNode{File, Green, Offset}) if a node must outlive the callback; see
// analyse/assignment_in_condition_rule.go. Green and Offset identify the
// visited node within root.File for the duration of the call only.
func Walk(root *RedNode, fn func(*RedNode) bool) {
	if root == nil || fn == nil {
		return
	}
	type walkFrame struct {
		green  *GreenNode
		offset int
	}
	scratch := &RedNode{File: root.File, Parent: nil, Green: root.Green, Offset: root.Offset}
	stack := []walkFrame{{green: root.Green, offset: root.Offset}}
	for len(stack) > 0 {
		top := len(stack) - 1
		f := stack[top]
		stack = stack[:top]
		scratch.Green = f.green
		scratch.Offset = f.offset
		if !fn(scratch) {
			continue
		}
		g := f.green
		if g == nil {
			continue
		}
		kids := g.Children()
		if len(kids) == 0 {
			continue
		}
		mark := len(stack)
		off := f.offset
		for _, child := range kids {
			if child == nil {
				continue
			}
			stack = append(stack, walkFrame{green: child, offset: off})
			off += child.Width()
		}
		for i, j := mark, len(stack)-1; i < j; i, j = i+1, j-1 {
			stack[i], stack[j] = stack[j], stack[i]
		}
	}
}

// FirstChildOfKind returns the first direct child with the given kind.
func (r *RedNode) FirstChildOfKind(k Kind) *RedNode {
	if r == nil {
		return nil
	}
	var found *RedNode
	r.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != k {
			return true
		}
		found = &RedNode{File: r.File, Green: green, Offset: offset}
		return false
	})
	return found
}

// ChildrenOfKind returns all direct children of kind k.
func (r *RedNode) ChildrenOfKind(k Kind) []*RedNode {
	var out []*RedNode
	for _, c := range r.Children() {
		if c.Kind() == k {
			out = append(out, c)
		}
	}
	return out
}

// AppendUseAliases merges class-like import aliases from a KindUseDecl into aliases.
// Used by the binder for per-namespace scopes (R2).
func AppendUseAliases(useDecl *RedNode, aliases map[string]string) {
	if useDecl == nil || aliases == nil {
		return
	}
	collectUseAliases(useDecl, aliases, nil, nil)
}

// AppendTypedUseAliases merges imports from a KindUseDecl into the alias map
// matching their PHP import namespace. Mixed group-use clauses are supported.
func AppendTypedUseAliases(useDecl *RedNode, classAliases, functionAliases, constAliases map[string]string) {
	if useDecl == nil {
		return
	}
	collectUseAliases(useDecl, classAliases, functionAliases, constAliases)
}

// NamespaceAndAliases walks a syntax File for the primary namespace name and
// class-like use aliases (including group-use expansions).
// Prefer per-namespace binding via analyse.BindSyntaxFile for multi-namespace files (R2).
func NamespaceAndAliases(f *File) (string, map[string]string) {
	aliases := map[string]string{}
	if f == nil || f.Root == nil {
		return "", aliases
	}
	ns := ""
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case KindNamespaceDecl:
			n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
				switch green.Kind() {
				case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName:
					ns = strings.TrimPrefix(NameText(n.bindChild(green, offset)), `\`)
				}
				return true
			})
		case KindUseDecl:
			collectUseAliases(n, aliases, nil, nil)
			return // don't double-walk clauses
		}
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			walk(n.bindChild(green, offset))
			return true
		})
	}
	walk(f.Root)
	return ns, aliases
}

func collectUseAliases(useDecl *RedNode, classAliases, functionAliases, constAliases map[string]string) {
	useType := "class"
	useDecl.ForEachChildDesc(func(green *GreenNode, _ int) bool {
		if green.Kind() != KindToken {
			return true
		}
		switch green.TokenType() {
		case token.T_FUNCTION:
			useType = "function"
		case token.T_CONST:
			useType = "const"
		}
		return true
	})
	for _, clause := range useDecl.ChildrenOfKind(KindUseClause) {
		collectUseClauseAliases(clause, "", useType, classAliases, functionAliases, constAliases)
	}
}

func collectUseClauseAliases(clause *RedNode, prefix, useType string, classAliases, functionAliases, constAliases map[string]string) {
	if clause == nil {
		return
	}
	itemType := useType
	var name *RedNode
	var alias *RedNode
	var group *RedNode
	clause.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch {
		case green.Kind() == KindToken && green.TokenType() == token.T_FUNCTION:
			itemType = "function"
		case green.Kind() == KindToken && green.TokenType() == token.T_CONST:
			itemType = "const"
		case green.Kind() == KindUnqualifiedName || green.Kind() == KindQualifiedName || green.Kind() == KindFullyQualifiedName || green.Kind() == KindRelativeName:
			c := clause.bindChild(green, offset)
			if name == nil {
				name = c
			} else {
				alias = c
			}
		case green.Kind() == KindUseGroup:
			group = clause.bindChild(green, offset)
		}
		return true
	})
	if group != nil {
		base := strings.TrimPrefix(NameText(name), `\`)
		if prefix != "" {
			base = strings.Trim(prefix, `\`) + `\` + strings.Trim(base, `\`)
		}
		base = strings.TrimSuffix(base, `\`)
		for _, inner := range group.ChildrenOfKind(KindUseClause) {
			collectUseClauseAliases(inner, base, itemType, classAliases, functionAliases, constAliases)
		}
		return
	}
	path := strings.TrimPrefix(NameText(name), `\`)
	if prefix != "" {
		path = strings.Trim(prefix, `\`) + `\` + path
	}
	path = strings.Trim(path, `\`)
	aliasName := ""
	if alias != nil {
		aliasName = NameText(alias)
	} else {
		aliasName = unqualifiedSyntaxName(path)
	}
	if aliasName == "" || path == "" {
		return
	}
	aliases := classAliases
	switch itemType {
	case "function":
		aliases = functionAliases
	case "const":
		aliases = constAliases
	}
	if aliases == nil {
		return
	}
	aliases[strings.ToLower(aliasName)] = path
}

func unqualifiedSyntaxName(path string) string {
	path = strings.Trim(path, `\`)
	if i := strings.LastIndexByte(path, '\\'); i >= 0 {
		return path[i+1:]
	}
	return path
}

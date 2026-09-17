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
		for _, c := range n.Children() {
			if c.Green != nil && c.Green.IsToken() {
				tok, ok := c.Green.Token()
				if !ok {
					continue
				}
				if tok.Type == token.T_STRING || tok.Type == token.T_NS_SEPARATOR ||
					tok.Type == token.T_NAMESPACE || tok.Type == token.T_SELF ||
					tok.Type == token.T_PARENT || tok.Type == token.T_STATIC {
					if tok.Literal != "" {
						b = append(b, tok.Literal...)
					} else {
						// Position-independent green: slice significant token text from red span.
						s := c.Span()
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
						start := s.Start + lead
						end := start + sig
						if c.File != nil && start >= 0 && end <= len(c.File.Source) && start <= end {
							b = append(b, c.File.Source[start:end]...)
						}
					}
				}
			}
		}
		return string(b)
	default:
		return n.Text()
	}
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

// Walk performs a pre-order traversal of the CST rooted at root, calling fn
// for each node. If fn returns false, that node's children are skipped
// (unlike the lowered ast.Node walkers, this supports early subtree exit).
func Walk(root *RedNode, fn func(*RedNode) bool) {
	if root == nil || fn == nil {
		return
	}
	if !fn(root) {
		return
	}
	for _, c := range root.Children() {
		Walk(c, fn)
	}
}

// FirstChildOfKind returns the first direct child with the given kind.
func (r *RedNode) FirstChildOfKind(k Kind) *RedNode {
	for _, c := range r.Children() {
		if c.Kind() == k {
			return c
		}
	}
	return nil
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
			for _, c := range n.Children() {
				switch c.Kind() {
				case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName:
					ns = strings.TrimPrefix(NameText(c), `\`)
				}
			}
		case KindUseDecl:
			collectUseAliases(n, aliases, nil, nil)
			return // don't double-walk clauses
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(f.Root)
	return ns, aliases
}

func collectUseAliases(useDecl *RedNode, classAliases, functionAliases, constAliases map[string]string) {
	useType := "class"
	for _, c := range useDecl.Children() {
		if c.Green != nil && c.Green.IsToken() {
			switch c.Green.TokenType() {
			case token.T_FUNCTION:
				useType = "function"
			case token.T_CONST:
				useType = "const"
			}
		}
	}
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
	for _, c := range clause.Children() {
		switch {
		case c.Green != nil && c.Green.IsToken() && c.Green.TokenType() == token.T_FUNCTION:
			itemType = "function"
		case c.Green != nil && c.Green.IsToken() && c.Green.TokenType() == token.T_CONST:
			itemType = "const"
		case c.Kind() == KindUnqualifiedName || c.Kind() == KindQualifiedName || c.Kind() == KindFullyQualifiedName || c.Kind() == KindRelativeName:
			if name == nil {
				name = c
			} else {
				alias = c
			}
		case c.Kind() == KindUseGroup:
			group = c
		}
	}
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

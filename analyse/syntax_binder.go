package analyse

import (
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/ayanozturk/go-php-parser/syntax"
	"github.com/ayanozturk/go-php-parser/token"
)

// BindMode selects declaration-tier vs complete reference binding (R3).
// Do not conflate: declaration indexing is for symbol discovery only;
// project-wide rename/refs require BindModeReferences (full parse+bind).
type BindMode int

const (
	// BindModeDeclarations parses with SkipFunctionBodies — symbol discovery only.
	BindModeDeclarations BindMode = iota
	// BindModeReferences fully parses and binds — required before project rename/refs.
	BindModeReferences
)

// NameUse records a bound name occurrence for references/rename.
type NameUse struct {
	NodeID    int
	URI       string
	Span      syntax.Span // exact leaf-identifier byte span (no trivia)
	StartLine int         // 0-based LSP line
	StartChar int         // 0-based UTF-16 code units
	EndLine   int
	EndChar   int
	Written   string
	Resolved  string // FQN when known (no leading \)
	Kind      string // class|interface|trait|enum|function|method|property|const|attr|type|name
	Owner     string // owning type FQN for members (no leading \)
}

// UsageGraph maps name-node ids to resolved symbols for one file (or merge unit).
type UsageGraph struct {
	Uses []NameUse
}

// ProjectUsageGraph is a project-scoped binder index: URI → uses, with
// reverse lookup by resolved FQN (+ kind/owner). Unqualified spelling maps are
// not used for project rename/refs (avoids A\Foo vs B\Foo collisions — R2).
type ProjectUsageGraph struct {
	mu        sync.RWMutex
	byURI     map[string][]NameUse
	byResolve map[string][]NameUse // lower(resolved) → uses
}

func NewProjectUsageGraph() *ProjectUsageGraph {
	return &ProjectUsageGraph{
		byURI:     make(map[string][]NameUse),
		byResolve: make(map[string][]NameUse),
	}
}

// PutFile replaces all uses for uri.
func (g *ProjectUsageGraph) PutFile(uri string, uses []NameUse) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeFileLocked(uri)
	if len(uses) == 0 {
		return
	}
	copied := append([]NameUse(nil), uses...)
	for i := range copied {
		copied[i].URI = uri
	}
	g.byURI[uri] = copied
	for _, u := range copied {
		if r := strings.ToLower(strings.TrimPrefix(u.Resolved, `\`)); r != "" {
			g.byResolve[r] = append(g.byResolve[r], u)
		}
	}
}

// RemoveFile drops uses for uri.
func (g *ProjectUsageGraph) RemoveFile(uri string) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeFileLocked(uri)
}

func (g *ProjectUsageGraph) removeFileLocked(uri string) {
	old := g.byURI[uri]
	delete(g.byURI, uri)
	if len(old) == 0 {
		return
	}
	for _, u := range old {
		if r := strings.ToLower(strings.TrimPrefix(u.Resolved, `\`)); r != "" {
			g.byResolve[r] = filterUsesURI(g.byResolve[r], uri)
			if len(g.byResolve[r]) == 0 {
				delete(g.byResolve, r)
			}
		}
	}
}

func filterUsesURI(in []NameUse, uri string) []NameUse {
	out := in[:0]
	for _, u := range in {
		if u.URI != uri {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return append([]NameUse(nil), out...)
}

// FindByResolved returns uses whose Resolved FQN equals name (no unqualified fallback).
func (g *ProjectUsageGraph) FindByResolved(name string) []NameUse {
	if g == nil || name == "" {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	key := strings.ToLower(strings.TrimPrefix(name, `\`))
	return append([]NameUse(nil), g.byResolve[key]...)
}

// FindMatching returns project uses matching the bound symbol identity at the cursor:
// same Resolved FQN, Kind, and Owner (when set). Never matches via unqualified spelling.
func (g *ProjectUsageGraph) FindMatching(needle NameUse) []NameUse {
	if g == nil {
		return nil
	}
	resolved := strings.ToLower(strings.TrimPrefix(needle.Resolved, `\`))
	if resolved == "" {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []NameUse
	for _, u := range g.byResolve[resolved] {
		if needle.Kind != "" && u.Kind != "" && !kindsCompatible(needle.Kind, u.Kind) {
			continue
		}
		if needle.Owner != "" || u.Owner != "" {
			if strings.ToLower(strings.TrimPrefix(needle.Owner, `\`)) !=
				strings.ToLower(strings.TrimPrefix(u.Owner, `\`)) {
				continue
			}
		}
		out = append(out, u)
	}
	return out
}

// FindByName returns project uses with the given resolved FQN.
// Unqualified names only match globally-resolved uses (Resolved == name), never
// every Foo spelling across namespaces (R2 gate).
func (g *ProjectUsageGraph) FindByName(name string) []NameUse {
	return g.FindByResolved(name)
}

func kindsCompatible(a, b string) bool {
	if a == b {
		return true
	}
	classLike := func(k string) bool {
		switch k {
		case "class", "interface", "trait", "enum", "type", "name", "attr":
			return true
		}
		return false
	}
	if classLike(a) && classLike(b) {
		return true
	}
	return false
}

// Binder resolves syntax.Name nodes using the current namespace + import aliases.
type Binder struct {
	Namespace string
	Aliases   map[string]string
	URI       string
	Owner     string // current class-like FQN for member ownership
	NextID    int
	Graph     UsageGraph
	src       []byte
	lines     token.LineTable
}

func NewBinder(namespace string, aliases map[string]string) *Binder {
	if aliases == nil {
		aliases = map[string]string{}
	}
	return &Binder{Namespace: namespace, Aliases: aliases}
}

// BindName resolves a syntax name node and records a usage with exact leaf span.
func (b *Binder) BindName(n *syntax.RedNode, kind string) string {
	if n == nil || b == nil {
		return ""
	}
	written := syntax.NameText(n)
	resolved := b.resolve(written)
	b.NextID++
	span := nameLeafSpan(n)
	use := NameUse{
		NodeID:   b.NextID,
		URI:      b.URI,
		Span:     span,
		Written:  written,
		Resolved: resolved,
		Kind:     kind,
		Owner:    b.Owner,
	}
	if len(b.src) > 0 && len(b.lines) > 0 {
		sl, sc := offsetToUTF16(b.src, b.lines, span.Start)
		el, ec := offsetToUTF16(b.src, b.lines, span.End)
		use.StartLine, use.StartChar = sl, sc
		use.EndLine, use.EndChar = el, ec
	} else if n.File != nil && len(n.File.Lines) > 0 {
		src := n.File.Source
		sl, sc := offsetToUTF16(src, n.File.Lines, span.Start)
		el, ec := offsetToUTF16(src, n.File.Lines, span.End)
		use.StartLine, use.StartChar = sl, sc
		use.EndLine, use.EndChar = el, ec
	}
	b.Graph.Uses = append(b.Graph.Uses, use)
	return resolved
}

// nameLeafSpan returns the byte span of the last identifier token in a name
// (exact rename/edit range — no leading trivia / namespace prefixes).
func nameLeafSpan(n *syntax.RedNode) syntax.Span {
	if n == nil {
		return syntax.Span{}
	}
	toks := n.Tokens()
	var last *token.Token
	for i := range toks {
		t := &toks[i]
		if t.Type == token.T_STRING {
			last = t
		}
	}
	if last != nil && last.End.Offset >= last.Pos.Offset {
		return syntax.Span{Start: last.Pos.Offset, End: last.End.Offset}
	}
	children := n.Children()
	for i := len(children) - 1; i >= 0; i-- {
		c := children[i]
		if c.Green != nil && c.Green.IsToken() {
			switch c.Green.TokenType() {
			case token.T_STRING:
				return significantTokenSpan(c)
			case token.T_NS_SEPARATOR, token.T_NAMESPACE:
				continue
			}
		}
		if c.Kind() == syntax.KindUnqualifiedName {
			return nameLeafSpan(c)
		}
	}
	return n.Span()
}

func significantTokenSpan(n *syntax.RedNode) syntax.Span {
	if n == nil || n.Green == nil || !n.Green.IsToken() {
		if n == nil {
			return syntax.Span{}
		}
		return n.Span()
	}
	tok, ok := n.Green.Token()
	if !ok {
		return n.Span()
	}
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
	start := n.Offset + lead
	return syntax.Span{Start: start, End: start + sig}
}

// offsetToUTF16 returns 0-based line and UTF-16 code-unit column for a byte offset.
func offsetToUTF16(src []byte, lines token.LineTable, offset int) (line, col int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	line, _ = offsetLineCol(lines, offset)
	lineStart := 0
	if line >= 0 && line < len(lines) {
		lineStart = lines[line]
	}
	if lineStart > offset {
		lineStart = 0
	}
	col = utf16CodeUnits(src[lineStart:offset])
	return line, col
}

func utf16CodeUnits(b []byte) int {
	units := 0
	for len(b) > 0 {
		r, size := decodeRune(b)
		units += utf16.RuneLen(r)
		if size <= 0 {
			break
		}
		b = b[size:]
	}
	return units
}

func decodeRune(b []byte) (r rune, size int) {
	if len(b) == 0 {
		return 0, 0
	}
	if b[0] < 0x80 {
		return rune(b[0]), 1
	}
	switch {
	case b[0]&0xE0 == 0xC0 && len(b) >= 2:
		return rune(b[0]&0x1F)<<6 | rune(b[1]&0x3F), 2
	case b[0]&0xF0 == 0xE0 && len(b) >= 3:
		return rune(b[0]&0x0F)<<12 | rune(b[1]&0x3F)<<6 | rune(b[2]&0x3F), 3
	case b[0]&0xF8 == 0xF0 && len(b) >= 4:
		return rune(b[0]&0x07)<<18 | rune(b[1]&0x3F)<<12 | rune(b[2]&0x3F)<<6 | rune(b[3]&0x3F), 4
	default:
		return rune(b[0]), 1
	}
}

// offsetLineCol returns 0-based line and byte column for a byte offset.
func offsetLineCol(lines token.LineTable, offset int) (line, col int) {
	if len(lines) == 0 {
		return 0, offset
	}
	lo, hi := 0, len(lines)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if lines[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	idx := hi
	if idx < 0 {
		idx = 0
	}
	return idx, offset - lines[idx]
}

// UseAtOffset returns the innermost NameUse whose Span covers offset.
// Spans are treated as inclusive on End so a caret after the last char still hits.
func UseAtOffset(uses []NameUse, offset int) (NameUse, bool) {
	var best NameUse
	found := false
	bestLen := int(^uint(0) >> 1)
	for _, u := range uses {
		if u.Span.End <= u.Span.Start {
			continue
		}
		if offset < u.Span.Start || offset > u.Span.End {
			continue
		}
		spanLen := u.Span.End - u.Span.Start
		if !found || spanLen < bestLen || (spanLen == bestLen && u.Span.Start >= best.Span.Start) {
			best = u
			bestLen = spanLen
			found = true
		}
	}
	return best, found
}

// BindModeName documents the two index modes (R3).
func BindModeName(m BindMode) string {
	switch m {
	case BindModeDeclarations:
		return "declarations"
	case BindModeReferences:
		return "references"
	default:
		return "unknown"
	}
}

// BindFile binds names under the given mode (R3 separation).
func BindFile(uri string, src []byte, mode BindMode) UsageGraph {
	switch mode {
	case BindModeDeclarations:
		return bindSyntaxFile(uri, src, true)
	default:
		return bindSyntaxFile(uri, src, false)
	}
}

// BindSyntaxFile fully parses src and records every name use (reference-complete).
// Namespace scopes and import aliases are taken per-namespace from the syntax tree.
func BindSyntaxFile(uri string, src []byte, namespace string, aliases map[string]string) UsageGraph {
	_ = namespace
	_ = aliases
	return bindSyntaxFile(uri, src, false)
}

// BindSyntaxFileForIndex parses src skipping function/method bodies (declaration index).
// Not sufficient alone for project-wide rename/refs (R3).
func BindSyntaxFileForIndex(uri string, src []byte, namespace string, aliases map[string]string) UsageGraph {
	_ = namespace
	_ = aliases
	return bindSyntaxFile(uri, src, true)
}

// BindSyntaxResult binds names from an already-parsed syntax tree (no re-lex/re-parse).
// Callers that already hold a ParseResult should use this instead of BindSyntaxFile*
// so index/rename paths share one parse (Strom dual-lex cutover vertical slice).
func BindSyntaxResult(uri string, res *syntax.ParseResult) UsageGraph {
	if res == nil || res.File == nil || res.File.Root == nil {
		return UsageGraph{}
	}
	b := NewBinder("", map[string]string{})
	b.URI = uri
	b.src = res.File.Source
	b.lines = res.File.Lines
	w := &binderWalk{b: b}
	w.walkFile(res.File.Root)
	return b.Graph
}

func bindSyntaxFile(uri string, src []byte, skipBodies bool) UsageGraph {
	var res *syntax.ParseResult
	if skipBodies {
		res = syntax.ParseForIndex(src)
	} else {
		res = syntax.Parse(src)
	}
	return BindSyntaxResult(uri, res)
}

type binderWalk struct {
	b *Binder
}

type scopeSnap struct {
	ns      string
	aliases map[string]string
	owner   string
}

func (w *binderWalk) saveScope() scopeSnap {
	aliases := make(map[string]string, len(w.b.Aliases))
	for k, v := range w.b.Aliases {
		aliases[k] = v
	}
	return scopeSnap{ns: w.b.Namespace, aliases: aliases, owner: w.b.Owner}
}

func (w *binderWalk) restoreScope(s scopeSnap) {
	w.b.Namespace = s.ns
	w.b.Aliases = s.aliases
	w.b.Owner = s.owner
}

func (w *binderWalk) walkFile(n *syntax.RedNode) {
	if n == nil {
		return
	}
	if n.Kind() == syntax.KindStatementList {
		w.walkStatementList(n)
		return
	}
	for _, c := range n.Children() {
		if c.Kind() == syntax.KindStatementList {
			w.walkStatementList(c)
			continue
		}
		w.walk(c)
	}
}

func (w *binderWalk) walkStatementList(n *syntax.RedNode) {
	for _, c := range n.Children() {
		switch c.Kind() {
		case syntax.KindNamespaceDecl:
			w.walkNamespace(c)
		case syntax.KindUseDecl:
			syntax.AppendUseAliases(c, w.b.Aliases)
			w.walkUseNames(c)
		default:
			w.walk(c)
		}
	}
}

func (w *binderWalk) walkNamespace(n *syntax.RedNode) {
	nsName := ""
	var body *syntax.RedNode
	for _, c := range n.Children() {
		switch c.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
			syntax.KindFullyQualifiedName, syntax.KindRelativeName:
			nsName = strings.TrimPrefix(syntax.NameText(c), `\`)
			w.b.BindName(c, "name")
		case syntax.KindStatementList:
			body = c
		}
	}
	if body != nil {
		prev := w.saveScope()
		w.b.Namespace = nsName
		w.b.Aliases = map[string]string{}
		w.b.Owner = ""
		w.walkStatementList(body)
		w.restoreScope(prev)
		return
	}
	w.b.Namespace = nsName
	w.b.Aliases = map[string]string{}
	w.b.Owner = ""
}

func (w *binderWalk) walkUseNames(useDecl *syntax.RedNode) {
	var walk func(*syntax.RedNode)
	walk = func(n *syntax.RedNode) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
			syntax.KindFullyQualifiedName, syntax.KindRelativeName:
			w.b.BindName(n, "name")
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(useDecl)
}

func (w *binderWalk) walk(n *syntax.RedNode) {
	if n == nil {
		return
	}
	switch n.Kind() {
	case syntax.KindNamespaceDecl:
		w.walkNamespace(n)
		return
	case syntax.KindUseDecl:
		syntax.AppendUseAliases(n, w.b.Aliases)
		w.walkUseNames(n)
		return
	case syntax.KindStatementList:
		w.walkStatementList(n)
		return
	case syntax.KindClassDecl, syntax.KindInterfaceDecl, syntax.KindTraitDecl, syntax.KindEnumDecl:
		w.walkClassLike(n)
		return
	case syntax.KindFunctionDecl, syntax.KindMethodDecl:
		w.walkFunctionLike(n)
		return
	case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
		syntax.KindFullyQualifiedName, syntax.KindRelativeName:
		w.b.BindName(n, nameKind(n))
		return
	}
	for _, c := range n.Children() {
		w.walk(c)
	}
}

func (w *binderWalk) walkClassLike(n *syntax.RedNode) {
	kind := "class"
	switch n.Kind() {
	case syntax.KindInterfaceDecl:
		kind = "interface"
	case syntax.KindTraitDecl:
		kind = "trait"
	case syntax.KindEnumDecl:
		kind = "enum"
	}
	var nameNode *syntax.RedNode
	for _, c := range n.Children() {
		switch c.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
			syntax.KindFullyQualifiedName, syntax.KindRelativeName:
			if nameNode == nil {
				nameNode = c
			}
		}
	}
	prevOwner := w.b.Owner
	if nameNode != nil {
		fqn := w.b.BindName(nameNode, kind)
		w.b.Owner = fqn
	}
	for _, c := range n.Children() {
		if c == nameNode {
			continue
		}
		switch c.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
			syntax.KindFullyQualifiedName, syntax.KindRelativeName:
			continue
		case syntax.KindMemberList, syntax.KindStatementList:
			for _, m := range c.Children() {
				w.walkMember(m)
			}
		default:
			w.walk(c)
		}
	}
	w.b.Owner = prevOwner
}

func (w *binderWalk) walkMember(n *syntax.RedNode) {
	if n == nil {
		return
	}
	switch n.Kind() {
	case syntax.KindFunctionDecl, syntax.KindMethodDecl:
		w.walkFunctionLike(n)
	case syntax.KindClassConstDecl:
		w.walkClassConst(n)
	case syntax.KindPropertyDecl:
		for _, c := range n.Children() {
			w.walk(c)
		}
	default:
		w.walk(n)
	}
}

func (w *binderWalk) walkFunctionLike(n *syntax.RedNode) {
	kind := "function"
	if w.b.Owner != "" || n.Kind() == syntax.KindMethodDecl {
		kind = "method"
	}
	var nameNode *syntax.RedNode
	for _, c := range n.Children() {
		if c.Kind() == syntax.KindUnqualifiedName && nameNode == nil {
			nameNode = c
		}
	}
	if nameNode != nil {
		w.b.BindName(nameNode, kind)
	}
	for _, c := range n.Children() {
		if c == nameNode {
			continue
		}
		w.walk(c)
	}
}

func (w *binderWalk) walkClassConst(n *syntax.RedNode) {
	seenNameish := false
	for _, c := range n.Children() {
		switch c.Kind() {
		case syntax.KindNamedType, syntax.KindNullableType, syntax.KindUnionType,
			syntax.KindIntersectionType, syntax.KindParenthesizedType, syntax.KindPrimitiveType,
			syntax.KindCallableType:
			seenNameish = true
			w.walk(c)
		case syntax.KindUnqualifiedName:
			w.b.BindName(c, "const")
			seenNameish = true
			_ = seenNameish
		default:
			w.walk(c)
		}
	}
}

func nameKind(n *syntax.RedNode) string {
	if n == nil {
		return "name"
	}
	if p := n.Parent; p != nil {
		switch p.Kind() {
		case syntax.KindNamedType:
			return "type"
		case syntax.KindAttribute:
			return "attr"
		case syntax.KindClassDecl:
			return "class"
		case syntax.KindInterfaceDecl:
			return "interface"
		case syntax.KindTraitDecl:
			return "trait"
		case syntax.KindEnumDecl:
			return "enum"
		case syntax.KindFunctionDecl:
			return "function"
		case syntax.KindMethodDecl:
			return "method"
		case syntax.KindClassConstDecl:
			return "const"
		case syntax.KindExtendsClause, syntax.KindImplementsClause, syntax.KindUseTraitClause:
			return "class"
		}
	}
	return "name"
}

// TypeFromSyntax builds analyse.Type from a syntax type node + binder.
// Native types never go through ParseType(raw string) at this boundary.
func (b *Binder) TypeFromSyntax(n *syntax.RedNode) Type {
	if n == nil {
		return Type{}
	}
	switch n.Kind() {
	case syntax.KindNullableType:
		inner := b.TypeFromSyntax(n.Child(1))
		return nullableType(inner)
	case syntax.KindUnionType:
		var parts []Type
		for _, c := range n.Children() {
			if c.Kind() == syntax.KindToken {
				continue
			}
			parts = append(parts, b.TypeFromSyntax(c))
		}
		return unionTypes(parts...)
	case syntax.KindIntersectionType:
		var parts []Type
		for _, c := range n.Children() {
			if c.Kind() == syntax.KindToken {
				continue
			}
			parts = append(parts, b.TypeFromSyntax(c))
		}
		return intersectionTypes(parts...)
	case syntax.KindParenthesizedType:
		for _, c := range n.Children() {
			if c.Kind() != syntax.KindToken {
				return b.TypeFromSyntax(c)
			}
		}
		return Type{}
	case syntax.KindPrimitiveType:
		return ParseType(strings.ToLower(strings.TrimSpace(n.Text())))
	case syntax.KindNamedType:
		name := n.FirstChildOfKind(syntax.KindUnqualifiedName)
		if name == nil {
			name = n.FirstChildOfKind(syntax.KindQualifiedName)
		}
		if name == nil {
			name = n.FirstChildOfKind(syntax.KindFullyQualifiedName)
		}
		if name == nil {
			name = n.FirstChildOfKind(syntax.KindRelativeName)
		}
		if name == nil && len(n.Children()) > 0 {
			name = n.Child(0)
		}
		fqn := b.BindName(name, "type")
		return ClassType(fqn)
	case syntax.KindCallableType:
		b.bindCallableTypeNames(n)
		return ParseType("callable")
	default:
		return ParseType(syntax.TypeText(n))
	}
}

func (b *Binder) bindCallableTypeNames(n *syntax.RedNode) {
	if n == nil {
		return
	}
	switch n.Kind() {
	case syntax.KindCallableParamList, syntax.KindCallableParam, syntax.KindCallableType:
		for _, c := range n.Children() {
			if c.Kind() == syntax.KindToken {
				continue
			}
			b.bindCallableTypeNames(c)
		}
	case syntax.KindNamedType, syntax.KindNullableType, syntax.KindUnionType,
		syntax.KindIntersectionType, syntax.KindParenthesizedType,
		syntax.KindPrimitiveType:
		b.TypeFromSyntax(n)
	default:
		for _, c := range n.Children() {
			if c.Kind() == syntax.KindToken {
				continue
			}
			b.bindCallableTypeNames(c)
		}
	}
}

func (b *Binder) resolve(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, `\`) {
		return strings.TrimPrefix(name, `\`)
	}
	if strings.HasPrefix(name, `namespace\`) {
		rest := strings.TrimPrefix(name, `namespace\`)
		if b.Namespace == "" {
			return rest
		}
		return b.Namespace + `\` + rest
	}
	if i := strings.IndexByte(name, '\\'); i >= 0 {
		first, rest := name[:i], name[i+1:]
		if target, ok := b.Aliases[strings.ToLower(first)]; ok {
			return target + `\` + rest
		}
		if b.Namespace != "" {
			return b.Namespace + `\` + name
		}
		return name
	}
	if target, ok := b.Aliases[strings.ToLower(name)]; ok {
		return target
	}
	switch strings.ToLower(name) {
	case "self", "static", "parent":
		return name
	}
	if b.Namespace != "" {
		return b.Namespace + `\` + name
	}
	return name
}

func nullableType(t Type) Type {
	if t.IsEmpty() {
		return ParseType("null")
	}
	return ParseType(t.dnfString() + "|null")
}

func unionTypes(parts ...Type) Type {
	var names []string
	for _, p := range parts {
		if p.IsEmpty() {
			continue
		}
		s := p.dnfString()
		if len(splitTopLevelTypes(stripBalancedOuterTypeParens(s), '&')) > 1 {
			s = "(" + stripBalancedOuterTypeParens(s) + ")"
		}
		names = append(names, s)
	}
	if len(names) == 0 {
		return Type{}
	}
	return ParseType(strings.Join(names, "|"))
}

func intersectionTypes(parts ...Type) Type {
	var names []string
	for _, p := range parts {
		if p.IsEmpty() {
			continue
		}
		names = append(names, p.dnfString())
	}
	if len(names) == 0 {
		return Type{}
	}
	return ParseType(strings.Join(names, "&"))
}

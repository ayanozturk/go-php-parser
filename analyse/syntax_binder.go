package analyse

import (
	"strings"
	"sync"

	"github.com/ayanozturk/go-php-parser/syntax"
	"github.com/ayanozturk/go-php-parser/token"
)

// NameUse records a bound name occurrence for references/rename.
type NameUse struct {
	NodeID    int
	URI       string
	Span      syntax.Span
	StartLine int // 0-based LSP line
	StartChar int // 0-based UTF-16-ish byte column (byte offset on line)
	EndLine   int
	EndChar   int
	Written   string
	Resolved  string // FQN when known (no leading \)
	Kind      string // class|function|const|attr|type|name
}

// UsageGraph maps name-node ids to resolved symbols for one file (or merge unit).
type UsageGraph struct {
	Uses []NameUse
}

// ProjectUsageGraph is a project-scoped binder index: URI → uses, with
// reverse lookup by resolved / written name for cross-file references.
type ProjectUsageGraph struct {
	mu        sync.RWMutex
	byURI     map[string][]NameUse
	byResolve map[string][]NameUse // lower(resolved) → uses
	byWritten map[string][]NameUse // lower(unqualified written) → uses
}

func NewProjectUsageGraph() *ProjectUsageGraph {
	return &ProjectUsageGraph{
		byURI:     make(map[string][]NameUse),
		byResolve: make(map[string][]NameUse),
		byWritten: make(map[string][]NameUse),
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
		w := u.Written
		if i := strings.LastIndexByte(w, '\\'); i >= 0 {
			w = w[i+1:]
		}
		if w = strings.ToLower(w); w != "" {
			g.byWritten[w] = append(g.byWritten[w], u)
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
		w := u.Written
		if i := strings.LastIndexByte(w, '\\'); i >= 0 {
			w = w[i+1:]
		}
		if w = strings.ToLower(w); w != "" {
			g.byWritten[w] = filterUsesURI(g.byWritten[w], uri)
			if len(g.byWritten[w]) == 0 {
				delete(g.byWritten, w)
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

// FindByName returns project uses matching an unqualified or FQN name.
func (g *ProjectUsageGraph) FindByName(name string) []NameUse {
	if g == nil || name == "" {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	key := strings.ToLower(strings.TrimPrefix(name, `\`))
	seen := map[string]struct{}{}
	var out []NameUse
	add := func(list []NameUse) {
		for _, u := range list {
			id := u.URI + ":" + itoa(u.NodeID) + ":" + itoa(u.Span.Start)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, u)
		}
	}
	add(g.byResolve[key])
	if i := strings.LastIndexByte(key, '\\'); i >= 0 {
		add(g.byWritten[key[i+1:]])
	} else {
		add(g.byWritten[key])
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

// Binder resolves syntax.Name nodes using namespace + imports.
type Binder struct {
	Namespace string
	Aliases   map[string]string
	URI       string
	NextID    int
	Graph     UsageGraph
}

func NewBinder(namespace string, aliases map[string]string) *Binder {
	if aliases == nil {
		aliases = map[string]string{}
	}
	return &Binder{Namespace: namespace, Aliases: aliases}
}

// BindName resolves a syntax name node and records a usage.
func (b *Binder) BindName(n *syntax.RedNode, kind string) string {
	if n == nil || b == nil {
		return ""
	}
	written := syntax.NameText(n)
	resolved := b.resolve(written)
	b.NextID++
	span := n.Span()
	use := NameUse{
		NodeID:   b.NextID,
		URI:      b.URI,
		Span:     span,
		Written:  written,
		Resolved: resolved,
		Kind:     kind,
	}
	if n.File != nil && len(n.File.Lines) > 0 {
		sl, sc := offsetLineCol(n.File.Lines, span.Start)
		el, ec := offsetLineCol(n.File.Lines, span.End)
		use.StartLine, use.StartChar = sl, sc
		use.EndLine, use.EndChar = el, ec
	}
	b.Graph.Uses = append(b.Graph.Uses, use)
	return resolved
}

// offsetLineCol returns 0-based line and column for a byte offset.
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

// BindSyntaxFile parses src into green/red and records every name use.
// Namespace and import aliases are taken from the syntax tree (including group-use).
func BindSyntaxFile(uri string, src []byte, namespace string, aliases map[string]string) UsageGraph {
	res := syntax.Parse(src)
	if res == nil || res.File == nil || res.File.Root == nil {
		return UsageGraph{}
	}
	syntaxNS, syntaxAliases := syntax.NamespaceAndAliases(res.File)
	if namespace == "" {
		namespace = syntaxNS
	}
	if len(aliases) == 0 {
		aliases = syntaxAliases
	} else {
		// Prefer caller aliases, fill missing from syntax (group-use).
		for k, v := range syntaxAliases {
			if _, ok := aliases[k]; !ok {
				aliases[k] = v
			}
		}
	}
	b := NewBinder(namespace, aliases)
	b.URI = uri
	var walk func(*syntax.RedNode)
	walk = func(n *syntax.RedNode) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
			syntax.KindFullyQualifiedName, syntax.KindRelativeName:
			kind := "name"
			if p := n.Parent; p != nil {
				switch p.Kind() {
				case syntax.KindNamedType:
					kind = "type"
				case syntax.KindAttribute:
					kind = "attr"
				case syntax.KindClassDecl, syntax.KindInterfaceDecl, syntax.KindTraitDecl, syntax.KindEnumDecl:
					kind = "class"
				case syntax.KindFunctionDecl, syntax.KindMethodDecl:
					kind = "function"
				case syntax.KindExtendsClause, syntax.KindImplementsClause, syntax.KindUseTraitClause:
					kind = "class"
				}
			}
			b.BindName(n, kind)
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	return b.Graph
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
		// Bind nested param/return type names; semantic type is callable.
		b.bindCallableTypeNames(n)
		return ParseType("callable")
	default:
		// Fallback for token-list coverage during cutover.
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
		// Keep intersection alternatives parenthesized so ParseType preserves DNF.
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

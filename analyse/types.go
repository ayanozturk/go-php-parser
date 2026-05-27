package analyse

import (
	"sort"
	"strings"
	"sync"
)

type typeKind int

const (
	typeKindBuiltin typeKind = iota + 1
	typeKindClass
)

type Type struct {
	atoms map[string]typeAtom
}

type typeAtom struct {
	key     string
	display string
	kind    typeKind
}

var builtinTypeNames = map[string]struct{}{
	"array":    {},
	"bool":     {},
	"callable": {},
	"false":    {},
	"float":    {},
	"int":      {},
	"iterable": {},
	"mixed":    {},
	"never":    {},
	"null":     {},
	"object":   {},
	"resource": {},
	"string":   {},
	"true":     {},
	"void":     {},
}

var parsedTypeCache sync.Map

func EmptyType() Type {
	return Type{}
}

func MixedType() Type {
	return ParseType("mixed")
}

func ClassType(name string) Type {
	return ParseType(name)
}

func ParseType(raw string) Type {
	raw = normalizeRawType(raw)
	if raw == "" {
		return EmptyType()
	}

	if strings.HasPrefix(raw, "?") {
		raw = "null|" + strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}

	if cached, ok := parsedTypeCache.Load(raw); ok {
		return cached.(Type)
	}

	t := Type{atoms: make(map[string]typeAtom)}
	for _, part := range splitTopLevelTypes(raw, '|') {
		for _, intersectionPart := range splitTopLevelTypes(part, '&') {
			for _, atom := range normalizeTypeAtoms(intersectionPart) {
				t.atoms[atom.key] = atom
			}
		}
	}

	if len(t.atoms) == 0 {
		return EmptyType()
	}
	parsedTypeCache.Store(raw, t)
	return t
}

func normalizeTypeAtoms(raw string) []typeAtom {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "\\")
	canonical := canonicalizeDocType(raw)
	parts := splitTopLevelTypes(canonical, '|')
	if len(parts) > 1 {
		atoms := make([]typeAtom, 0, len(parts))
		for _, part := range parts {
			for _, intersectionPart := range splitTopLevelTypes(part, '&') {
				atom, ok := normalizeTypeAtom(intersectionPart)
				if ok {
					atoms = append(atoms, atom)
				}
			}
		}
		return atoms
	}
	parts = splitTopLevelTypes(canonical, '&')
	if len(parts) > 1 {
		atoms := make([]typeAtom, 0, len(parts))
		for _, part := range parts {
			atom, ok := normalizeTypeAtom(part)
			if ok {
				atoms = append(atoms, atom)
			}
		}
		return atoms
	}
	atom, ok := normalizeTypeAtom(canonical)
	if !ok {
		return nil
	}
	return []typeAtom{atom}
}

func normalizeRawType(raw string) string {
	raw = strings.TrimSpace(raw)
	for strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")") {
		inner := strings.TrimSpace(raw[1 : len(raw)-1])
		if inner == "" {
			break
		}
		raw = inner
	}
	return raw
}

func (t Type) IsEmpty() bool {
	return len(t.atoms) == 0
}

func (t Type) String() string {
	if t.IsEmpty() {
		return ""
	}

	parts := make([]string, 0, len(t.atoms))
	for _, atom := range t.atoms {
		parts = append(parts, atom.display)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func (t Type) Accepts(actual Type) bool {
	return t.AcceptsWithContext(actual, nil, nil)
}

func (t Type) AcceptsWithContext(actual Type, scope *functionScope, ctx *AnalysisContext) bool {
	if t.IsEmpty() || actual.IsEmpty() {
		return true
	}
	if t.hasBuiltin("mixed") || actual.hasBuiltin("mixed") {
		return true
	}

	for _, actualAtom := range actual.sortedAtoms() {
		// PHPUnit / Mockery mock types are always compatible with any declared
		// type — they implement whatever interface or extend whatever class they
		// were created for. Skipping them avoids false positives in test code.
		if isMockObjectType(actualAtom.display) {
			continue
		}
		matched := false
		for _, declaredAtom := range t.sortedAtoms() {
			if atomsCompatibleWithContext(declaredAtom, actualAtom, scope, ctx) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// isMockObjectType returns true for PHPUnit and Mockery mock framework types
// that are always valid substitutes for the type they were created from.
func isMockObjectType(name string) bool {
	switch name {
	case `PHPUnit\Framework\MockObject\MockObject`,
		`PHPUnit\Framework\MockObject\MockObjectForAbstractClass`,
		`PHPUnit\Framework\MockObject\MockObjectForTrait`,
		`Mockery\MockInterface`,
		`Mockery\LegacyMockInterface`:
		return true
	}
	return false
}

func (t Type) SingleClassName() (string, bool) {
	if len(t.atoms) != 1 {
		return "", false
	}
	for _, atom := range t.atoms {
		if atom.kind == typeKindClass {
			return atom.display, true
		}
	}
	return "", false
}

func (t Type) hasBuiltin(name string) bool {
	_, ok := t.atoms[name]
	return ok
}

func (t Type) withoutBuiltin(name string) Type {
	if t.IsEmpty() || !t.hasBuiltin(name) {
		return t
	}

	refined := Type{atoms: make(map[string]typeAtom, len(t.atoms)-1)}
	for key, atom := range t.atoms {
		if key == name {
			continue
		}
		refined.atoms[key] = atom
	}
	return refined
}

func (t Type) sortedAtoms() []typeAtom {
	atoms := make([]typeAtom, 0, len(t.atoms))
	for _, atom := range t.atoms {
		atoms = append(atoms, atom)
	}
	sort.Slice(atoms, func(i, j int) bool {
		return atoms[i].key < atoms[j].key
	})
	return atoms
}

func normalizeTypeAtom(raw string) (typeAtom, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return typeAtom{}, false
	}

	raw = strings.TrimPrefix(raw, "\\")
	raw = canonicalizeDocType(raw)

	lower := strings.ToLower(raw)
	if _, ok := builtinTypeNames[lower]; ok {
		return typeAtom{key: lower, display: lower, kind: typeKindBuiltin}, true
	}

	trimmed := strings.TrimPrefix(raw, "\\")
	if trimmed == "" {
		return typeAtom{}, false
	}

	return typeAtom{
		key:     "class:" + strings.ToLower(trimmed),
		display: trimmed,
		kind:    typeKindClass,
	}, true
}

func atomsCompatible(declared, actual typeAtom) bool {
	return atomsCompatibleWithContext(declared, actual, nil, nil)
}

func atomsCompatibleWithContext(declared, actual typeAtom, scope *functionScope, ctx *AnalysisContext) bool {
	if declared.key == actual.key {
		return true
	}
	if declared.kind == typeKindBuiltin && actual.kind == typeKindBuiltin {
		if declared.key == "float" && actual.key == "int" {
			return true
		}
		if declared.key == "void" && actual.key == "null" {
			return true
		}
		if declared.key == "bool" && (actual.key == "true" || actual.key == "false") {
			return true
		}
		if declared.key == "iterable" && actual.key == "array" {
			return true
		}
	}
	if declared.kind == typeKindBuiltin && actual.kind == typeKindClass {
		if declared.key == "object" {
			return true
		}
		if declared.key == "iterable" {
			return classHierarchyCompatible("Traversable", actual.display, scope, ctx)
		}
	}
	if declared.kind == typeKindClass && actual.kind == typeKindClass {
		return classHierarchyCompatible(declared.display, actual.display, scope, ctx)
	}
	return false
}

func canonicalizeDocType(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(lower, "[") && strings.HasSuffix(lower, "]") {
		return "array"
	}
	if strings.HasSuffix(lower, "[]") {
		return "array"
	}

	base := lower
	if idx := strings.IndexAny(base, "<("); idx >= 0 {
		base = base[:idx]
	}
	if idx := strings.Index(base, "{"); idx >= 0 {
		base = base[:idx]
	}

	switch base {
	case "boolean":
		return "bool"
	case "integer":
		return "int"
	case "double", "real":
		return "float"
	case "callback":
		return "callable"
	case "list", "non-empty-list", "array-key", "array-shape":
		if base == "array-key" {
			return "int|string"
		}
		return "array"
	case "array", "non-empty-array", "associative-array":
		return "array"
	case "class-string", "interface-string", "trait-string", "literal-string", "non-empty-string", "numeric-string", "lowercase-string":
		return "string"
	case "positive-int", "negative-int", "non-negative-int", "non-positive-int":
		return "int"
	case "scalar":
		return "bool|float|int|string"
	}

	return raw
}

func splitTopLevelTypes(raw string, sep rune) []string {
	var parts []string
	start := 0
	depthAngle := 0
	depthParen := 0
	depthBrace := 0
	for idx, r := range raw {
		switch r {
		case '<':
			depthAngle++
		case '>':
			if depthAngle > 0 {
				depthAngle--
			}
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		default:
			if r == sep && depthAngle == 0 && depthParen == 0 && depthBrace == 0 {
				parts = append(parts, raw[start:idx])
				start = idx + len(string(r))
			}
		}
	}
	parts = append(parts, raw[start:])
	return parts
}

func classHierarchyCompatible(declaredName, actualName string, scope *functionScope, ctx *AnalysisContext) bool {
	declaredName = canonicalClassName(declaredName, scope, ctx)
	actualName = canonicalClassName(actualName, scope, ctx)
	if declaredName == "" || actualName == "" {
		return false
	}
	if strings.EqualFold(declaredName, actualName) {
		return true
	}

	seen := map[string]struct{}{}
	queue := []string{actualName}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		key := strings.ToLower(strings.TrimPrefix(current, `\`))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		resolved, ok := resolveHierarchyClass(current, scope, ctx)
		if !ok {
			continue
		}
		for _, parent := range resolved.Extends {
			parent = canonicalClassName(parent, scope, ctx)
			if parent == "" {
				continue
			}
			if strings.EqualFold(parent, declaredName) {
				return true
			}
			queue = append(queue, parent)
		}
		for _, implemented := range resolved.Implements {
			implemented = canonicalClassName(implemented, scope, ctx)
			if implemented == "" {
				continue
			}
			if strings.EqualFold(implemented, declaredName) {
				return true
			}
			queue = append(queue, implemented)
		}
	}

	return false
}

func canonicalClassName(name string, scope *functionScope, ctx *AnalysisContext) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if name == "" {
		return ""
	}
	if ctx != nil && ctx.Resolver != nil {
		if resolved, ok := ctx.Resolver.ResolveClass(name); ok && resolved.Name != "" {
			return strings.TrimPrefix(resolved.Name, `\`)
		}
	}
	if scope == nil {
		return name
	}
	switch strings.ToLower(name) {
	case "self", "static":
		return strings.TrimPrefix(scope.className, `\`)
	case "parent":
		if resolved, ok := resolveHierarchyClass(scope.className, scope, ctx); ok && len(resolved.Extends) > 0 {
			return strings.TrimPrefix(resolved.Extends[0], `\`)
		}
		return name
	default:
		return strings.TrimPrefix(name, `\`)
	}
}

func resolveHierarchyClass(name string, scope *functionScope, ctx *AnalysisContext) (ResolvedClass, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if trimmed == "" {
		return ResolvedClass{}, false
	}
	if scope != nil {
		if resolved, ok := scope.typeCtx.resolveClass(trimmed); ok {
			return resolved, true
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		return ctx.Resolver.ResolveClass(trimmed)
	}
	return ResolvedClass{}, false
}

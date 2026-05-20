package analyse

import (
	"sort"
	"strings"
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
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return EmptyType()
	}

	if strings.HasPrefix(raw, "?") {
		raw = "null|" + strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}

	t := Type{atoms: make(map[string]typeAtom)}
	for _, part := range strings.Split(raw, "|") {
		atom, ok := normalizeTypeAtom(part)
		if !ok {
			continue
		}
		t.atoms[atom.key] = atom
	}

	if len(t.atoms) == 0 {
		return EmptyType()
	}
	return t
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
	}
	if declared.kind == typeKindClass && actual.kind == typeKindClass {
		return classHierarchyCompatible(declared.display, actual.display, scope, ctx)
	}
	return false
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

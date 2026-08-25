package analyse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// SymbolID is a stable, case-insensitive semantic identity. It deliberately
// excludes declaration offsets so references survive edits that only move a
// declaration within its file.
type SymbolID string

// SourceLocation is a declaration's half-open source span. A blank File and
// zero positions identify synthetic or built-in symbols with no source.
type SourceLocation struct {
	File  string
	Start ast.Position
	End   ast.Position
}

const (
	// FactKindInferredType stores a type inferred for a source expression.
	FactKindInferredType FactKind = "inferred-type"
	// FactKindReference associates a source span with a resolved symbol.
	FactKindReference FactKind = "reference"
)

// FactKind identifies the interpretation of a semantic fact. Callers may
// define additional namespaced kinds without changing the store.
type FactKind string

// SemanticFactKey identifies a fact at an exact source span. Offsets are byte
// offsets, matching the parser's canonical source coordinate.
type SemanticFactKey struct {
	File        string
	StartOffset int
	EndOffset   int
	Kind        FactKind
}

// SemanticFact is intentionally a value-only record so an immutable snapshot
// never returns caller-mutable maps, slices, or pointers.
type SemanticFact struct {
	Key     SemanticFactKey
	Subject SymbolID
	Type    string
	Value   string
}

// SemanticFactReader is the read-only contract shared by analysis passes.
// Facts must come from the same content snapshot as the AST being analysed;
// exact byte-span keys prevent unrelated or shifted facts from being reused.
type SemanticFactReader interface {
	Fact(key SemanticFactKey) (SemanticFact, bool)
	FactsForFile(filename string) []SemanticFact
}

// SemanticSnapshot is an immutable, concurrency-safe view of a project graph
// and its reusable semantic facts. Construction owns the mutable ProjectIndex;
// callers can only query defensive value copies through this facade.
type SemanticSnapshot struct {
	project   *ProjectIndex
	facts     map[SemanticFactKey]SemanticFact
	filenames []string
}

// NewSemanticSnapshot builds a project graph, derives reusable semantic facts,
// and freezes the result. Duplicate explicit fact keys are rejected; explicit
// facts take precedence over generated facts at the same exact source span.
func NewSemanticSnapshot(parsed map[string][]ast.Node, facts []SemanticFact) (*SemanticSnapshot, error) {
	store := make(map[SemanticFactKey]SemanticFact, len(facts))
	for _, fact := range facts {
		if err := validateSemanticFactKey(fact.Key); err != nil {
			return nil, err
		}
		if _, exists := store[fact.Key]; exists {
			return nil, fmt.Errorf("duplicate semantic fact key: %s:%d-%d:%s", fact.Key.File, fact.Key.StartOffset, fact.Key.EndOffset, fact.Key.Kind)
		}
		store[fact.Key] = fact
	}

	filenames := make([]string, 0, len(parsed))
	for filename := range parsed {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	snapshot := &SemanticSnapshot{
		project:   BuildProjectIndex(parsed),
		facts:     store,
		filenames: filenames,
	}
	snapshot.generateInferredTypeFacts(parsed)
	return snapshot, nil
}

func (s *SemanticSnapshot) generateInferredTypeFacts(parsed map[string][]ast.Node) {
	for _, filename := range s.filenames {
		nodes := parsed[filename]
		ctx := s.NewAnalysisContext()
		fileCtx := analysisFileTypeContext(ctx, nodes)

		var walk func(ast.Node, *ast.ClassNode)
		walk = func(node ast.Node, class *ast.ClassNode) {
			switch n := node.(type) {
			case *ast.NamespaceNode:
				for _, child := range n.Body {
					walk(child, class)
				}
			case *ast.ClassNode:
				for _, method := range n.Methods {
					walk(method, n)
				}
			case *ast.FunctionNode:
				scope := analysisFunctionScope(ctx, class, n, fileCtx)
				walkStatementsForArgTypesUsing(n.Body, scope, ctx, filename, nil, func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
					s.addGeneratedInferredTypeFact(filename, expr, inferType(expr, scope, ctx), fileCtx, class, n)
				})
			}
		}

		for _, node := range nodes {
			walk(node, nil)
		}
	}
}

func (s *SemanticSnapshot) addGeneratedInferredTypeFact(filename string, expr ast.Node, inferred Type, fileCtx fileTypeContext, class *ast.ClassNode, function *ast.FunctionNode) {
	if expr == nil || inferred.IsEmpty() {
		return
	}
	start, end := expr.GetPos(), expr.GetEndPos()
	if end.Offset <= start.Offset {
		return
	}
	key := SemanticFactKey{File: filename, StartOffset: start.Offset, EndOffset: end.Offset, Kind: FactKindInferredType}
	if _, explicitOrGenerated := s.facts[key]; explicitOrGenerated {
		return
	}
	s.facts[key] = SemanticFact{Key: key, Subject: s.functionSymbolID(fileCtx, class, function), Type: inferred.String()}
}

func (s *SemanticSnapshot) functionSymbolID(fileCtx fileTypeContext, class *ast.ClassNode, function *ast.FunctionNode) SymbolID {
	if function == nil {
		return ""
	}
	if class != nil {
		className := fileCtx.resolveClassLike(class.Name)
		if method, ok := s.ResolveMethod(className, function.Name); ok {
			return method.ID
		}
		return stableSymbolID("method", className, function.Name)
	}
	functionName := fileCtx.resolveClassLike(function.Name)
	if resolved, ok := s.ResolveFunction(functionName); ok {
		return resolved.ID
	}
	return stableSymbolID("function", "", functionName)
}

func validateSemanticFactKey(key SemanticFactKey) error {
	if key.File == "" {
		return fmt.Errorf("semantic fact file must not be empty")
	}
	if key.Kind == "" {
		return fmt.Errorf("semantic fact kind must not be empty")
	}
	if key.StartOffset < 0 || key.EndOffset < key.StartOffset {
		return fmt.Errorf("invalid semantic fact span %d-%d", key.StartOffset, key.EndOffset)
	}
	return nil
}

// Files returns the snapshot's files in stable lexical order.
func (s *SemanticSnapshot) Files() []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s.filenames...)
}

// NewAnalysisContext creates the compatibility bridge used while rules move
// from direct ProjectIndex access to snapshot and fact-reader queries.
func (s *SemanticSnapshot) NewAnalysisContext() *AnalysisContext {
	if s == nil {
		return &AnalysisContext{}
	}
	return &AnalysisContext{Resolver: s, Project: s.project, Facts: s}
}

// Fact returns the fact registered for an exact source-span key.
func (s *SemanticSnapshot) Fact(key SemanticFactKey) (SemanticFact, bool) {
	if s == nil {
		return SemanticFact{}, false
	}
	fact, ok := s.facts[key]
	return fact, ok
}

// FactsForFile returns facts in deterministic source order.
func (s *SemanticSnapshot) FactsForFile(filename string) []SemanticFact {
	if s == nil {
		return nil
	}
	facts := make([]SemanticFact, 0)
	for key, fact := range s.facts {
		if key.File == filename {
			facts = append(facts, fact)
		}
	}
	sort.Slice(facts, func(i, j int) bool {
		left, right := facts[i].Key, facts[j].Key
		if left.StartOffset != right.StartOffset {
			return left.StartOffset < right.StartOffset
		}
		if left.EndOffset != right.EndOffset {
			return left.EndOffset < right.EndOffset
		}
		return left.Kind < right.Kind
	})
	return facts
}

func (s *SemanticSnapshot) ClassExists(name string) bool {
	_, ok := s.ResolveClass(name)
	return ok
}

func (s *SemanticSnapshot) FunctionExists(name string) bool {
	_, ok := s.ResolveFunction(name)
	return ok
}

func (s *SemanticSnapshot) ConstantExists(name string) bool {
	return s != nil && s.project.ConstantExists(name)
}

func (s *SemanticSnapshot) ResolveClass(name string) (ResolvedClass, bool) {
	if s == nil {
		return ResolvedClass{}, false
	}
	class, ok := s.project.ResolveClass(name)
	if !ok {
		return ResolvedClass{}, false
	}
	if class.ID == "" {
		class.ID = stableSymbolID("class", "", class.Name)
	}
	class.Extends = append([]string(nil), class.Extends...)
	class.Implements = append([]string(nil), class.Implements...)
	class.TemplateParams = append([]string(nil), class.TemplateParams...)
	class.Traits = append([]string(nil), class.Traits...)
	class.GenericParents = cloneGenericParents(class.GenericParents)
	return class, true
}

func (s *SemanticSnapshot) ResolveMethod(className, methodName string) (ResolvedMethod, bool) {
	if s == nil {
		return ResolvedMethod{}, false
	}
	method, ok := s.project.ResolveMethod(className, methodName)
	if !ok {
		return ResolvedMethod{}, false
	}
	if method.ID == "" {
		method.ID = stableSymbolID("method", method.DeclaringClass, method.Name)
	}
	method.Params = append([]ResolvedParam(nil), method.Params...)
	return method, true
}

func (s *SemanticSnapshot) ResolveProperty(className, propertyName string) (ResolvedProperty, bool) {
	if s == nil {
		return ResolvedProperty{}, false
	}
	property, ok := s.project.ResolveProperty(className, propertyName)
	if !ok {
		return ResolvedProperty{}, false
	}
	if property.DeclaringClass == "" {
		property.DeclaringClass = resolvedPropertyOwner(s.project, className, propertyName)
	}
	if property.ID == "" {
		property.ID = stableSymbolID("property", property.DeclaringClass, strings.TrimPrefix(property.Name, "$"))
	}
	return property, true
}

func (s *SemanticSnapshot) ResolveFunction(name string) (ResolvedFunction, bool) {
	if s == nil {
		return ResolvedFunction{}, false
	}
	fn, ok := s.project.ResolveFunction(name)
	if !ok {
		return ResolvedFunction{}, false
	}
	if fn.ID == "" {
		fn.ID = stableSymbolID("function", "", fn.Name)
	}
	fn.Params = append([]ResolvedParam(nil), fn.Params...)
	return fn, true
}

// ResolveConstant resolves a class constant and attaches its stable identity.
func (s *SemanticSnapshot) ResolveConstant(className, constantName string) (ResolvedConstant, bool) {
	if s == nil {
		return ResolvedConstant{}, false
	}
	constant, ok := s.project.ResolveConstant(className, constantName)
	if !ok {
		return ResolvedConstant{}, false
	}
	if constant.ID == "" {
		constant.ID = stableSymbolID("constant", constant.DeclaringClass, constant.Name)
	}
	return constant, true
}

func stableSymbolID(kind, owner, name string) SymbolID {
	parts := []string{strings.ToLower(kind)}
	if owner != "" {
		parts = append(parts, indexKey(owner))
	}
	parts = append(parts, indexKey(name))
	return SymbolID(strings.Join(parts, ":"))
}

func cloneGenericParents(parents []ResolvedGenericParent) []ResolvedGenericParent {
	cloned := make([]ResolvedGenericParent, len(parents))
	for i, parent := range parents {
		cloned[i] = parent
		cloned[i].TypeArguments = append([]string(nil), parent.TypeArguments...)
	}
	return cloned
}

func resolvedPropertyOwner(project *ProjectIndex, className, propertyName string) string {
	for _, candidate := range project.classLineage(className) {
		if _, ok := project.Properties[indexKey(candidate)][strings.ToLower(strings.TrimPrefix(propertyName, "$"))]; ok {
			if class, found := project.ResolveClass(candidate); found {
				return class.Name
			}
			return candidate
		}
	}
	if class, ok := project.ResolveClass(className); ok {
		return class.Name
	}
	return className
}

var _ SymbolResolver = (*SemanticSnapshot)(nil)
var _ SemanticFactReader = (*SemanticSnapshot)(nil)

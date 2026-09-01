package analyse

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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
	// FactKindNarrowed stores type narrowing from instanceof/null checks.
	FactKindNarrowed FactKind = "narrowed-type"
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

// semanticFactSpanKey omits the filename because facts are partitioned by
// file internally. SemanticFactKey remains the public lookup contract, while
// the compact key avoids storing the same filename twice in every map entry.
type semanticFactSpanKey struct {
	StartOffset int
	EndOffset   int
	Kind        FactKind
}

// semanticFactOffsetKey is the compact key used by the three built-in fact
// kinds. Generated facts always address ordinary source files, so their two
// uint32 offsets fit in one map word. The uncommon larger-offset case falls
// back to the fully general custom map and keeps the public int-based contract.
type semanticFactOffsetKey uint64

const maxCompactSemanticFactOffset = uint64(^uint32(0))

func compactSemanticFactOffset(start, end int) (semanticFactOffsetKey, bool) {
	if start < 0 || end < 0 || uint64(start) > maxCompactSemanticFactOffset || uint64(end) > maxCompactSemanticFactOffset {
		return 0, false
	}
	return semanticFactOffsetKey(uint64(uint32(start))<<32 | uint64(uint32(end))), true
}

func (k semanticFactOffsetKey) offsets() (int, int) {
	return int(uint32(uint64(k) >> 32)), int(uint32(k))
}

type storedSemanticFact struct {
	Subject SymbolID
	Type    string
	Value   string
}

type storedGeneratedInferredFact struct {
	Subject SymbolID
	Type    string
}

type semanticFactFileStore struct {
	inferred          map[semanticFactOffsetKey]storedSemanticFact
	generatedInferred map[semanticFactOffsetKey]storedGeneratedInferredFact
	reference         map[semanticFactOffsetKey]storedSemanticFact
	narrowed          map[semanticFactOffsetKey]storedSemanticFact
	custom            map[semanticFactSpanKey]storedSemanticFact
}

type semanticFactStore map[string]*semanticFactFileStore

func (s semanticFactStore) has(key SemanticFactKey) bool {
	_, ok := s.stored(key)
	return ok
}

func (s semanticFactStore) fact(key SemanticFactKey) (SemanticFact, bool) {
	stored, ok := s.stored(key)
	if !ok {
		return SemanticFact{}, false
	}
	return SemanticFact{Key: key, Subject: stored.Subject, Type: stored.Type, Value: stored.Value}, true
}

func (s semanticFactStore) stored(key SemanticFactKey) (storedSemanticFact, bool) {
	fileFacts := s[key.File]
	if fileFacts == nil {
		return storedSemanticFact{}, false
	}
	offset, compact := compactSemanticFactOffset(key.StartOffset, key.EndOffset)
	if !compact {
		fact, ok := fileFacts.custom[semanticFactSpanKey{StartOffset: key.StartOffset, EndOffset: key.EndOffset, Kind: key.Kind}]
		return fact, ok
	}
	switch key.Kind {
	case FactKindInferredType:
		if fact, ok := fileFacts.inferred[offset]; ok {
			return fact, true
		}
		fact, ok := fileFacts.generatedInferred[offset]
		return storedSemanticFact{Subject: fact.Subject, Type: fact.Type}, ok
	case FactKindReference:
		fact, ok := fileFacts.reference[offset]
		return fact, ok
	case FactKindNarrowed:
		fact, ok := fileFacts.narrowed[offset]
		return fact, ok
	default:
		fact, ok := fileFacts.custom[semanticFactSpanKey{StartOffset: key.StartOffset, EndOffset: key.EndOffset, Kind: key.Kind}]
		return fact, ok
	}
}

func (s semanticFactStore) put(fact SemanticFact) bool {
	return s.putParts(fact.Key, fact.Subject, fact.Type, fact.Value)
}

func (s semanticFactStore) putGeneratedInferred(key SemanticFactKey, subject SymbolID, typ string) bool {
	fileFacts := s[key.File]
	if fileFacts == nil {
		fileFacts = &semanticFactFileStore{}
		s[key.File] = fileFacts
	}
	offset, compact := compactSemanticFactOffset(key.StartOffset, key.EndOffset)
	if !compact {
		return s.putParts(key, subject, typ, "")
	}
	if _, exists := fileFacts.inferred[offset]; exists {
		return false
	}
	if fileFacts.generatedInferred == nil {
		fileFacts.generatedInferred = make(map[semanticFactOffsetKey]storedGeneratedInferredFact)
	}
	if _, exists := fileFacts.generatedInferred[offset]; exists {
		return false
	}
	fileFacts.generatedInferred[offset] = storedGeneratedInferredFact{Subject: subject, Type: typ}
	return true
}

func (s semanticFactStore) putParts(key SemanticFactKey, subject SymbolID, typ, value string) bool {
	fileFacts := s[key.File]
	if fileFacts == nil {
		fileFacts = &semanticFactFileStore{}
		s[key.File] = fileFacts
	}
	stored := storedSemanticFact{Subject: subject, Type: typ, Value: value}
	offset, compact := compactSemanticFactOffset(key.StartOffset, key.EndOffset)
	if !compact {
		if fileFacts.custom == nil {
			fileFacts.custom = make(map[semanticFactSpanKey]storedSemanticFact)
		}
		span := semanticFactSpanKey{StartOffset: key.StartOffset, EndOffset: key.EndOffset, Kind: key.Kind}
		if _, exists := fileFacts.custom[span]; exists {
			return false
		}
		fileFacts.custom[span] = stored
		return true
	}
	switch key.Kind {
	case FactKindInferredType:
		if _, exists := fileFacts.generatedInferred[offset]; exists {
			return false
		}
		if fileFacts.inferred == nil {
			fileFacts.inferred = make(map[semanticFactOffsetKey]storedSemanticFact)
		}
		if _, exists := fileFacts.inferred[offset]; exists {
			return false
		}
		fileFacts.inferred[offset] = stored
	case FactKindReference:
		if fileFacts.reference == nil {
			fileFacts.reference = make(map[semanticFactOffsetKey]storedSemanticFact)
		}
		if _, exists := fileFacts.reference[offset]; exists {
			return false
		}
		fileFacts.reference[offset] = stored
	case FactKindNarrowed:
		if fileFacts.narrowed == nil {
			fileFacts.narrowed = make(map[semanticFactOffsetKey]storedSemanticFact)
		}
		if _, exists := fileFacts.narrowed[offset]; exists {
			return false
		}
		fileFacts.narrowed[offset] = stored
	default:
		if fileFacts.custom == nil {
			fileFacts.custom = make(map[semanticFactSpanKey]storedSemanticFact)
		}
		span := semanticFactSpanKey{StartOffset: key.StartOffset, EndOffset: key.EndOffset, Kind: key.Kind}
		if _, exists := fileFacts.custom[span]; exists {
			return false
		}
		fileFacts.custom[span] = stored
	}
	return true
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
	project                 *ProjectIndex
	facts                   semanticFactStore
	flowGraphs              map[FlowScopeKey]ControlFlowGraph
	statementReachability   map[FlowStatementKey]bool
	ambiguousFlowStatements map[FlowStatementKey]struct{}
	scopeNesting            map[FlowScopeKey]FlowScopeKey
	variableReads           map[string][]variableReadFact
	completeVariableReads   map[string]*lazyVariableReadFacts
	filenames               []string
}

type lazyVariableReadFacts struct {
	once     sync.Once
	filename string
	nodes    []ast.Node
	resolver SymbolResolver
	reads    []variableReadFact
}

func (f *lazyVariableReadFacts) complete() []variableReadFact {
	if f == nil {
		return nil
	}
	f.once.Do(func() {
		f.reads = buildVariableFlowFacts(f.filename, f.nodes, true, f.resolver)
		f.nodes = nil
	})
	return f.reads
}

// NewSemanticSnapshot builds a project graph, derives reusable semantic facts,
// and freezes the result. Duplicate explicit fact keys are rejected; explicit
// facts take precedence over generated facts at the same exact source span.
func NewSemanticSnapshot(parsed map[string][]ast.Node, facts []SemanticFact) (*SemanticSnapshot, error) {
	return NewSemanticSnapshotScoped(parsed, facts, nil)
}

// NewSemanticSnapshotScoped is like NewSemanticSnapshot, but when targets is
// non-empty it restricts the expensive per-file semantic work (control-flow
// graphs, variable-flow facts, inferred-type facts, narrowing facts, and the
// resulting Files() list callers iterate to run analysis rules) to just
// those files. parsed still builds the full ProjectIndex regardless, so
// cross-file symbol resolution against files outside targets (e.g. a large
// project indexed only so a single file's class references resolve) keeps
// working — only the O(project size) fact/CFG generation and rule execution
// gets scoped down to what the caller actually wants diagnostics for. A nil
// or empty targets behaves exactly like NewSemanticSnapshot (all parsed
// files are in scope).
func NewSemanticSnapshotScoped(parsed map[string][]ast.Node, facts []SemanticFact, targets []string) (*SemanticSnapshot, error) {
	return newSemanticSnapshot(BuildProjectIndex(parsed), parsed, facts, targets)
}

// NewSemanticSnapshotWithIndex is like NewSemanticSnapshotScoped, but reuses
// an already-built ProjectIndex instead of deriving one from parsed. This
// lets a caller with a warm on-disk symbol-table cache and a small targets
// set (e.g. `analyze <file>` re-run with no project changes) skip parsing
// and indexing every other file in the project: parsed only needs to
// contain the target files themselves, since idx already carries the full
// project's classes/methods/properties/functions for cross-file resolution.
func NewSemanticSnapshotWithIndex(idx *ProjectIndex, parsed map[string][]ast.Node, facts []SemanticFact, targets []string) (*SemanticSnapshot, error) {
	return newSemanticSnapshot(idx, parsed, facts, targets)
}

func newSemanticSnapshot(idx *ProjectIndex, parsed map[string][]ast.Node, facts []SemanticFact, targets []string) (*SemanticSnapshot, error) {
	scoped := parsed
	filenames := make([]string, 0, len(parsed))
	if len(targets) > 0 {
		scoped = make(map[string][]ast.Node, len(targets))
		for _, target := range targets {
			if nodes, ok := parsed[target]; ok {
				scoped[target] = nodes
				filenames = append(filenames, target)
			}
		}
	} else {
		for filename := range parsed {
			filenames = append(filenames, filename)
		}
	}
	sort.Strings(filenames)

	store := make(semanticFactStore, len(filenames))

	// Explicit facts (currently unused by any caller, but kept strict for
	// future callers) must not collide with one another.
	for _, fact := range facts {
		if err := validateSemanticFactKey(fact.Key); err != nil {
			return nil, err
		}
		if !store.put(fact) {
			return nil, fmt.Errorf("duplicate semantic fact key: %s:%d-%d:%s", fact.Key.File, fact.Key.StartOffset, fact.Key.EndOffset, fact.Key.Kind)
		}
	}

	// Generate narrowing facts from control flow. Overlapping conditions
	// (e.g. an elseif re-testing the same expression) can legitimately
	// produce two narrowing facts for the exact same source span; keep the
	// first rather than aborting analysis of the entire project over it.
	for filename, nodes := range scoped {
		insertNarrowingFacts(store, filename, nodes)
	}

	snapshot := &SemanticSnapshot{
		project:   idx,
		facts:     store,
		filenames: filenames,
	}
	snapshot.generateControlFlowGraphs(scoped)
	snapshot.generateVariableFlowFacts(scoped)
	snapshot.generateInferredTypeFacts(scoped)
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
					s.addGeneratedInferredTypeFact(filename, expr, fileCtx, class, n, func() Type {
						return inferType(expr, scope, ctx)
					})
				})
			}
		}

		for _, node := range nodes {
			walk(node, nil)
		}
	}
}

func (s *SemanticSnapshot) addGeneratedInferredTypeFact(filename string, expr ast.Node, fileCtx FileTypeContext, class *ast.ClassNode, function *ast.FunctionNode, infer func() Type) {
	if expr == nil || infer == nil {
		return
	}
	start, end := expr.GetPos(), expr.GetEndPos()
	if end.Offset <= start.Offset {
		return
	}
	key := SemanticFactKey{File: filename, StartOffset: start.Offset, EndOffset: end.Offset, Kind: FactKindInferredType}
	if s.facts.has(key) {
		return
	}
	inferred := infer()
	if inferred.IsEmpty() {
		return
	}
	s.facts.putGeneratedInferred(key, s.functionSymbolID(fileCtx, class, function), inferred.dnfString())
}

func (s *SemanticSnapshot) functionSymbolID(fileCtx FileTypeContext, class *ast.ClassNode, function *ast.FunctionNode) SymbolID {
	if function == nil {
		return ""
	}
	if class != nil {
		className := fileCtx.resolveClassLike(class.Name)
		if method, ok := s.resolveMethodView(className, function.Name); ok {
			return method.ID
		}
		return stableSymbolID("method", className, function.Name)
	}
	functionName := fileCtx.resolveClassLike(function.Name)
	if resolved, ok := s.resolveFunctionView(functionName); ok {
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

// NewAnalysisContext creates a read-only analysis context backed by this
// snapshot's symbol resolver and semantic facts.
func (s *SemanticSnapshot) NewAnalysisContext() *AnalysisContext {
	if s == nil {
		return &AnalysisContext{}
	}
	return &AnalysisContext{Resolver: s, Facts: s, Flow: s, VariableFlow: s}
}

func (s *SemanticSnapshot) generateVariableFlowFacts(parsed map[string][]ast.Node) {
	s.variableReads = make(map[string][]variableReadFact, len(s.filenames))
	s.completeVariableReads = make(map[string]*lazyVariableReadFacts, len(s.filenames))
	for _, filename := range s.filenames {
		nodes := parsed[filename]
		s.variableReads[filename] = buildVariableFlowFacts(filename, nodes, false, s)
		s.completeVariableReads[filename] = &lazyVariableReadFacts{filename: filename, nodes: nodes, resolver: s}
	}
}

// VariableReadsForFile returns variable reads in deterministic source order.
func (s *SemanticSnapshot) VariableReadsForFile(filename string) []VariableReadFact {
	if s == nil {
		return nil
	}
	reads := s.completeVariableReads[filename].complete()
	result := make([]VariableReadFact, len(reads))
	for i, read := range reads {
		result[i] = read.public(filename)
	}
	return result
}

func (s *SemanticSnapshot) rangeVariableReadsForFile(filename string, visit func(VariableReadFact)) {
	if s == nil {
		return
	}
	for _, read := range s.variableReads[filename] {
		visit(read.public(filename))
	}
}

// Fact returns the fact registered for an exact source-span key.
func (s *SemanticSnapshot) Fact(key SemanticFactKey) (SemanticFact, bool) {
	if s == nil {
		return SemanticFact{}, false
	}
	return s.facts.fact(key)
}

// FactsForFile returns facts in deterministic source order.
func (s *SemanticSnapshot) FactsForFile(filename string) []SemanticFact {
	if s == nil {
		return nil
	}
	facts := make([]SemanticFact, 0)
	fileFacts := s.facts[filename]
	if fileFacts == nil {
		return facts
	}
	appendBuiltIn := func(kind FactKind, entries map[semanticFactOffsetKey]storedSemanticFact) {
		for span, stored := range entries {
			start, end := span.offsets()
			key := SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: kind}
			facts = append(facts, SemanticFact{Key: key, Subject: stored.Subject, Type: stored.Type, Value: stored.Value})
		}
	}
	appendBuiltIn(FactKindInferredType, fileFacts.inferred)
	for span, stored := range fileFacts.generatedInferred {
		start, end := span.offsets()
		key := SemanticFactKey{File: filename, StartOffset: start, EndOffset: end, Kind: FactKindInferredType}
		facts = append(facts, SemanticFact{Key: key, Subject: stored.Subject, Type: stored.Type})
	}
	appendBuiltIn(FactKindReference, fileFacts.reference)
	appendBuiltIn(FactKindNarrowed, fileFacts.narrowed)
	for span, stored := range fileFacts.custom {
		key := SemanticFactKey{File: filename, StartOffset: span.StartOffset, EndOffset: span.EndOffset, Kind: span.Kind}
		facts = append(facts, SemanticFact{Key: key, Subject: stored.Subject, Type: stored.Type, Value: stored.Value})
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
	return s != nil && s.project != nil && s.project.FunctionExists(name)
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
	return method, true
}

func (s *SemanticSnapshot) methodReferenceParams(className, methodName string) ([]ResolvedParam, bool) {
	if s == nil || s.project == nil {
		return nil, false
	}
	return s.project.methodReferenceParams(className, methodName)
}

func (s *SemanticSnapshot) resolveMethodView(className, methodName string) (ResolvedMethod, bool) {
	if s == nil || s.project == nil {
		return ResolvedMethod{}, false
	}
	method, ok := s.project.resolveMethodView(className, methodName)
	if !ok {
		return ResolvedMethod{}, false
	}
	if method.ID == "" {
		method.ID = stableSymbolID("method", method.DeclaringClass, method.Name)
	}
	return method, true
}

func (s *SemanticSnapshot) ResolveOwnMethod(className, methodName string) (ResolvedMethod, bool) {
	if s == nil {
		return ResolvedMethod{}, false
	}
	method, ok := s.project.ResolveOwnMethod(className, methodName)
	if !ok {
		return ResolvedMethod{}, false
	}
	if method.ID == "" {
		method.ID = stableSymbolID("method", method.DeclaringClass, method.Name)
	}
	return method, true
}

func (s *SemanticSnapshot) resolveOwnMethodView(className, methodName string) (ResolvedMethod, bool) {
	if s == nil || s.project == nil {
		return ResolvedMethod{}, false
	}
	method, ok := s.project.resolveOwnMethodView(className, methodName)
	if !ok {
		return ResolvedMethod{}, false
	}
	if method.ID == "" {
		method.ID = stableSymbolID("method", method.DeclaringClass, method.Name)
	}
	return method, true
}

func (s *SemanticSnapshot) MethodsDeclaredBy(className string) []ResolvedMethod {
	if s == nil {
		return nil
	}
	methods := s.project.methodsDeclaredView(className)
	result := make([]ResolvedMethod, len(methods))
	for i := range methods {
		result[i] = methods[i]
		result[i].Params = append([]ResolvedParam(nil), methods[i].Params...)
	}
	return result
}

func (s *SemanticSnapshot) rangeMethodsDeclaredBy(className string, visit func(ResolvedMethod) bool) {
	if s == nil || visit == nil {
		return
	}
	for _, method := range s.project.methodsDeclaredView(className) {
		if !visit(method) {
			return
		}
	}
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

func (s *SemanticSnapshot) resolveFunctionView(name string) (ResolvedFunction, bool) {
	if s == nil || s.project == nil {
		return ResolvedFunction{}, false
	}
	return s.project.ResolveFunction(name)
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

func (s *SemanticSnapshot) ResolveOwnConstant(className, constantName string) (ResolvedConstant, bool) {
	if s == nil {
		return ResolvedConstant{}, false
	}
	constant, ok := s.project.ResolveOwnConstant(className, constantName)
	if !ok {
		return ResolvedConstant{}, false
	}
	if constant.ID == "" {
		constant.ID = stableSymbolID("constant", constant.DeclaringClass, constant.Name)
	}
	return constant, true
}

func (s *SemanticSnapshot) DuplicateClasses(filename string) []DuplicateSymbol {
	if s == nil {
		return nil
	}
	return append([]DuplicateSymbol(nil), s.project.DuplicateClasses(filename)...)
}

func stableSymbolID(kind, owner, name string) SymbolID {
	parts := []string{asciiLowerIdent(kind)}
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
		if _, ok := project.Properties[indexKey(candidate)][asciiLowerIdent(strings.TrimPrefix(propertyName, "$"))]; ok {
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
var _ VariableFlowReader = (*SemanticSnapshot)(nil)

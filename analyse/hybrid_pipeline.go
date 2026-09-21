package analyse

import (
	"fmt"
	"sort"

	"github.com/ayanozturk/go-php-parser/ast"
)

// NewSemanticSnapshotWithIndexOnly builds a snapshot that carries the project
// index (and optional explicit facts) but does not generate per-file CFG /
// variable-flow / inferred-type semantics. Call AnalysisContextForFile with
// full-body trees before running rules that need those facts.
//
// This is the hybrid A/B shape: declaration-tier ingest fills the index;
// exactly one full ParseAndLower per host file owns facts + diagnostics.
func NewSemanticSnapshotWithIndexOnly(idx *ProjectIndex, hostFiles []string, facts []SemanticFact) (*SemanticSnapshot, error) {
	filenames := HostFiles(hostFiles)
	sort.Strings(filenames)

	store := make(semanticFactStore, len(filenames))
	for _, fact := range facts {
		if err := validateSemanticFactKey(fact.Key); err != nil {
			return nil, err
		}
		if !store.put(fact) {
			return nil, fmt.Errorf("duplicate semantic fact key: %s:%d-%d:%s", fact.Key.File, fact.Key.StartOffset, fact.Key.EndOffset, fact.Key.Kind)
		}
	}

	return &SemanticSnapshot{
		project:               idx,
		facts:                 store,
		flow:                  make(map[string]*flowFileStore, len(filenames)),
		variableReads:         make(map[string][]variableReadFact, len(filenames)),
		completeVariableReads: make(map[string]*lazyVariableReadFacts, len(filenames)),
		filenames:             filenames,
	}, nil
}

// AnalysisContextForFile derives full-body semantics for one host file into a
// file-local snapshot view (Facts / Flow / VariableFlow) while reusing this
// snapshot's project index as SymbolResolver. Concurrent calls for distinct
// files are safe: each allocates its own fact/flow stores.
//
// Callers should set Content and Parsed on the returned context to the same
// ParseAndLower result used for nodes so rules do not rebuild the CST.
func (s *SemanticSnapshot) AnalysisContextForFile(filename string, nodes []ast.Node) *AnalysisContext {
	if s == nil {
		return &AnalysisContext{}
	}
	sem := s.buildFileSemantics(filename, nodes)
	if sem.complete != nil {
		// Match R2.5c releasing RSS: do not pin full AST for lazy complete reads.
		sem.complete.nodes = nil
	}
	fileSnap := &SemanticSnapshot{
		project: s.project,
		facts:   semanticFactStore{},
		flow:    make(map[string]*flowFileStore, 1),
		variableReads: make(map[string][]variableReadFact, 1),
		completeVariableReads: make(map[string]*lazyVariableReadFacts, 1),
		filenames: []string{filename},
	}
	if sem.facts != nil {
		fileSnap.facts[filename] = sem.facts
	}
	if sem.flow != nil {
		fileSnap.flow[filename] = sem.flow
	}
	fileSnap.variableReads[filename] = sem.reads
	if sem.complete != nil {
		sem.complete.resolver = fileSnap
		fileSnap.completeVariableReads[filename] = sem.complete
	}
	return fileSnap.NewAnalysisContext()
}

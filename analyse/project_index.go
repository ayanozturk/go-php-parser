package analyse

import (
	"sort"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/phpstubs"
)

type ProjectIndex struct {
	Classes         map[string]ResolvedClass
	Methods         map[string]map[string]ResolvedMethod
	Properties      map[string]map[string]ResolvedProperty
	ClassConsts     map[string]map[string]ResolvedConstant
	Functions       map[string]ResolvedFunction
	Constants       map[string]struct{}
	FileTypes       map[string]FileTypeContext
	Duplicates      []DuplicateSymbol
	methodsDeclared map[string][]ResolvedMethod
	classLineages   map[string][]string
	phpVersion      string
	// fileClasses maps file path → class names defined in that file
	fileClasses map[string]map[string]struct{}
	// sourceFiles retains the immutable parsed inputs used to build this view.
	// The slices and nodes are shared, never mutated by incremental updates.
	sourceFiles map[string][]ast.Node
	// collidingDefinitions records symbols whose deterministic winner depends
	// on file order. Incremental updates touching one fall back to a full build.
	collidingDefinitions map[string]struct{}
	// globalConstantFiles supplies ownership missing from Constants' set values.
	globalConstantFiles map[string]string
}

type DuplicateSymbol struct {
	File string
	Name string
	Pos  ast.Position
}

// ExportedSymbolChange identifies a project-level symbol whose semantic
// signature was added, removed, or changed by an incremental update.
type ExportedSymbolChange struct {
	ID    SymbolID
	Kind  string
	Owner string
	Name  string
}

// ProjectIndexChanges describes the dependency surface of an incremental
// update. Complete is false when missing source metadata forces callers to
// invalidate every cached semantic consumer. FullRebuild reports that the
// requested update used the deterministic full-build path instead of replacing
// only the listed file contributions. DependencyNames includes changed symbols,
// their owners, and transitive class descendants.
type ProjectIndexChanges struct {
	Complete        bool
	FullRebuild     bool
	Symbols         []ExportedSymbolChange
	DependencyNames []string
}

// SemanticChanged reports whether cached cross-file semantic facts may be stale.
func (changes ProjectIndexChanges) SemanticChanged() bool {
	return !changes.Complete || len(changes.Symbols) > 0
}

func NewProjectIndex() *ProjectIndex {
	return NewProjectIndexForVersion(phpstubs.DefaultPHPVersion)
}

func NewProjectIndexForVersion(phpVersion string) *ProjectIndex {
	idx := newProjectIndex()
	idx.phpVersion = phpstubs.NormalizePHPVersion(phpVersion)
	idx.indexPHPStubs(idx.phpVersion)
	idx.seedBuiltins()
	return idx
}

func newProjectIndex() *ProjectIndex {
	idx := &ProjectIndex{
		Classes:              make(map[string]ResolvedClass),
		fileClasses:          make(map[string]map[string]struct{}),
		Methods:              make(map[string]map[string]ResolvedMethod),
		Properties:           make(map[string]map[string]ResolvedProperty),
		ClassConsts:          make(map[string]map[string]ResolvedConstant),
		Functions:            make(map[string]ResolvedFunction),
		Constants:            make(map[string]struct{}),
		FileTypes:            make(map[string]FileTypeContext),
		collidingDefinitions: make(map[string]struct{}),
		globalConstantFiles:  make(map[string]string),
	}
	return idx
}

// BuildProjectIndex indexes every parsed file into a single ProjectIndex.
// Files are processed in sorted filename order rather than native Go map
// iteration order (which is randomized per run): symbol registration below
// is order-dependent (addClass keeps the first definition and records
// later same-name definitions as duplicates; addFunction/addMethod/
// addProperty/addClassConstant let the last definition win), so an
// unsorted, randomized iteration order made duplicate-symbol resolution -
// and therefore some diagnostics computed relative to it - vary between
// otherwise-identical runs over the same corpus.
func BuildProjectIndex(parsed map[string][]ast.Node) *ProjectIndex {
	return BuildProjectIndexForVersion(parsed, phpstubs.DefaultPHPVersion)
}

func BuildProjectIndexForVersion(parsed map[string][]ast.Node, phpVersion string) *ProjectIndex {
	idx := NewProjectIndexForVersion(phpVersion)
	idx.sourceFiles = make(map[string][]ast.Node, len(parsed))
	filenames := make([]string, 0, len(parsed))
	for filename, nodes := range parsed {
		filenames = append(filenames, filename)
		idx.sourceFiles[filename] = nodes
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		nodes := parsed[filename]
		ft := CollectFileTypeContext(nodes)
		idx.FileTypes[filename] = ft
		idx.indexNodes(filename, nodes, ft, "")
	}
	idx.methodsDeclared = buildMethodsDeclaredViews(idx)
	idx.classLineages = buildClassLineageViews(idx)
	return idx
}

package analyse

import (
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
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
	idx := newProjectIndex()
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
	idx := NewProjectIndex()
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

// BuildProjectIndexIncremental returns a new immutable project index after
// replacing only the listed files' symbol contributions. parsed is the full
// current file set; entries absent from it are treated as removals. The bool
// reports whether exported symbol semantics changed, excluding declaration
// positions. If the previous index lacks contribution metadata, the function
// safely falls back to a complete deterministic build.
func BuildProjectIndexIncremental(previous *ProjectIndex, parsed map[string][]ast.Node, changedFiles []string) (*ProjectIndex, bool) {
	idx, changes := BuildProjectIndexIncrementalWithChanges(previous, parsed, changedFiles)
	return idx, changes.SemanticChanged()
}

// BuildProjectIndexIncrementalWithChanges is BuildProjectIndexIncremental with
// deterministic exported-symbol change details for dependency-scoped caches.
func BuildProjectIndexIncrementalWithChanges(previous *ProjectIndex, parsed map[string][]ast.Node, changedFiles []string) (*ProjectIndex, ProjectIndexChanges) {
	if previous == nil || previous.sourceFiles == nil {
		return BuildProjectIndex(parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
	}

	changed := make(map[string]struct{}, len(changedFiles))
	for _, filename := range changedFiles {
		changed[filename] = struct{}{}
	}
	if len(changed) == 0 {
		return previous, ProjectIndexChanges{Complete: true}
	}
	for filename := range parsed {
		if _, exists := previous.sourceFiles[filename]; !exists {
			if _, listed := changed[filename]; !listed {
				return BuildProjectIndex(parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
			}
		}
	}
	for filename := range previous.sourceFiles {
		if _, exists := parsed[filename]; !exists {
			if _, listed := changed[filename]; !listed {
				return BuildProjectIndex(parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
			}
		}
	}

	changes := ProjectIndexChanges{Complete: true}
	requiresFullBuild := false
	filenames := make([]string, 0, len(changed))
	for filename := range changed {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	newContributions := make(map[string]*ProjectIndex, len(filenames))
	for _, filename := range filenames {
		oldContribution := buildProjectFileIndex(filename, previous.sourceFiles[filename])
		newContribution := buildProjectFileIndex(filename, parsed[filename])
		newContributions[filename] = newContribution
		changes.Symbols = append(changes.Symbols, projectFileExportChanges(oldContribution, newContribution)...)
		if projectFileTouchesCollisions(previous, oldContribution) || len(newContribution.collidingDefinitions) > 0 {
			requiresFullBuild = true
		}
	}
	if requiresFullBuild {
		idx := BuildProjectIndex(parsed)
		changes.FullRebuild = true
		return idx, finalizeProjectIndexChanges(previous, idx, changes)
	}

	idx := cloneProjectIndex(previous)
	for _, filename := range filenames {
		idx.removeProjectFile(filename)
	}
	for _, filename := range filenames {
		contribution := newContributions[filename]
		if projectFileCollidesWithIndex(idx, contribution) {
			fresh := BuildProjectIndex(parsed)
			changes.FullRebuild = true
			return fresh, finalizeProjectIndexChanges(previous, fresh, changes)
		}
		nodes, remains := parsed[filename]
		if !remains {
			delete(idx.sourceFiles, filename)
			continue
		}
		ft := CollectFileTypeContext(nodes)
		idx.FileTypes[filename] = ft
		idx.sourceFiles[filename] = nodes
		idx.indexNodes(filename, nodes, ft, "")
	}
	idx.methodsDeclared = buildMethodsDeclaredViews(idx)
	idx.classLineages = buildClassLineageViews(idx)
	return idx, finalizeProjectIndexChanges(previous, idx, changes)
}

func buildProjectFileIndex(filename string, nodes []ast.Node) *ProjectIndex {
	idx := newProjectIndex()
	ft := CollectFileTypeContext(nodes)
	idx.FileTypes[filename] = ft
	idx.indexNodes(filename, nodes, ft, "")
	return idx
}

func cloneProjectIndex(previous *ProjectIndex) *ProjectIndex {
	idx := newProjectIndex()
	idx.Classes = mapsClone(previous.Classes)
	idx.Methods = cloneNestedMap(previous.Methods)
	idx.Properties = cloneNestedMap(previous.Properties)
	idx.ClassConsts = cloneNestedMap(previous.ClassConsts)
	idx.Functions = mapsClone(previous.Functions)
	idx.Constants = mapsClone(previous.Constants)
	idx.FileTypes = mapsClone(previous.FileTypes)
	idx.Duplicates = append([]DuplicateSymbol(nil), previous.Duplicates...)
	idx.fileClasses = cloneNestedSet(previous.fileClasses)
	idx.sourceFiles = mapsClone(previous.sourceFiles)
	idx.collidingDefinitions = mapsClone(previous.collidingDefinitions)
	idx.globalConstantFiles = mapsClone(previous.globalConstantFiles)
	return idx
}

func mapsClone[K comparable, V any](source map[K]V) map[K]V {
	cloned := make(map[K]V, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneNestedMap[V any](source map[string]map[string]V) map[string]map[string]V {
	cloned := make(map[string]map[string]V, len(source))
	for outerKey, values := range source {
		cloned[outerKey] = mapsClone(values)
	}
	return cloned
}

func cloneNestedSet(source map[string]map[string]struct{}) map[string]map[string]struct{} {
	return cloneNestedMap(source)
}

func (idx *ProjectIndex) removeProjectFile(filename string) {
	for key, class := range idx.Classes {
		if class.Declaration.File == filename {
			delete(idx.Classes, key)
		}
	}
	removeNestedDeclarations(idx.Methods, filename, func(value ResolvedMethod) SourceLocation { return value.Declaration })
	removeNestedDeclarations(idx.Properties, filename, func(value ResolvedProperty) SourceLocation { return value.Declaration })
	for _, constants := range idx.ClassConsts {
		for _, constant := range constants {
			if constant.Declaration.File == filename {
				delete(idx.Constants, indexKey(constant.DeclaringClass+"::"+constant.Name))
			}
		}
	}
	removeNestedDeclarations(idx.ClassConsts, filename, func(value ResolvedConstant) SourceLocation { return value.Declaration })
	for key, fn := range idx.Functions {
		if fn.Declaration.File == filename {
			delete(idx.Functions, key)
		}
	}
	for key, owner := range idx.globalConstantFiles {
		if owner == filename {
			delete(idx.globalConstantFiles, key)
			delete(idx.Constants, key)
		}
	}
	delete(idx.FileTypes, filename)
	delete(idx.fileClasses, filename)
	delete(idx.sourceFiles, filename)
	idx.Duplicates = slices.DeleteFunc(idx.Duplicates, func(duplicate DuplicateSymbol) bool { return duplicate.File == filename })
}

func removeNestedDeclarations[V any](values map[string]map[string]V, filename string, location func(V) SourceLocation) {
	for outerKey, entries := range values {
		for key, value := range entries {
			if location(value).File == filename {
				delete(entries, key)
			}
		}
		if len(entries) == 0 {
			delete(values, outerKey)
		}
	}
}

func projectFileTouchesCollisions(project, contribution *ProjectIndex) bool {
	for key := range projectFileDefinitionKeys(contribution) {
		if _, collision := project.collidingDefinitions[key]; collision {
			return true
		}
	}
	return false
}

func projectFileCollidesWithIndex(project, contribution *ProjectIndex) bool {
	for key := range projectFileDefinitionKeys(contribution) {
		if project.definitionExists(key) {
			return true
		}
	}
	return false
}

func projectFileDefinitionKeys(contribution *ProjectIndex) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, class := range contribution.Classes {
		keys[classDefinitionKey(class.Name)] = struct{}{}
	}
	for _, fn := range contribution.Functions {
		keys[functionDefinitionKey(fn.Name)] = struct{}{}
	}
	for classKey, methods := range contribution.Methods {
		for methodKey := range methods {
			keys[memberDefinitionKey("method", classKey, methodKey)] = struct{}{}
		}
	}
	for classKey, properties := range contribution.Properties {
		for propertyKey := range properties {
			keys[memberDefinitionKey("property", classKey, propertyKey)] = struct{}{}
		}
	}
	for classKey, constants := range contribution.ClassConsts {
		for constantKey := range constants {
			keys[memberDefinitionKey("class-constant", classKey, constantKey)] = struct{}{}
		}
	}
	for key := range contribution.globalConstantFiles {
		keys[globalConstantDefinitionKey(key)] = struct{}{}
	}
	return keys
}

func (idx *ProjectIndex) definitionExists(key string) bool {
	parts := strings.Split(key, "\x00")
	switch parts[0] {
	case "class":
		_, ok := idx.Classes[parts[1]]
		return ok
	case "function":
		_, ok := idx.Functions[parts[1]]
		return ok
	case "method":
		_, ok := idx.Methods[parts[1]][parts[2]]
		return ok
	case "property":
		_, ok := idx.Properties[parts[1]][parts[2]]
		return ok
	case "class-constant":
		_, ok := idx.ClassConsts[parts[1]][parts[2]]
		return ok
	case "global-constant":
		_, ok := idx.globalConstantFiles[parts[1]]
		return ok
	default:
		return false
	}
}

func classDefinitionKey(name string) string    { return "class\x00" + indexKey(name) }
func functionDefinitionKey(name string) string { return "function\x00" + indexKey(name) }
func memberDefinitionKey(kind, className, memberName string) string {
	return kind + "\x00" + indexKey(className) + "\x00" + strings.ToLower(memberName)
}
func globalConstantDefinitionKey(name string) string { return "global-constant\x00" + indexKey(name) }

func projectFileExportChanges(left, right *ProjectIndex) []ExportedSymbolChange {
	var changes []ExportedSymbolChange
	classKeys := unionMapKeys(left.Classes, right.Classes)
	for _, key := range classKeys {
		oldValue, oldOK := left.Classes[key]
		newValue, newOK := right.Classes[key]
		if oldOK && newOK && resolvedClassSemanticallyEqual(oldValue, newValue) {
			continue
		}
		if oldOK {
			changes = append(changes, exportedClassChange(oldValue))
		}
		if newOK && (!oldOK || oldValue.ID != newValue.ID) {
			changes = append(changes, exportedClassChange(newValue))
		}
	}

	functionKeys := unionMapKeys(left.Functions, right.Functions)
	for _, key := range functionKeys {
		oldValue, oldOK := left.Functions[key]
		newValue, newOK := right.Functions[key]
		if oldOK && newOK && resolvedFunctionSemanticallyEqual(oldValue, newValue) {
			continue
		}
		if oldOK {
			changes = append(changes, ExportedSymbolChange{ID: oldValue.ID, Kind: "function", Name: oldValue.Name})
		}
		if newOK && (!oldOK || oldValue.ID != newValue.ID) {
			changes = append(changes, ExportedSymbolChange{ID: newValue.ID, Kind: "function", Name: newValue.Name})
		}
	}

	changes = append(changes, changedMethods(left.Methods, right.Methods)...)
	changes = append(changes, changedProperties(left.Properties, right.Properties)...)
	changes = append(changes, changedClassConstants(left.ClassConsts, right.ClassConsts)...)
	for _, key := range unionMapKeys(left.globalConstantFiles, right.globalConstantFiles) {
		_, oldOK := left.globalConstantFiles[key]
		_, newOK := right.globalConstantFiles[key]
		if oldOK == newOK {
			continue
		}
		changes = append(changes, ExportedSymbolChange{ID: stableSymbolID("constant", "", key), Kind: "constant", Name: key})
	}
	return changes
}

func exportedClassChange(class ResolvedClass) ExportedSymbolChange {
	return ExportedSymbolChange{ID: class.ID, Kind: "class", Name: class.Name}
}

func changedMethods(left, right map[string]map[string]ResolvedMethod) []ExportedSymbolChange {
	var changes []ExportedSymbolChange
	for _, classKey := range unionMapKeys(left, right) {
		for _, memberKey := range unionMapKeys(left[classKey], right[classKey]) {
			oldValue, oldOK := left[classKey][memberKey]
			newValue, newOK := right[classKey][memberKey]
			if oldOK && newOK && resolvedMethodSemanticallyEqual(oldValue, newValue) {
				continue
			}
			if oldOK {
				changes = append(changes, ExportedSymbolChange{ID: oldValue.ID, Kind: "method", Owner: oldValue.DeclaringClass, Name: oldValue.Name})
			}
			if newOK && (!oldOK || oldValue.ID != newValue.ID) {
				changes = append(changes, ExportedSymbolChange{ID: newValue.ID, Kind: "method", Owner: newValue.DeclaringClass, Name: newValue.Name})
			}
		}
	}
	return changes
}

func changedProperties(left, right map[string]map[string]ResolvedProperty) []ExportedSymbolChange {
	var changes []ExportedSymbolChange
	for _, classKey := range unionMapKeys(left, right) {
		for _, memberKey := range unionMapKeys(left[classKey], right[classKey]) {
			oldValue, oldOK := left[classKey][memberKey]
			newValue, newOK := right[classKey][memberKey]
			if oldOK && newOK && resolvedPropertySemanticallyEqual(oldValue, newValue) {
				continue
			}
			if oldOK {
				changes = append(changes, ExportedSymbolChange{ID: oldValue.ID, Kind: "property", Owner: oldValue.DeclaringClass, Name: strings.TrimPrefix(oldValue.Name, "$")})
			}
			if newOK && (!oldOK || oldValue.ID != newValue.ID) {
				changes = append(changes, ExportedSymbolChange{ID: newValue.ID, Kind: "property", Owner: newValue.DeclaringClass, Name: strings.TrimPrefix(newValue.Name, "$")})
			}
		}
	}
	return changes
}

func changedClassConstants(left, right map[string]map[string]ResolvedConstant) []ExportedSymbolChange {
	var changes []ExportedSymbolChange
	for _, classKey := range unionMapKeys(left, right) {
		for _, memberKey := range unionMapKeys(left[classKey], right[classKey]) {
			oldValue, oldOK := left[classKey][memberKey]
			newValue, newOK := right[classKey][memberKey]
			if oldOK && newOK && resolvedConstantSemanticallyEqual(oldValue, newValue) {
				continue
			}
			if oldOK {
				changes = append(changes, ExportedSymbolChange{ID: oldValue.ID, Kind: "class-constant", Owner: oldValue.DeclaringClass, Name: oldValue.Name})
			}
			if newOK && (!oldOK || oldValue.ID != newValue.ID) {
				changes = append(changes, ExportedSymbolChange{ID: newValue.ID, Kind: "class-constant", Owner: newValue.DeclaringClass, Name: newValue.Name})
			}
		}
	}
	return changes
}

func unionMapKeys[V any](left, right map[string]V) []string {
	keys := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		keys[key] = struct{}{}
	}
	for key := range right {
		keys[key] = struct{}{}
	}
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func finalizeProjectIndexChanges(previous, current *ProjectIndex, changes ProjectIndexChanges) ProjectIndexChanges {
	deduplicated := make(map[string]ExportedSymbolChange, len(changes.Symbols))
	dependencyNames := make(map[string]string)
	classRoots := make(map[string]struct{})
	for _, change := range changes.Symbols {
		key := change.Kind + "\x00" + strings.ToLower(change.Owner) + "\x00" + strings.ToLower(change.Name)
		deduplicated[key] = change
		addDependencyName(dependencyNames, change.Name)
		addDependencyName(dependencyNames, change.Owner)
		if change.Kind == "class" {
			classRoots[indexKey(change.Name)] = struct{}{}
		} else if change.Owner != "" {
			classRoots[indexKey(change.Owner)] = struct{}{}
		}
	}
	changes.Symbols = changes.Symbols[:0]
	for _, change := range deduplicated {
		changes.Symbols = append(changes.Symbols, change)
	}
	sort.Slice(changes.Symbols, func(i, j int) bool {
		left := changes.Symbols[i]
		right := changes.Symbols[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if !strings.EqualFold(left.Owner, right.Owner) {
			return strings.ToLower(left.Owner) < strings.ToLower(right.Owner)
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})
	addDescendantDependencyNames(previous, classRoots, dependencyNames)
	addDescendantDependencyNames(current, classRoots, dependencyNames)
	changes.DependencyNames = make([]string, 0, len(dependencyNames))
	for _, name := range dependencyNames {
		changes.DependencyNames = append(changes.DependencyNames, name)
	}
	sort.Slice(changes.DependencyNames, func(i, j int) bool {
		return strings.ToLower(changes.DependencyNames[i]) < strings.ToLower(changes.DependencyNames[j])
	})
	return changes
}

func addDescendantDependencyNames(project *ProjectIndex, roots map[string]struct{}, names map[string]string) {
	if project == nil || len(roots) == 0 {
		return
	}
	for _, class := range project.Classes {
		for _, ancestor := range project.classLineage(class.Name) {
			if _, affected := roots[indexKey(ancestor)]; affected {
				addDependencyName(names, class.Name)
				break
			}
		}
	}
}

func addDependencyName(names map[string]string, name string) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "\\")
	if name == "" {
		return
	}
	key := strings.ToLower(name)
	names[key] = name
	if index := strings.LastIndex(name, "\\"); index >= 0 && index+1 < len(name) {
		short := name[index+1:]
		names[strings.ToLower(short)] = short
	}
}

func projectFileSemanticsEqual(left, right *ProjectIndex) bool {
	if left == nil || right == nil {
		return left == right
	}
	if !resolvedClassesSemanticallyEqual(left.Classes, right.Classes) ||
		!duplicatesSemanticallyEqual(left.Duplicates, right.Duplicates) ||
		!resolvedFunctionsSemanticallyEqual(left.Functions, right.Functions) ||
		!resolvedMethodsSemanticallyEqual(left.Methods, right.Methods) ||
		!resolvedPropertiesSemanticallyEqual(left.Properties, right.Properties) ||
		!resolvedConstantsSemanticallyEqual(left.ClassConsts, right.ClassConsts) ||
		!reflect.DeepEqual(left.Constants, right.Constants) {
		return false
	}
	return true
}

// FilesAffectedByChangedFile returns all files that may need re-analysis
// if the given file changes. This includes the file itself and any files
// that inherit from classes defined in the changed file.
func (idx *ProjectIndex) FilesAffectedByChangedFile(changedFile string) []string {
	affected := make(map[string]struct{})
	affected[changedFile] = struct{}{}

	// Find all classes defined in changed file
	classesInFile := idx.fileClasses[changedFile]
	if len(classesInFile) == 0 {
		return []string{changedFile}
	}

	// Find all files that define classes extending/implementing those classes
	for className := range classesInFile {
		for _, otherClass := range idx.Classes {
			// Check if any other class extends or implements the changed class
			for _, ext := range otherClass.Extends {
				if strings.EqualFold(ext, className) {
					affected[otherClass.Declaration.File] = struct{}{}
				}
			}
			for _, impl := range otherClass.Implements {
				if strings.EqualFold(impl, className) {
					affected[otherClass.Declaration.File] = struct{}{}
				}
			}
		}
	}

	result := make([]string, 0, len(affected))
	for f := range affected {
		result = append(result, f)
	}
	return result
}

// MergeIncremental merges newly parsed files into this index.
// For each file in newParsed, re-indexes it, updating all maps.
// Returns updated index (this is modified in-place).
func (idx *ProjectIndex) MergeIncremental(filesToReparse map[string][]ast.Node, FileTypeContexts map[string]FileTypeContext) {
	// Clear old entries for files being re-parsed
	for filePath := range filesToReparse {
		// Remove classes from this file
		classesToRemove := []string{}
		for className, class := range idx.Classes {
			if class.Declaration.File == filePath {
				classesToRemove = append(classesToRemove, className)
			}
		}
		for _, className := range classesToRemove {
			delete(idx.Classes, className)
		}

		// Clear file from fileClasses map
		delete(idx.fileClasses, filePath)

		// Remove methods/properties/constants from this file's classes
		// (simplified: rely on Classes removal to cascade)
	}

	// Re-index new files
	for filePath, nodes := range filesToReparse {
		ft := FileTypeContexts[filePath]
		if ft.ClassNodes == nil {
			ft.ClassNodes = make(map[string]*ast.ClassNode)
		}
		idx.indexNodes(filePath, nodes, ft, "")
	}

	// Invalidate caches (will be recomputed if needed)
	idx.methodsDeclared = nil
	idx.classLineages = nil
	idx.sourceFiles = nil
}

func resolvedClassesSemanticallyEqual(left, right map[string]ResolvedClass) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		rightValue, ok := right[key]
		if !ok || !resolvedClassSemanticallyEqual(leftValue, rightValue) {
			return false
		}
	}
	return true
}

func resolvedClassSemanticallyEqual(left, right ResolvedClass) bool {
	left.Declaration = SourceLocation{}
	right.Declaration = SourceLocation{}
	return reflect.DeepEqual(left, right)
}

func duplicatesSemanticallyEqual(left, right []DuplicateSymbol) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name != right[i].Name {
			return false
		}
	}
	return true
}

func resolvedFunctionsSemanticallyEqual(left, right map[string]ResolvedFunction) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		rightValue, ok := right[key]
		if !ok || !resolvedFunctionSemanticallyEqual(leftValue, rightValue) {
			return false
		}
	}
	return true
}

func resolvedFunctionSemanticallyEqual(left, right ResolvedFunction) bool {
	left.Declaration = SourceLocation{}
	right.Declaration = SourceLocation{}
	return reflect.DeepEqual(left, right)
}

func resolvedMethodsSemanticallyEqual(left, right map[string]map[string]ResolvedMethod) bool {
	if len(left) != len(right) {
		return false
	}
	for classKey, leftMethods := range left {
		rightMethods, ok := right[classKey]
		if !ok || len(leftMethods) != len(rightMethods) {
			return false
		}
		for methodKey, leftValue := range leftMethods {
			rightValue, ok := rightMethods[methodKey]
			if !ok {
				return false
			}
			if !resolvedMethodSemanticallyEqual(leftValue, rightValue) {
				return false
			}
		}
	}
	return true
}

func resolvedMethodSemanticallyEqual(left, right ResolvedMethod) bool {
	left.Declaration = SourceLocation{}
	right.Declaration = SourceLocation{}
	return reflect.DeepEqual(left, right)
}

func resolvedPropertiesSemanticallyEqual(left, right map[string]map[string]ResolvedProperty) bool {
	if len(left) != len(right) {
		return false
	}
	for classKey, leftProperties := range left {
		rightProperties, ok := right[classKey]
		if !ok || len(leftProperties) != len(rightProperties) {
			return false
		}
		for propertyKey, leftValue := range leftProperties {
			rightValue, ok := rightProperties[propertyKey]
			if !ok {
				return false
			}
			if !resolvedPropertySemanticallyEqual(leftValue, rightValue) {
				return false
			}
		}
	}
	return true
}

func resolvedPropertySemanticallyEqual(left, right ResolvedProperty) bool {
	left.Declaration = SourceLocation{}
	right.Declaration = SourceLocation{}
	return reflect.DeepEqual(left, right)
}

func resolvedConstantsSemanticallyEqual(left, right map[string]map[string]ResolvedConstant) bool {
	if len(left) != len(right) {
		return false
	}
	for classKey, leftConstants := range left {
		rightConstants, ok := right[classKey]
		if !ok || len(leftConstants) != len(rightConstants) {
			return false
		}
		for constantKey, leftValue := range leftConstants {
			rightValue, ok := rightConstants[constantKey]
			if !ok {
				return false
			}
			if !resolvedConstantSemanticallyEqual(leftValue, rightValue) {
				return false
			}
		}
	}
	return true
}

func resolvedConstantSemanticallyEqual(left, right ResolvedConstant) bool {
	left.Declaration = SourceLocation{}
	right.Declaration = SourceLocation{}
	return reflect.DeepEqual(left, right)
}

func (idx *ProjectIndex) ClassExists(name string) bool {
	_, ok := idx.ResolveClass(name)
	return ok
}

func (idx *ProjectIndex) FunctionExists(name string) bool {
	_, ok := idx.ResolveFunction(name)
	return ok
}

func (idx *ProjectIndex) ConstantExists(name string) bool {
	if _, ok := idx.Constants[indexKey(name)]; ok {
		return true
	}
	if className, constName, ok := strings.Cut(name, "::"); ok {
		_, ok := idx.ResolveConstant(className, constName)
		return ok
	}
	return false
}

func (idx *ProjectIndex) ResolveClass(name string) (ResolvedClass, bool) {
	key := indexKey(name)
	if class, ok := idx.Classes[key]; ok {
		return class, true
	}
	if short := unqualifiedName(key); short != key && isBuiltinClassName(short) {
		class, ok := idx.Classes[short]
		return class, ok
	}
	if class, ok := idx.resolveKnownClassSuffix(key); ok {
		return class, true
	}
	return ResolvedClass{}, false
}

func (idx *ProjectIndex) ResolveMethod(className, methodName string) (ResolvedMethod, bool) {
	return idx.resolveMethodWithTemplates(className, methodName, nil, make(map[string]struct{}))
}

// ResolveMethodWithGenerics resolves a method on a generic class instance.
// className is the fully qualified class name (e.g., "Repository").
// typeArguments are the generic type arguments (e.g., ["User"] for Repository<User>).
func (idx *ProjectIndex) ResolveMethodWithGenerics(className, methodName string, typeArguments []string) (ResolvedMethod, bool) {
	class, ok := idx.ResolveClass(className)
	if !ok || len(class.TemplateParams) == 0 || len(typeArguments) == 0 {
		return idx.ResolveMethod(className, methodName)
	}

	// Build bindings: T -> User, K -> string, etc.
	bindings := make(map[string]string, len(class.TemplateParams))
	for i, param := range class.TemplateParams {
		if i < len(typeArguments) {
			bindings[param] = typeArguments[i]
		}
	}

	return idx.resolveMethodWithTemplates(className, methodName, bindings, make(map[string]struct{}))
}

func (idx *ProjectIndex) methodReferenceParams(className, methodName string) ([]ResolvedParam, bool) {
	var seen [32]string
	return idx.methodReferenceParamsSeen(className, methodName, seen[:0])
}

func (idx *ProjectIndex) methodReferenceParamsSeen(className, methodName string, seen []string) ([]ResolvedParam, bool) {
	if idx == nil {
		return nil, false
	}
	if len(seen) == cap(seen) {
		method, ok := idx.ResolveMethod(className, methodName)
		return method.Params, ok
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return nil, false
	}
	key := indexKey(class.Name)
	for _, visited := range seen {
		if visited == key {
			return nil, false
		}
	}
	seen = append(seen, key)
	if method, found := idx.Methods[key][strings.ToLower(methodName)]; found {
		return method.Params, true
	}
	for _, parentName := range class.Extends {
		if params, found := idx.methodReferenceParamsSeen(parentName, methodName, seen); found {
			return params, true
		}
	}
	for _, parentName := range class.Implements {
		if params, found := idx.methodReferenceParamsSeen(parentName, methodName, seen); found {
			return params, true
		}
	}
	return nil, false
}

func (idx *ProjectIndex) ResolveOwnMethod(className, methodName string) (ResolvedMethod, bool) {
	if idx == nil {
		return ResolvedMethod{}, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	method, ok := idx.Methods[indexKey(class.Name)][strings.ToLower(methodName)]
	if !ok {
		return ResolvedMethod{}, false
	}
	method.DeclaringClass = class.Name
	method.Params = append([]ResolvedParam(nil), method.Params...)
	return method, true
}

func (idx *ProjectIndex) MethodsDeclaredBy(className string) []ResolvedMethod {
	if idx == nil {
		return nil
	}
	methods := idx.methodsDeclaredView(className)
	result := make([]ResolvedMethod, len(methods))
	for i := range methods {
		result[i] = methods[i]
		result[i].Params = append([]ResolvedParam(nil), methods[i].Params...)
	}
	return result
}

func (idx *ProjectIndex) rangeMethodsDeclaredBy(className string, visit func(ResolvedMethod) bool) {
	if idx == nil || visit == nil {
		return
	}
	for _, method := range idx.methodsDeclaredView(className) {
		if !visit(method) {
			return
		}
	}
}

func (idx *ProjectIndex) methodsDeclaredView(className string) []ResolvedMethod {
	if idx == nil {
		return nil
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return nil
	}
	key := indexKey(class.Name)
	if idx.methodsDeclared != nil {
		return idx.methodsDeclared[key]
	}
	// Mutable indexes constructed directly retain compatibility. Immutable
	// indexes returned by BuildProjectIndex always use the precomputed path.
	return buildMethodsDeclaredView(idx, key, class.Name)
}

func (idx *ProjectIndex) resolveMethodWithTemplates(className, methodName string, bindings map[string]string, seen map[string]struct{}) (ResolvedMethod, bool) {
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	key := indexKey(class.Name)
	if _, exists := seen[key]; exists {
		return ResolvedMethod{}, false
	}
	seen[key] = struct{}{}
	defer delete(seen, key)
	if method, found := idx.Methods[key][strings.ToLower(methodName)]; found {
		// ResolvedMethod is returned by value, but Params is a slice. Clone it
		// before applying call-specific generic bindings so resolution cannot
		// mutate the project index or race with concurrent snapshot readers.
		method.Params = append([]ResolvedParam(nil), method.Params...)
		method.DeclaringClass = class.Name
		method.ReturnType = ApplyTemplateBindings(method.ReturnType, bindings)
		for i := range method.Params {
			method.Params[i].Type = ApplyTemplateBindings(method.Params[i].Type, bindings)
		}
		return method, true
	}
	parents := append(append([]string(nil), class.Extends...), class.Implements...)
	for _, parentName := range parents {
		parent, parentOK := idx.ResolveClass(parentName)
		if !parentOK {
			continue
		}
		parentBindings := map[string]string(nil)
		if relation, relationOK := genericRelationTo(class, parentName); relationOK {
			parentBindings = bindGenericParent(parent, relation, bindings)
		}
		if method, found := idx.resolveMethodWithTemplates(parent.Name, methodName, parentBindings, seen); found {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func (idx *ProjectIndex) ResolveProperty(className, propertyName string) (ResolvedProperty, bool) {
	if class, ok := idx.ResolveClass(className); ok && class.Kind == "enum" && strings.EqualFold(strings.TrimPrefix(propertyName, "$"), "value") {
		return ResolvedProperty{ID: stableSymbolID("property", class.Name, "value"), DeclaringClass: class.Name, Name: "value", Visibility: "public", Readonly: true}, true
	}
	for _, candidate := range idx.classLineage(className) {
		properties := idx.Properties[indexKey(candidate)]
		if properties == nil {
			continue
		}
		if property, ok := properties[strings.ToLower(strings.TrimPrefix(propertyName, "$"))]; ok {
			return property, true
		}
	}
	return ResolvedProperty{}, false
}

func (idx *ProjectIndex) ResolveConstant(className, constantName string) (ResolvedConstant, bool) {
	for _, candidate := range idx.classLineage(className) {
		constants := idx.ClassConsts[indexKey(candidate)]
		if constants == nil {
			continue
		}
		if constant, ok := constants[strings.ToLower(constantName)]; ok {
			constant.DeclaringClass = candidate
			return constant, true
		}
	}
	return ResolvedConstant{}, false
}

func (idx *ProjectIndex) ResolveOwnConstant(className, constantName string) (ResolvedConstant, bool) {
	if idx == nil {
		return ResolvedConstant{}, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedConstant{}, false
	}
	constant, ok := idx.ClassConsts[indexKey(class.Name)][strings.ToLower(constantName)]
	if !ok {
		return ResolvedConstant{}, false
	}
	constant.DeclaringClass = class.Name
	return constant, true
}

func (idx *ProjectIndex) DuplicateClasses(filename string) []DuplicateSymbol {
	if idx == nil {
		return nil
	}
	duplicates := make([]DuplicateSymbol, 0)
	for _, duplicate := range idx.Duplicates {
		if filename == "" || duplicate.File == filename {
			duplicates = append(duplicates, duplicate)
		}
	}
	return duplicates
}

func (idx *ProjectIndex) ResolveFunction(name string) (ResolvedFunction, bool) {
	fn, ok := idx.Functions[indexKey(name)]
	return fn, ok
}

func (idx *ProjectIndex) classLineage(className string) []string {
	if idx == nil {
		return nil
	}
	key := indexKey(className)
	if class, ok := idx.ResolveClass(className); ok {
		key = indexKey(class.Name)
	}
	if lineage, ok := idx.classLineages[key]; ok {
		return lineage
	}
	// Mutable indexes constructed directly retain compatibility. Immutable
	// indexes returned by BuildProjectIndex always use the precomputed path.
	return buildClassLineage(idx, className)
}

func buildClassLineageViews(idx *ProjectIndex) map[string][]string {
	views := make(map[string][]string, len(idx.Classes))
	for key, class := range idx.Classes {
		views[key] = buildClassLineage(idx, class.Name)
	}
	return views
}

func buildClassLineage(idx *ProjectIndex, className string) []string {
	var out []string
	seen := map[string]struct{}{}
	var walk func(string)
	walk = func(name string) {
		key := indexKey(name)
		if key == "" {
			return
		}
		class, ok := idx.ResolveClass(name)
		if ok {
			key = indexKey(class.Name)
			name = class.Name
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
		if !ok {
			return
		}
		for _, parent := range class.Extends {
			walk(parent)
		}
		for _, iface := range class.Implements {
			walk(iface)
		}
		for _, trait := range class.Traits {
			walk(trait)
		}
	}
	walk(className)
	return out
}

func (idx *ProjectIndex) indexNodes(filename string, nodes []ast.Node, ft FileTypeContext, currentClass string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := CollectFileTypeContext(n.Body)
			if nft.Namespace == "" {
				nft.Namespace = n.Name
			}
			idx.indexNodes(filename, n.Body, nft, currentClass)
		case *ast.ClassNode:
			name := ft.resolveClassLike(n.Name)
			templates, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			class := ResolvedClass{
				Name:                  name,
				Extends:               resolvedList(ft, optionalList(n.Extends)),
				Implements:            resolvedList(ft, n.Implements),
				TemplateParams:        templates,
				GenericParents:        genericParents,
				Traits:                traitUsesFromMembers(n.Properties, ft),
				Kind:                  "class",
				Final:                 strings.Contains(n.Modifier, "final"),
				Abstract:              strings.Contains(n.Modifier, "abstract"),
				Readonly:              strings.Contains(n.Modifier, "readonly"),
				ConsistentConstructor: hasPHPStanConsistentConstructorTag(n.PHPDoc),
			}
			idx.addClass(filename, class, n)
			idx.indexClassMembers(filename, name, n.Properties, n.Methods, n.Constants, ft, templates)
		case *ast.InterfaceNode:
			name := ft.resolveClassLike(n.Name)
			templates, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			idx.addClass(filename, ResolvedClass{Name: name, Extends: resolvedList(ft, n.Extends), TemplateParams: templates, GenericParents: genericParents, Kind: "interface"}, n)
			idx.indexInterfaceMembers(filename, name, n.Members, ft, templates)
		case *ast.TraitNode:
			if n.Name != nil {
				name := ft.resolveClassLike(n.Name.Name)
				idx.addClass(filename, ResolvedClass{Name: name, Kind: "trait"}, n)
				idx.indexClassMembers(filename, name, n.Body, nil, nil, ft, nil)
			}
		case *ast.EnumNode:
			name := ft.resolveClassLike(n.Name)
			idx.addClass(filename, ResolvedClass{Name: name, Implements: resolvedList(ft, n.Implements), Kind: "enum", Final: true}, n)
			idx.indexClassMembers(filename, name, nil, n.Methods, nil, ft, nil)
			for _, enumCase := range n.Cases {
				idx.addClassConstant(name, ResolvedConstant{Name: enumCase.Name, DeclaringClass: name, Declaration: sourceLocation(filename, enumCase), Visibility: "public"})
			}
			declaration := sourceLocation(filename, n)
			idx.addMethod(name, ResolvedMethod{Name: "cases", DeclaringClass: name, Declaration: declaration, ReturnType: "array", Visibility: "public", IsStatic: true})
			idx.addMethod(name, ResolvedMethod{Name: "from", DeclaringClass: name, Declaration: declaration, ReturnType: name, Params: []ResolvedParam{{Name: "value"}}, Visibility: "public", IsStatic: true})
			idx.addMethod(name, ResolvedMethod{Name: "tryFrom", DeclaringClass: name, Declaration: declaration, ReturnType: "?" + name, Params: []ResolvedParam{{Name: "value"}}, Visibility: "public", IsStatic: true})
		case *ast.FunctionNode:
			if currentClass != "" {
				idx.addMethod(currentClass, methodFromFunction(filename, currentClass, n, ft, nil))
				continue
			}
			name := ft.resolveClassLike(n.Name)
			returnType := n.ReturnType
			if n.PHPDoc != nil && n.PHPDoc.ReturnType != "" {
				returnType = n.PHPDoc.ReturnType
			}
			callableReturn := callableReturnType(returnType, ft)
			normalizedReturn := normalizeTypeWithContext(returnType, ft)
			if !callableReturn.IsEmpty() {
				normalizedReturn = "callable"
			}
			fn := ResolvedFunction{Name: name, Declaration: sourceLocation(filename, n), ReturnType: normalizedReturn, CallableReturnType: callableReturn.dnfString(), Params: paramsFromNodesWithPHPDoc(n.Params, n.PHPDoc, ft, nil)}
			if n.PHPDoc != nil {
				fn.Deprecated = n.PHPDoc.Deprecated
				fn.DeprecationMessage = n.PHPDoc.DeprecationMessage
			}
			idx.addFunction(fn)
		case *ast.ConstantNode:
			idx.addGlobalConstant(filename, ft.resolveClassLike(n.Name))
		}
	}
}

// indexPromotedProperties registers PHP 8 constructor-promoted parameters
// (e.g. `private readonly Foo $x` in a __construct signature) as properties,
// mirroring promotedClassProperties' handling for return-type inference so
// property-existence checks don't false-positive on them.
func (idx *ProjectIndex) indexPromotedProperties(filename, className string, constructor *ast.FunctionNode, ft FileTypeContext) {
	for _, paramNode := range constructor.Params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok || !param.IsPromoted {
			continue
		}
		idx.addProperty(className, ResolvedProperty{
			Declaration: sourceLocation(filename, param),
			Name:        param.Name,
			Type:        normalizeTypeWithContext(param.TypeHint, ft),
			Visibility:  defaultVisibility(param.Visibility),
			Readonly:    param.IsReadonly,
		})
	}
}

func (idx *ProjectIndex) indexClassMembers(filename, className string, properties, methods, constants []ast.Node, ft FileTypeContext, templateParams []string) {
	for _, propNode := range properties {
		switch p := propNode.(type) {
		case *ast.PropertyNode:
			rawType := p.TypeHint
			if p.PHPDoc != nil && p.PHPDoc.VarType != "" {
				rawType = p.PHPDoc.VarType
			}
			callableReturn := callableReturnType(rawType, ft)
			normalizedType := normalizeTypeWithContext(rawType, ft)
			if !callableReturn.IsEmpty() {
				normalizedType = "callable"
			}
			idx.addProperty(className, ResolvedProperty{
				Declaration:        sourceLocation(filename, p),
				Name:               p.Name,
				Type:               normalizedType,
				CallableReturnType: callableReturn.dnfString(),
				Visibility:         defaultVisibility(p.Visibility),
				IsStatic:           p.IsStatic,
				Readonly:           p.IsReadonly,
			})
		case *ast.TraitUseNode:
			// Trait use is checked by level-0 rules; no index entry needed.
		case *ast.FunctionNode:
			idx.addMethod(className, methodFromFunction(filename, className, p, ft, templateParams))
		}
	}
	for _, methodNode := range methods {
		fn, ok := methodNode.(*ast.FunctionNode)
		if !ok {
			continue
		}
		idx.addMethod(className, methodFromFunction(filename, className, fn, ft, templateParams))
		if strings.EqualFold(fn.Name, "__construct") {
			idx.indexPromotedProperties(filename, className, fn, ft)
		}
	}
	for _, constNode := range constants {
		if c, ok := constNode.(*ast.ConstantNode); ok {
			idx.addClassConstant(className, constantFromNode(filename, className, c, ft))
		}
	}
}

func (idx *ProjectIndex) indexInterfaceMembers(filename, className string, members []ast.Node, ft FileTypeContext, templateParams []string) {
	templates := templateNames(templateParams)
	for _, member := range members {
		switch m := member.(type) {
		case *ast.InterfaceMethodNode:
			returnType := ""
			if m.ReturnType != nil {
				returnType = m.ReturnType.TokenLiteral()
			}
			if m.PHPDoc != nil && m.PHPDoc.ReturnType != "" {
				returnType = m.PHPDoc.ReturnType
			}
			callableReturn := callableReturnType(returnType, ft)
			normalizedReturn := normalizeTemplateAwareType(returnType, ft, templates)
			if !callableReturn.IsEmpty() {
				normalizedReturn = "callable"
			}
			idx.addMethod(className, ResolvedMethod{Name: m.Name, DeclaringClass: className, Declaration: sourceLocation(filename, m), ReturnType: normalizedReturn, CallableReturnType: callableReturn.dnfString(), Params: paramsFromNodesWithPHPDoc(m.Params, m.PHPDoc, ft, templates), Visibility: "public", Abstract: true})
		case *ast.PropertyNode:
			rawType := m.TypeHint
			if m.PHPDoc != nil && m.PHPDoc.VarType != "" {
				rawType = m.PHPDoc.VarType
			}
			callableReturn := callableReturnType(rawType, ft)
			normalizedType := normalizeTypeWithContext(rawType, ft)
			if !callableReturn.IsEmpty() {
				normalizedType = "callable"
			}
			idx.addProperty(className, ResolvedProperty{Name: m.Name, DeclaringClass: className, Declaration: sourceLocation(filename, m), Type: normalizedType, CallableReturnType: callableReturn.dnfString(), Visibility: "public", Readonly: m.IsReadonly})
		case *ast.ConstantNode:
			idx.addClassConstant(className, constantFromNode(filename, className, m, ft))
		}
	}
}

func (idx *ProjectIndex) addClass(filename string, class ResolvedClass, node ast.Node) {
	key := indexKey(class.Name)
	class.ID = stableSymbolID("class", "", class.Name)
	class.Declaration = sourceLocation(filename, node)
	_, exists := idx.Classes[key]
	if exists {
		idx.collidingDefinitions[classDefinitionKey(class.Name)] = struct{}{}
		idx.Duplicates = append(idx.Duplicates, DuplicateSymbol{File: filename, Name: class.Name, Pos: node.GetPos()})
		return
	}
	idx.Classes[key] = class
	idx.classLineages = nil

	// Track class → file mapping for incremental analysis
	if idx.fileClasses[filename] == nil {
		idx.fileClasses[filename] = make(map[string]struct{})
	}
	idx.fileClasses[filename][class.Name] = struct{}{}
}

func (idx *ProjectIndex) addFunction(fn ResolvedFunction) {
	fn.ID = stableSymbolID("function", "", fn.Name)
	key := indexKey(fn.Name)
	_, exists := idx.Functions[key]
	if exists {
		idx.collidingDefinitions[functionDefinitionKey(fn.Name)] = struct{}{}
	}
	idx.Functions[key] = fn
}

func (idx *ProjectIndex) addMethod(className string, method ResolvedMethod) {
	key := indexKey(className)
	method.DeclaringClass = className
	method.ID = stableSymbolID("method", className, method.Name)
	if idx.Methods[key] == nil {
		idx.Methods[key] = make(map[string]ResolvedMethod)
	}
	methodKey := strings.ToLower(method.Name)
	_, exists := idx.Methods[key][methodKey]
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("method", key, methodKey)] = struct{}{}
	}
	idx.Methods[key][methodKey] = method
	idx.methodsDeclared = nil
}

func buildMethodsDeclaredViews(project *ProjectIndex) map[string][]ResolvedMethod {
	views := make(map[string][]ResolvedMethod, len(project.Methods))
	for classKey := range project.Methods {
		className := classKey
		if class, ok := project.Classes[classKey]; ok {
			className = class.Name
		}
		views[classKey] = buildMethodsDeclaredView(project, classKey, className)
	}
	return views
}

func buildMethodsDeclaredView(project *ProjectIndex, classKey, className string) []ResolvedMethod {
	methodMap := project.Methods[classKey]
	methodKeys := make([]string, 0, len(methodMap))
	for methodKey := range methodMap {
		methodKeys = append(methodKeys, methodKey)
	}
	sort.Strings(methodKeys)
	methods := make([]ResolvedMethod, 0, len(methodKeys))
	for _, methodKey := range methodKeys {
		method := methodMap[methodKey]
		method.DeclaringClass = className
		if method.ID == "" {
			method.ID = stableSymbolID("method", className, method.Name)
		}
		methods = append(methods, method)
	}
	return methods
}

func (idx *ProjectIndex) addProperty(className string, property ResolvedProperty) {
	key := indexKey(className)
	property.DeclaringClass = className
	property.ID = stableSymbolID("property", className, strings.TrimPrefix(property.Name, "$"))
	if idx.Properties[key] == nil {
		idx.Properties[key] = make(map[string]ResolvedProperty)
	}
	propertyKey := strings.ToLower(property.Name)
	_, exists := idx.Properties[key][propertyKey]
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("property", key, propertyKey)] = struct{}{}
	}
	idx.Properties[key][propertyKey] = property
}

func (idx *ProjectIndex) addClassConstant(className string, constant ResolvedConstant) {
	key := indexKey(className)
	constant.DeclaringClass = className
	constant.ID = stableSymbolID("constant", className, constant.Name)
	if idx.ClassConsts[key] == nil {
		idx.ClassConsts[key] = make(map[string]ResolvedConstant)
	}
	constantKey := strings.ToLower(constant.Name)
	_, exists := idx.ClassConsts[key][constantKey]
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("class-constant", key, constantKey)] = struct{}{}
	}
	idx.ClassConsts[key][constantKey] = constant
	idx.Constants[indexKey(className+"::"+constant.Name)] = struct{}{}
}

func (idx *ProjectIndex) addGlobalConstant(filename, name string) {
	key := indexKey(name)
	_, exists := idx.globalConstantFiles[key]
	if exists {
		idx.collidingDefinitions[globalConstantDefinitionKey(key)] = struct{}{}
	}
	idx.Constants[key] = struct{}{}
	idx.globalConstantFiles[key] = filename
}

func methodFromFunction(filename, className string, fn *ast.FunctionNode, ft FileTypeContext, templateParams []string) ResolvedMethod {
	returnType := fn.ReturnType
	if fn.PHPDoc != nil && fn.PHPDoc.ReturnType != "" {
		returnType = fn.PHPDoc.ReturnType
	}
	templates := templateNames(templateParams)
	callableReturn := callableReturnType(returnType, ft)
	normalizedReturn := normalizeTemplateAwareType(returnType, ft, templates)
	if !callableReturn.IsEmpty() {
		normalizedReturn = "callable"
	}
	method := ResolvedMethod{
		Name:               fn.Name,
		DeclaringClass:     className,
		Declaration:        sourceLocation(filename, fn),
		ReturnType:         normalizedReturn,
		CallableReturnType: callableReturn.dnfString(),
		Params:             paramsFromNodesWithPHPDoc(fn.Params, fn.PHPDoc, ft, templates),
		Visibility:         functionVisibility(fn),
		IsStatic:           hasModifier(fn.Modifiers, "static"),
		Abstract:           hasModifier(fn.Modifiers, "abstract"),
		Final:              hasModifier(fn.Modifiers, "final"),
	}
	if fn.PHPDoc != nil {
		method.Deprecated = fn.PHPDoc.Deprecated
		method.DeprecationMessage = fn.PHPDoc.DeprecationMessage
	}
	return method
}

func resolvedGenericMetadata(doc *ast.PHPDocNode, ft FileTypeContext) ([]string, []ResolvedGenericParent) {
	if doc == nil {
		return nil, nil
	}
	templates := make([]string, 0, len(doc.Templates))
	for _, template := range doc.Templates {
		templates = append(templates, template.Name)
	}
	templateSet := templateNames(templates)
	references := append(append([]ast.PHPDocTypeReference(nil), doc.Extends...), doc.Implements...)
	parents := make([]ResolvedGenericParent, 0, len(references))
	for _, ref := range references {
		parent := ResolvedGenericParent{Name: ft.resolveClassLike(ref.Name)}
		for _, argument := range ref.TypeArguments {
			parent.TypeArguments = append(parent.TypeArguments, normalizeTemplateAwareType(argument, ft, templateSet))
		}
		parents = append(parents, parent)
	}
	return templates, parents
}

func constantFromNode(filename, className string, c *ast.ConstantNode, ft FileTypeContext) ResolvedConstant {
	return ResolvedConstant{
		Name:           c.Name,
		DeclaringClass: className,
		Declaration:    sourceLocation(filename, c),
		Type:           normalizeTypeWithContext(c.Type, ft),
		Visibility:     defaultVisibility(c.Visibility),
		Final:          hasModifier(c.Modifiers, "final"),
	}
}

func sourceLocation(filename string, node ast.Node) SourceLocation {
	if node == nil {
		return SourceLocation{}
	}
	return SourceLocation{File: filename, Start: node.GetPos(), End: node.GetEndPos()}
}

func functionVisibility(fn *ast.FunctionNode) string {
	if fn.Visibility != "" {
		return defaultVisibility(fn.Visibility)
	}
	if hasModifier(fn.Modifiers, "private") {
		return "private"
	}
	if hasModifier(fn.Modifiers, "protected") {
		return "protected"
	}
	return "public"
}

func paramsFromNodes(nodes []ast.Node, ft FileTypeContext) []ResolvedParam {
	return paramsFromNodesWithPHPDoc(nodes, nil, ft, nil)
}

func paramsFromNodesWithPHPDoc(nodes []ast.Node, doc *ast.PHPDocNode, ft FileTypeContext, templates map[string]struct{}) []ResolvedParam {
	params := make([]ResolvedParam, 0, len(nodes))
	for _, node := range nodes {
		param, ok := node.(*ast.ParamNode)
		if !ok {
			continue
		}
		typ := param.TypeHint
		if typ == "" && param.UnionType != nil {
			typ = param.UnionType.TokenLiteral()
		}
		if doc != nil {
			if documented := doc.GetParamTypeFromPHPDoc(param.Name); documented != "" {
				typ = documented
			}
		}
		params = append(params, ResolvedParam{
			Name:       param.Name,
			Type:       normalizeTemplateAwareType(typ, ft, templates),
			HasDefault: param.DefaultValue != nil,
			IsVariadic: param.IsVariadic,
			IsByRef:    param.IsByRef,
			IsOut:      param.IsByRef,
		})
	}
	return params
}

func resolvedList(ft FileTypeContext, names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, ft.resolveClassLike(name))
	}
	return out
}

func traitUsesFromMembers(members []ast.Node, ft FileTypeContext) []string {
	var traits []string
	for _, member := range members {
		use, ok := member.(*ast.TraitUseNode)
		if !ok {
			continue
		}
		for _, trait := range use.Traits {
			traits = append(traits, ft.resolveClassLike(trait))
		}
	}
	return traits
}

func optionalList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}

func hasModifier(modifiers []string, wanted string) bool {
	for _, modifier := range modifiers {
		if strings.EqualFold(modifier, wanted) {
			return true
		}
	}
	return false
}

func defaultVisibility(visibility string) string {
	if visibility == "" {
		return "public"
	}
	return visibility
}

func indexKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), `\`))
}

func unqualifiedName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if i := strings.LastIndex(name, `\`); i >= 0 {
		return name[i+1:]
	}
	return name
}

var builtinClassNames = map[string]struct{}{
	"arrayaccess":         {},
	"arrayiterator":       {},
	"arrayobject":         {},
	"closure":             {},
	"countable":           {},
	"dateinterval":        {},
	"datetime":            {},
	"datetimeimmutable":   {},
	"datetimeinterface":   {},
	"datetimezone":        {},
	"error":               {},
	"exception":           {},
	"generator":           {},
	"iterator":            {},
	"iteratoraggregate":   {},
	"jsonexception":       {},
	"jsonserializable":    {},
	"reflectionclass":     {},
	"reflectionexception": {},
	"reflectionfunction":  {},
	"reflectionmethod":    {},
	"reflectionnamedtype": {},
	"reflectionobject":    {},
	"reflectionparameter": {},
	"reflectionproperty":  {},
	"sensitiveparameter":  {},
	"simplexmlelement":    {},
	"stdclass":            {},
	"stringable":          {},
	"throwable":           {},
	"traversable":         {},
	"valueerror":          {},
}

func isBuiltinClassName(name string) bool {
	_, ok := builtinClassNames[indexKey(name)]
	return ok
}

func (idx *ProjectIndex) resolveKnownClassSuffix(key string) (ResolvedClass, bool) {
	parts := strings.Split(strings.TrimPrefix(key, `\`), `\`)
	for i := 1; i < len(parts)-1; i++ {
		suffix := strings.Join(parts[i:], `\`)
		if class, ok := idx.Classes[suffix]; ok && strings.Contains(suffix, `\`) {
			return class, true
		}
	}
	return ResolvedClass{}, false
}

func (idx *ProjectIndex) seedBuiltins() {
	for _, class := range []ResolvedClass{
		{Name: "stdClass", Kind: "class"},
		{Name: "Exception", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "Throwable", Kind: "interface"},
		{Name: "Error", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "DateTime", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeImmutable", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeInterface", Kind: "interface"},
		{Name: "DateTimeZone", Kind: "class"},
		{Name: "Closure", Kind: "class", Final: true},
		{Name: "Stringable", Kind: "interface"},
		{Name: "ReflectionClass", Kind: "class"},
		{Name: "ReflectionException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "ReflectionFunction", Kind: "class"},
		{Name: "ReflectionMethod", Kind: "class"},
		{Name: "ReflectionNamedType", Kind: "class"},
		{Name: "ReflectionObject", Kind: "class", Extends: []string{"ReflectionClass"}},
		{Name: "ReflectionParameter", Kind: "class"},
		{Name: "ReflectionProperty", Kind: "class"},
		{Name: "ArrayAccess", Kind: "interface"},
		{Name: "ArrayIterator", Kind: "class", Implements: []string{"Iterator", "Traversable"}},
		{Name: "ArrayObject", Kind: "class", Implements: []string{"IteratorAggregate", "Traversable"}},
		{Name: "Countable", Kind: "interface"},
		{Name: "DateInterval", Kind: "class"},
		{Name: "Generator", Kind: "class", Final: true},
		{Name: "Iterator", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "IteratorAggregate", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "JsonException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "JsonSerializable", Kind: "interface"},
		{Name: "SensitiveParameter", Kind: "class", Final: true},
		{Name: "SimpleXMLElement", Kind: "class"},
		{Name: "Traversable", Kind: "interface"},
		{Name: "ValueError", Kind: "class", Extends: []string{"Error"}},
	} {
		class.ID = stableSymbolID("class", "", class.Name)
		idx.Classes[indexKey(class.Name)] = class
	}
	for _, className := range []string{"DateTime", "DateTimeImmutable"} {
		idx.addMethod(className, ResolvedMethod{Name: "createFromFormat", DeclaringClass: className, ReturnType: className + "|false", Params: []ResolvedParam{{Name: "format"}, {Name: "datetime"}, {Name: "timezone", HasDefault: true}}, Visibility: "public", IsStatic: true})
		idx.addMethod(className, ResolvedMethod{Name: "createFromInterface", DeclaringClass: className, ReturnType: className, Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
		idx.addMethod(className, ResolvedMethod{Name: "getLastErrors", DeclaringClass: className, ReturnType: "array|false", Visibility: "public", IsStatic: true})
	}
	idx.addMethod("DateTime", ResolvedMethod{Name: "createFromImmutable", DeclaringClass: "DateTime", ReturnType: "DateTime", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("DateTimeImmutable", ResolvedMethod{Name: "createFromMutable", DeclaringClass: "DateTimeImmutable", ReturnType: "DateTimeImmutable", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("Closure", ResolvedMethod{Name: "fromCallable", DeclaringClass: "Closure", ReturnType: "Closure", Params: []ResolvedParam{{Name: "callback"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("DateTimeZone", ResolvedMethod{Name: "__construct", DeclaringClass: "DateTimeZone", Params: []ResolvedParam{{Name: "timezone"}}, Visibility: "public"})
	idx.addMethod("DateInterval", ResolvedMethod{Name: "__construct", DeclaringClass: "DateInterval", Params: []ResolvedParam{{Name: "duration"}}, Visibility: "public"})
	idx.addMethod("ArrayObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ArrayObject", Params: []ResolvedParam{{Name: "array", HasDefault: true}, {Name: "flags", HasDefault: true}, {Name: "iteratorClass", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("Error", ResolvedMethod{Name: "__construct", DeclaringClass: "Error", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("Exception", ResolvedMethod{Name: "__construct", DeclaringClass: "Exception", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("ReflectionClass", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionClass", Params: []ResolvedParam{{Name: "objectOrClass"}}, Visibility: "public"})
	idx.addMethod("ReflectionMethod", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionMethod", Params: []ResolvedParam{{Name: "objectOrMethod"}, {Name: "method", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("ReflectionProperty", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionProperty", Params: []ResolvedParam{{Name: "class"}, {Name: "property"}}, Visibility: "public"})
	idx.addMethod("ReflectionObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionObject", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public"})
	for _, constant := range []string{"ATOM", "COOKIE", "ISO8601", "RFC822", "RFC850", "RFC1036", "RFC1123", "RFC7231", "RFC2822", "RFC3339", "RFC3339_EXTENDED", "RSS", "W3C"} {
		idx.addClassConstant("DateTime", ResolvedConstant{Name: constant, DeclaringClass: "DateTime", Visibility: "public"})
		idx.addClassConstant("DateTimeImmutable", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeImmutable", Visibility: "public"})
		idx.addClassConstant("DateTimeInterface", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeInterface", Visibility: "public"})
	}
	for _, fn := range []ResolvedFunction{
		{Name: "abs", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "addslashes", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "array_any", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}}},
		{Name: "array_chunk", Params: []ResolvedParam{{Name: "array"}, {Name: "length"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_column", Params: []ResolvedParam{{Name: "array"}, {Name: "column_key"}, {Name: "index_key", HasDefault: true}}},
		{Name: "array_diff", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_fill", Params: []ResolvedParam{{Name: "start_index"}, {Name: "count"}, {Name: "value"}}},
		{Name: "array_fill_keys", Params: []ResolvedParam{{Name: "keys"}, {Name: "value"}}},
		{Name: "array_filter", Params: []ResolvedParam{{Name: "array"}, {Name: "callback", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "array_intersect_assoc", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_intersect_key", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_key_exists", Params: []ResolvedParam{{Name: "key"}, {Name: "array"}}},
		{Name: "array_key_first", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_map", Params: []ResolvedParam{{Name: "callback"}, {Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge_recursive", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_pop", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_push", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_reduce", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}, {Name: "initial", HasDefault: true}}},
		{Name: "array_search", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "array_shift", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_splice", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "offset"}, {Name: "length", HasDefault: true}, {Name: "replacement", HasDefault: true}}},
		{Name: "array_keys", Params: []ResolvedParam{{Name: "array"}, {Name: "filter_value", HasDefault: true}, {Name: "strict", HasDefault: true}}},
		{Name: "array_slice", Params: []ResolvedParam{{Name: "array"}, {Name: "offset"}, {Name: "length", HasDefault: true}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_unique", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}}},
		{Name: "array_unshift", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_sum", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_values", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_walk", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}, {Name: "arg", HasDefault: true}}},
		{Name: "array_walk_recursive", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}, {Name: "arg", HasDefault: true}}},
		{Name: "arsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "asort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "assert", Params: []ResolvedParam{{Name: "assertion"}, {Name: "description", HasDefault: true}}},
		{Name: "base64_encode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "basename", Params: []ResolvedParam{{Name: "path"}, {Name: "suffix", HasDefault: true}}},
		{Name: "bin2hex", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ceil", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "checkdate", Params: []ResolvedParam{{Name: "month"}, {Name: "day"}, {Name: "year"}}},
		{Name: "class_exists", Params: []ResolvedParam{{Name: "class"}, {Name: "autoload", HasDefault: true}}},
		{Name: "compact", Params: []ResolvedParam{{Name: "var_name"}, {Name: "var_names", IsVariadic: true}}},
		{Name: "constant", Params: []ResolvedParam{{Name: "name"}}},
		{Name: "count", Params: []ResolvedParam{{Name: "value"}, {Name: "mode", HasDefault: true}}},
		{Name: "crc32", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "define", Params: []ResolvedParam{{Name: "constant_name"}, {Name: "value"}, {Name: "case_insensitive", HasDefault: true}}},
		{Name: "defined", Params: []ResolvedParam{{Name: "constant_name"}}},
		{Name: "die", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "dirname", Params: []ResolvedParam{{Name: "path"}, {Name: "levels", HasDefault: true}}},
		{Name: "empty", Params: []ResolvedParam{{Name: "var"}}},
		{Name: "enum_exists", Params: []ResolvedParam{{Name: "enum"}, {Name: "autoload", HasDefault: true}}},
		{Name: "end", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "eval", Params: []ResolvedParam{{Name: "code"}}},
		{Name: "exec", Params: []ResolvedParam{{Name: "command"}, {Name: "output", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "exit", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "explode", Params: []ResolvedParam{{Name: "separator"}, {Name: "string"}, {Name: "limit", HasDefault: true}}},
		{Name: "extract", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}, {Name: "prefix", HasDefault: true}}},
		{Name: "extension_loaded", Params: []ResolvedParam{{Name: "extension"}}},
		{Name: "file_exists", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "filter_var", Params: []ResolvedParam{{Name: "value"}, {Name: "filter", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "floor", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "fpassthru", Params: []ResolvedParam{{Name: "stream"}}},
		{Name: "fscanf", Params: []ResolvedParam{{Name: "stream"}, {Name: "format"}, {Name: "vars", HasDefault: true, IsVariadic: true, IsByRef: true, IsOut: true}}},
		{Name: "func_get_args"},
		{Name: "function_exists", Params: []ResolvedParam{{Name: "function"}}},
		{Name: "get_class", Params: []ResolvedParam{{Name: "object", HasDefault: true}}},
		{Name: "get_object_vars", Params: []ResolvedParam{{Name: "object"}}},
		{Name: "getenv", Params: []ResolvedParam{{Name: "name", HasDefault: true}, {Name: "local_only", HasDefault: true}}},
		{Name: "getopt", Params: []ResolvedParam{{Name: "short_options"}, {Name: "long_options", HasDefault: true}, {Name: "rest_index", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "glob", Params: []ResolvedParam{{Name: "pattern"}, {Name: "flags", HasDefault: true}}},
		{Name: "hash", Params: []ResolvedParam{{Name: "algo"}, {Name: "data"}, {Name: "binary", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "headers_sent", Params: []ResolvedParam{{Name: "filename", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "line", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "http_build_query", Params: []ResolvedParam{{Name: "data"}, {Name: "numeric_prefix", HasDefault: true}, {Name: "arg_separator", HasDefault: true}, {Name: "encoding_type", HasDefault: true}}},
		{Name: "htmlspecialchars", Params: []ResolvedParam{{Name: "string"}, {Name: "flags", HasDefault: true}, {Name: "encoding", HasDefault: true}, {Name: "double_encode", HasDefault: true}}},
		{Name: "implode", Params: []ResolvedParam{{Name: "separator"}, {Name: "array", HasDefault: true}}},
		{Name: "in_array", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "intdiv", Params: []ResolvedParam{{Name: "num1"}, {Name: "num2"}}},
		{Name: "intval", Params: []ResolvedParam{{Name: "value"}, {Name: "base", HasDefault: true}}},
		{Name: "interface_exists", Params: []ResolvedParam{{Name: "interface"}, {Name: "autoload", HasDefault: true}}},
		{Name: "isset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "is_array", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_bool", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_callable", Params: []ResolvedParam{{Name: "value"}, {Name: "syntax_only", HasDefault: true}, {Name: "callable_name", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "is_countable", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_dir", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_file", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_finite", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "is_float", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_int", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_null", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_numeric", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_object", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_resource", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_scalar", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_string", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "iterator_count", Params: []ResolvedParam{{Name: "iterator"}}},
		{Name: "iterator_to_array", Params: []ResolvedParam{{Name: "iterator"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "json_last_error"},
		{Name: "krsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "ksort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "lcfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "libxml_clear_errors"},
		{Name: "libxml_get_errors"},
		{Name: "libxml_use_internal_errors", Params: []ResolvedParam{{Name: "use_errors", HasDefault: true}}},
		{Name: "ltrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "max", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "mb_strtoupper", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_strlen", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_substr", Params: []ResolvedParam{{Name: "string"}, {Name: "start"}, {Name: "length", HasDefault: true}, {Name: "encoding", HasDefault: true}}},
		{Name: "md5", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "method_exists", Params: []ResolvedParam{{Name: "object_or_class"}, {Name: "method"}}},
		{Name: "microtime", Params: []ResolvedParam{{Name: "as_float", HasDefault: true}}},
		{Name: "min", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "natcasesort", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "natsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "next", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "number_format", Params: []ResolvedParam{{Name: "num"}, {Name: "decimals", HasDefault: true}, {Name: "decimal_separator", HasDefault: true}, {Name: "thousands_separator", HasDefault: true}}},
		{Name: "parse_str", Params: []ResolvedParam{{Name: "string"}, {Name: "result", IsByRef: true, IsOut: true}}},
		{Name: "pathinfo", Params: []ResolvedParam{{Name: "path"}, {Name: "flags", HasDefault: true}}},
		{Name: "passthru", Params: []ResolvedParam{{Name: "command"}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_filter", Params: []ResolvedParam{{Name: "pattern"}, {Name: "replacement"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_match", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_match_all", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}, {Name: "offset", HasDefault: true}}},
		{Name: "preg_quote", Params: []ResolvedParam{{Name: "str"}, {Name: "delimiter", HasDefault: true}}},
		{Name: "preg_replace", Params: []ResolvedParam{{Name: "pattern"}, {Name: "replacement"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_replace_callback", Params: []ResolvedParam{{Name: "pattern"}, {Name: "callback"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}}},
		{Name: "preg_replace_callback_array", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}}},
		{Name: "prev", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "printf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "proc_open", Params: []ResolvedParam{{Name: "command"}, {Name: "descriptor_spec"}, {Name: "pipes", IsByRef: true, IsOut: true}, {Name: "cwd", HasDefault: true}, {Name: "env_vars", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "random_bytes", Params: []ResolvedParam{{Name: "length"}}},
		{Name: "reset", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "range", Params: []ResolvedParam{{Name: "start"}, {Name: "end"}, {Name: "step", HasDefault: true}}},
		{Name: "round", Params: []ResolvedParam{{Name: "num"}, {Name: "precision", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "rsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "rtrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "serialize", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "settype", Params: []ResolvedParam{{Name: "var", IsByRef: true}, {Name: "type"}}},
		{Name: "sha1", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "shuffle", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "similar_text", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}, {Name: "percent", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "sort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "sprintf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "sscanf", Params: []ResolvedParam{{Name: "string"}, {Name: "format"}, {Name: "vars", HasDefault: true, IsVariadic: true, IsByRef: true, IsOut: true}}},
		{Name: "stripos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "str_contains", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_ends_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_pad", Params: []ResolvedParam{{Name: "string"}, {Name: "length"}, {Name: "pad_string", HasDefault: true}, {Name: "pad_type", HasDefault: true}}},
		{Name: "str_repeat", Params: []ResolvedParam{{Name: "string"}, {Name: "times"}}},
		{Name: "str_replace", Params: []ResolvedParam{{Name: "search"}, {Name: "replace"}, {Name: "subject"}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "str_starts_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "strcasecmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strcmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strlen", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strrpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strtolower", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strtoupper", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strtr", Params: []ResolvedParam{{Name: "string"}, {Name: "from"}, {Name: "to", HasDefault: true}}},
		{Name: "substr", Params: []ResolvedParam{{Name: "string"}, {Name: "offset"}, {Name: "length", HasDefault: true}}},
		{Name: "substr_count", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}, {Name: "length", HasDefault: true}}},
		{Name: "sys_get_temp_dir"},
		{Name: "system", Params: []ResolvedParam{{Name: "command"}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "time"},
		{Name: "trait_exists", Params: []ResolvedParam{{Name: "trait"}, {Name: "autoload", HasDefault: true}}},
		{Name: "trim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "trigger_error", Params: []ResolvedParam{{Name: "message"}, {Name: "error_level", HasDefault: true}}},
		{Name: "uasort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "ucfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ucwords", Params: []ResolvedParam{{Name: "string"}, {Name: "separators", HasDefault: true}}},
		{Name: "uniqid", Params: []ResolvedParam{{Name: "prefix", HasDefault: true}, {Name: "more_entropy", HasDefault: true}}},
		{Name: "uksort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "urlencode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "usort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "unset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "usleep", Params: []ResolvedParam{{Name: "microseconds"}}},
	} {
		idx.addFunction(fn)
	}
	for _, constant := range []string{"PHP_VERSION", "PHP_VERSION_ID", "PHP_MAJOR_VERSION", "PHP_MINOR_VERSION", "PHP_OS", "PHP_EOL", "true", "false", "null"} {
		idx.addGlobalConstant("", constant)
	}
}

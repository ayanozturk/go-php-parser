package analyse

import (
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/phpstubs"
)

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

func rebuildProjectIndex(previous *ProjectIndex, parsed map[string][]ast.Node) *ProjectIndex {
	version := phpstubs.DefaultPHPVersion
	if previous != nil && previous.phpVersion != "" {
		version = previous.phpVersion
	}
	idx := BuildProjectIndexForVersion(parsed, version)
	if previous != nil {
		idx.importSymbolsFromUnparsedFiles(previous, parsed)
		idx.methodsDeclared = buildMethodsDeclaredViews(idx)
		idx.classLineages = buildClassLineageViews(idx)
	}
	return idx
}

// BuildProjectIndexIncrementalWithChanges is BuildProjectIndexIncremental with
// deterministic exported-symbol change details for dependency-scoped caches.
func BuildProjectIndexIncrementalWithChanges(previous *ProjectIndex, parsed map[string][]ast.Node, changedFiles []string) (*ProjectIndex, ProjectIndexChanges) {
	if previous == nil || previous.sourceFiles == nil {
		return rebuildProjectIndex(previous, parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
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
				return rebuildProjectIndex(previous, parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
			}
		}
	}
	for filename := range previous.sourceFiles {
		if _, exists := parsed[filename]; !exists {
			if _, listed := changed[filename]; !listed {
				return rebuildProjectIndex(previous, parsed), ProjectIndexChanges{Complete: false, FullRebuild: true}
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
		idx := rebuildProjectIndex(previous, parsed)
		changes.FullRebuild = true
		return idx, finalizeProjectIndexChanges(previous, idx, changes)
	}

	idx := cloneProjectIndexForFiles(previous, filenames)
	for _, filename := range filenames {
		idx.removeProjectFile(filename)
	}
	for _, filename := range filenames {
		contribution := newContributions[filename]
		if projectFileCollidesWithIndex(idx, contribution) {
			fresh := rebuildProjectIndex(previous, parsed)
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
	return cloneProjectIndexForFiles(previous, nil)
}

func cloneProjectIndexForFiles(previous *ProjectIndex, exclusiveFiles []string) *ProjectIndex {
	idx := newProjectIndex()
	idx.Classes = mapsClone(previous.Classes)
	idx.Functions = mapsClone(previous.Functions)
	idx.Constants = mapsClone(previous.Constants)
	idx.FileTypes = mapsClone(previous.FileTypes)
	idx.Duplicates = append([]DuplicateSymbol(nil), previous.Duplicates...)
	idx.fileClasses = cloneNestedSet(previous.fileClasses)
	idx.sourceFiles = mapsClone(previous.sourceFiles)
	idx.collidingDefinitions = mapsClone(previous.collidingDefinitions)
	idx.globalConstantFiles = mapsClone(previous.globalConstantFiles)
	idx.phpVersion = previous.phpVersion
	exclusive := exclusiveNestedKeys(previous, exclusiveFiles)
	idx.Methods = cloneNestedMapExclusive(previous.Methods, exclusive)
	idx.Properties = cloneNestedMapExclusive(previous.Properties, exclusive)
	idx.ClassConsts = cloneNestedMapExclusive(previous.ClassConsts, exclusive)
	return idx
}

func exclusiveNestedKeys(previous *ProjectIndex, files []string) map[string]struct{} {
	if previous == nil || len(files) == 0 {
		return nil
	}
	keys := make(map[string]struct{})
	owned := make(map[string]struct{}, len(files))
	for _, filename := range files {
		owned[filename] = struct{}{}
		for className := range previous.fileClasses[filename] {
			keys[indexKey(className)] = struct{}{}
		}
	}
	for key, class := range previous.Classes {
		if _, ok := owned[class.Declaration.File]; ok {
			keys[key] = struct{}{}
		}
	}
	addNestedKeysOwnedByFiles(keys, previous.Methods, owned, func(value ResolvedMethod) string { return value.Declaration.File })
	addNestedKeysOwnedByFiles(keys, previous.Properties, owned, func(value ResolvedProperty) string { return value.Declaration.File })
	addNestedKeysOwnedByFiles(keys, previous.ClassConsts, owned, func(value ResolvedConstant) string { return value.Declaration.File })
	return keys
}

func addNestedKeysOwnedByFiles[V any](keys map[string]struct{}, values map[string]map[string]V, files map[string]struct{}, fileOf func(V) string) {
	for outerKey, entries := range values {
		if _, already := keys[outerKey]; already {
			continue
		}
		for _, value := range entries {
			if _, ok := files[fileOf(value)]; ok {
				keys[outerKey] = struct{}{}
				break
			}
		}
	}
}

func cloneNestedMapExclusive[V any](source map[string]map[string]V, exclusive map[string]struct{}) map[string]map[string]V {
	if exclusive == nil {
		return cloneNestedMap(source)
	}
	cloned := make(map[string]map[string]V, len(source))
	for outerKey, values := range source {
		if _, copyInner := exclusive[outerKey]; copyInner {
			cloned[outerKey] = mapsClone(values)
			continue
		}
		cloned[outerKey] = values
	}
	return cloned
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

// DropRetainedTrees releases parsed ASTs for selected files while keeping their
// extracted symbols. Incremental updates must not list those files as changed
// unless they are reparsed; full rebuilds reimport the dropped files from the
// previous index.
func (idx *ProjectIndex) DropRetainedTrees(drop func(filename string) bool) {
	if idx == nil || drop == nil {
		return
	}
	for filename := range idx.sourceFiles {
		if !drop(filename) {
			if ft, ok := idx.FileTypes[filename]; ok {
				ft.ClassNodes = nil
				idx.FileTypes[filename] = ft
			}
			continue
		}
		delete(idx.sourceFiles, filename)
		delete(idx.FileTypes, filename)
	}
}

func (idx *ProjectIndex) importSymbolsFromUnparsedFiles(previous *ProjectIndex, parsed map[string][]ast.Node) {
	if idx == nil || previous == nil {
		return
	}
	for key, class := range previous.Classes {
		if _, exists := idx.Classes[key]; exists {
			continue
		}
		if _, parsedFile := parsed[class.Declaration.File]; parsedFile && class.Declaration.File != "" {
			continue
		}
		idx.Classes[key] = class
		if methods := previous.Methods[key]; methods != nil {
			idx.Methods[key] = methods
		}
		if properties := previous.Properties[key]; properties != nil {
			idx.Properties[key] = properties
		}
		if constants := previous.ClassConsts[key]; constants != nil {
			idx.ClassConsts[key] = constants
			for _, constant := range constants {
				idx.Constants[indexKey(constant.DeclaringClass+"::"+constant.Name)] = struct{}{}
			}
		}
		if owners := previous.fileClasses[class.Declaration.File]; owners != nil && class.Declaration.File != "" {
			if idx.fileClasses[class.Declaration.File] == nil {
				idx.fileClasses[class.Declaration.File] = mapsClone(owners)
			}
		}
	}
	for key, fn := range previous.Functions {
		if _, exists := idx.Functions[key]; exists {
			continue
		}
		if _, parsedFile := parsed[fn.Declaration.File]; parsedFile && fn.Declaration.File != "" {
			continue
		}
		idx.Functions[key] = fn
	}
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
	return kind + "\x00" + indexKey(className) + "\x00" + asciiLowerIdent(memberName)
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
		key := change.Kind + "\x00" + asciiLowerIdent(change.Owner) + "\x00" + asciiLowerIdent(change.Name)
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
			return asciiLowerIdent(left.Owner) < asciiLowerIdent(right.Owner)
		}
		return asciiLowerIdent(left.Name) < asciiLowerIdent(right.Name)
	})
	addDescendantDependencyNames(previous, classRoots, dependencyNames)
	addDescendantDependencyNames(current, classRoots, dependencyNames)
	changes.DependencyNames = make([]string, 0, len(dependencyNames))
	for _, name := range dependencyNames {
		changes.DependencyNames = append(changes.DependencyNames, name)
	}
	sort.Slice(changes.DependencyNames, func(i, j int) bool {
		return asciiLowerIdent(changes.DependencyNames[i]) < asciiLowerIdent(changes.DependencyNames[j])
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
	key := asciiLowerIdent(name)
	names[key] = name
	if index := strings.LastIndex(name, "\\"); index >= 0 && index+1 < len(name) {
		short := name[index+1:]
		names[asciiLowerIdent(short)] = short
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

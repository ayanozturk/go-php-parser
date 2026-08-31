package analyse

import (
	"crypto/md5"
	"sync"
)

// FileDependency tracks which symbols are defined/used in each file.
type FileDependency struct {
	// File path (key in FileDeps map)
	Path string

	// Symbols defined in this file
	DefinedClasses    map[string]struct{}
	DefinedFunctions  map[string]struct{}
	DefinedConstants  map[string]struct{}
	DefinedNamespaces map[string]struct{}

	// Symbols used/referenced in this file
	UsedClasses   map[string]struct{}
	UsedFunctions map[string]struct{}
	UsedConstants map[string]struct{}

	// File hash for detecting changes
	ContentHash [16]byte
}

// DependencyGraph tracks symbol definitions and usages across the project.
type DependencyGraph struct {
	// FileDeps maps file path → FileDependency
	FileDeps map[string]*FileDependency

	// ReverseClassDeps maps class name → files that define it
	ReverseClassDeps map[string][]string

	// ReverseClassUsage maps class name → files that use it
	ReverseClassUsage map[string][]string

	mu sync.RWMutex
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		FileDeps:          make(map[string]*FileDependency),
		ReverseClassDeps:  make(map[string][]string),
		ReverseClassUsage: make(map[string][]string),
	}
}

// AddFile records symbols in a file.
func (dg *DependencyGraph) AddFile(path string, contentHash [16]byte, defined, used SymbolSet) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	fd := &FileDependency{
		Path:              path,
		ContentHash:       contentHash,
		DefinedClasses:    defined.Classes,
		DefinedFunctions:  defined.Functions,
		DefinedConstants:  defined.Constants,
		DefinedNamespaces: defined.Namespaces,
		UsedClasses:       used.Classes,
		UsedFunctions:     used.Functions,
		UsedConstants:     used.Constants,
	}
	dg.FileDeps[path] = fd

	// Update reverse indices
	for className := range defined.Classes {
		dg.ReverseClassDeps[className] = append(dg.ReverseClassDeps[className], path)
	}
	for className := range used.Classes {
		dg.ReverseClassUsage[className] = append(dg.ReverseClassUsage[className], path)
	}
}

// FilesAffectedByChange returns files that need re-analysis when changedFile is modified.
// Includes the changed file itself and all files that depend on it.
func (dg *DependencyGraph) FilesAffectedByChange(changedFile string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	affected := make(map[string]struct{})
	affected[changedFile] = struct{}{}

	fd, ok := dg.FileDeps[changedFile]
	if !ok {
		return []string{changedFile}
	}

	// Find all files that use symbols from changed file
	for className := range fd.DefinedClasses {
		for _, usageFile := range dg.ReverseClassUsage[className] {
			affected[usageFile] = struct{}{}
		}
	}
	for funcName := range fd.DefinedFunctions {
		// Similar logic for functions (simplified for now)
		_ = funcName
	}

	result := make([]string, 0, len(affected))
	for f := range affected {
		result = append(result, f)
	}
	return result
}

// SymbolSet tracks classes, functions, constants, namespaces.
type SymbolSet struct {
	Classes    map[string]struct{}
	Functions  map[string]struct{}
	Constants  map[string]struct{}
	Namespaces map[string]struct{}
}

// FileChecksum computes MD5 hash of file content.
func FileChecksum(content []byte) [16]byte {
	h := md5.New()
	h.Write(content)
	var result [16]byte
	copy(result[:], h.Sum(nil))
	return result
}

// ContentChangedSince checks if file content has changed since last hash.
func ContentChangedSince(newContent []byte, oldHash [16]byte) bool {
	return FileChecksum(newContent) != oldHash
}

// FilesChanged detects which files have changed based on checksums.
// Returns changed file paths.
func FilesChanged(newChecksums, oldChecksums map[string]string) []string {
	var changed []string
	// Find files with different checksums
	for path, newHash := range newChecksums {
		oldHash, existed := oldChecksums[path]
		if !existed || newHash != oldHash {
			changed = append(changed, path)
		}
	}
	// Also include files that were removed (in old but not new)
	for path := range oldChecksums {
		if _, exists := newChecksums[path]; !exists {
			changed = append(changed, path)
		}
	}
	return changed
}

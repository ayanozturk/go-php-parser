package analyse

import (
	"path/filepath"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// IsVendoredPath reports whether path is under a Composer vendor tree.
// Matching Mago's FileType::Vendored, these files are indexed for symbols
// and never type-checked or linted.
func IsVendoredPath(path string) bool {
	if path == "" {
		return false
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "vendor" {
			return true
		}
	}
	return false
}

// HostFiles returns the paths that should receive per-file semantic analysis.
func HostFiles(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	host := make([]string, 0, len(paths))
	for _, path := range paths {
		if !IsVendoredPath(path) {
			host = append(host, path)
		}
	}
	return host
}

func hostSnapshotTargets(parsed map[string][]ast.Node, targets []string) []string {
	if len(targets) == 0 {
		targets = make([]string, 0, len(parsed))
		for filename := range parsed {
			targets = append(targets, filename)
		}
	}
	host := make([]string, 0, len(targets))
	for _, target := range targets {
		if IsVendoredPath(target) {
			continue
		}
		if _, ok := parsed[target]; !ok {
			continue
		}
		host = append(host, target)
	}
	return host
}

package main

import (
	"path/filepath"
	"strings"
)

func benchmarkPathExcluded(path string, excludes []string) bool {
	return benchmarkPathExcludedDir(path, excludes, false)
}

func benchmarkPathExcludedDir(path string, excludes []string, isDir bool) bool {
	for _, excluded := range excludes {
		if path == excluded || strings.HasPrefix(path, excluded+"/") {
			return true
		}
		if !strings.ContainsAny(excluded, "*?[") {
			continue
		}
		if matchPathGlob(excluded, path) {
			return true
		}
		if isDir && matchPathGlob(excluded, path+"/x") {
			return true
		}
	}
	return false
}

func matchPathGlob(pattern, path string) bool {
	return matchGlobSegments(splitPathSegments(pattern), splitPathSegments(path))
}

func splitPathSegments(path string) []string {
	path = strings.Trim(filepath.ToSlash(path), "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func matchGlobSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		if matchGlobSegments(pattern[1:], path) {
			return true
		}
		if len(path) == 0 {
			return false
		}
		return matchGlobSegments(pattern, path[1:])
	}
	if len(path) == 0 {
		return false
	}
	ok, err := filepath.Match(pattern[0], path[0])
	if err != nil || !ok {
		return false
	}
	return matchGlobSegments(pattern[1:], path[1:])
}

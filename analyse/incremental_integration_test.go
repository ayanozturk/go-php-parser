package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func TestFilesChanged(t *testing.T) {
	oldChecksums := map[string]string{
		"file1.php": "abc123",
		"file2.php": "def456",
		"file3.php": "ghi789",
	}

	// File1 changed, file2 unchanged, file3 removed, file4 added
	newChecksums := map[string]string{
		"file1.php": "changed",
		"file2.php": "def456",
		"file4.php": "new123",
	}

	changed := FilesChanged(newChecksums, oldChecksums)
	changedSet := make(map[string]struct{})
	for _, f := range changed {
		changedSet[f] = struct{}{}
	}

	if _, ok := changedSet["file1.php"]; !ok {
		t.Fatalf("file1.php should be in changed set (content changed)")
	}
	if _, ok := changedSet["file2.php"]; ok {
		t.Fatalf("file2.php should not be in changed set (unchanged)")
	}
	if _, ok := changedSet["file3.php"]; !ok {
		t.Fatalf("file3.php should be in changed set (removed)")
	}
	if _, ok := changedSet["file4.php"]; !ok {
		t.Fatalf("file4.php should be in changed set (added)")
	}
}

func TestProjectIndexMergeIncremental(t *testing.T) {
	// Test that MergeIncremental correctly removes old classes and adds new ones
	file1 := "test.php"

	// Create minimal index with one test class
	idx := &ProjectIndex{
		Classes:     make(map[string]ResolvedClass),
		Methods:     make(map[string]map[string]ResolvedMethod),
		Properties:  make(map[string]map[string]ResolvedProperty),
		ClassConsts: make(map[string]map[string]ResolvedConstant),
		Functions:   make(map[string]ResolvedFunction),
		fileClasses: make(map[string]map[string]struct{}),
	}

	// Add a class to the index
	idx.fileClasses[file1] = make(map[string]struct{})
	idx.fileClasses[file1]["OldClass"] = struct{}{}
	idx.Classes["oldclass"] = ResolvedClass{
		Name:        "OldClass",
		Kind:        "class",
		Declaration: SourceLocation{File: file1},
	}

	if _, ok := idx.Classes["oldclass"]; !ok {
		t.Fatalf("OldClass should exist initially")
	}

	// Parse updated file with new class
	source := `<?php
class NewClass {}
`
	p := parser.New(lexer.New(source), false)
	nodes := p.Parse()

	// Merge: re-index file1
	newParsed := map[string][]ast.Node{
		file1: nodes,
	}
	fileContexts := map[string]fileTypeContext{
		file1: collectFileTypeContext(nodes),
	}

	idx.MergeIncremental(newParsed, fileContexts)

	// Old class should be gone
	if _, ok := idx.Classes["oldclass"]; ok {
		t.Fatalf("OldClass should be removed after merge")
	}

	// New class should be added
	if _, ok := idx.Classes["newclass"]; !ok {
		t.Fatalf("NewClass should be added after merge (got %d classes)", len(idx.Classes))
	}
}

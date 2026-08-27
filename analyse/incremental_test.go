package analyse

import (
	"testing"
)

func TestFilesAffectedByChangedFile(t *testing.T) {
	// Build a project with inheritance
	idx := NewProjectIndex()

	// File 1: Base class
	file1 := "base.php"
	baseClass := ResolvedClass{
		Name:        "BaseEntity",
		Kind:        "class",
		Declaration: SourceLocation{File: file1},
	}
	idx.fileClasses[file1] = make(map[string]struct{})
	idx.fileClasses[file1]["BaseEntity"] = struct{}{}
	idx.Classes["baseentity"] = baseClass

	// File 2: Child class extending Base
	file2 := "derived.php"
	derivedClass := ResolvedClass{
		Name:        "UserEntity",
		Extends:     []string{"BaseEntity"},
		Kind:        "class",
		Declaration: SourceLocation{File: file2},
	}
	idx.fileClasses[file2] = make(map[string]struct{})
	idx.fileClasses[file2]["UserEntity"] = struct{}{}
	idx.Classes["userentity"] = derivedClass

	// File 3: Unrelated class
	file3 := "other.php"
	otherClass := ResolvedClass{
		Name:        "OtherClass",
		Kind:        "class",
		Declaration: SourceLocation{File: file3},
	}
	idx.fileClasses[file3] = make(map[string]struct{})
	idx.fileClasses[file3]["OtherClass"] = struct{}{}
	idx.Classes["otherclass"] = otherClass

	// When BaseEntity changes, both file1 and file2 (which extends it) are affected
	affected := idx.FilesAffectedByChangedFile(file1)
	hasFile1 := false
	hasFile2 := false
	for _, f := range affected {
		if f == file1 {
			hasFile1 = true
		}
		if f == file2 {
			hasFile2 = true
		}
	}
	if !hasFile1 {
		t.Fatalf("affected files should include changed file itself, got %v", affected)
	}
	if !hasFile2 {
		t.Fatalf("affected files should include dependent file, got %v", affected)
	}

	// When OtherClass changes, only file3 is affected
	affected2 := idx.FilesAffectedByChangedFile(file3)
	if len(affected2) != 1 || affected2[0] != file3 {
		t.Fatalf("unrelated file change should only affect itself, got %v", affected2)
	}
}

func TestContentChangedSince(t *testing.T) {
	content1 := []byte("hello world")
	hash1 := FileChecksum(content1)

	// Same content
	if ContentChangedSince(content1, hash1) {
		t.Fatalf("same content should not register as changed")
	}

	// Different content
	content2 := []byte("goodbye world")
	if !ContentChangedSince(content2, hash1) {
		t.Fatalf("different content should register as changed")
	}
}

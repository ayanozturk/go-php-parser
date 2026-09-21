package syntax

import (
	"strconv"
	"testing"
)

func walkRecursivePreOrder(root *RedNode, fn func(*RedNode) bool) {
	if root == nil || fn == nil {
		return
	}
	if !fn(root) {
		return
	}
	root.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		child := RedNode{File: root.File, Green: green, Offset: offset}
		walkRecursivePreOrder(&child, fn)
		return true
	})
}

func walkVisitKeys(root *RedNode, walk func(*RedNode, func(*RedNode) bool)) []string {
	var keys []string
	walk(root, func(n *RedNode) bool {
		keys = append(keys, n.Kind().String()+":"+strconv.Itoa(n.Offset))
		return true
	})
	return keys
}

func TestWalkPreOrderMatchesRecursiveReference(t *testing.T) {
	src := []byte("<?php\nif ($a = 1) {\n    echo $a;\n}\n")
	res := Parse(src)
	if res.File == nil || res.File.Root == nil {
		t.Fatal("parse failed")
	}
	root := res.File.Root
	recursive := walkVisitKeys(root, walkRecursivePreOrder)
	iterative := walkVisitKeys(root, Walk)
	if len(recursive) != len(iterative) {
		t.Fatalf("visit count: recursive %d, iterative %d", len(recursive), len(iterative))
	}
	for i := range recursive {
		if recursive[i] != iterative[i] {
			t.Fatalf("visit %d: recursive %q, iterative %q", i, recursive[i], iterative[i])
		}
	}
}

func TestWalkSkipsChildrenWhenFnReturnsFalse(t *testing.T) {
	src := []byte("<?php\nif ($a) {}\n")
	res := Parse(src)
	root := res.File.Root
	var kinds []Kind
	Walk(root, func(n *RedNode) bool {
		kinds = append(kinds, n.Kind())
		if n.Kind() == KindIfStmt {
			return false
		}
		return true
	})
	for _, k := range kinds {
		if k == KindStatementList {
			t.Fatalf("expected subtree skipped after IfStmt, saw %v", k)
		}
	}
}

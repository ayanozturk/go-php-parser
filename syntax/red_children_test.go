package syntax

import "testing"

func TestChildrenFreshWrappersSameShape(t *testing.T) {
	res := Parse([]byte("<?php\nfunction f() {}\n"))
	if res == nil || res.File == nil || res.File.Root == nil {
		t.Fatal("parse failed")
	}
	root := res.File.Root
	a := root.Children()
	b := root.Children()
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Kind() != b[i].Kind() || a[i].Offset != b[i].Offset {
			t.Fatalf("child %d: kind/offset mismatch %v/%d vs %v/%d",
				i, a[i].Kind(), a[i].Offset, b[i].Kind(), b[i].Offset)
		}
		if a[i].Parent != root || b[i].Parent != root {
			t.Fatalf("child %d: parent not receiver", i)
		}
		if a[i] == b[i] {
			t.Fatalf("child %d: expected distinct *RedNode wrappers", i)
		}
	}
}

func TestBuildChildDescsStableShape(t *testing.T) {
	res := Parse([]byte("<?php\nfunction f() {}\n"))
	if res == nil || res.File == nil || res.File.Root == nil {
		t.Fatal("parse failed")
	}
	root := res.File.Root
	if root.Green == nil {
		t.Fatal("nil green root")
	}
	a := buildChildDescs(root.Green, root.Offset)
	b := buildChildDescs(root.Green, root.Offset)
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].green != b[i].green || a[i].offset != b[i].offset {
			t.Fatalf("desc %d: green/offset mismatch", i)
		}
	}
	ch := root.Children()
	if len(ch) != len(a) {
		t.Fatalf("Children len %d != descs len %d", len(ch), len(a))
	}
	for i := range ch {
		if ch[i].Green != a[i].green || ch[i].Offset != a[i].offset {
			t.Fatalf("child %d: does not match buildChildDescs entry", i)
		}
	}
}

func TestForEachChildDescMatchesChildren(t *testing.T) {
	res := Parse([]byte("<?php\nclass C { public function m() {} }\n"))
	if res == nil || res.File == nil || res.File.Root == nil {
		t.Fatal("parse failed")
	}
	var fnCount int
	var descKinds []Kind
	res.File.Root.ForEachChildDesc(func(g *GreenNode, offset int) bool {
		fnCount++
		descKinds = append(descKinds, g.kind)
		return true
	})
	ch := res.File.Root.Children()
	if fnCount != len(ch) {
		t.Fatalf("ForEachChildDesc count %d != Children len %d", fnCount, len(ch))
	}
	for i := range ch {
		if ch[i].Kind() != descKinds[i] {
			t.Fatalf("child %d: kind mismatch", i)
		}
	}
}

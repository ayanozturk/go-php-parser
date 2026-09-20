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

package analyse

import "testing"

func TestProjectIndexLoadsBundledPHPStubs(t *testing.T) {
	idx := NewProjectIndex()
	if _, ok := idx.ResolveClass("UnitEnum"); !ok {
		t.Fatal("expected bundled UnitEnum from Core stubs")
	}
	if _, ok := idx.ResolveClass("BackedEnum"); !ok {
		t.Fatal("expected bundled BackedEnum from Core stubs")
	}
	if _, ok := idx.ResolveClass("ArrayObject"); !ok {
		t.Fatal("expected bundled ArrayObject from SPL stubs")
	}
	if _, ok := idx.ResolveMethod("DateTime", "createFromFormat"); !ok {
		t.Fatal("expected DateTime::createFromFormat")
	}
}

func TestPHPStubCatalogFollowsLanguageVersion(t *testing.T) {
	php82 := NewProjectIndexForVersion("8.2")
	if _, ok := php82.ResolveClass("Override"); ok {
		t.Fatal("PHP 8.2 must not expose Override")
	}
	php83 := NewProjectIndexForVersion("8.3")
	if _, ok := php83.ResolveClass("Override"); !ok {
		t.Fatal("expected Override in PHP 8.3 stubs")
	}
	if _, ok := php83.ResolveClass("Deprecated"); ok {
		t.Fatal("did not expect Deprecated before PHP 8.4")
	}
	php84 := NewProjectIndexForVersion("8.4")
	if _, ok := php84.ResolveClass("Deprecated"); !ok {
		t.Fatal("expected Deprecated in PHP 8.4 stubs")
	}
	php85 := NewProjectIndexForVersion("8.5")
	if _, ok := php85.ResolveClass("NoDiscard"); !ok {
		t.Fatal("expected NoDiscard in PHP 8.5 stubs")
	}
}

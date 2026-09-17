package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// walkSyntaxConfigured has no ast.Node equivalent to diff against yet (that
// requires the Phase 2 CST-direct FileTypeContext), so this test validates
// what Phase 1 actually promises: every CST node is visited, container-only
// kinds are skipped, and the enclosing class/function are threaded down
// correctly across namespace/class/method/closure boundaries.
func TestWalkSyntaxConfiguredContextPropagation(t *testing.T) {
	src := `<?php
function topLevelFn() {
    echo "top";
}

class Foo {
    public function bar() {
        echo "in bar";
        $cb = function () {
            echo "in closure";
        };
    }

    public function baz() {
        $x = 1;
    }
}
`
	res := syntax.Parse([]byte(src))

	type ctxCapture struct {
		classText string
		fnText    string
	}
	captured := map[string]ctxCapture{}
	walkSyntaxConfigured(res.File.Root, FileTypeContext{}, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext) {
		text := strings.TrimSpace(n.Text())
		key := ""
		switch {
		case n.Kind() == syntax.KindEchoStmt:
			key = text
		case n.Kind() == syntax.KindAssignExpr && strings.HasPrefix(text, "$x"):
			key = "$x = 1"
		default:
			return
		}
		var classText, fnText string
		if class != nil {
			classText = strings.TrimSpace(class.Text())
		}
		if currentFn != nil {
			fnText = strings.TrimSpace(currentFn.Text())
		}
		captured[key] = ctxCapture{classText: classText, fnText: fnText}
	})

	echoTop, ok := captured[`echo "top";`]
	if !ok {
		t.Fatal("expected to capture the top-level echo statement")
	}
	if echoTop.classText != "" {
		t.Fatalf("expected no enclosing class for top-level echo, got %q", echoTop.classText)
	}
	if !strings.Contains(echoTop.fnText, "topLevelFn") {
		t.Fatalf("expected enclosing function topLevelFn, got %q", echoTop.fnText)
	}

	echoBar, ok := captured[`echo "in bar";`]
	if !ok {
		t.Fatal("expected to capture the echo statement inside bar()")
	}
	if !strings.Contains(echoBar.classText, "class Foo") {
		t.Fatalf("expected enclosing class Foo, got %q", echoBar.classText)
	}
	if !strings.HasPrefix(echoBar.fnText, "public function bar") {
		t.Fatalf("expected enclosing function bar, got %q", echoBar.fnText)
	}

	echoClosure, ok := captured[`echo "in closure";`]
	if !ok {
		t.Fatal("expected to capture the echo statement inside the closure")
	}
	if !strings.Contains(echoClosure.classText, "class Foo") {
		t.Fatalf("expected the closure to still see enclosing class Foo, got %q", echoClosure.classText)
	}
	if !strings.HasPrefix(echoClosure.fnText, "function ()") {
		t.Fatalf("expected enclosing function to be the closure itself, got %q", echoClosure.fnText)
	}

	assignBaz, ok := captured[`$x = 1`]
	if !ok {
		t.Fatal("expected to capture the assignment inside baz()")
	}
	if !strings.Contains(assignBaz.classText, "class Foo") {
		t.Fatalf("expected enclosing class Foo, got %q", assignBaz.classText)
	}
	if !strings.HasPrefix(assignBaz.fnText, "public function baz") {
		t.Fatalf("expected enclosing function baz, got %q", assignBaz.fnText)
	}
}

// TestWalkSyntaxConfiguredSkipsContainerKinds asserts that pure grouping
// kinds (param lists, member lists, statement lists, …) never reach fn,
// which is what lets rule callbacks assume every visited node is a
// "semantic" one — the same guarantee walkAllConfigured gives for free by
// only ever calling its callback with concrete ast.Node values.
func TestWalkSyntaxConfiguredSkipsContainerKinds(t *testing.T) {
	src := `<?php
class Foo {
    public function bar(int $a, string $b) {
        return $a;
    }
}
`
	res := syntax.Parse([]byte(src))
	walkSyntaxConfigured(res.File.Root, FileTypeContext{}, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext) {
		if syntaxContainerOnlyKinds[n.Kind()] {
			t.Fatalf("container-only kind %v reached fn (text %q)", n.Kind(), n.Text())
		}
	})
}

// TestWalkSyntaxConfiguredNilRoot ensures the nil-root guard doesn't panic.
func TestWalkSyntaxConfiguredNilRoot(t *testing.T) {
	walkSyntaxConfigured(nil, FileTypeContext{}, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext) {
		t.Fatal("fn should never be called for a nil root")
	})
}

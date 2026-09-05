package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestTernaryBranchNarrowing(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"truthy receiver", `return $lamp ? $lamp->glow() : '';`, 0},
		{"non-null receiver", `return $lamp !== null ? $lamp->glow() : '';`, 0},
		{"null false branch", `return $lamp === null ? '' : $lamp->glow();`, 0},
		{"negated receiver", `return !$lamp ? '' : $lamp->glow();`, 0},
		{"predicate false branch", `return is_null($lamp) ? '' : $lamp->glow();`, 0},
		{"nested branches", `return $lamp ? ($lamp !== null ? $lamp->glow() : '') : '';`, 0},
		{"argument", `return $lamp ? illuminate($lamp) : '';`, 0},
		{"false branch argument", `return $lamp === null ? '' : illuminate($lamp);`, 0},
		{"branch isolation", `$lamp ? $lamp->glow() : ''; return illuminate($lamp);`, 1},
		{"opposite branch", `return $lamp ? '' : $lamp->glow();`, 1},
		{"unrelated condition", `return $flag ? $lamp->glow() : '';`, 1},
		{"known invalid method", `return $lamp ? $lamp->missing() : '';`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": `<?php
class Lantern { public function glow(): string { return 'lit'; } }
function illuminate(Lantern $lamp): string { return $lamp->glow(); }
function inspect(?Lantern $lamp, bool $flag): string { ` + strings.ReplaceAll(tc.body, "; ", ";\n") + ` }
`}, 8)
			if len(issues) != tc.want {
				t.Fatalf("got %d issues, want %d: %#v", len(issues), tc.want, issues)
			}
		})
	}
}

func TestTernaryNarrowingReturnAndPropertyTypes(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": `<?php
class Lantern {
    public Lantern $peer;
    public function setPeer(?Lantern $lamp): void { $this->peer = $lamp ?: new Lantern(); }
}
function fallback(?Lantern $lamp): Lantern { return $lamp ?: new Lantern(); }
function choose(?Lantern $lamp): Lantern { return $lamp ? $lamp : new Lantern(); }
function reverse(?Lantern $lamp): Lantern { return $lamp === null ? new Lantern() : $lamp; }
`}, 8)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestTernaryHoverUsesBranchScope(t *testing.T) {
	const source = `<?php
class Lantern {}
function choose(?Lantern $lamp): Lantern {
    return $lamp ? $lamp : new Lantern();
}
`
	nodes := parseHoverFixture(t, source)
	project := BuildProjectIndex(map[string][]ast.Node{"test.php": nodes})
	ctx := &AnalysisContext{Resolver: project}
	for _, tc := range []struct {
		column int
		want   string
	}{{13, "Lantern|null"}, {21, "Lantern"}} {
		target, ok := InferHoverTargetAtPosition(nodes, 4, tc.column, "lamp", ctx)
		if !ok || target.Type != tc.want {
			t.Fatalf("column %d: got %#v, %t; want %s", tc.column, target, ok, tc.want)
		}
	}
}

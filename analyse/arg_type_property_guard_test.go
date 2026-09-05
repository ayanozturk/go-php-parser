package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestDirectPropertyGuards(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"non-null branch", `if ($this->beacon !== null) { return useBeacon($this->beacon); } return '';`, ""},
		{"truthy branch", `if ($this->beacon) { return $this->beacon->ping(); } return '';`, ""},
		{"early return", `if ($this->beacon === null) { return ''; } return useBeacon($this->beacon);`, ""},
		{"reverse null guard", `if (null === $this->beacon) { return ''; } return useBeacon($this->beacon);`, ""},
		{"negated early return", `if (!$this->beacon) { return ''; } return useBeacon($this->beacon);`, ""},
		{"else branch", `if ($this->beacon === null) { return ''; } else { return useBeacon($this->beacon); }`, ""},
		{"invalid null branch", `if ($this->beacon === null) { return useBeacon($this->beacon); } return '';`, "A.ARG.TYPE"},
		{"and branch", `if ($this->beacon !== null && $other->beacon !== null) { return useBeacon($this->beacon); } return '';`, ""},
		{"or early return", `if ($this->beacon === null || $other->beacon === null) { return ''; } return useBeacon($this->beacon);`, ""},
		{"or is not a guard", `if ($this->beacon !== null || $other->beacon !== null) { return useBeacon($this->beacon); } return '';`, "A.ARG.TYPE"},
		{"ternary", `return $this->beacon ? $this->beacon->ping() : '';`, ""},
		{"ternary false", `return $this->beacon === null ? '' : useBeacon($this->beacon);`, ""},
		{"negated ternary", `return !$this->beacon ? '' : useBeacon($this->beacon);`, ""},
		{"other receiver", `if ($this->beacon === null) { return ''; } return useBeacon($other->beacon);`, "A.ARG.TYPE"},
		{"other property", `if ($this->beacon === null) { return ''; } return useBeacon($this->Beacon);`, "A.ARG.TYPE"},
		{"reassigned property", `if ($this->beacon === null) { return ''; } $this->beacon = null; return useBeacon($this->beacon);`, "A.ARG.TYPE"},
		{"branch isolation", `if ($this->beacon !== null) { useBeacon($this->beacon); } return useBeacon($this->beacon);`, "A.ARG.TYPE"},
		{"nonterminating guard", `if ($this->beacon === null) { $label = ''; } return useBeacon($this->beacon);`, "A.ARG.TYPE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `<?php
class Beacon { public function ping(): string { return 'ready'; } }
function useBeacon(Beacon $beacon): string { return $beacon->ping(); }
class Harbor {
    public ?Beacon $beacon;
    public ?Beacon $Beacon;
    public function inspect(Harbor $other): string { ` + tc.body + ` }
}`
			nodes := parsePHPForLevel0(t, source)
			files := map[string][]ast.Node{"guard.php": nodes}
			snapshot, err := NewSemanticSnapshot(files, nil)
			if err != nil {
				t.Fatal(err)
			}
			level := 8
			for _, ctx := range []*AnalysisContext{{Resolver: BuildProjectIndex(files)}, snapshot.NewAnalysisContext()} {
				ctx.AnalysisLevel = &level
				issues := RunAnalysisRulesWithContext("guard.php", nodes, ctx)
				if tc.want == "" && len(issues) != 0 {
					t.Fatalf("unexpected issues: %#v", issues)
				}
				if tc.want != "" && (len(issues) != 1 || issues[0].Code != tc.want) {
					t.Fatalf("want one %s, got %#v", tc.want, issues)
				}
			}
		})
	}
}

func TestPropertyGuardReturnAndAssignmentTypes(t *testing.T) {
	source := `<?php
class Beacon {}
class Harbor {
    public ?Beacon $beacon;
    public Beacon $selected;
    public function select(): Beacon {
        if ($this->beacon === null) { return new Beacon(); }
        $this->selected = $this->beacon;
        return $this->beacon;
    }
    public function choose(): Beacon {
        if ($this->beacon !== null) { return $this->beacon; }
        return new Beacon();
    }
    public function fallback(): Beacon { return $this->beacon ?: new Beacon(); }
}`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"guard.php": source}, 8)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestPropertyGuardHoverPreservesReceiverIdentity(t *testing.T) {
	source := `<?php
class Beacon {}
class Harbor {
    public ?Beacon $beacon;
    public function inspect(Harbor $other): ?Beacon {
        if ($this->beacon === null) { return null; }
        $selected = $this->beacon;
        return $other->beacon;
    }
}`
	nodes := parseHoverFixture(t, source)
	ctx := &AnalysisContext{Resolver: BuildProjectIndex(map[string][]ast.Node{"guard.php": nodes})}
	for _, tc := range []struct {
		line, column int
		want         string
	}{
		{6, 21, "Beacon|null"}, {7, 28, "Beacon"}, {8, 25, "Beacon|null"},
	} {
		target, ok := InferHoverTargetAtPosition(nodes, tc.line, tc.column, "beacon", ctx)
		if !ok || target.Type != tc.want {
			t.Fatalf("line %d: got %#v, %t; want %s", tc.line, target, ok, tc.want)
		}
	}
}

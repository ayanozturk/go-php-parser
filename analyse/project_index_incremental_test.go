package analyse

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func parseProjectSources(t *testing.T, sources map[string]string) map[string][]ast.Node {
	t.Helper()
	parsed := make(map[string][]ast.Node, len(sources))
	for filename, source := range sources {
		parsed[filename] = parsePHPForProjectIndex(t, source)
	}
	return parsed
}

func BenchmarkBuildProjectIndexIncremental(b *testing.B) {
	const fileCount = 1000
	sources := make(map[string]string, fileCount)
	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("src/Class%04d.php", i)
		sources[filename] = fmt.Sprintf("<?php\nclass Class%04d { public function value(): string {} }\n", i)
	}
	parsed := make(map[string][]ast.Node, len(sources))
	for filename, source := range sources {
		parsed[filename] = parser.New(lexer.New(source), false).Parse()
	}
	previous := BuildProjectIndex(parsed)
	updated := make(map[string][]ast.Node, len(parsed))
	for filename, nodes := range parsed {
		updated[filename] = nodes
	}
	const changedFile = "src/Class0500.php"
	updated[changedFile] = parser.New(lexer.New("<?php\nclass Class0500 { public function value(int $input): string {} }\n"), false).Parse()
	bodyOnly := make(map[string][]ast.Node, len(parsed))
	for filename, nodes := range parsed {
		bodyOnly[filename] = nodes
	}
	bodyOnly[changedFile] = parser.New(lexer.New("<?php\nclass Class0500 { public function value(): string { return \"updated\"; } }\n"), false).Parse()

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = BuildProjectIndex(updated)
		}
	})
	b.Run("one-file-incremental", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = BuildProjectIndexIncremental(previous, updated, []string{changedFile})
		}
	})
	b.Run("one-file-body-only", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = BuildProjectIndexIncremental(previous, bodyOnly, []string{changedFile})
		}
	})
}

func assertProjectIndexEquivalent(t *testing.T, got, want *ProjectIndex) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incremental project index differs from fresh build:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildProjectIndexIncrementalMatchesFreshBuildAfterChangedFile(t *testing.T) {
	initialSources := map[string]string{
		"src/Base.php": `<?php
class Base
{
    public function label(): string {}
}
`,
		"src/Child.php": `<?php
class Child extends Base {}
`,
	}
	updatedSources := map[string]string{
		"src/Base.php": `<?php
class Base
{
    public function label(): string {}
    public function version(): int {}
}
`,
		"src/Child.php": initialSources["src/Child.php"],
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	got, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{"src/Base.php"})
	if !semanticChanged {
		t.Fatal("expected exported symbols to change after adding a public method")
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))

	method, ok := got.ResolveMethod("Child", "version")
	if !ok || method.ReturnType != "int" {
		t.Fatalf("expected inherited changed method in incremental index, got %#v, %v", method, ok)
	}
}

func TestBuildProjectIndexIncrementalAddsAndRemovesFiles(t *testing.T) {
	initialSources := map[string]string{
		"a.php": "<?php\nclass Alpha {}\n",
		"b.php": "<?php\nclass Beta {}\n",
	}
	updatedSources := map[string]string{
		"a.php": initialSources["a.php"],
		"c.php": "<?php\nclass Gamma {}\n",
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	got, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{"b.php", "c.php"})
	if !semanticChanged {
		t.Fatal("expected add/remove file changes to alter exported symbols")
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
	if got.ClassExists("Beta") {
		t.Fatal("expected removed file's class to be absent")
	}
	if !got.ClassExists("Gamma") {
		t.Fatal("expected added file's class to be indexed")
	}
}

func TestBuildProjectIndexIncrementalIgnoresBodyAndPositionChangesSemantically(t *testing.T) {
	const filename = "src/Widget.php"
	baseSource := `<?php
class Widget
{
    public function label(): string
    {
        return "base";
    }
}
`
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "body-only",
			source: `<?php
class Widget
{
    public function label(): string
    {
        return "updated body";
    }
}
`,
		},
		{
			name: "position-only",
			source: `<?php

// Keep the declaration semantics unchanged while moving its source span.
class Widget
{
    public function label(): string
    {
        return "base";
    }
}
`,
		},
	}

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: baseSource}))
	previousClass, previousClassOK := previous.ResolveClass("Widget")
	previousMethod, previousMethodOK := previous.ResolveMethod("Widget", "label")
	if !previousClassOK || !previousMethodOK {
		t.Fatalf("expected baseline declarations, got class=%v method=%v", previousClassOK, previousMethodOK)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updatedParsed := parseProjectSources(t, map[string]string{filename: tc.source})
			got, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{filename})
			if semanticChanged {
				t.Fatal("expected body/position-only edit to preserve exported semantics")
			}
			fresh := BuildProjectIndex(updatedParsed)
			assertProjectIndexEquivalent(t, got, fresh)

			gotClass, gotClassOK := got.ResolveClass("Widget")
			freshClass, freshClassOK := fresh.ResolveClass("Widget")
			gotMethod, gotMethodOK := got.ResolveMethod("Widget", "label")
			freshMethod, freshMethodOK := fresh.ResolveMethod("Widget", "label")
			if !gotClassOK || !freshClassOK || !gotMethodOK || !freshMethodOK {
				t.Fatalf("expected updated declarations, got class=%v/%v method=%v/%v", gotClassOK, freshClassOK, gotMethodOK, freshMethodOK)
			}
			if gotClass.Declaration != freshClass.Declaration || gotMethod.Declaration != freshMethod.Declaration {
				t.Fatalf("expected declaration locations to match fresh build, got class=%#v method=%#v", gotClass.Declaration, gotMethod.Declaration)
			}
			if gotClass.Declaration == previousClass.Declaration && gotMethod.Declaration == previousMethod.Declaration {
				t.Fatal("expected changed source to return updated declaration locations")
			}
		})
	}
}

func TestBuildProjectIndexIncrementalReportsExportedMethodSignatureChange(t *testing.T) {
	const filename = "src/Formatter.php"
	initialSource := `<?php
class Formatter
{
    public function format(int $value): string {}
}
`
	updatedSource := `<?php
class Formatter
{
    public function format(string $value): string {}
}
`

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: initialSource}))
	updatedParsed := parseProjectSources(t, map[string]string{filename: updatedSource})
	got, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{filename})
	if !semanticChanged {
		t.Fatal("expected exported method parameter change to be semantic")
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
	method, ok := got.ResolveMethod("Formatter", "format")
	if !ok || len(method.Params) != 1 || method.Params[0].Type != "string" {
		t.Fatalf("expected updated exported method signature, got %#v, %v", method, ok)
	}
}

func TestBuildProjectIndexIncrementalPreservesDuplicateOrdering(t *testing.T) {
	initialSources := map[string]string{
		"a.php": `<?php
class Duplicate
{
    public function origin(): string { return "a"; }
}
`,
		"m.php": `<?php
class Duplicate
{
    public function origin(): string { return "m"; }
}
`,
		"z.php": `<?php
class Duplicate
{
    public function origin(): string { return "z"; }
}
`,
	}
	updatedSources := map[string]string{
		"a.php": initialSources["a.php"],
		"m.php": `<?php
class Duplicate
{
    public function origin(): string { return "middle"; }
}
`,
		"z.php": initialSources["z.php"],
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	got, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{"m.php"})
	if semanticChanged {
		t.Fatal("expected duplicate body-only edit to preserve exported semantics")
	}
	fresh := BuildProjectIndex(updatedParsed)
	assertProjectIndexEquivalent(t, got, fresh)
	if !reflect.DeepEqual(got.Duplicates, fresh.Duplicates) {
		t.Fatalf("expected duplicate ordering to match fresh build, got %#v versus %#v", got.Duplicates, fresh.Duplicates)
	}
	if len(got.Duplicates) != 2 || got.Duplicates[0].File != "m.php" || got.Duplicates[1].File != "z.php" {
		t.Fatalf("expected sorted duplicate ordering [m.php, z.php], got %#v", got.Duplicates)
	}
}

func TestBuildProjectIndexIncrementalLeavesPreviousIndexConcurrentlyReadable(t *testing.T) {
	const filename = "src/Stable.php"
	initialSource := `<?php
class Stable
{
    public function value(): string { return "before"; }
}
`
	updatedSource := `<?php
class Stable
{
    public function value(): string { return "after"; }
}
`
	initialParsed := parseProjectSources(t, map[string]string{filename: initialSource})
	previous := BuildProjectIndex(initialParsed)
	baseline := BuildProjectIndex(initialParsed)
	updatedParsed := parseProjectSources(t, map[string]string{filename: updatedSource})

	readErrors := make(chan string, 1)
	reportReadError := func(message string) {
		select {
		case readErrors <- message:
		default:
		}
	}
	var readers sync.WaitGroup
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for j := 0; j < 100; j++ {
				class, ok := previous.ResolveClass("Stable")
				if !ok || class.Declaration.File != filename {
					reportReadError("previous class resolution changed during incremental build")
					return
				}
				method, ok := previous.ResolveMethod("Stable", "value")
				if !ok || method.ReturnType != "string" {
					reportReadError("previous method resolution changed during incremental build")
					return
				}
				if len(previous.MethodsDeclaredBy("Stable")) != 1 {
					reportReadError("previous declared-method view changed during incremental build")
					return
				}
			}
		}()
	}

	if _, semanticChanged := BuildProjectIndexIncremental(previous, updatedParsed, []string{filename}); semanticChanged {
		readers.Wait()
		t.Fatal("expected changed method body to preserve exported semantics")
	}
	readers.Wait()
	select {
	case message := <-readErrors:
		t.Fatal(message)
	default:
	}
	assertProjectIndexEquivalent(t, previous, baseline)
}

func TestBuildProjectIndexIncrementalWithChangesIgnoresBodyAndPositionEdits(t *testing.T) {
	const filename = "src/Widget.php"
	baseSource := `<?php
class Widget
{
    public function label(): string
    {
        return "base";
    }
}
`
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "body-only",
			source: `<?php
class Widget
{
    public function label(): string
    {
        return "updated body";
    }
}
`,
		},
		{
			name: "position-only",
			source: `<?php

// Keep declaration semantics unchanged while moving its source span.
class Widget
{
    public function label(): string
    {
        return "base";
    }
}
`,
		},
	}

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: baseSource}))
	previousClass, previousClassOK := previous.ResolveClass("Widget")
	previousMethod, previousMethodOK := previous.ResolveMethod("Widget", "label")
	if !previousClassOK || !previousMethodOK {
		t.Fatalf("expected baseline declarations, got class=%v method=%v", previousClassOK, previousMethodOK)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updatedParsed := parseProjectSources(t, map[string]string{filename: tc.source})
			got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{filename})
			if !changes.Complete {
				t.Fatal("expected complete change metadata when source metadata is available")
			}
			if changes.SemanticChanged() {
				t.Fatalf("expected body/position-only edit to preserve exported semantics, got %#v", changes)
			}
			if len(changes.Symbols) != 0 || len(changes.DependencyNames) != 0 {
				t.Fatalf("expected no exported symbols or dependency names, got %#v", changes)
			}

			fresh := BuildProjectIndex(updatedParsed)
			assertProjectIndexEquivalent(t, got, fresh)
			gotClass, gotClassOK := got.ResolveClass("Widget")
			freshClass, freshClassOK := fresh.ResolveClass("Widget")
			gotMethod, gotMethodOK := got.ResolveMethod("Widget", "label")
			freshMethod, freshMethodOK := fresh.ResolveMethod("Widget", "label")
			if !gotClassOK || !freshClassOK || !gotMethodOK || !freshMethodOK {
				t.Fatalf("expected updated declarations, got class=%v/%v method=%v/%v", gotClassOK, freshClassOK, gotMethodOK, freshMethodOK)
			}
			if gotClass.Declaration != freshClass.Declaration || gotMethod.Declaration != freshMethod.Declaration {
				t.Fatalf("expected declaration locations to match fresh build, got class=%#v method=%#v", gotClass.Declaration, gotMethod.Declaration)
			}
			if gotClass.Declaration == previousClass.Declaration && gotMethod.Declaration == previousMethod.Declaration {
				t.Fatal("expected changed source to return updated declaration locations")
			}
		})
	}
}

func TestBuildProjectIndexIncrementalWithChangesReportsStableMethodIdentity(t *testing.T) {
	const filename = "src/Formatter.php"
	initialSource := `<?php
class Formatter
{
    public function format(int $value): string {}
}
`
	updatedSource := `<?php
class Formatter
{
    public function format(string $value): string {}
}
`

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: initialSource}))
	previousMethod, ok := previous.ResolveMethod("Formatter", "format")
	if !ok {
		t.Fatal("expected baseline method")
	}
	updatedParsed := parseProjectSources(t, map[string]string{filename: updatedSource})
	got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{filename})
	if !changes.Complete || !changes.SemanticChanged() {
		t.Fatalf("expected complete semantic method change, got %#v", changes)
	}
	if len(changes.Symbols) != 1 {
		t.Fatalf("expected one method change, got %#v", changes.Symbols)
	}
	change := changes.Symbols[0]
	if change.ID != previousMethod.ID || change.Kind != "method" || change.Owner != "Formatter" || change.Name != "format" {
		t.Fatalf("expected stable method identity, got %#v; previous method=%#v", change, previousMethod)
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
}

func TestBuildProjectIndexIncrementalWithChangesReportsFunctionAndConstantAddRemove(t *testing.T) {
	const filename = "src/Definitions.php"
	initialSource := `<?php
function oldHelper(): int { return 1; }
const OLD_FLAG = 1;
class Settings
{
    public const OLD_MODE = "old";
}
`
	updatedSource := `<?php
function newHelper(): int { return 2; }
const NEW_FLAG = 2;
class Settings
{
    public const NEW_MODE = "new";
}
`

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: initialSource}))
	updatedParsed := parseProjectSources(t, map[string]string{filename: updatedSource})
	got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{filename})
	if !changes.Complete || !changes.SemanticChanged() {
		t.Fatalf("expected complete semantic add/remove changes, got %#v", changes)
	}
	for _, expected := range []ExportedSymbolChange{
		{ID: stableSymbolID("function", "", "oldHelper"), Kind: "function", Name: "oldHelper"},
		{ID: stableSymbolID("function", "", "newHelper"), Kind: "function", Name: "newHelper"},
		{ID: stableSymbolID("constant", "", "old_flag"), Kind: "constant", Name: "old_flag"},
		{ID: stableSymbolID("constant", "", "new_flag"), Kind: "constant", Name: "new_flag"},
		{ID: stableSymbolID("constant", "Settings", "OLD_MODE"), Kind: "class-constant", Owner: "Settings", Name: "OLD_MODE"},
		{ID: stableSymbolID("constant", "Settings", "NEW_MODE"), Kind: "class-constant", Owner: "Settings", Name: "NEW_MODE"},
	} {
		if !hasExportedSymbolChange(changes.Symbols, expected) {
			t.Fatalf("expected change %#v in %#v", expected, changes.Symbols)
		}
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
	if got.FunctionExists("oldHelper") || !got.FunctionExists("newHelper") {
		t.Fatal("expected function removal and addition in updated index")
	}
	if got.ConstantExists("OLD_FLAG") || !got.ConstantExists("NEW_FLAG") {
		t.Fatal("expected global constant removal and addition in updated index")
	}
	if _, ok := got.ResolveConstant("Settings", "OLD_MODE"); ok {
		t.Fatal("expected removed class constant to be absent")
	}
	if _, ok := got.ResolveConstant("Settings", "NEW_MODE"); !ok {
		t.Fatal("expected added class constant to be indexed")
	}
}

func TestBuildProjectIndexIncrementalWithChangesReportsClassRename(t *testing.T) {
	const filename = "src/Renamed.php"
	initialSource := `<?php
class OldName {}
`
	updatedSource := `<?php
class NewName {}
`

	previous := BuildProjectIndex(parseProjectSources(t, map[string]string{filename: initialSource}))
	updatedParsed := parseProjectSources(t, map[string]string{filename: updatedSource})
	got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{filename})
	if !changes.Complete || !changes.SemanticChanged() {
		t.Fatalf("expected complete semantic rename change, got %#v", changes)
	}
	for _, expected := range []ExportedSymbolChange{
		{ID: stableSymbolID("class", "", "OldName"), Kind: "class", Name: "OldName"},
		{ID: stableSymbolID("class", "", "NewName"), Kind: "class", Name: "NewName"},
	} {
		if !hasExportedSymbolChange(changes.Symbols, expected) {
			t.Fatalf("expected class rename change %#v in %#v", expected, changes.Symbols)
		}
	}
	if got.ClassExists("OldName") || !got.ClassExists("NewName") {
		t.Fatal("expected old class to be removed and new class to be indexed")
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
}

func TestBuildProjectIndexIncrementalWithChangesIncludesTransitiveDescendants(t *testing.T) {
	initialSources := map[string]string{
		"src/Base.php": `<?php
class Base
{
    public function render(): string {}
}
`,
		"src/Child.php":      "<?php\nclass Child extends Base {}\n",
		"src/Grandchild.php": "<?php\nclass Grandchild extends Child {}\n",
	}
	updatedSources := map[string]string{
		"src/Base.php": `<?php
class Base
{
    public function render(): int {}
}
`,
		"src/Child.php":      initialSources["src/Child.php"],
		"src/Grandchild.php": initialSources["src/Grandchild.php"],
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{"src/Base.php"})
	if !changes.Complete || !changes.SemanticChanged() {
		t.Fatalf("expected complete semantic base-member change, got %#v", changes)
	}
	if want := []string{"Base", "Child", "Grandchild", "render"}; !reflect.DeepEqual(changes.DependencyNames, want) {
		t.Fatalf("expected sorted transitive dependency names %#v, got %#v", want, changes.DependencyNames)
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
}

func TestBuildProjectIndexIncrementalWithChangesOrderingIsDeterministic(t *testing.T) {
	initialSources := map[string]string{
		"b.php": "<?php\nclass Beta { public function render(int $value): string {} }\n",
		"a.php": "<?php\nfunction helper(): int { return 1; }\n",
	}
	updatedSources := map[string]string{
		"b.php": "<?php\nclass Beta { public function render(string $value): string {} }\n",
		"a.php": "<?php\nfunction helper(): string { return \"ready\"; }\n",
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	var want ProjectIndexChanges
	for i, changedFiles := range [][]string{{"b.php", "a.php"}, {"a.php", "b.php"}, {"b.php", "a.php", "b.php"}} {
		_, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, changedFiles)
		if i == 0 {
			want = changes
			continue
		}
		if !reflect.DeepEqual(changes, want) {
			t.Fatalf("expected deterministic change ordering for iteration %d, got %#v versus %#v", i, changes, want)
		}
	}
}

func TestBuildProjectIndexIncrementalWithChangesMissingSourceMetadataIsIncomplete(t *testing.T) {
	const filename = "src/Recovered.php"
	parsed := parseProjectSources(t, map[string]string{filename: "<?php\nclass Recovered {}\n"})

	got, changes := BuildProjectIndexIncrementalWithChanges(NewProjectIndex(), parsed, []string{filename})
	if changes.Complete {
		t.Fatal("expected missing prior source metadata to force incomplete change metadata")
	}
	if !changes.SemanticChanged() {
		t.Fatal("expected incomplete change metadata to require semantic invalidation")
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(parsed))
}

func TestBuildProjectIndexIncrementalWithChangesReportsFullRebuildForMissingMetadata(t *testing.T) {
	const filename = "src/Recovered.php"
	parsed := parseProjectSources(t, map[string]string{filename: "<?php\nclass Recovered {}\n"})

	for _, tc := range []struct {
		name     string
		previous *ProjectIndex
	}{
		{name: "nil previous", previous: nil},
		{name: "untracked previous metadata", previous: NewProjectIndex()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, changes := BuildProjectIndexIncrementalWithChanges(tc.previous, parsed, []string{filename})
			if changes.Complete {
				t.Fatalf("expected missing prior source metadata to be incomplete, got %#v", changes)
			}
			if !changes.FullRebuild {
				t.Fatalf("expected missing prior source metadata to force a full rebuild, got %#v", changes)
			}
			assertProjectIndexEquivalent(t, got, BuildProjectIndex(parsed))
		})
	}
}

func TestBuildProjectIndexIncrementalWithChangesDoesNotReportFullRebuildWithoutChanges(t *testing.T) {
	const filename = "src/Stable.php"
	parsed := parseProjectSources(t, map[string]string{filename: "<?php\nclass Stable {}\n"})
	previous := BuildProjectIndex(parsed)

	for _, tc := range []struct {
		name         string
		changedFiles []string
	}{
		{name: "no changed files"},
		{name: "unchanged file", changedFiles: []string{filename}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, changes := BuildProjectIndexIncrementalWithChanges(previous, parsed, tc.changedFiles)
			if !changes.Complete {
				t.Fatalf("expected complete change metadata, got %#v", changes)
			}
			if changes.FullRebuild {
				t.Fatalf("expected no full rebuild for %s, got %#v", tc.name, changes)
			}
			if changes.SemanticChanged() {
				t.Fatalf("expected unchanged sources to preserve exported semantics, got %#v", changes)
			}
			assertProjectIndexEquivalent(t, got, previous)
		})
	}
}

func TestBuildProjectIndexIncrementalWithChangesDoesNotReportFullRebuildForOrdinaryUpdate(t *testing.T) {
	const filename = "src/Widget.php"
	initialSources := map[string]string{
		filename:        "<?php\nclass Widget { public function label(): string {} }\n",
		"src/Other.php": "<?php\nclass Other {}\n",
	}
	updatedSources := map[string]string{
		filename:        "<?php\nclass Widget { public function label(int $value): string {} }\n",
		"src/Other.php": initialSources["src/Other.php"],
	}

	previous := BuildProjectIndex(parseProjectSources(t, initialSources))
	updatedParsed := parseProjectSources(t, updatedSources)
	got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{filename})
	if !changes.Complete || !changes.SemanticChanged() {
		t.Fatalf("expected complete semantic incremental update, got %#v", changes)
	}
	if changes.FullRebuild {
		t.Fatalf("expected ordinary single-file update to stay incremental, got %#v", changes)
	}
	assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
}

func TestBuildProjectIndexIncrementalWithChangesReportsFullRebuildForCollidingDefinitions(t *testing.T) {
	t.Run("existing duplicate touched", func(t *testing.T) {
		initialSources := map[string]string{
			"a.php": "<?php\nclass Shared {}\n",
			"b.php": "<?php\nclass Shared {}\n",
		}
		updatedSources := map[string]string{
			"a.php": initialSources["a.php"],
			"b.php": "<?php\nclass Shared { public function label(): string {} }\n",
		}

		previous := BuildProjectIndex(parseProjectSources(t, initialSources))
		updatedParsed := parseProjectSources(t, updatedSources)
		got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{"b.php"})
		if !changes.Complete || !changes.FullRebuild {
			t.Fatalf("expected duplicate definition update to use a full rebuild, got %#v", changes)
		}
		assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
	})

	t.Run("new definition collides", func(t *testing.T) {
		initialSources := map[string]string{
			"a.php": "<?php\nclass Shared {}\n",
		}
		updatedSources := map[string]string{
			"a.php": initialSources["a.php"],
			"b.php": "<?php\nclass Shared {}\n",
		}

		previous := BuildProjectIndex(parseProjectSources(t, initialSources))
		updatedParsed := parseProjectSources(t, updatedSources)
		got, changes := BuildProjectIndexIncrementalWithChanges(previous, updatedParsed, []string{"b.php"})
		if !changes.Complete || !changes.FullRebuild {
			t.Fatalf("expected colliding definition addition to use a full rebuild, got %#v", changes)
		}
		assertProjectIndexEquivalent(t, got, BuildProjectIndex(updatedParsed))
	})
}

func hasExportedSymbolChange(changes []ExportedSymbolChange, want ExportedSymbolChange) bool {
	for _, change := range changes {
		if change.ID == want.ID && change.Kind == want.Kind && change.Owner == want.Owner && change.Name == want.Name {
			return true
		}
	}
	return false
}

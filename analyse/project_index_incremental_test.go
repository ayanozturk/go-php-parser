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

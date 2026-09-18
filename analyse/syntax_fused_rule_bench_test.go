package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func BenchmarkEnsureSharedFileDiagnosticsFromCST(b *testing.B) {
	content := []byte(benchmarkFixtureSource)
	nodes, _ := syntax.ParseAST(content)
	project := BuildProjectIndex(map[string][]ast.Node{"bench.php": nodes})
	level := 6
	for i := 0; i < b.N; i++ {
		ctx := &AnalysisContext{Content: content, AnalysisLevel: &level, Resolver: project}
		ensureSharedFileDiagnosticsFromCST("bench.php", content, nodes, ctx)
	}
}

const benchmarkFixtureSource = `<?php
// a moderately complex fixture exercising class model, type refs, symbols,
// language checks, property-callable, empty-statement, method-visibility,
// throw-type, phpdoc, missing-types, and return-type checks all in one file
namespace App;

interface Greetable {
    public function greet(): string;
}

abstract class Base implements Greetable {
    /** @var string */
    protected $name;

    public function __construct(string $name) {
        $this->name = $name;
    }

    abstract public function greet(): string;

    protected function throwsSomething(): void {
        throw new \RuntimeException('nope');
    }
}

class Person extends Base {
    public function greet(): string {
        return "Hello, {$this->name}!";
    }

    public function process(array $items): array {
        $result = [];
        foreach ($items as $item) {
            if ($item) {
                $result[] = strtoupper($item);
            }
        }
        return $result;
    }
}
`
